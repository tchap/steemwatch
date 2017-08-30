package notifications

import (
	"github.com/tchap/steemwatch/server/users"

	"github.com/go-steem/rpc"
	"github.com/steemwatch/blockfetcher"
	"gopkg.in/mgo.v2"
)

func Run(
	client *rpc.Client,
	db *mgo.Database,
	userChangedCh <-chan *users.User,
	opts ...Option,
) (*blockfetcher.Context, error) {

	processor, err := New(client, db, userChangedCh, opts...)
	if err != nil {
		return nil, err
	}
	return blockfetcher.Run(client, processor)
}
