package events

import (
	"regexp"

	"github.com/go-steem/rpc"
)

type UserMentioned struct {
	Op      *rpc.CommentOperation
	Content *rpc.Content
	User    string
}

type UserMentionedEventMiner struct {
	re *regexp.Regexp
}

func NewUserMentionedEventMiner() *UserMentionedEventMiner {
	return &UserMentionedEventMiner{
		re: regexp.MustCompile(`@([a-z0-9\-]+)`),
	}
}

func (miner *UserMentionedEventMiner) MineEvent(operation *rpc.Operation, content *rpc.Content) []interface{} {
	op, ok := operation.Body.(*rpc.CommentOperation)
	if !ok {
		return nil
	}

	match := miner.re.FindAllStringSubmatch(content.Body, -1)

	events := make([]interface{}, 0, len(match))
	for _, m := range match {
		events = append(events, &UserMentioned{op, content, m[1]})
	}
	return events
}
