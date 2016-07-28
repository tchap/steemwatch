package steemitchat

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/tchap/steemwatch/server/context"
	"github.com/tchap/steemwatch/server/users"

	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const NotifierID = "steemit-chat"

const VerificationRequestTimeout = 10 * time.Second

type Settings struct {
	Username  string `json:"username,omitempty"  bson:"username"`
	UserID    string `json:"userID,omitempty"    bson:"-"`
	AuthToken string `json:"authToken,omitempty" bson:"-"`
}

func (settings *Settings) Validate() error {
	switch {
	case settings.Username == "":
		return errors.New("field not set: username")
	case settings.UserID == "":
		return errors.New("field not set: userID")
	case settings.AuthToken == "":
		return errors.New("field not set: authToken")
	default:
		return nil
	}
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
	default:
		return doc.Settings.Validate()
	}
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

	root.PUT("/", func(ctx echo.Context) error {
		profile := ctx.Get("user").(*users.User)

		// Prepare the document to be inserted.
		var doc Document
		if err := json.NewDecoder(ctx.Request().Body()).Decode(&doc); err != nil {
			return errors.Wrap(err, "failed to decode request body")
		}
		doc.OwnerId = bson.ObjectIdHex(profile.Id)
		doc.NotifierId = NotifierID

		// Make sure the document is valid and complete.
		if err := doc.Validate(); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		// Make sure once more that the given username matches the credentials.
		if err := verifyCredentials(doc.Settings); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		// Replace the document.
		selector := bson.M{
			"ownerId":    doc.OwnerId,
			"notifierId": doc.NotifierId,
		}

		_, err := serverCtx.DB.C("notifiers").Upsert(selector, &doc)
		return errors.Wrapf(err, "failed to upsert doc [select=%+v, upsert=%+v]", selector, doc)
	})

	root.PATCH("/", func(ctx echo.Context) error {
		profile := ctx.Get("user").(*users.User)

		// Unmarshal the update.
		var doc Document
		if err := json.NewDecoder(ctx.Request().Body()).Decode(&doc); err != nil {
			return errors.Wrap(err, "failed to decode request body")
		}

		// In case the credentials are being updated, make sure they are valid.
		if doc.Settings != nil {
			if err := verifyCredentials(doc.Settings); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
		}

		// Update the document.
		selector := bson.M{
			"ownerId":    bson.ObjectIdHex(profile.Id),
			"notifierId": NotifierID,
		}

		update := bson.M{
			"$set": &doc,
		}

		err := serverCtx.DB.C("notifiers").Update(selector, update)
		return errors.Wrapf(err, "failed to update doc [select=%+v, update=%+v]", selector, doc)
	})

	root.DELETE("/", func(ctx echo.Context) error {
		profile := ctx.Get("user").(*users.User)

		selector := bson.M{
			"ownerId":    bson.ObjectIdHex(profile.Id),
			"notifierId": NotifierID,
		}

		err := serverCtx.DB.C("notifiers").Remove(selector)
		return errors.Wrapf(err, "failed to remove doc [select=%+v]", selector)
	})
}

func verifyCredentials(settings *Settings) error {
	// Make sure the settings are complete.
	if err := settings.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// GET /me
	req := fasthttp.AcquireRequest()
	res := fasthttp.AcquireResponse()

	cleanup := func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(res)
	}

	const uri = "https://steemit.chat/api/v1/me"

	req.Header.SetMethod("GET")
	req.Header.Set("X-User-Id", settings.UserID)
	req.Header.Set("X-Auth-Token", settings.AuthToken)
	req.SetRequestURI(uri)
	req.SetConnectionClose()

	if err := fasthttp.DoTimeout(req, res, VerificationRequestTimeout); err != nil {
		cleanup()
		return errors.Wrap(err, "failed to verify Steemit Chat credentials")
	}

	if code := res.StatusCode(); code < 200 || code >= 300 {
		cleanup()
		return errors.Errorf("GET %v -> %v", uri, code)
	}

	// Unmarshal the response.
	var body struct {
		Username string `json:"username"`
	}
	if err := json.Unmarshal(res.Body(), &body); err != nil {
		cleanup()
		return errors.Wrap(err, "failed to unmarshal Steemit Chat profile")
	}

	// Verify the username.
	if body.Username != settings.Username {
		cleanup()
		return errors.New("Steemit Chat credentials do not match")
	}

	cleanup()
	return nil
}
