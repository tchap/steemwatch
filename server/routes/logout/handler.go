package logout

import (
	"net/http"

	"github.com/tchap/steemwatch/server/context"

	"github.com/labstack/echo"
)

type Handler struct {
	ctx *context.Context
}

func NewHandlerFunc(serverCtx *context.Context) echo.HandlerFunc {
	handler := &Handler{serverCtx}
	return handler.HandlerFunc
}

func (handler *Handler) HandlerFunc(ctx echo.Context) error {
	profile, err := handler.ctx.SessionManager.GetProfile(ctx)
	if err != nil {
		return err
	}
	if profile != nil {
		if err := handler.ctx.SessionManager.ClearProfile(ctx); err != nil {
			return err
		}
	}
	return ctx.Redirect(http.StatusTemporaryRedirect, handler.ctx.CanonicalURL.String())
}
