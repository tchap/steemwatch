package telegram

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/pkg/errors"
	"github.com/tchap/steemwatch/server/context"
	"github.com/tchap/steemwatch/server/users"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/labstack/echo"
)

const NotifierID = "telegram"

type Settings struct {
	StartToken string `json:"startToken" bson:"startToken,omitempty"`

	UserID    int64  `json:"-"                   bson:"userId,omitempty"`
	FirstName string `json:"firstName,omitempty" bson:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"  bson:"lastName,omitempty"`
	Username  string `json:"username,omitempty"  bson:"username,omitempty"`
	ChatID    int64  `json:"-"                   bson:"chatId,omitempty"`
}

type Document struct {
	OwnerId    bson.ObjectId `json:"-"          bson:"ownerId,omitempty"`
	NotifierId string        `json:"-"          bson:"notifierId,omitempty"`
	Enabled    *bool         `json:"enabled"    bson:"enabled,omitempty"`
	Settings   *Settings     `json:"settings"   bson:"settings,omitempty"`
}

func BindWebhook(serverCtx *context.Context, root *echo.Group) {
	u, _ := url.Parse("/notifications")
	notifiersURL := serverCtx.CanonicalURL.ResolveReference(u)

	startText := fmt.Sprintf(`
Hey there!

I am the official [SteemWatch](%v) bot. Glad you decided to have a chat with me.

Anyway, you want to get set up so that you can start receiving SteemWatch notifications over Telegram, right?

To achieve that, you need to visit the SteemWatch [Notifications](%v) page, then go to the Telegram section.`,
		serverCtx.CanonicalURL, notifiersURL)

	settingsText := fmt.Sprintf(`
There is nothing to configure here.
Perhaps you want to visit the SteemWatch [web application](%v)?`,
		serverCtx.CanonicalURL)

	const enabledText = `
The Telegram notifier is set up and enabled, yay!

You will be receiving SteemWatch notifications in this channel.`

	root.POST("/", func(ctx echo.Context) error {
		var update tgbotapi.Update
		if err := json.NewDecoder(ctx.Request().Body).Decode(&update); err != nil {
			return errors.Wrap(err, "failed to decode request body")
		}

		text := update.Message.Text

		switch text {
		case "/start", "/help":
			return ctx.JSON(http.StatusOK, map[string]interface{}{
				"method":     "sendMessage",
				"chat_id":    update.Message.Chat.ID,
				"text":       startText,
				"parse_mode": "Markdown",
			})

		case "/settings":
			return ctx.JSON(http.StatusOK, map[string]interface{}{
				"method":     "sendMessage",
				"chat_id":    update.Message.Chat.ID,
				"text":       settingsText,
				"parse_mode": "Markdown",
			})

			// TODO:
			/*
				case "/enable":

				case "/disable":

				case "/disconnect":
			*/
		}

		// /start TOKEN
		if strings.HasPrefix(text, "/start ") {
			parts := strings.SplitN(text, " ", 2)
			token := parts[1]

			selector := bson.M{
				"settings.startToken": token,
			}

			change := bson.M{
				"$set": bson.M{
					"enabled":            true,
					"settings.userId":    update.Message.From.ID,
					"settings.firstName": update.Message.From.FirstName,
					"settings.lastName":  update.Message.From.LastName,
					"settings.username":  update.Message.From.UserName,
					"settings.chatId":    update.Message.Chat.ID,
				},
			}

			if err := serverCtx.DB.C("notifiers").Update(selector, change); err != nil {
				if err == mgo.ErrNotFound {
					return ctx.JSON(http.StatusOK, map[string]interface{}{
						"method":  "sendMessage",
						"chat_id": update.Message.Chat.ID,
						"text":    "The token you passed to /start is unknown to me.",
					})
				}
				return errors.Wrap(err, "failed to enable Telegram")
			}

			return ctx.JSON(http.StatusOK, map[string]interface{}{
				"method":  "sendMessage",
				"chat_id": update.Message.Chat.ID,
				"text":    enabledText,
			})
		}

		return nil
	})
}

func BindAPI(serverCtx *context.Context, root *echo.Group) {
	root.GET("/", func(ctx echo.Context) error {
		profile := ctx.Get("user").(*users.User)

		query := bson.M{
			"ownerId":    bson.ObjectIdHex(profile.Id),
			"notifierId": NotifierID,
		}

		var doc Document
		err := serverCtx.DB.C("notifiers").Find(query).One(&doc)
		if err != nil {
			if err == mgo.ErrNotFound {
				doc.OwnerId = bson.ObjectIdHex(profile.Id)
				doc.NotifierId = NotifierID
				enabled := false
				doc.Enabled = &enabled
				doc.Settings = &Settings{
					StartToken: bson.NewObjectId().Hex(),
				}
				if err := serverCtx.DB.C("notifiers").Insert(&doc); err != nil {
					return errors.Wrap(err, "failed to store start token")
				}
			} else {
				return errors.Wrapf(err, "failed to get doc [query=%+v]", query)
			}
		}

		err = json.NewEncoder(ctx.Response().Writer).Encode(&doc)
		return errors.Wrapf(err, "failed to encode doc [doc=%+v]", doc)
	})

	root.PATCH("/", func(ctx echo.Context) error {
		profile := ctx.Get("user").(*users.User)

		var doc Document
		if err := json.NewDecoder(ctx.Request().Body).Decode(&doc); err != nil {
			return errors.Wrap(err, "failed to decode request body")
		}
		if doc.Enabled == nil {
			return errors.New("invalid request")
		}

		selector := bson.M{
			"ownerId":    bson.ObjectIdHex(profile.Id),
			"notifierId": NotifierID,
		}

		update := bson.M{
			"$set": bson.M{
				"enabled": *doc.Enabled,
			},
		}

		err := serverCtx.DB.C("notifiers").Update(selector, update)
		return errors.Wrapf(err, "failed to update doc [select=%+v, update=%+v]", selector, update)
	})

	root.DELETE("/", func(ctx echo.Context) error {
		profile := ctx.Get("user").(*users.User)

		selector := bson.M{
			"ownerId":    bson.ObjectIdHex(profile.Id),
			"notifierId": NotifierID,
		}

		update := bson.M{
			"$set": bson.M{
				"enabled": false,
			},
			"$unset": bson.M{
				"settings.userId":    "",
				"settings.firstName": "",
				"settings.lastName":  "",
				"settings.username":  "",
				"settings.chatId":    "",
			},
		}

		err := serverCtx.DB.C("notifiers").Update(selector, update)
		return errors.Wrapf(err, "failed to update doc [select=%+v, update=%+v]", selector, update)
	})
}
