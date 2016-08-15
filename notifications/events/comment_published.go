package events

import (
	"github.com/go-steem/rpc/apis/database"
)

type CommentPublished struct {
	Op      *database.CommentOperation
	Content *database.Content
}

type CommentPublishedEventMiner struct{}

func NewCommentPublishedEventMiner() *CommentPublishedEventMiner {
	return &CommentPublishedEventMiner{}
}

func (miner *CommentPublishedEventMiner) MineEvent(
	operation *database.Operation,
	content *database.Content,
) ([]interface{}, error) {

	if content.IsStory() {
		return nil, nil
	}

	op, ok := operation.Body.(*database.CommentOperation)
	if !ok {
		return nil, nil
	}

	return []interface{}{&CommentPublished{op, content}}, nil
}
