package profile

import (
	"encoding/json"
	"io/ioutil"

	"github.com/tchap/steemwatch/server/context"
	"github.com/tchap/steemwatch/server/users"

	"github.com/labstack/echo"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Profile struct {
	Accounts []string `json:"accounts" bson:"accounts"`
}

func Bind(serverCtx *context.Context, group *echo.Group) {
	group.GET("/", func(ctx echo.Context) error {
		profile := ctx.Get("user").(*users.User)

		query := bson.M{
			"_id": bson.ObjectIdHex(profile.Id),
		}

		selector := bson.M{
			"accounts": 1,
		}

		var doc Profile
		err := serverCtx.DB.C("users").Find(query).Select(selector).One(&doc)
		if err != nil && err != mgo.ErrNotFound {
			return err
		}

		// Send the chosen list as a response.
		ctx.Response().Header().Set(echo.HeaderContentType, "application/json")
		return json.NewEncoder(ctx.Response().Writer()).Encode(&doc)
	})

	group.GET("/accounts/", func(ctx echo.Context) error {
		profile := ctx.Get("user").(*users.User)

		query := bson.M{
			"_id": bson.ObjectIdHex(profile.Id),
		}

		selector := bson.M{
			"accounts": 1,
		}

		var (
			doc  Profile
			list []string
		)
		err := serverCtx.DB.C("users").Find(query).Select(selector).One(&doc)
		if err != nil && err != mgo.ErrNotFound {
			return err
		}
		if doc.Accounts == nil {
			list = []string{}
		} else {
			list = doc.Accounts
		}

		// Send the chosen list as a response.
		ctx.Response().Header().Set(echo.HeaderContentType, "application/json")
		return json.NewEncoder(ctx.Response().Writer()).Encode(list)
	})

	group.POST("/accounts/", func(ctx echo.Context) error {
		// Read the request body.
		body, err := ioutil.ReadAll(ctx.Request().Body())
		if err != nil {
			return err
		}

		// Push to the database.
		profile := ctx.Get("user").(*users.User)

		selector := bson.M{
			"_id": bson.ObjectIdHex(profile.Id),
		}

		update := bson.M{
			"$push": bson.M{
				"accounts": string(body),
			},
		}

		_, err = serverCtx.DB.C("users").Upsert(selector, update)
		return err
	})

	group.DELETE("/accounts/:item/", func(ctx echo.Context) error {
		// Push to the database.
		var (
			profile = ctx.Get("user").(*users.User)
			item    = ctx.Param("item")
		)

		selector := bson.M{
			"_id": bson.ObjectIdHex(profile.Id),
		}

		update := bson.M{
			"$pull": bson.M{
				"accounts": item,
			},
		}

		return serverCtx.DB.C("users").Update(selector, update)
	})
}
