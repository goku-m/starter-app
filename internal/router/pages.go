package router

import (
	"github.com/goku-m/starter/internal/handler"
	"github.com/goku-m/starter/internal/middleware"

	"github.com/labstack/echo/v4"
)

func registerPagesRoutes(r *echo.Echo, h *handler.Handlers, auth *middleware.AuthMiddleware) {

	r.GET("/login", h.Auth.LoginPage)
	r.GET("/", h.Todo.GetTodoPage)
	r.GET("/create", h.Todo.CreateTodoPage)
	r.Use(auth.RequireAuthIP)
	r.GET("/update/:id", h.Todo.UpdateTodoPage)
}
