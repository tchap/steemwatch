package events

import (
	"github.com/go-steem/rpc/apis/database"
	"github.com/go-steem/rpc/types"
)

type TransferMade struct {
	Op *types.TransferOperation
}

type TransferMadeEventMiner struct{}

func NewTransferMadeEventMiner() *TransferMadeEventMiner {
	return &TransferMadeEventMiner{}
}

func (miner *TransferMadeEventMiner) MineEvent(
	operation types.Operation,
	content *database.Content, // nil
) ([]interface{}, error) {

	op, ok := operation.Data().(*types.TransferOperation)
	if !ok {
		return nil, nil
	}
	return []interface{}{&TransferMade{op}}, nil
}
