package slack

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/tchap/steemwatch/server/context"
	"github.com/tchap/steemwatch/server/users"

	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Settings struct {
	WebhookURL string `json:"webhookURL" bson:"webhookURL,omitempty"`
}

type Document struct {
	OwnerId    bson.ObjectId `json:"-"          bson:"ownerId,omitempty"`
	NotifierId string        `json:"-"          bson:"notifierId,omitempty"`
	Enabled    *bool         `json:"enabled"    bson:"enabled,omitempty"`
	Settings   *Settings     `json:"settings"   bson:"settings,omitempty"`
}

func (doc *Document) Validate() error {
	switch {
	case doc.Enabled == nil:
		return errors.New("field not set: enabled")
	case doc.Settings == nil || doc.Settings.WebhookURL == "":
		return errors.New("field not set: settings.webhookURL")
	}

	if _, err := url.Parse(doc.Settings.WebhookURL); err != nil {
		return errors.Wrap(err, "settings.webhookURL is not a valid URL")
	}
	return nil
}

func Bind(serverCtx *context.Context, root *echo.Group) {
	root.GET("/", func(ctx echo.Context) error {
		profile := ctx.Get("user").(*users.User)

		query := bson.M{
			"ownerId":    bson.ObjectIdHex(profile.Id),
			"notifierId": "slack",
		}

		var doc Document
		err := serverCtx.DB.C("notifiers").Find(query).One(&doc)
		if err != nil {
			if err == mgo.ErrNotFound {
				enabled := false
				doc.Enabled = &enabled
				doc.Settings = &Settings{}
			} else {
				return errors.Wrapf(err, "failed to get doc [query=%+v]", query)
			}
		}

		err = json.NewEncoder(ctx.Response().Writer()).Encode(&doc)
		return errors.Wrapf(err, "failed to encode doc [doc=%+v]", doc)
	})

	root.PUT("/", func(ctx echo.Context) error {
		profile := ctx.Get("user").(*users.User)

		var doc Document
		if err := json.NewDecoder(ctx.Request().Body()).Decode(&doc); err != nil {
			return errors.Wrap(err, "failed to decode request body")
		}
		doc.OwnerId = bson.ObjectIdHex(profile.Id)
		doc.NotifierId = "slack"

		if err := doc.Validate(); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		selector := bson.M{
			"ownerId":    doc.OwnerId,
			"notifierId": doc.NotifierId,
		}

		_, err := serverCtx.DB.C("notifiers").Upsert(selector, &doc)
		return errors.Wrapf(err, "failed to upsert doc [select=%+v, upsert=%+v]", selector, doc)
	})

	root.PATCH("/", func(ctx echo.Context) error {
		profile := ctx.Get("user").(*users.User)

		var doc Document
		if err := json.NewDecoder(ctx.Request().Body()).Decode(&doc); err != nil {
			return errors.Wrap(err, "failed to decode request body")
		}

		selector := bson.M{
			"ownerId":    bson.ObjectIdHex(profile.Id),
			"notifierId": "slack",
		}

		update := bson.M{
			"$set": &doc,
		}

		err := serverCtx.DB.C("notifiers").Update(selector, update)
		return errors.Wrapf(err, "failed to update doc [select=%+v, update=%+v]", selector, doc)
	})
}
