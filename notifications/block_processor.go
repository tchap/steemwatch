package notifications

import (
	"log"
	"sync"
	"time"

	"github.com/tchap/steemwatch/notifications/events"

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

	eventMiners []EventMiner

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
	eventMiners := []EventMiner{
		events.NewUserMentionedEventMiner(),
		events.NewStoryPublishedEventMiner(),
		events.NewStoryVotedEventMiner(),
		events.NewCommentPublishedEventMiner(),
		events.NewCommentVotedEventMiner(),
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
			// Fetch the associated content.
			var (
				content *database.Content
				err     error
			)
			switch op := op.Body.(type) {
			case *database.CommentOperation:
				content, err = processor.client.Database.GetContent(op.Author, op.Permlink)
			case *database.VoteOperation:
				content, err = processor.client.Database.GetContent(op.Author, op.Permlink)
			default:
				continue
			}
			if err != nil {
				return err
			}

			// Mine events.
			for _, eventMiner := range processor.eventMiners {
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
	return errors.Wrap(iter.Err(), "failed get target users for comment.published")
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
