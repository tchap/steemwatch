package steemitchat

import (
	"encoding/json"

	"github.com/tchap/steemwatch/server/context"
	"github.com/tchap/steemwatch/server/users"

	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const NotifierID = "steemit-chat"

type Settings struct {
	Username  string `json:"username,omitempty"  bson:"username"`
	UserID    string `json:"userID,omitempty"    bson:"userID"`
	AuthToken string `json:"authToken,omitempty" bson:"authToken"`
}

type Document struct {
	OwnerId    bson.ObjectId `json:"-"                  bson:"ownerId,omitempty"`
	NotifierId string        `json:"-"                  bson:"notifierId,omitempty"`
	Enabled    *bool         `json:"enabled,omitempty"  bson:"enabled,omitempty"`
	Settings   *Settings     `json:"settings,omitempty" bson:"settings,omitempty"`
}

func (doc *Document) Validate() error {
	switch {
	case doc.Enabled == nil:
		return errors.New("field not set: enabled")
	case doc.Settings == nil:
		return errors.New("filed not set: settings")
	case doc.Settings.Username == "":
		return errors.New("field not set: settings.username")
	case doc.Settings.UserID == "":
		return errors.New("field not set: settings.userID")
	case doc.Settings.AuthToken == "":
		return errors.New("field not set: settings.authToken")
	}
	return nil
}

func Bind(serverCtx *context.Context, root *echo.Group) {
	root.GET("/", func(ctx echo.Context) error {
		profile := ctx.Get("user").(*users.User)

		query := bson.M{
			"ownerId":    bson.ObjectIdHex(profile.Id),
			"notifierId": NotifierID,
		}

		var doc Document
		err := serverCtx.DB.C("notifiers").Find(query).One(&doc)
		if err != nil {
			if err != mgo.ErrNotFound {
				return errors.Wrapf(err, "failed to get doc [query=%+v]", query)
			}
		}

		err = json.NewEncoder(ctx.Response().Writer()).Encode(&doc)
		return errors.Wrapf(err, "failed to encode doc [doc=%+v]", doc)
	})
}
