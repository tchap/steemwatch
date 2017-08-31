package notifications

import (
	"log"
	"time"

	"github.com/tchap/steemwatch/server/routes/api/events/descendantpublished"

	"github.com/cznic/ql"
	"github.com/pkg/errors"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func buildDB(db *mgo.Database) (*ql.DB, error) {
	start := time.Now()

	mem, err := ql.OpenMem()
	if err != nil {
		return nil, errors.Wrap(err, "failed to open a QL in-memory database")
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
		return nil, errors.Wrap(err, "failed to create a table")
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
		return nil, errors.Wrap(err, "failed to create a table")
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
		return nil, errors.Wrap(err, "failed to create a table")
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
		return nil, errors.Wrap(err, "failed to create a table")
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
		return nil, errors.Wrap(err, "failed to create a table")
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
		return nil, errors.Wrap(err, "failed to create a table")
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
		return nil, errors.Wrap(err, "failed to create a table")
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
		return nil, errors.Wrap(err, "failed to create a table")
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
		return nil, errors.Wrap(err, "failed to create a table")
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
		return nil, errors.Wrap(err, "failed to create a table")
	}

	// Now we need to fill the database.
	if _, _, err := mem.Run(tctx, "BEGIN TRANSACTION"); err != nil {
		mem.Close()
		return nil, errors.Wrap(err, "failed to begin a transaction")
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
		Authors         []string                       `bson:"authors"`
		Tags            []string                       `bson:"tags"`
		Voters          []string                       `bson:"voters"`
		ParentAuthors   []string                       `bson:"parentAuthors"`
		Selectors       []descendantpublished.Selector `bson:"selectors"`
	}
	iter := db.C("events").Find(nil).Iter()
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
					return nil, errors.Wrap(err, "failed to insert internal DB value")
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
					return nil, errors.Wrap(err, "failed to insert internal DB value")
				}
			}
			for _, v := range result.Witnesses {
				if _, _, err := mem.Run(
					tctx,
					`INSERT INTO AccountWitnessVoted VALUES ($1, NULL, $2)`,
					ownerID, v,
				); err != nil {
					mem.Close()
					return nil, errors.Wrap(err, "failed to insert internal DB value")
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
					return nil, errors.Wrap(err, "failed to insert internal DB value")
				}
			}
			for _, v := range result.To {
				if _, _, err := mem.Run(
					tctx,
					`INSERT INTO TransferMade VALUES ($1, NULL, $2)`,
					ownerID, v,
				); err != nil {
					mem.Close()
					return nil, errors.Wrap(err, "failed to insert internal DB value")
				}
			}

		case "user.mentioned":
			for _, v := range result.Users {
				if _, _, err := mem.Run(
					tctx,
					`INSERT INTO UserMentioned VALUES ($1, $2, NULL)`,
					ownerID, v,
				); err != nil {
					mem.Close()
					return nil, errors.Wrap(err, "failed to insert internal DB value")
				}
			}
			for _, v := range result.AuthorBlacklist {
				if _, _, err := mem.Run(
					tctx,
					`INSERT INTO UserMentioned VALUES ($1, NULL, $2)`,
					ownerID, v,
				); err != nil {
					mem.Close()
					return nil, errors.Wrap(err, "failed to insert internal DB value")
				}
			}

		case "user.follow_changed":
			for _, v := range result.Users {
				if _, _, err := mem.Run(
					tctx,
					`INSERT INTO UserFollowChanged VALUES ($1, $2)`,
					ownerID, v,
				); err != nil {
					mem.Close()
					return nil, errors.Wrap(err, "failed to insert internal DB value")
				}
			}

		case "story.published":
			for _, v := range result.Authors {
				if _, _, err := mem.Run(
					tctx,
					`INSERT INTO StoryPublished VALUES ($1, $2, NULL)`,
					ownerID, v,
				); err != nil {
					mem.Close()
					return nil, errors.Wrap(err, "failed to insert internal DB value")
				}
			}
			for _, v := range result.Tags {
				if _, _, err := mem.Run(
					tctx,
					`INSERT INTO StoryPublished VALUES ($1, NULL, $2)`,
					ownerID, v,
				); err != nil {
					mem.Close()
					return nil, errors.Wrap(err, "failed to insert internal DB value")
				}
			}

		case "story.voted":
			for _, v := range result.Authors {
				if _, _, err := mem.Run(
					tctx,
					`INSERT INTO StoryVoted VALUES ($1, $2, NULL)`,
					ownerID, v,
				); err != nil {
					mem.Close()
					return nil, errors.Wrap(err, "failed to insert internal DB value")
				}
			}
			for _, v := range result.Voters {
				if _, _, err := mem.Run(
					tctx,
					`INSERT INTO StoryVoted VALUES ($1, NULL, $2)`,
					ownerID, v,
				); err != nil {
					mem.Close()
					return nil, errors.Wrap(err, "failed to insert internal DB value")
				}
			}

		case "comment.published":
			for _, v := range result.Authors {
				if _, _, err := mem.Run(
					tctx,
					`INSERT INTO CommentPublished VALUES ($1, $2, NULL)`,
					ownerID, v,
				); err != nil {
					mem.Close()
					return nil, errors.Wrap(err, "failed to insert internal DB value")
				}
			}
			for _, v := range result.ParentAuthors {
				if _, _, err := mem.Run(
					tctx,
					`INSERT INTO CommentPublished VALUES ($1, NULL, $2)`,
					ownerID, v,
				); err != nil {
					mem.Close()
					return nil, errors.Wrap(err, "failed to insert internal DB value")
				}
			}

		case "comment.voted":
			for _, v := range result.Authors {
				if _, _, err := mem.Run(
					tctx,
					`INSERT INTO CommentVoted VALUES ($1, $2, NULL)`,
					ownerID, v,
				); err != nil {
					mem.Close()
					return nil, errors.Wrap(err, "failed to insert internal DB value")
				}
			}
			for _, v := range result.Voters {
				if _, _, err := mem.Run(
					tctx,
					`INSERT INTO CommentVoted VALUES ($1, NULL, $2)`,
					ownerID, v,
				); err != nil {
					mem.Close()
					return nil, errors.Wrap(err, "failed to insert internal DB value")
				}
			}

		case "descendant.published":
			for _, v := range result.Selectors {
				var err error
				if v.Mode == descendantpublished.SelectorModeDepthLimit {
					_, _, err = mem.Run(
						tctx,
						`INSERT INTO DescendantPublished VALUES ($1, $2, $3)`,
						ownerID, v.ContentID, uint8(v.DepthLimit),
					)
				} else {
					_, _, err = mem.Run(
						tctx,
						`INSERT INTO DescendantPublished VALUES ($1, $2, NULL)`,
						ownerID, v.ContentID,
					)
				}
				if err != nil {
					mem.Close()
					return nil, errors.Wrap(err, "failed to insert internal DB value")
				}
			}

		default:
			log.Printf("[notifications] unknown event kind: %v", result.Kind)
		}
	}
	if err := iter.Err(); err != nil {
		mem.Close()
		return nil, errors.Wrap(err, "failed get all event documents")
	}

	if _, _, err := mem.Run(tctx, "COMMIT"); err != nil {
		mem.Close()
		return nil, errors.Wrap(err, "failed to commit the transaction")
	}

	log.Printf("notifications: internal DB initialized, it took %v", time.Since(start))
	return mem, nil
}
