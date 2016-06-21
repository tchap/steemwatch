package notifications

import (
	"github.com/go-steem/rpc"
	"github.com/steemwatch/blockfetcher"
	"gopkg.in/mgo.v2"
)

func Run(client *rpc.Client, db *mgo.Database) (*blockfetcher.Context, error) {
	processor, err := New(client, db)
	if err != nil {
		return nil, err
	}
	return blockfetcher.Run(client, processor)
}
