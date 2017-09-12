package notifications

import (
	"github.com/go-steem/rpc/apis/database"
	"github.com/go-steem/rpc/types"
)

type EventMiner interface {
	MineEvent(types.Operation, *database.Content) (events []interface{}, err error)
}
