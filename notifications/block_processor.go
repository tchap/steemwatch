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
	NextBlockNum       uint32     `bson:"nextBlockNum"`
	LastBlockTimestamp *time.Time `bson:"lastBlockTimestamp,omitempty"`
}

func (config *BlockProcessorConfig) Clone() *BlockProcessorConfig {
	clone := &BlockProcessorConfig{
		NextBlockNum: config.NextBlockNum,
	}
	if ts := config.LastBlockTimestamp; ts != nil {
		lastBlockTimestamp := *ts
		clone.LastBlockTimestamp = &lastBlockTimestamp
	}
	return clone
}

type BlockProcessor struct {
	client *rpc.Client
	db     *mgo.Database
	config *BlockProcessorConfig

	eventMiners         map[string][]EventMiner
	additionalNotifiers map[string]Notifier

	blockProcessingLock *sync.Mutex
	lastBlockCh         chan *database.Block

	t *tomb.Tomb
}

type Option func(*BlockProcessor)

func AddNotifier(id string, notifier Notifier) Option {
	return func(processor *BlockProcessor) {
		if processor.additionalNotifiers == nil {
			processor.additionalNotifiers = map[string]Notifier{
				id: notifier,
			}
		} else {
			processor.additionalNotifiers[id] = notifier
		}
	}
}

func New(client *rpc.Client, db *mgo.Database, opts ...Option) (*BlockProcessor, error) {
	// Ensure DB indexes exist.
	indexes := []struct {
		Key    string
		Sparse bool
	}{
		{"kind", false},
		{"accounts", true},
		{"witnesses", true},
		{"from", true},
		{"to", true},
		{"users", true},
		{"authorBlacklist", true},
		{"tags", true},
		{"authors", true},
		{"voters", true},
		{"parentAuthors", true},
		{"selectors.contentID", true},
	}

	for _, index := range indexes {
		log.Printf("Creating index for events.%v ...", index.Key)
		err := db.C("events").EnsureIndex(mgo.Index{
			Key:        []string{index.Key},
			Background: true,
			Sparse:     index.Sparse,
		})
		if err != nil {
			log.Printf("Failed creating index for events.%v: %v", index.Key, err)
		}
	}

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
		database.OpTypeAccountUpdate: []EventMiner{
			events.NewAccountUpdatedEventMiner(),
		},
		database.OpTypeAccountWitnessVote: []EventMiner{
			events.NewAccountWitnessVotedEventMiner(),
		},
		database.OpTypeTransfer: []EventMiner{
			events.NewTransferMadeEventMiner(),
		},
		database.OpTypeComment: []EventMiner{
			events.NewUserMentionedEventMiner(),
			events.NewStoryPublishedEventMiner(),
			events.NewCommentPublishedEventMiner(),
		},
		database.OpTypeVote: []EventMiner{
			events.NewStoryVotedEventMiner(),
			events.NewCommentVotedEventMiner(),
		},
		database.OpTypeCustomJSON: []EventMiner{
			events.NewUserFollowStatusChangedEventMiner(),
		},
	}

	// Create a new BlockProcessor instance.
	processor := &BlockProcessor{
		client:              client,
		db:                  db,
		config:              &config,
		eventMiners:         eventMiners,
		blockProcessingLock: new(sync.Mutex),
		lastBlockCh:         make(chan *database.Block, 1),
		t:                   new(tomb.Tomb),
	}

	// Apply the options.
	for _, opt := range opts {
		opt(processor)
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
				err = errors.Wrapf(err, "block %v: failed to get content: @%v/%v",
					block.Number, body.Author, body.Permlink)
			case *database.VoteOperation:
				content, err = processor.client.Database.GetContent(body.Author, body.Permlink)
				err = errors.Wrapf(err, "block %v: failed to get content: @%v/%v",
					block.Number, body.Author, body.Permlink)
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
				events, err := eventMiner.MineEvent(op, content)
				if err == nil {
					for _, event := range events {
						err = processor.handleEvent(event)
						if err != nil {
							break
						}
					}
				}
				if err != nil {
					return errors.Wrapf(err, "block %v: %v", block.Number, err.Error())
				}
			}
		}
	}

	processor.lastBlockCh <- block
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
	config := processor.config.Clone()
	updateConfig := func(lastProcessedBlock *database.Block) {
		config.NextBlockNum = lastProcessedBlock.Number + 1
		config.LastBlockTimestamp = lastProcessedBlock.Timestamp.Time
	}

	var timeoutCh <-chan time.Time
	resetTimeout := func() {
		timeoutCh = time.After(1 * time.Minute)
	}
	resetTimeout()

	for {
		select {
		// Store the last processed block number every time it is received.
		case block := <-processor.lastBlockCh:
			updateConfig(block)

		// Flush config every minute.
		case <-timeoutCh:
			if err := processor.flushConfig(config); err != nil {
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
			case block := <-processor.lastBlockCh:
				updateConfig(block)
			default:
			}

			// Flush the config at last.
			return processor.flushConfig(config)
		}
	}
}

