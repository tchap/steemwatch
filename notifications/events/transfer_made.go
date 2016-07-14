package events

import (
	"fmt"
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
) []interface{} {

	fmt.Printf("%T\n", operation.Body)
	op, ok := operation.Body.(*database.TransferOperation)
	if !ok {
		return nil
	}
	return []interface{}{&TransferMade{op}}
}
