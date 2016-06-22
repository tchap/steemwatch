package events

import (
	"github.com/go-steem/rpc"
)

type StoryPublished struct {
	Op      *rpc.CommentOperation
	Content *rpc.Content
}

type StoryPublishedEventMiner struct{}

func NewStoryPublishedEventMiner() *StoryPublishedEventMiner {
	return &StoryPublishedEventMiner{}
}

func (miner *StoryPublishedEventMiner) MineEvent(operation *rpc.Operation, content *rpc.Content) []interface{} {
	if !content.IsStory() {
		return nil
	}

	op, ok := operation.Body.(*rpc.CommentOperation)
	if !ok {
		return nil
	}

	return []interface{}{&StoryPublished{op, content}}
}
