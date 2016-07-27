package sessions

import (
	"strings"

	"github.com/tchap/steemwatch/server/users"

	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"github.com/tchap/securecookie"
)

const SessionCookieName = "SID"

type SessionManager struct {
	cookie *securecookie.SecureCookie
	store  users.Store
	secure bool
}

func NewSessionManager(hashKey, blockKey []byte, store users.Store) (*SessionManager, error) {
	// Make sure the keys are of correct length.
	switch {
	case len(hashKey) != 64:
		return nil, errors.New("the hash key must be 64 bytes long")
	case len(blockKey) != 32:
		return nil, errors.New("the block key must be 32 bytes long")
	}

	// Create a SecureCookie.
	cookie := securecookie.New(hashKey, blockKey)

	// Return a new SessionManager.
	return &SessionManager{
		cookie: cookie,
		store:  store,
	}, nil
}

func (manager *SessionManager) SecureCookie(secure bool) {
	manager.secure = secure
}

func (manager *SessionManager) GetProfile(ctx echo.Context) (*users.User, error) {
	// Get the session cookie value.
	cookie, err := ctx.Cookie(SessionCookieName)
	if err != nil {
		if err == echo.ErrCookieNotFound {
			return nil, nil
		} else {
			return nil, err
		}
	}
	// Empty value is the same as no value at all.
	cookieValue := cookie.Value()
	if cookieValue == "" {
		return nil, nil
	}
	// Replace '0' with '='.
	cookieValue = strings.Replace(cookieValue, "0", "=", -1)

	// Decode the cookie value.
	var session string
	if err := manager.cookie.Decode(SessionCookieName, cookieValue, &session); err != nil {
		if ex, ok := err.(securecookie.Error); ok && ex.IsDecode() {
			manager.ClearProfile(ctx)
			return nil, nil
		} else {
			return nil, err
		}
	}

	// Load the user profile.
	return manager.store.LoadUser(session)
}

func (manager *SessionManager) SetProfile(ctx echo.Context, profile *users.User) error {
	// Store the profile.
	session, err := manager.store.StoreUser(profile)
	if err != nil {
		return err
	}

	// Encode the profile to get the cookie value.
	cookieValue, err := manager.cookie.Encode(SessionCookieName, session)
	if err != nil {
		return err
	}
	// Replace '=' with '0'.
	cookieValue = strings.Replace(cookieValue, "=", "0", -1)

	// Assemble the cookie object.
	cookie := &echo.Cookie{}
	cookie.SetName(SessionCookieName)
	cookie.SetValue(cookieValue)
	cookie.SetHTTPOnly(true)
	cookie.SetSecure(manager.secure)

	// And finally, set the cookie.
	ctx.SetCookie(cookie)
	return nil
}

func (manager *SessionManager) ClearProfile(ctx echo.Context) error {
	// Assemble the cookie object so that the value is empty and expires is in the past.
	cookie := &echo.Cookie{}
	cookie.SetName(SessionCookieName)
	cookie.SetValue("")

	// Set the cookie.
	ctx.SetCookie(cookie)
	return nil
}
