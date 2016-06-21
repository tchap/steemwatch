package context

import (
	"net/url"

	"github.com/tchap/steemwatch/server/sessions"

	"gopkg.in/mgo.v2"
)

type Context struct {
	CanonicalURL   *url.URL
	SessionManager *sessions.SessionManager
	DB             *mgo.Database
	SSLEnabled     bool
}
