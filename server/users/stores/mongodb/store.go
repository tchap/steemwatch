package mongodb

import (
	"github.com/tchap/steemwatch/server/users"

	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type SocialLink struct {
	UserKey  string `bson:"userKey"`
	UserName string `bson:"userName"`
}

type User struct {
	Id          bson.ObjectId          `bson:"_id"`
	Email       string                 `bson:"email"`
	SocialLinks map[string]*SocialLink `bson:"links,omitempty"`
}

type UserStore struct {
	users *mgo.Collection
}

func NewUserStore(users *mgo.Collection) *UserStore {
	return &UserStore{users}
}

func (store *UserStore) LoadUser(sessionCookie string) (*users.User, error) {
	id := bson.ObjectIdHex(sessionCookie)

	var user User
	err := store.users.FindId(id).One(&user)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to load user profile by ID")
	}

	normalized := &users.User{
		Id:    id.Hex(),
		Email: user.Email,
	}
	if n := len(user.SocialLinks); n != 0 {
		normalized.SocialLinks = make(map[string]*users.SocialLink, n)
		for k, v := range user.SocialLinks {
			normalized.SocialLinks[k] = &users.SocialLink{
				UserKey:  v.UserKey,
				UserName: v.UserName,
			}
		}
	}
	return normalized, nil
}

func (store *UserStore) StoreUser(user *users.User) (string, error) {
	var (
		selector bson.M
		update   bson.M
	)

	switch {
	case user.Email != "":
		selector = bson.M{
			"email": user.Email,
		}
		update = bson.M{
			"$set": selector,
		}

	case len(user.SocialLinks) != 0:
		var (
			serviceName string
			link        *users.SocialLink
		)
		for k, v := range user.SocialLinks {
			serviceName, link = k, v
		}

		selector = bson.M{
			"links." + serviceName + ".userKey": link.UserKey,
		}
		update = bson.M{
			"$set": bson.M{
				"links": bson.M{
					serviceName: bson.M{
						"userKey":  link.UserKey,
						"userName": link.UserName,
					},
				},
			},
		}

	default:
		return "", errors.Errorf("invalid user object: %+v", *user)
	}

	_, err := store.users.Upsert(selector, update)
	if err != nil {
		return "", errors.Wrap(err, "failed to upsert user profile")
	}

	var doc User
	if err := store.users.Find(selector).One(&doc); err != nil {
		return "", errors.Wrap(err, "failed to get user id")
	}

	return doc.Id.Hex(), nil
}
