package notifications

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tchap/steemwatch/notifications/events"
	"github.com/tchap/steemwatch/server/routes/api/events/descendantpublished"

	"github.com/go-steem/rpc"
	"github.com/go-steem/rpc/apis/database"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/tomb.v2"
)

type BlockProcessorConfig struct {
	NextBlockNum uint32 `bson:"nextBlockNum"`
}

type BlockProcessor struct {
	client *rpc.Client
	db     *mgo.Database
	config *BlockProcessorConfig

	eventMiners map[string][]EventMiner

	blockProcessingLock *sync.Mutex
	lastBlockNumCh      chan uint32

	t *tomb.Tomb
}

func New(client *rpc.Client, db *mgo.Database) (*BlockProcessor, error) {
	// Load config from the database.
	var config BlockProcessorConfig
	if err := db.C("configuration").FindId("BlockProcessor").One(&config); err != nil {
		if err == mgo.ErrNotFound {
			// We need to get the last irreversible block number
			// to know where to start processing blocks from initially.
			props, err := client.Database.GetDynamicGlobalProperties()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get steemd dynamic global properties")
			}
			config.NextBlockNum = props.LastIrreversibleBlockNum
		} else {
			return nil, errors.Wrap(err, "failed to load BlockProcessor configuration")
		}
	}

	// Instantiate event miners.
	eventMiners := map[string][]EventMiner{
		database.OpTypeComment: []EventMiner{
			events.NewUserMentionedEventMiner(),
			events.NewStoryPublishedEventMiner(),
			events.NewCommentPublishedEventMiner(),
		},
		database.OpTypeVote: []EventMiner{
			events.NewStoryVotedEventMiner(),
			events.NewCommentVotedEventMiner(),
		},
		database.OpTypeTransfer: []EventMiner{
			events.NewTransferMadeEventMiner(),
		},
	}

	// Create a new BlockProcessor instance.
	processor := &BlockProcessor{
		client:              client,
		db:                  db,
		config:              &config,
		eventMiners:         eventMiners,
		blockProcessingLock: new(sync.Mutex),
		lastBlockNumCh:      make(chan uint32, 1),
		t:                   new(tomb.Tomb),
	}

	// Start the config flusher.
	processor.t.Go(processor.configFlusher)

	// Return the new BlockProcessor.
	return processor, nil
}

func (processor *BlockProcessor) BlockRange() (from, to uint32) {
	return processor.config.NextBlockNum, 0
}

func (processor *BlockProcessor) ProcessBlock(block *database.Block) error {
	processor.blockProcessingLock.Lock()
	defer processor.blockProcessingLock.Unlock()

	for _, tx := range block.Transactions {
		for _, op := range tx.Operations {
			// Fetch the associated content in case
			// this is a content-related operation.
			var (
				content *database.Content
				err     error
			)
			switch body := op.Body.(type) {
			case *database.CommentOperation:
				content, err = processor.client.Database.GetContent(body.Author, body.Permlink)
			case *database.VoteOperation:
				content, err = processor.client.Database.GetContent(body.Author, body.Permlink)
			}
			if err != nil {
				return err
			}

			// Get miners associated with the given operation.
			miners, ok := processor.eventMiners[op.Type]
			if !ok {
				continue
			}
			// Mine events and handle them.
			for _, eventMiner := range miners {
				for _, event := range eventMiner.MineEvent(op, content) {
					if err := processor.handleEvent(event); err != nil {
						return err
					}
				}
			}
		}
	}

	processor.lastBlockNumCh <- block.Number
	return nil
}

func (processor *BlockProcessor) Finalize() error {
	processor.t.Kill(nil)

	for _, notifier := range availableNotifiers {
		notifier.Close()
	}

	return processor.t.Wait()
}

func (processor *BlockProcessor) configFlusher() error {
	lastProcessedBlockNum := processor.config.NextBlockNum

	var timeoutCh <-chan time.Time
	resetTimeout := func() {
		timeoutCh = time.After(1 * time.Minute)
	}
	resetTimeout()

	for {
		select {
		// Store the last processed block number every time it is received.
		case blockNum := <-processor.lastBlockNumCh:
			lastProcessedBlockNum = blockNum

		// Flush config every minute.
		case <-timeoutCh:
			if err := processor.flushConfig(lastProcessedBlockNum); err != nil {
				return err
			}
			resetTimeout()

		// Flush the config on exit.
		case <-processor.t.Dying():

			// Make sure ProcessBlock() is not running any more.
			processor.blockProcessingLock.Lock()
			defer processor.blockProcessingLock.Unlock()

			// Check whether there is any additional block number in the queue.
			select {
			case blockNum := <-processor.lastBlockNumCh:
				lastProcessedBlockNum = blockNum
			default:
			}

			// Flush the config at last.
			return processor.flushConfig(lastProcessedBlockNum)
		}
	}
}

func (processor *BlockProcessor) flushConfig(lastProcessedBlockNum uint32) error {
	config := &BlockProcessorConfig{
		NextBlockNum: lastProcessedBlockNum + 1,
	}
	if _, err := processor.db.C("configuration").UpsertId("BlockProcessor", config); err != nil {
		return errors.Wrapf(err, "failed to store BlockProcessor configuration: %+v", config)
	}

	return nil
}

