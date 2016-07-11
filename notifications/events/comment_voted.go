package events

import (
	"github.com/go-steem/rpc/apis/database"
)

type CommentVoted struct {
	Op      *database.VoteOperation
	Content *database.Content
}

type CommentVotedEventMiner struct{}

func NewCommentVotedEventMiner() *CommentVotedEventMiner {
	return &CommentVotedEventMiner{}
}

func (miner *CommentVotedEventMiner) MineEvent(
	operation *database.Operation,
	content *database.Content,
) []interface{} {

	if content.IsStory() {
		return nil
	}

	op, ok := operation.Body.(*database.VoteOperation)
	if !ok {
		return nil
	}

	return []interface{}{&CommentVoted{op, content}}
}
