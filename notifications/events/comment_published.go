package events

import (
	"github.com/go-steem/rpc/apis/database"
	"github.com/go-steem/rpc/types"
)

type CommentPublished struct {
	Op      *types.CommentOperation
	Content *database.Content
}

type CommentPublishedEventMiner struct{}

func NewCommentPublishedEventMiner() *CommentPublishedEventMiner {
	return &CommentPublishedEventMiner{}
}

func (miner *CommentPublishedEventMiner) MineEvent(
	operation types.Operation,
	content *database.Content,
) ([]interface{}, error) {

	if content.IsStory() {
		return nil, nil
	}

	op, ok := operation.Data().(*types.CommentOperation)
	if !ok {
		return nil, nil
	}

	return []interface{}{&CommentPublished{op, content}}, nil
}
