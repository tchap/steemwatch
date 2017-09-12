package events

import (
	"github.com/go-steem/rpc/apis/database"
	"github.com/go-steem/rpc/types"
)

type StoryPublished struct {
	Op      *types.CommentOperation
	Content *database.Content
}

type StoryPublishedEventMiner struct{}

func NewStoryPublishedEventMiner() *StoryPublishedEventMiner {
	return &StoryPublishedEventMiner{}
}

func (miner *StoryPublishedEventMiner) MineEvent(
	operation types.Operation,
	content *database.Content,
) ([]interface{}, error) {

	if !content.IsStory() {
		return nil, nil
	}

	op, ok := operation.Data().(*types.CommentOperation)
	if !ok {
		return nil, nil
	}

	return []interface{}{&StoryPublished{op, content}}, nil
}
