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
) []interface{} {

	if content.IsStory() {
		return nil
	}

	op, ok := operation.Body.(*database.CommentOperation)
	if !ok {
		return nil
	}

	return []interface{}{&CommentPublished{op, content}}
}
