package sessions

import (
	"github.com/tchap/steemwatch/server/users"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
)

const SessionName = "session"

type SessionManager struct {
	store  users.Store
	secure bool
}

func NewSessionManager(store users.Store) (*SessionManager, error) {
	return &SessionManager{
		store: store,
	}, nil
}

func (manager *SessionManager) SecureCookie(secure bool) {
	manager.secure = secure
}

func (manager *SessionManager) GetProfile(ctx echo.Context) (*users.User, error) {
	// Get the session.
	s, err := session.Get(SessionName, ctx)
	if err != nil {
		return nil, err
	}

	// Make sure this is not a new session.
	if s.IsNew {
		return nil, nil
	}

	// Load the user profile.
	return manager.store.LoadUser(s.Values["id"].(string))
}

func (manager *SessionManager) SetProfile(ctx echo.Context, profile *users.User) error {
	// Store the profile.
	id, err := manager.store.StoreUser(profile)
	if err != nil {
		return err
	}

	// Get a sessions.
	s, err := session.Get(SessionName, ctx)
	if err != nil {
		return err
	}
	s.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   manager.secure,
	}

	// Update and save the session.
	s.Values["id"] = id
	return s.Save(ctx.Request(), ctx.Response())
}

func (manager *SessionManager) ClearProfile(ctx echo.Context) error {
	s, err := session.Get(SessionName, ctx)
	if err != nil {
		return err
	}
	if s.IsNew {
		return nil
	}

	s.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   manager.secure,
	}
	return s.Save(ctx.Request(), ctx.Response())
}
