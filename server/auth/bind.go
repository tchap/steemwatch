package auth

import (
	"net/http"

	"github.com/tchap/steemwatch/server/context"

	"github.com/labstack/echo"
)

type Authenticator interface {
	Authenticate(ctx echo.Context) error
	Callback(ctx echo.Context) (*UserProfile, error)
}

func Bind(serverCtx *context.Context, group *echo.Group, auth Authenticator) {
	group.GET("/", func(ctx echo.Context) error {
		// Make sure the session is not established yet.
		profile, err := serverCtx.SessionManager.GetProfile(ctx)
		if err != nil {
			return err
		}
		if profile != nil {
			return ctx.Redirect(http.StatusTemporaryRedirect, serverCtx.CanonicalURL.String())
		}

		// Proceed to the authentication step.
		return auth.Authenticate(ctx)
	})

	group.GET("/callback/", func(ctx echo.Context) error {
		// Process the callback request.
		profile, err := auth.Callback(ctx)
		if err != nil {
			return err
		}

		// Create a session.
		if err := serverCtx.SessionManager.SetProfile(ctx, profile.AsUser()); err != nil {
			return err
		}

		// Redirect to home.
		return ctx.Redirect(http.StatusTemporaryRedirect, serverCtx.CanonicalURL.String())
	})
}