//==============================================================================
// Event handling
//==============================================================================

func (processor *BlockProcessor) handleEvent(event interface{}) error {
	switch event := event.(type) {
	case *events.UserMentioned:
		return processor.HandleUserMentionedEvent(event)
	case *events.TransferMade:
		return processor.HandleTransferMadeEvent(event)
	case *events.StoryPublished:
		return processor.HandleStoryPublishedEvent(event)
	case *events.StoryVoted:
		return processor.HandleStoryVotedEvent(event)
	case *events.CommentPublished:
		return processor.HandleCommentPublishedEvent(event)
	case *events.CommentVoted:
		return processor.HandleCommentVotedEvent(event)
	default:
		return errors.Errorf("unknown event type: %T", event)
	}
}

func (processor *BlockProcessor) HandleUserMentionedEvent(event *events.UserMentioned) error {
	query := bson.M{
		"kind":  "user.mentioned",
		"users": event.User,
	}

	log.Println(query)

	var result struct {
		OwnerId bson.ObjectId `bson:"ownerId"`
	}
	iter := processor.db.C("events").Find(query).Iter()
	for iter.Next(&result) {
		processor.DispatchUserMentionedEvent(result.OwnerId.Hex(), event)
	}
	return errors.Wrap(iter.Err(), "failed get target users for user.mentioned")
}

func (processor *BlockProcessor) HandleTransferMadeEvent(event *events.TransferMade) error {
	query := bson.M{
		"kind": "transfer.made",
		"$or": []interface{}{
			bson.M{
				"from": event.Op.From,
			},
			bson.M{
				"to": event.Op.To,
			},
		},
	}

	log.Println(query)

	var result struct {
		OwnerId bson.ObjectId `bson:"ownerId"`
	}
	iter := processor.db.C("events").Find(query).Iter()
	for iter.Next(&result) {
		processor.DispatchTransferMadeEvent(result.OwnerId.Hex(), event)
	}
	return errors.Wrap(iter.Err(), "failed get target users for transfer.made")
}

func (processor *BlockProcessor) HandleStoryPublishedEvent(event *events.StoryPublished) error {
	query := bson.M{
		"kind": "story.published",
		"$or": []interface{}{
			bson.M{
				"authors": event.Content.Author,
			},
			bson.M{
				"tags": bson.M{
					"$in": event.Content.JsonMetadata.Tags,
				},
			},
		},
	}

	log.Println(query)

	var result struct {
		OwnerId bson.ObjectId `bson:"ownerId"`
	}
	iter := processor.db.C("events").Find(query).Iter()
	for iter.Next(&result) {
		processor.DispatchStoryPublishedEvent(result.OwnerId.Hex(), event)
	}
	return errors.Wrap(iter.Err(), "failed get target users for story.published")
}

func (processor *BlockProcessor) HandleStoryVotedEvent(event *events.StoryVoted) error {
	query := bson.M{
		"kind": "story.voted",
		"$or": []interface{}{
			bson.M{
				"authors": event.Content.Author,
			},
			bson.M{
				"voters": event.Op.Voter,
			},
		},
	}

	log.Println(query)

	var result struct {
		OwnerId bson.ObjectId `bson:"ownerId"`
	}
	iter := processor.db.C("events").Find(query).Iter()
	for iter.Next(&result) {
		processor.DispatchStoryVotedEvent(result.OwnerId.Hex(), event)
	}
	return errors.Wrap(iter.Err(), "failed get target users for story.voted")
}

func (processor *BlockProcessor) HandleCommentPublishedEvent(event *events.CommentPublished) error {
	query := bson.M{
		"kind": "comment.published",
		"$or": []interface{}{
			bson.M{
				"authors": event.Content.Author,
			},
			bson.M{
				"parentAuthors": event.Content.ParentAuthor,
			},
		},
	}

	log.Println(query)

	var result struct {
		OwnerId bson.ObjectId `bson:"ownerId"`
	}
	iter := processor.db.C("events").Find(query).Iter()
	for iter.Next(&result) {
		processor.DispatchCommentPublishedEvent(result.OwnerId.Hex(), event)
	}
	if err := iter.Err(); err != nil {
		return errors.Wrap(err, "failed get target users for comment.published")
	}

	// We also need to check for descendant.published here.
	return processor.handleDescendantPublished(event)
}

