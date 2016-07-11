package notifications

import (
	"github.com/go-steem/rpc/apis/database"
)

type EventMiner interface {
	MineEvent(*database.Operation, *database.Content) (events []interface{})
}
