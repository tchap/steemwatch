package notifications

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tchap/steemwatch/notifications/events"
	"github.com/tchap/steemwatch/server/routes/api/events/descendantpublished"
	"github.com/tchap/steemwatch/server/users"

	"github.com/cznic/ql"
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

	mem *ql.DB

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

func New(
	client *rpc.Client,
	db *mgo.Database,
	userChangedCh <-chan *users.User,
	opts ...Option,
) (*BlockProcessor, error) {
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

	// Build in-memory indexes.
	if err := processor.buildDB(); err != nil {
		return nil, err
	}

	// Start the config flusher.
	processor.t.Go(processor.configFlusher)

	// Start the index reloader.
	processor.t.Go(func() error {
		return processor.indexReloader(userChangedCh)
	})

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
// Metadata in-memory database
//==============================================================================

func (processor *BlockProcessor) buildDB() error {
	start := time.Now()

	mem, err := ql.OpenMem()
	if err != nil {
		return errors.Wrap(err, "failed to open a QL in-memory database")
	}

	tctx := ql.NewRWCtx()

	// account.updated
	query := `
	BEGIN TRANSACTION;
		CREATE TABLE AccountUpdated (
			UserID  string NOT NULL,
			Account string NOT NULL
		);
		CREATE INDEX AccountUpdatedUserID  ON AccountUpdated (UserID);
		CREATE INDEX AccountUpdatedAccount ON AccountUpdated (Account);
	COMMIT;
	`

	if _, _, err := mem.Run(tctx, query); err != nil {
		mem.Close()
		return errors.Wrap(err, "failed to create a table")
	}

	// account.witness_voted
	query = `
	BEGIN TRANSACTION;
		CREATE TABLE AccountWitnessVoted (
			UserID  string NOT NULL,
			Account string,
			Witness string
		);
		CREATE INDEX AccountWitnessVotedUserID  ON AccountWitnessVoted (UserID);
		CREATE INDEX AccountWitnessVotedAccount ON AccountWitnessVoted (Account);
		CREATE INDEX AccountWitnessVotedWitness ON AccountWitnessVoted (Witness);
	COMMIT;
	`

	if _, _, err := mem.Run(tctx, query); err != nil {
		mem.Close()
		return errors.Wrap(err, "failed to create a table")
	}

	// transfer.made
	query = `
	BEGIN TRANSACTION;
		CREATE TABLE TransferMade (
			UserID      string NOT NULL,
			FromAccount string,
			ToAccount   string
		);
		CREATE INDEX TransferMadeUserID      ON TransferMade (UserID);
		CREATE INDEX TransferMadeFromAccount ON TransferMade (FromAccount);
		CREATE INDEX TransferMadeToAccount   ON TransferMade (ToAccount);
	COMMIT;
	`

	if _, _, err := mem.Run(tctx, query); err != nil {
		mem.Close()
		return errors.Wrap(err, "failed to create a table")
	}

	// user.mentioned
	query = `
	BEGIN TRANSACTION;
		CREATE TABLE UserMentioned (
			UserID   string NOT NULL,
			User     string,
			BLAuthor string
		);
		CREATE INDEX UserMentionedUserID   ON UserMentioned (UserID);
		CREATE INDEX UserMentionedUser     ON UserMentioned (User);
		CREATE INDEX UserMentionedBLAuthor ON UserMentioned (BLAuthor);
	COMMIT;
	`

	if _, _, err := mem.Run(tctx, query); err != nil {
		mem.Close()
		return errors.Wrap(err, "failed to create a table")
	}

	// user.follow_changed
	query = `
	BEGIN TRANSACTION;
		CREATE TABLE UserFollowChanged (
			UserID string NOT NULL,
			User   string NOT NULL
		);
		CREATE INDEX UserFollowChangedUserID ON UserFollowChanged (UserID);
		CREATE INDEX UserFollowChangedUser   ON UserFollowChanged (User);
	COMMIT;
	`

	if _, _, err := mem.Run(tctx, query); err != nil {
		mem.Close()
		return errors.Wrap(err, "failed to create a table")
	}

	// story.published
	query = `
	BEGIN TRANSACTION;
		CREATE TABLE StoryPublished (
			UserID string NOT NULL,
			Author string,
			Tag    string
		);
		CREATE INDEX StoryPublishedUserID ON StoryPublished (UserID);
		CREATE INDEX StoryPublishedAuthor ON StoryPublished (Author);
		CREATE INDEX StoryPublishedTag    ON StoryPublished (Tag);
	COMMIT;
	`

	if _, _, err := mem.Run(tctx, query); err != nil {
		mem.Close()
		return errors.Wrap(err, "failed to create a table")
	}

	// story.voted
	query = `
	BEGIN TRANSACTION;
		CREATE TABLE StoryVoted (
			UserID string NOT NULL,
			Author string,
			Voter  string
		);
		CREATE INDEX StoryVotedUserID ON StoryVoted (UserID);
		CREATE INDEX StoryVotedAuthor ON StoryVoted (Author);
		CREATE INDEX StoryVotedVoter  ON StoryVoted (Voter);
	COMMIT;
	`

	if _, _, err := mem.Run(tctx, query); err != nil {
		mem.Close()
		return errors.Wrap(err, "failed to create a table")
	}

	// comment.published
	query = `
	BEGIN TRANSACTION;
		CREATE TABLE CommentPublished (
			UserID       string NOT NULL,
			Author       string,
			ParentAuthor string
		);
		CREATE INDEX CommentPublishedUserID       ON CommentPublished (UserID);
		CREATE INDEX CommentPublishedAuthor       ON CommentPublished (Author);
		CREATE INDEX CommentPublishedParentAuthor ON CommentPublished (ParentAuthor);
	COMMIT;
	`

	if _, _, err := mem.Run(tctx, query); err != nil {
		mem.Close()
		return errors.Wrap(err, "failed to create a table")
	}

	// comment.voted
	query = `
	BEGIN TRANSACTION;
		CREATE TABLE CommentVoted (
			UserID string NOT NULL,
			Author string,
			Voter  string
		);
		CREATE INDEX CommentVotedUserID ON CommentVoted (UserID);
		CREATE INDEX CommentVotedAuthor ON CommentVoted (Author);
		CREATE INDEX CommentVotedVoter  ON CommentVoted (Voter);
	COMMIT;
	`

	if _, _, err := mem.Run(tctx, query); err != nil {
		mem.Close()
		return errors.Wrap(err, "failed to create a table")
	}

	// descendant.published
	query = `
	BEGIN TRANSACTION;
		CREATE TABLE DescendantPublished (
			UserID     string NOT NULL,
			ContentID  string,
			DepthLimit uint8
		);
		CREATE INDEX DescendantPublishedUserID   ON DescendantPublished (UserID);
		CREATE INDEX DescendantPublishedContenID ON DescendantPublished (ContentID);
	COMMIT;
	`

	if _, _, err := mem.Run(tctx, query); err != nil {
		mem.Close()
		return errors.Wrap(err, "failed to create a table")
	}

	// Now we need to fill the database.
	if _, _, err := mem.Run(tctx, "BEGIN TRANSACTION"); err != nil {
		mem.Close()
		return errors.Wrap(err, "failed to begin a transaction")
	}

	var result struct {
		OwnerID         bson.ObjectId                  `bson:"ownerId"`
		Kind            string                         `bson:"kind"`
		Accounts        []string                       `bson:"accounts"`
		Witnesses       []string                       `bson:"witnesses"`
		From            []string                       `bson:"from"`
		To              []string                       `bson:"to"`
		Users           []string                       `bson:"users"`
		AuthorBlacklist []string                       `bson:"authorBlacklist"`
		Tags            []string                       `bson:"tags"`
		Voters          []string                       `bson:"voters"`
		ParentAuthors   []string                       `bson:"parentAuthors"`
		Selectors       []descendantpublished.Selector `bson:"selectors"`
	}
	iter := processor.db.C("events").Find(nil).Iter()
	for iter.Next(&result) {
		ownerID := result.OwnerID.Hex()

		switch result.Kind {
		case "account.updated":
			for _, v := range result.Accounts {
				if _, _, err := mem.Run(
					tctx,
					`INSERT INTO AccountUpdated VALUES ($1, $2)`,
					ownerID, v,
				); err != nil {
					mem.Close()
					return errors.Wrap(err, "failed to insert internal DB value")
				}
			}

		case "account.witness_voted":
			for _, v := range result.Accounts {
				if _, _, err := mem.Run(
					tctx,
					`INSERT INTO AccountWitnessVoted VALUES ($1, $2, NULL)`,
					ownerID, v,
				); err != nil {
					mem.Close()
					return errors.Wrap(err, "failed to insert internal DB value")
				}
			}
			for _, v := range result.Witnesses {
				if _, _, err := mem.Run(
					tctx,
					`INSERT INTO AccountWitnessVoted VALUES ($1, NULL, $2)`,
					ownerID, v,
				); err != nil {
					mem.Close()
					return errors.Wrap(err, "failed to insert internal DB value")
				}
			}

		case "transfer.made":
			for _, v := range result.From {
				if _, _, err := mem.Run(
					tctx,
					`INSERT INTO TransferMade VALUES ($1, $2, NULL)`,
					ownerID, v,
				); err != nil {
					mem.Close()
					return errors.Wrap(err, "failed to insert internal DB value")
				}
			}
			for _, v := range result.To {
				if _, _, err := mem.Run(
					tctx,
					`INSERT INTO TransferMade VALUES ($1, NULL, $2)`,
					ownerID, v,
				); err != nil {
					mem.Close()
					return errors.Wrap(err, "failed to insert internal DB value")
				}
			}

		case "user.mentioned":

		case "user.follow_changed":

		case "story.published":

		case "story.voted":

		case "comment.published":

		case "comment.voted":

		case "descendant.published":

		default:
		}
	}
	if err := iter.Err(); err != nil {
		mem.Close()
		return errors.Wrap(err, "failed get all event documents")
	}

	if _, _, err := mem.Run(tctx, "COMMIT"); err != nil {
		mem.Close()
		return errors.Wrap(err, "failed to commit the transaction")
	}

	log.Printf("notifications: internal DB initialized, it took %v", time.Since(start))
	processor.mem = mem
	return nil
}

func (processor *BlockProcessor) indexReloader(<-chan *users.User) error {
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