func (processor *BlockProcessor) handleDescendantPublished(event *events.CommentPublished) error {
	var (
		parentAuthor        = event.Content.ParentAuthor
		parentPermlink      = event.Content.ParentPermlink
		distance       uint = 1
	)

	for {
		contentID := fmt.Sprintf("@%v/%v", parentAuthor, parentPermlink)

		query := bson.M{
			"kind":                "descendant.published",
			"selectors.contentID": contentID,
		}

		selector := bson.M{
			"ownerId": 1,
			"selectors": bson.M{
				"$elemMatch": bson.M{
					"contentID": contentID,
				},
			},
		}

		log.Println(query)

		var result descendantpublished.Document
		iter := processor.db.C("events").Find(query).Select(selector).Iter()
		for iter.Next(&result) {
			// It should be perfectly fine to just index the first item in the array
			// since $elemMatch should return just the first match and the array
			// should never be empty, otherwise the document would not be returned.
			selector := result.Selectors[0]
			switch selector.Mode {
			case descendantpublished.SelectorModeAny:
				processor.DispatchCommentPublishedEvent(result.OwnerID.Hex(), event)

			case descendantpublished.SelectorModeDepthLimit:
				if distance <= selector.DepthLimit {
					processor.DispatchCommentPublishedEvent(result.OwnerID.Hex(), event)
				}

			default:
				panic("unreachable code reached")
			}
		}
		if err := iter.Err(); err != nil {
			return errors.Wrap(iter.Err(), "failed get target users for descendant.published")
		}

		parent, err := processor.client.Database.GetContent(parentAuthor, parentPermlink)
		if err != nil {
			return errors.Wrap(err, "failed to call get_content over RPC")
		}
		if parent.IsStory() {
			return nil
		}

		parentAuthor = parent.ParentAuthor
		parentPermlink = parent.ParentPermlink
		distance++
	}
}

func (processor *BlockProcessor) HandleCommentVotedEvent(event *events.CommentVoted) error {
	query := bson.M{
		"kind": "comment.voted",
		"$or": []interface{}{
			bson.M{
				"authors": event.Content.Author,
			},
			bson.M{
				"voters": event.Op.Voter,
			},
		},
	}

	log.Println(query)

	var result struct {
		OwnerId bson.ObjectId `bson:"ownerId"`
	}
	iter := processor.db.C("events").Find(query).Iter()
	for iter.Next(&result) {
		processor.DispatchCommentVotedEvent(result.OwnerId.Hex(), event)
	}
	return errors.Wrap(iter.Err(), "failed get target users for comment.voted")
}

//==============================================================================
// Notification dispatch
//==============================================================================

type NotifierDoc struct {
	NotifierId string   `bson:"notifierId"`
	Settings   bson.Raw `bson:"settings"`
}

func (processor *BlockProcessor) getActiveNotifiersForUser(userId string) ([]*NotifierDoc, error) {
	query := bson.M{
		"ownerId": bson.ObjectIdHex(userId),
		"enabled": true,
	}

	var result []*NotifierDoc
	if err := processor.db.C("notifiers").Find(query).All(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (processor *BlockProcessor) dispatchEvent(userId string, dispatch func(Notifier, bson.Raw) error) error {
	notifiers, err := processor.getActiveNotifiersForUser(userId)
	if err != nil {
		return errors.Wrapf(err, "failed to get notifiers for user %v", userId)
	}

	for _, notifier := range notifiers {
		id := notifier.NotifierId

		dispatcher, ok := availableNotifiers[id]
		if !ok {
			log.Printf("dispatcher not found: id=%v", id)
			continue
		}

		if err := dispatch(dispatcher, notifier.Settings); err != nil {
			log.Printf("dispatcher %v failed: %+v", id, err)
		}
	}

	return nil
}

func (processor *BlockProcessor) DispatchUserMentionedEvent(userId string, event *events.UserMentioned) {
	processor.t.Go(func() error {
		return processor.dispatchEvent(userId, func(notifier Notifier, settings bson.Raw) error {
			return notifier.DispatchUserMentionedEvent(userId, settings, event)
		})
	})
}

func (processor *BlockProcessor) DispatchTransferMadeEvent(userId string, event *events.TransferMade) {
	processor.t.Go(func() error {
		return processor.dispatchEvent(userId, func(notifier Notifier, settings bson.Raw) error {
			return notifier.DispatchTransferMadeEvent(userId, settings, event)
		})
	})
}

func (processor *BlockProcessor) DispatchStoryPublishedEvent(userId string, event *events.StoryPublished) {
	processor.t.Go(func() error {
		return processor.dispatchEvent(userId, func(notifier Notifier, settings bson.Raw) error {
			return notifier.DispatchStoryPublishedEvent(userId, settings, event)
		})
	})
}

func (processor *BlockProcessor) DispatchStoryVotedEvent(userId string, event *events.StoryVoted) {
	processor.t.Go(func() error {
		return processor.dispatchEvent(userId, func(notifier Notifier, settings bson.Raw) error {
			return notifier.DispatchStoryVotedEvent(userId, settings, event)
		})
	})
}

func (processor *BlockProcessor) DispatchCommentPublishedEvent(userId string, event *events.CommentPublished) {
	processor.t.Go(func() error {
		return processor.dispatchEvent(userId, func(notifier Notifier, settings bson.Raw) error {
			return notifier.DispatchCommentPublishedEvent(userId, settings, event)
		})
	})
}

func (processor *BlockProcessor) DispatchCommentVotedEvent(userId string, event *events.CommentVoted) {
	processor.t.Go(func() error {
		return processor.dispatchEvent(userId, func(notifier Notifier, settings bson.Raw) error {
			return notifier.DispatchCommentVotedEvent(userId, settings, event)
		})
	})
}
