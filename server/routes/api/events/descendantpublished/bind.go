package descendantpublished

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/tchap/steemwatch/server/context"
	"github.com/tchap/steemwatch/server/users"

	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	OpTypeAdd    = "add"
	OpTypeRemove = "remove"
)

const (
	SelectorModeAny        = "any"
	SelectorModeDepthLimit = "depthLimit"
)

type Operation struct {
	Type     string    `json:"type"`
	Selector *Selector `json:"selector"`
}

func (op *Operation) Validate() error {
	if op.Selector == nil {
		return errors.New("selector is not set")
	}

	switch op.Type {
	case OpTypeAdd:
		return op.validateAdd()
	case OpTypeRemove:
		return op.validateRemove()
	default:
		return errors.New("unknown operation type: " + op.Type)
	}
}

func (op *Operation) validateAdd() error {
	switch op.Selector.Mode {
	case SelectorModeAny:
		if op.Selector.DepthLimit != 0 {
			return errors.New("selector.depthLimit is set")
		}
	case SelectorModeDepthLimit:
		if op.Selector.DepthLimit == 0 {
			return errors.New("selector.depthLimit must be larger than 0")
		}
	default:
		return errors.New("unknown selection mode: " + op.Selector.Mode)
	}
	return nil
}

func (op *Operation) validateRemove() error {
	if op.Selector.ContentURL == "" {
		return errors.New("selector.contentURL is not set")
	}
	return nil
}

type Selector struct {
	ContentURL string `json:"contentURL"           bson:"contentURL"`
	ContentID  string `json:"-"                    bson:"contentID"`
	Mode       string `json:"mode"                 bson:"mode"`
	DepthLimit uint   `json:"depthLimit,omitempty" bson:"depthLimit,omitempty"`
}

type Document struct {
	OwnerID   bson.ObjectId `bson:"ownerId"`
	Selectors []*Selector   `bson:"selectors"`
}

func Bind(serverCtx *context.Context, root *echo.Group) {
	events := serverCtx.DB.C("events")

	root.GET("/", func(ctx echo.Context) error {
		profile := ctx.Get("user").(*users.User)

		items, err := list(profile, events)
		if err != nil {
			return err
		}

		resp := ctx.Response()
		resp.Header().Set("Content-Type", "application/json")
		return json.NewEncoder(resp.Writer()).Encode(items)
	})

	root.POST("/edit/", func(ctx echo.Context) error {
		profile := ctx.Get("user").(*users.User)

		var op Operation
		if err := json.NewDecoder(ctx.Request().Body()).Decode(&op); err != nil {
			return err
		}
		if err := op.Validate(); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		switch op.Type {
		case OpTypeAdd:
			return add(profile, events, op.Selector)
		case OpTypeRemove:
			return remove(profile, events, op.Selector)
		default:
			panic("unreachable code reached")
		}
	})
}

func add(profile *users.User, events *mgo.Collection, args *Selector) error {
	ownerID := bson.ObjectIdHex(profile.Id)

	// Parse the URL.
	contentURL, err := url.Parse(args.ContentURL)
	if err != nil {
		return err
	}
	if contentURL.Fragment != "" {
		args.ContentID = contentURL.Fragment
	} else {
		parts := strings.SplitN(contentURL.Path, "@", 2)
		if len(parts) != 2 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid content URL")
		}
		args.ContentID = "@" + parts[1]
	}

	// Make sure the thing is not in the array yet.
	query := bson.M{
		"ownerId":             ownerID,
		"kind":                "descendant.published",
		"selectors.contentID": args.ContentID,
	}

	n, err := events.Find(query).Count()
	if err != nil {
		return err
	}
	if n != 0 {
		return echo.NewHTTPError(http.StatusConflict, "content URL already in the list")
	}

	// Insert.
	selector := bson.M{
		"ownerId": ownerID,
		"kind":    "descendant.published",
	}
	update := bson.M{
		"$push": bson.M{
			"selectors": args,
		},
	}
	_, err = events.Upsert(selector, update)
	return err
}

func remove(profile *users.User, events *mgo.Collection, args *Selector) error {
	ownerID := bson.ObjectIdHex(profile.Id)

	// Remove.
	selector := bson.M{
		"ownerId": ownerID,
		"kind":    "descendant.published",
	}
	update := bson.M{
		"$pull": bson.M{
			"selectors": bson.M{
				"contentURL": args.ContentURL,
			},
		},
	}
	return events.Update(selector, update)
}

func list(profile *users.User, events *mgo.Collection) ([]*Selector, error) {
	ownerID := bson.ObjectIdHex(profile.Id)

	// Find.
	query := bson.M{
		"ownerId": ownerID,
		"kind":    "descendant.published",
	}
	var doc Document
	if err := events.Find(query).One(&doc); err != nil {
		if err == mgo.ErrNotFound {
			return []*Selector{}, nil
		}
		return nil, err
	}
	if doc.Selectors == nil {
		return []*Selector{}, nil
	}
	return doc.Selectors, nil
}
