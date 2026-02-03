package router

import (
	"github.com/goku-m/starter/internal/handler"
	"github.com/goku-m/starter/internal/middleware"
	"github.com/labstack/echo/v4"
)

func registerTodoRoutes(r *echo.Group, h *handler.TodoHandler, auth *middleware.AuthMiddleware) {
	// Todo operations
	todos := r.Group("/todos")
	todos.Use(auth.RequireAuthIP)

	// Collection operations
	todos.POST("/create", h.CreateTodo)
	todos.GET("", h.GetTodos)
	todos.POST("/delete", h.DeleteTodo)
	todos.POST("/update/:id", h.UpdateTodo)

	// Individual todo operations
	// dynamicTodo := todos.Group("/:id")
	// dynamicTodo.GET("", h.GetTodoByID)
	// dynamicTodo.PATCH("", h.UpdateTodo)
	// dynamicTodo.DELETE("", h.DeleteTodo)

}
