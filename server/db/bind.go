package db

import (
	"encoding/json"
	"io/ioutil"

	"github.com/tchap/steemwatch/server/context"
	"github.com/tchap/steemwatch/server/users"

	"github.com/labstack/echo"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func BindList(serverCtx *context.Context, group *echo.Group) {
	group.GET("/", func(ctx echo.Context) error {
		// Get the list from the database and unmarshal it.
		var (
			profile   = ctx.Get("user").(*users.User)
			eventKind = ctx.Param("kind")
			listName  = ctx.Param("list")
		)

		query := bson.M{
			"ownerId": bson.ObjectIdHex(profile.Id),
			"kind":    eventKind,
		}

		selector := bson.M{
			listName: 1,
		}

		var (
			doc  map[string][]string
			list []string
		)
		err := serverCtx.DB.C("events").Find(query).Select(selector).One(&doc)
		if err != nil && err != mgo.ErrNotFound {
			return err
		}
		if lx, ok := doc[listName]; ok {
			list = lx
		} else {
			list = []string{}
		}

		// Send the chosen list as a response.
		ctx.Response().Header().Set(echo.HeaderContentType, "application/json")
		return json.NewEncoder(ctx.Response().Writer).Encode(list)
	})

	group.POST("/", func(ctx echo.Context) error {
		// Read the request body.
		body, err := ioutil.ReadAll(ctx.Request().Body)
		if err != nil {
			return err
		}

		// Push to the database.
		var (
			profile   = ctx.Get("user").(*users.User)
			eventKind = ctx.Param("kind")
			listName  = ctx.Param("list")
		)

		selector := bson.M{
			"ownerId": bson.ObjectIdHex(profile.Id),
			"kind":    eventKind,
		}

		update := bson.M{
			"$push": bson.M{
				listName: string(body),
			},
		}

		_, err = serverCtx.DB.C("events").Upsert(selector, update)
		return err
	})

	group.DELETE("/:item/", func(ctx echo.Context) error {
		// Push to the database.
		var (
			profile   = ctx.Get("user").(*users.User)
			eventKind = ctx.Param("kind")
			listName  = ctx.Param("list")
			item      = ctx.Param("item")
		)

		selector := bson.M{
			"ownerId": bson.ObjectIdHex(profile.Id),
			"kind":    eventKind,
		}

		update := bson.M{
			"$pull": bson.M{
				listName: item,
			},
		}

		return serverCtx.DB.C("events").Update(selector, update)
	})
}
