package events

import (
	"github.com/go-steem/rpc/apis/database"
	"github.com/go-steem/rpc/types"
)

type StoryVoted struct {
	Op      *types.VoteOperation
	Content *database.Content
}

type StoryVotedEventMiner struct{}

func NewStoryVotedEventMiner() *StoryVotedEventMiner {
	return &StoryVotedEventMiner{}
}

func (miner *StoryVotedEventMiner) MineEvent(
	operation types.Operation,
	content *database.Content,
) ([]interface{}, error) {

	if !content.IsStory() {
		return nil, nil
	}

	op, ok := operation.Data().(*types.VoteOperation)
	if !ok {
		return nil, nil
	}

	return []interface{}{&StoryVoted{op, content}}, nil
}
