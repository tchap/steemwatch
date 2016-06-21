package notifications

import (
	"github.com/go-steem/rpc"
)

type EventMiner interface {
	MineEvent(*rpc.Operation, *rpc.Content) (event interface{})
}
