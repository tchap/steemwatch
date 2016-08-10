package context

import (
	"net/url"

	"github.com/tchap/steemwatch/server/sessions"

	"gopkg.in/mgo.v2"
)

type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentProduction  Environment = "production"
)

type Context struct {
	CanonicalURL   *url.URL
	Env            Environment
	SessionManager *sessions.SessionManager
	DB             *mgo.Database
	SSLEnabled     bool
}
