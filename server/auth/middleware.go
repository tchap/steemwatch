package auth

import (
	"net/http"

	"github.com/tchap/steemwatch/server/context"

	"github.com/labstack/echo"
)

func Required(serverCtx *context.Context) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			profile, err := serverCtx.SessionManager.GetProfile(ctx)
			if err != nil {
				return err
			}

			if profile == nil {
				return echo.NewHTTPError(http.StatusForbidden, "session not found")
			}

			ctx.Set("user", profile)
			return next(ctx)
		}
	}
}
