package events

import (
	"github.com/go-steem/rpc"
)

type CommentVoted struct {
	Op      *rpc.VoteOperation
	Content *rpc.Content
}

type CommentVotedEventMiner struct{}

func NewCommentVotedEventMiner() *CommentVotedEventMiner {
	return &CommentVotedEventMiner{}
}

func (miner *CommentVotedEventMiner) MineEvent(operation *rpc.Operation, content *rpc.Content) []interface{} {
	if content.IsStory() {
		return nil
	}

	op, ok := operation.Body.(*rpc.VoteOperation)
	if !ok {
		return nil
	}

	return []interface{}{&CommentVoted{op, content}}
}
