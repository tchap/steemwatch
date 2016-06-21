package mongodb

import (
	"github.com/tchap/steemwatch/server/users"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type User struct {
	Id    bson.ObjectId `bson:"_id"`
	Email string        `bson:"email"`
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
		return nil, err
	}

	return &users.User{
		Id:    id.Hex(),
		Email: user.Email,
	}, nil
}

func (store *UserStore) StoreUser(user *users.User) (string, error) {
	selector := bson.M{
		"email": user.Email,
	}

	update := bson.M{
		"$set": bson.M{
			"email": user.Email,
		},
	}

	_, err := store.users.Upsert(selector, update)
	if err != nil {
		return "", err
	}

	var doc User
	if err := store.users.Find(selector).One(&doc); err != nil {
		return "", err
	}

	return doc.Id.Hex(), nil
}
