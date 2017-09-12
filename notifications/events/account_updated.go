package events

import (
	"github.com/go-steem/rpc/apis/database"
	"github.com/go-steem/rpc/types"
)

type AccountUpdated struct {
	Op *types.AccountUpdateOperation
}

type AccountUpdatedEventMiner struct{}

func NewAccountUpdatedEventMiner() *AccountUpdatedEventMiner {
	return &AccountUpdatedEventMiner{}
}

func (miner *AccountUpdatedEventMiner) MineEvent(
	operation types.Operation,
	content *database.Content, // nil
) ([]interface{}, error) {

	op, ok := operation.Data().(*types.AccountUpdateOperation)
	if !ok {
		return nil, nil
	}
	return []interface{}{&AccountUpdated{op}}, nil
}
