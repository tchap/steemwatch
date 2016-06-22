package events

import (
	"github.com/go-steem/rpc"
)

type StoryVoted struct {
	Op      *rpc.VoteOperation
	Content *rpc.Content
}

type StoryVotedEventMiner struct{}

func NewStoryVotedEventMiner() *StoryVotedEventMiner {
	return &StoryVotedEventMiner{}
}

func (miner *StoryVotedEventMiner) MineEvent(operation *rpc.Operation, content *rpc.Content) []interface{} {
	if !content.IsStory() {
		return nil
	}

	op, ok := operation.Body.(*rpc.VoteOperation)
	if !ok {
		return nil
	}

	return []interface{}{&StoryVoted{op, content}}
}
