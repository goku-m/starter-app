package handler

import (
	"github.com/goku-m/starter/internal/server"
	"github.com/goku-m/starter/internal/service"
)

type Handlers struct {
	Health  *HealthHandler
	Todo    *TodoHandler
	Auth    *AuthHandler
}

func NewHandlers(s *server.Server, services *service.Services) *Handlers {
	return &Handlers{
		Health:  NewHealthHandler(s),
		Todo:    NewTodoHandler(s, services.Todo),
		Auth:    NewAuthHandler(s),
	}
}
