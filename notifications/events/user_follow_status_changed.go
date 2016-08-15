package events

import (
	"github.com/go-steem/rpc/apis/database"
)

type UserFollowStatusChanged struct {
	Op *database.FollowOperation
}

func (event *UserFollowStatusChanged) Followed() bool {
	return len(event.Op.What) == 1 && event.Op.What[0] == "blog"
}

func (event *UserFollowStatusChanged) Muted() bool {
	return len(event.Op.What) == 1 && event.Op.What[0] == "ignore"
}

func (event *UserFollowStatusChanged) Reset() bool {
	return len(event.Op.What) == 0
}

type UserFollowStatusChangedEventMiner struct{}

func NewUserFollowStatusChangedEventMiner() *UserFollowStatusChangedEventMiner {
	return &UserFollowStatusChangedEventMiner{}
}

func (miner *UserFollowStatusChangedEventMiner) MineEvent(
	operation *database.Operation,
	content *database.Content, // nil
) ([]interface{}, error) {

	op, ok := operation.Body.(*database.CustomJSONOperation)
	if !ok || op.ID != database.CustomJSONOperationIDFollow {
		return nil, nil
	}

	body, err := op.UnmarshalBody()
	if err != nil {
		return nil, err
	}

	return []interface{}{&UserFollowStatusChanged{body.(*database.FollowOperation)}}, nil
}
