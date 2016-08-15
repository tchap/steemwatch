package events

import (
	"github.com/go-steem/rpc/apis/database"
)

type TransferMade struct {
	Op *database.TransferOperation
}

type TransferMadeEventMiner struct{}

func NewTransferMadeEventMiner() *TransferMadeEventMiner {
	return &TransferMadeEventMiner{}
}

func (miner *TransferMadeEventMiner) MineEvent(
	operation *database.Operation,
	content *database.Content, // nil
) ([]interface{}, error) {

	op, ok := operation.Body.(*database.TransferOperation)
	if !ok {
		return nil, nil
	}
	return []interface{}{&TransferMade{op}}, nil
}
