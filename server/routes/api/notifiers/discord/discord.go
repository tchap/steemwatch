package discord

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	tomb "gopkg.in/tomb.v2"

	"github.com/bwmarrin/discordgo"
	lru "github.com/hashicorp/golang-lru"
	"github.com/kr/pretty"
	"github.com/pkg/errors"
	"github.com/tchap/steemwatch/server/context"
	"github.com/tchap/steemwatch/server/users"

	"github.com/labstack/echo"
)

const NotifierID = "discord"

type Settings struct {
	StartToken string `json:"startToken" bson:"startToken,omitempty"`

	UserID   string `json:"-"                  bson:"userId,omitempty"`
	Username string `json:"username,omitempty" bson:"username,omitempty"`
	ChatID   string `json:"-"                  bson:"chatId,omitempty"`
}

type Document struct {
	OwnerId    bson.ObjectId `json:"-"        bson:"ownerId,omitempty"`
	NotifierId string        `json:"-"        bson:"notifierId,omitempty"`
	Enabled    *bool         `json:"enabled"  bson:"enabled,omitempty"`
	Settings   *Settings     `json:"settings" bson:"settings,omitempty"`
}

func InitBot(
	t *tomb.Tomb,
	botToken string,
	serverCtx *context.Context,
) (*discordgo.Session, error) {

	// Ensure indexes.
	if err := serverCtx.DB.C("notifiers").EnsureIndex(mgo.Index{
		Key:        []string{"settings.startToken"},
		Unique:     true,
		Background: true,
		Sparse:     true,
	}); err != nil {
		log.Printf("%# v", errors.Wrap(err, "failed to create index for Discord"))
	}

	if err := serverCtx.DB.C("notifiers").EnsureIndex(mgo.Index{
		Key:        []string{"settings.userId"},
		Unique:     true,
		Background: true,
		Sparse:     true,
	}); err != nil {
		log.Printf("%# v", errors.Wrap(err, "failed to create index for Discord"))
	}

	// Discord now!
	dg, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize Discord")
	}

	dg.LogLevel = discordgo.LogWarning

	var botID string
	dg.AddHandler(func(s *discordgo.Session, msg *discordgo.Ready) {
		fmt.Printf("DISCORD READY: %# v\n", pretty.Formatter(msg))
		botID = msg.User.ID
	})

	channelCache, err := lru.New(10000)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init an LRU cache")
	}

	dg.AddHandler(func(s *discordgo.Session, msg *discordgo.MessageCreate) {
		// Drop messages posted by the bot.
		if msg.Author.ID == botID {
			return
		}

		fmt.Printf("DISCORD MESSAGE: %# v\n", pretty.Formatter(msg))

		// Helpers.
		channelID := msg.ChannelID
		send := func(text string) {
			if _, err := dg.ChannelMessageSend(channelID, text); err != nil {
				log.Printf("discord: %# v", errors.Wrap(err, "failed to send a message"))
			}
		}

		// Is this a direct message?
		var channel *discordgo.Channel
		v, ok := channelCache.Get(channelID)
		if ok {
			channel = v.(*discordgo.Channel)
		} else {
			var err error
			channel, err = dg.Channel(channelID)
			if err != nil {
				log.Printf("discord: %# v", errors.Wrap(err, "failed get channel by ID"))
				send("Something went terribly wrong, sorry!")
				return
			}

			channelCache.Add(channelID, channel)
		}

		isDM := channel.Type == discordgo.ChannelTypeDM

		// Is the bot mentioned?
		var isMentioned bool
		for _, user := range msg.Mentions {
			if user.ID == botID {
				isMentioned = true
				break
			}
		}

		// Decide what to do next.
		if !isDM {
			if isMentioned {
				send("Don't bother people here, send me a direct message instead.")
			}
			return
		}

		const help = `I am the official SteemWatch bot for Discord.
If you don't know what SteemWatch is, feel free to discover it at https://steemwatch.com

Anyway, I recognize the following commands:

	**ping** - A good-for-nothing command. I will simply reply with "pong".
	
    **status** - I will show you current configuration.

    **link <token>** - I will link your Discord and SteemWatch accounts together.
        You have to provide me with a token you can get in the SteemWatch web application.
		Just go to the Notifications tab, find the Discord section and there it is!

    **enable** - Make me enable SteemWatch notifications. This is the default.

    **disable** - Make me disable SteemWatch notifications.

    **unlink** - I will break the link with SteemWatch.

    **help** - I will print this command help and that's it.

To command me, simply send me a message containing one of the commands above.`

		content := strings.TrimSpace(msg.Content)

		var text string
		switch content {
		case "help":
			text = help

		case "ping":
			text = "pong"

		case "status":
			query := bson.M{
				"notifierId":      NotifierID,
				"settings.userId": msg.Author.ID,
			}

			var doc Document
			if err := serverCtx.DB.C("notifiers").Find(query).One(&doc); err != nil {
				if err == mgo.ErrNotFound {
					text = "You are not linked with SteemWatch yet!"
				} else {
					log.Printf("%# v", err)
					text = "Something went terribly wrong, sorry!"
				}
			} else {
				if doc.Settings.UserID == "" {
					text = "You are not linked with SteemWatch yet!"
				} else if *doc.Enabled {
					text = "Discord notifications enabled :-)"
				} else {
					text = "Discord notifications disabled :-("
				}
			}

		case "enable":
			selector := bson.M{
				"settings.userId": msg.Author.ID,
			}

			if err := setEnabled(serverCtx.DB.C("notifiers"), selector, true); err != nil {
				if errors.Cause(err) == mgo.ErrNotFound {
					text = "SteemWatch link not found, did you call **link**?"
				} else {
					log.Printf("%# v", err)
					text = "Something went terribly wrong, sorry!"
				}
			} else {
				text = "Enabled, as you wanted."
			}

		case "disable":
			selector := bson.M{
				"settings.userId": msg.Author.ID,
			}

			if err := setEnabled(serverCtx.DB.C("notifiers"), selector, false); err != nil {
				if errors.Cause(err) == mgo.ErrNotFound {
					text = "SteemWatch link not found, did you call **link**?"
				} else {
					log.Printf("%# v", err)
					text = "Something went terribly wrong, sorry!"
				}
			} else {
				text = "Disabled, as you wanted."
			}

		case "unlink":
			selector := bson.M{
				"settings.userId": msg.Author.ID,
			}

			if err := unlink(serverCtx.DB.C("notifiers"), selector); err != nil {
				if errors.Cause(err) == mgo.ErrNotFound {
					text = "SteemWatch link not found, did you call **link**?"
				} else {
					log.Printf("%# v", err)
					text = "Something went terribly wrong, sorry!"
				}
			} else {
				text = "Unlinked from SteemWatch, as you wanted."
			}

		default:
			parts := strings.SplitN(content, " ", -1)
			if len(parts) != 2 {
				text = "I don't understand.\n\n" + help
				break
			}

			cmd, arg := parts[0], parts[1]
			switch cmd {
			case "link":
				// /start TOKEN
				selector := bson.M{
					"settings.startToken": arg,
				}

				change := bson.M{
					"$set": bson.M{
						"enabled":           true,
						"settings.userId":   msg.Author.ID,
						"settings.username": msg.Author.Username,
						"settings.chatId":   msg.ChannelID,
					},
				}

				if err := serverCtx.DB.C("notifiers").Update(selector, change); err != nil {
					if err == mgo.ErrNotFound {
						send("I don't recognize the token you provided.")
					} else {
						send("Something went terribly wrong, sorry!")
						log.Printf("%# v", errors.Wrap(err, "failed to enable Discord"))
					}
					return
				}

				text = "SteemWatch account linked successfully."

			default:
				text = "I don't understand.\n\n" + help
			}
		}

		// Reply.
		send(text)

		// TODO: Do we need to ack?
	})

	// Open the gateway connection.
	if err := dg.Open(); err != nil {
		return nil, errors.Wrap(err, "failed to connect to Discord")
	}

	// Make sure the Discord session is ok.
	t.Go(func() error {
		me, err := dg.User("@me")
		if err != nil {
			log.Println("discord: failed to get @me")
			return nil
		}

		fmt.Printf("DISCORD: I am %# v\n", pretty.Formatter(me))
		return nil
	})

	// Close the Discord session when exiting.
	t.Go(func() error {
		<-t.Dying()
		dg.Close()
		return nil
	})

	return dg, nil
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
			"ownerId": bson.ObjectIdHex(profile.Id),
		}

		return setEnabled(serverCtx.DB.C("notifiers"), selector, *doc.Enabled)
	})

	root.DELETE("/", func(ctx echo.Context) error {
		profile := ctx.Get("user").(*users.User)

		selector := bson.M{
			"ownerId": bson.ObjectIdHex(profile.Id),
		}

		return unlink(serverCtx.DB.C("notifiers"), selector)
	})
}

func setEnabled(c *mgo.Collection, selector bson.M, enabled bool) error {
	selector["notifierId"] = NotifierID

	update := bson.M{
		"$set": bson.M{
			"enabled": enabled,
		},
	}

	err := c.Update(selector, update)
	return errors.Wrapf(err, "failed to update doc [select=%+v, update=%+v]", selector, update)
}

func unlink(c *mgo.Collection, selector bson.M) error {
	selector["notifierId"] = NotifierID

	update := bson.M{
		"$set": bson.M{
			"enabled": false,
		},
		"$unset": bson.M{
			"settings.userId":   "",
			"settings.username": "",
			"settings.chatId":   "",
		},
	}

	err := c.Update(selector, update)
	return errors.Wrapf(err, "failed to update doc [select=%+v, update=%+v]", selector, update)
}
