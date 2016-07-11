package events

import (
	"github.com/go-steem/rpc/apis/database"
)

type StoryVoted struct {
	Op      *database.VoteOperation
	Content *database.Content
}

type StoryVotedEventMiner struct{}

func NewStoryVotedEventMiner() *StoryVotedEventMiner {
	return &StoryVotedEventMiner{}
}

func (miner *StoryVotedEventMiner) MineEvent(
	operation *database.Operation,
	content *database.Content,
) []interface{} {

	if !content.IsStory() {
		return nil
	}

	op, ok := operation.Body.(*database.VoteOperation)
	if !ok {
		return nil
	}

	return []interface{}{&StoryVoted{op, content}}
}
