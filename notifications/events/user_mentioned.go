package events

import (
	"regexp"

	"github.com/go-steem/rpc/apis/database"
)

type UserMentioned struct {
	Op      *database.CommentOperation
	Content *database.Content
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

func (miner *UserMentionedEventMiner) MineEvent(
	operation *database.Operation,
	content *database.Content,
) ([]interface{}, error) {

	op, ok := operation.Body.(*database.CommentOperation)
	if !ok {
		return nil, nil
	}

	match := miner.re.FindAllStringSubmatch(content.Body, -1)

	events := make([]interface{}, 0, len(match))
	for _, m := range match {
		events = append(events, &UserMentioned{op, content, m[1]})
	}
	return events, nil
}