func (processor *BlockProcessor) flushConfig(config *BlockProcessorConfig) error {
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
	case *events.AccountUpdated:
		return processor.HandleAccountUpdatedEvent(event)
	case *events.AccountWitnessVoted:
		return processor.HandleAccountWitnessVotedEvent(event)
	case *events.TransferMade:
		return processor.HandleTransferMadeEvent(event)
	case *events.UserMentioned:
		return processor.HandleUserMentionedEvent(event)
	case *events.UserFollowStatusChanged:
		return processor.HandleUserFollowStatusChangedEvent(event)
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

func (processor *BlockProcessor) HandleAccountUpdatedEvent(event *events.AccountUpdated) error {
	query := bson.M{
		"kind":     "account.updated",
		"accounts": event.Op.Account,
	}

	log.Println(query)

	var result struct {
		OwnerId bson.ObjectId `bson:"ownerId"`
	}
	iter := processor.db.C("events").Find(query).Iter()
	for iter.Next(&result) {
		processor.DispatchAccountUpdatedEvent(result.OwnerId.Hex(), event)
	}
	return errors.Wrap(iter.Err(), "failed get target users for account.updated")
}

func (processor *BlockProcessor) HandleAccountWitnessVotedEvent(event *events.AccountWitnessVoted) error {
	query := bson.M{
		"kind": "account.witness_voted",
		"$or": []interface{}{
			bson.M{
				"accounts": event.Op.Account,
			},
			bson.M{
				"witnesses": event.Op.Witness,
			},
		},
	}

	log.Println(query)

	var result struct {
		OwnerId bson.ObjectId `bson:"ownerId"`
	}
	iter := processor.db.C("events").Find(query).Iter()
	for iter.Next(&result) {
		processor.DispatchAccountWitnessVotedEvent(result.OwnerId.Hex(), event)
	}
	return errors.Wrap(iter.Err(), "failed get target users for account.witness_voted")
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

func (processor *BlockProcessor) HandleUserMentionedEvent(event *events.UserMentioned) error {
	query := bson.M{
		"kind":            "user.mentioned",
		"users":           event.User,
		"authorBlacklist": bson.M{"$ne": event.Content.Author},
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

func (processor *BlockProcessor) HandleUserFollowStatusChangedEvent(
	event *events.UserFollowStatusChanged,
) error {

	query := bson.M{
		"kind":  "user.follow_changed",
		"users": event.Op.Following,
	}

	log.Println(query)

	var result struct {
		OwnerId bson.ObjectId `bson:"ownerId"`
	}
	iter := processor.db.C("events").Find(query).Iter()
	for iter.Next(&result) {
		processor.DispatchUserFollowStatusChangedEvent(result.OwnerId.Hex(), event)
	}
	return errors.Wrap(iter.Err(), "failed get target users for user.follow_changed")
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

	return nil
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
			dispatcher, ok = processor.additionalNotifiers[id]
			if !ok {
				log.Printf("dispatcher not found: id=%v", id)
				continue
			}
		}

		if err := dispatch(dispatcher, notifier.Settings); err != nil {
			log.Printf("dispatcher %v failed: %+v", id, err)
		}
	}

	var settings bson.Raw
	for id, dispatcher := range processor.additionalNotifiers {
		if err := dispatch(dispatcher, settings); err != nil {
			log.Printf("dispatcher %v failed: %+v", id, err)
		}
	}

	return nil
}

func (processor *BlockProcessor) DispatchAccountUpdatedEvent(userId string, event *events.AccountUpdated) {
	processor.t.Go(func() error {
		return processor.dispatchEvent(userId, func(notifier Notifier, settings bson.Raw) error {
			return notifier.DispatchAccountUpdatedEvent(userId, settings, event)
		})
	})
}

func (processor *BlockProcessor) DispatchAccountWitnessVotedEvent(
	userId string,
	event *events.AccountWitnessVoted,
) {
	processor.t.Go(func() error {
		return processor.dispatchEvent(userId, func(notifier Notifier, settings bson.Raw) error {
			return notifier.DispatchAccountWitnessVotedEvent(userId, settings, event)
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

func (processor *BlockProcessor) DispatchUserMentionedEvent(userId string, event *events.UserMentioned) {
	processor.t.Go(func() error {
		return processor.dispatchEvent(userId, func(notifier Notifier, settings bson.Raw) error {
			return notifier.DispatchUserMentionedEvent(userId, settings, event)
		})
	})
}

func (processor *BlockProcessor) DispatchUserFollowStatusChangedEvent(
	userId string,
	event *events.UserFollowStatusChanged,
) {
	processor.t.Go(func() error {
		return processor.dispatchEvent(userId, func(notifier Notifier, settings bson.Raw) error {
			return notifier.DispatchUserFollowStatusChangedEvent(userId, settings, event)
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
