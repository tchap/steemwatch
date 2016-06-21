package server

import (
	"github.com/gorilla/securecookie"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
)

type CookieConfig struct {
	Id       string `bson:"_id"`
	HashKey  []byte `bson:"hashKey"`
	BlockKey []byte `bson:"blockKey"`
}

func getSecureCookieKeys(mongo *mgo.Database) (hashKey, blockKey []byte, err error) {
	var doc CookieConfig
	err = mongo.C("configuration").FindId("SecureCookie").One(&doc)
	switch err {
	case nil:
		return doc.HashKey, doc.BlockKey, nil
	case mgo.ErrNotFound:
		// Generate new cookie keys.
		hashRandomKey := securecookie.GenerateRandomKey(64)
		blockRandomKey := securecookie.GenerateRandomKey(32)
		if hashRandomKey == nil || blockRandomKey == nil {
			return nil, nil, errors.New("failed to generate cookie keys")
		}

		// Store them in the database.
		doc := &CookieConfig{
			Id:       "SecureCookie",
			HashKey:  hashRandomKey,
			BlockKey: blockRandomKey,
		}
		if err := mongo.C("configuration").Insert(doc); err != nil {
			return nil, nil, errors.Wrap(err, "failed to store securecookie keys")
		}

		return hashRandomKey, blockRandomKey, nil
	default:
		return nil, nil, errors.Wrap(err, "failed to load securecookie configuration")
	}
}
