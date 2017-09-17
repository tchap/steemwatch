package info

import (
	"encoding/json"
	"time"

	"github.com/tchap/steemwatch/notifications"
	"github.com/tchap/steemwatch/server/context"

	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2/bson"
)

type Info struct {
	NextBlockNumber    uint32     `json:"nextBlockNumber"`
	LastBlockTimestamp *time.Time `json:"lastBlockTimestamp,omitempty"`
}

func Bind(serverCtx *context.Context, root *echo.Group) {
	root.GET("/", func(ctx echo.Context) error {
		var config notifications.BlockProcessorConfig
		err := serverCtx.DB.C("configuration").Find(bson.M{"_id": "BlockProcessor"}).One(&config)
		if err != nil {
			return errors.Wrap(err, "failed to get BlockProcessor config")
		}

		info := &Info{
			config.NextBlockNum,
			config.LastBlockTimestamp,
		}

		resp := ctx.Response()
		resp.Header().Set("Content-Type", "application/json")
		return json.NewEncoder(resp.Writer).Encode(&info)
	})
}
