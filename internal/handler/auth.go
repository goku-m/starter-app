package handler

import (
	"net/http"

	"github.com/goku-m/starter/internal/render"
	"github.com/goku-m/starter/internal/server"
	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	Handler
}

func NewAuthHandler(s *server.Server) *AuthHandler {
	return &AuthHandler{
		Handler: NewHandler(s),
	}
}

func (h *Handler) LoginPage(c echo.Context) error {
	td := &render.TemplateData{
		Data: map[string]interface{}{
			"key": "",
		},
	}

	// Will look for ./views/home/index.jet
	err := c.Render(http.StatusOK, "login", td)
	if err != nil {
		c.Logger().Error("LoginPage render error: ", err)
	}
	return err
}
