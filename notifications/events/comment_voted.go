package events

import (
	"github.com/go-steem/rpc/apis/database"
	"github.com/go-steem/rpc/types"
)

type CommentVoted struct {
	Op      *types.VoteOperation
	Content *database.Content
}

type CommentVotedEventMiner struct{}

func NewCommentVotedEventMiner() *CommentVotedEventMiner {
	return &CommentVotedEventMiner{}
}

func (miner *CommentVotedEventMiner) MineEvent(
	operation types.Operation,
	content *database.Content,
) ([]interface{}, error) {

	if content.IsStory() {
		return nil, nil
	}

	op, ok := operation.Data().(*types.VoteOperation)
	if !ok {
		return nil, nil
	}

	return []interface{}{&CommentVoted{op, content}}, nil
}
