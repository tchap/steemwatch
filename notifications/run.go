package notifications

import (
	"github.com/go-steem/rpc"
	"github.com/steemwatch/blockfetcher"
	"gopkg.in/mgo.v2"
)

type ConnectFunc func() (*rpc.Client, error)

func Run(
	client *rpc.Client,
	connect ConnectFunc,
	db *mgo.Database,
	opts ...Option,
) (*blockfetcher.Context, error) {
	initNotifiers()

	processor, err := New(client, connect, db, opts...)
	if err != nil {
		return nil, err
	}
	return blockfetcher.Run(client, processor)
}
