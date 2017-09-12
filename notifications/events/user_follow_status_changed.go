package events

import (
	"github.com/go-steem/rpc/apis/database"
	"github.com/go-steem/rpc/types"
)

type UserFollowStatusChanged struct {
	Op *types.FollowOperation
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
	operation types.Operation,
	content *database.Content, // nil
) ([]interface{}, error) {

	op, ok := operation.Data().(*types.CustomJSONOperation)
	if !ok || op.Type() != types.TypeFollow {
		return nil, nil
	}

	data, err := op.UnmarshalData()
	if err != nil {
		return nil, err
	}

	return []interface{}{&UserFollowStatusChanged{data.(*types.FollowOperation)}}, nil
}
