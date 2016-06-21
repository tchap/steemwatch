package events

import (
	"github.com/go-steem/rpc"
)

type CommentPublished struct {
	Op      *rpc.CommentOperation
	Content *rpc.Content
}

type CommentPublishedEventMiner struct{}

func NewCommentPublishedEventMiner() *CommentPublishedEventMiner {
	return &CommentPublishedEventMiner{}
}

func (miner *CommentPublishedEventMiner) MineEvent(operation *rpc.Operation, content *rpc.Content) interface{} {
	if content.IsStory() {
		return nil
	}

	op, ok := operation.Body.(*rpc.CommentOperation)
	if !ok {
		return nil
	}

	return &CommentPublished{op, content}
}
