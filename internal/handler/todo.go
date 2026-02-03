package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/goku-m/starter/internal/middleware"
	"github.com/goku-m/starter/internal/model"
	"github.com/goku-m/starter/internal/model/todo"
	"github.com/goku-m/starter/internal/render"
	"github.com/google/uuid"

	"github.com/goku-m/starter/internal/server"
	"github.com/goku-m/starter/internal/service"
	"github.com/labstack/echo/v4"
)

type TodoHandler struct {
	Handler
	todoService *service.TodoService
}

func NewTodoHandler(s *server.Server, todoService *service.TodoService) *TodoHandler {
	return &TodoHandler{
		Handler:     NewHandler(s),
		todoService: todoService,
	}
}

//PAGE HANDLERS

func (h *TodoHandler) GetTodoPage(c echo.Context) error {
	// 1) Guard service nil (super common during wiring)
	if h.todoService == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "todoService is nil")
	}

	query := &todo.GetTodosQuery{}
	if err := c.Bind(query); err != nil {
		return err
	}

	todos, err := h.todoService.GetTodos(c, query)
	if err != nil {
		return err
	}

	// // 2) Avoid panicking when there are no todos
	// var firstTitle string
	// if todos != nil && len(todos.Data) > 0 {
	// 	firstTitle = todos.Data[0].Title
	// }

	td := &render.TemplateData{
		Data: map[string]interface{}{
			"todos": todos.Data, // could be "" when none exist
			// or pass the whole list:
			// "todos": todos.Data,
		},
	}

	if err := c.Render(http.StatusOK, "home", td); err != nil {
		c.Logger().Error("TodoPage render error: ", err)
		return err
	}

	return nil
}

func (h *TodoHandler) CreateTodoPage(c echo.Context) error {

	if err := c.Render(http.StatusOK, "addTodo", nil); err != nil {
		c.Logger().Error("TodoPage render error: ", err)
		return err
	}

	return nil
}

func (h *TodoHandler) UpdateTodoPage(c echo.Context) error {
	userID := middleware.GetUserID(c)
	idParam := c.Param("id")
	fmt.Println(idParam)
	todoID, err := uuid.Parse(idParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid todo id")
	}

	t, err := h.todoService.GetTodoByID(c, userID, todoID) // you need this service method
	if err != nil {
		return err
	}

	td := &render.TemplateData{
		Data: map[string]interface{}{
			"todo": t,
		},
	}

	if err := c.Render(http.StatusOK, "updateTodo", td); err != nil {
		c.Logger().Error("TodoPage render error: ", err)
		return err
	}

	return nil
}

func (h *TodoHandler) CreateTodo(c echo.Context) error {
	userID := middleware.GetUserID(c)

	title := c.FormValue("title")
	description := c.FormValue("description")
	priority := c.FormValue("priority")

	if strings.TrimSpace(title) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}

	payload := &todo.CreateTodoPayload{
		Title:       title,
		Description: &description, // only if your struct uses *string
	}

	// only set priority if provided (depends on your types)
	if strings.TrimSpace(priority) != "" {
		p := todo.Priority(priority)
		payload.Priority = &p
	}

	if _, err := h.todoService.CreateTodo(c, userID, payload); err != nil {
		return err
	}

	// Redirect back to list (refresh)
	return c.Redirect(http.StatusSeeOther, "/")
}

func (h *TodoHandler) UpdateTodo(c echo.Context) error {
	userID := middleware.GetUserID(c)

	// 1) Get ID from route param
	idParam := c.Param("id")
	todoID, err := uuid.Parse(idParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid todo id")
	}

	// 2) Read form values
	title := strings.TrimSpace(c.FormValue("title"))
	description := strings.TrimSpace(c.FormValue("description"))
	priorityStr := strings.TrimSpace(c.FormValue("priority"))
	statusStr := strings.TrimSpace(c.FormValue("status"))

	// For update, you can decide whether title is required.
	// If you want to require it on the form:
	if title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}

	// 3) Build payload (NOTE: pointer fields!)
	payload := &todo.UpdateTodoPayload{
		ID: todoID,
	}

	// Set Title (payload.Title is *string)
	payload.Title = &title

	// Set Description only if provided (or always set if you want to allow clearing)
	if description != "" {
		payload.Description = &description
	}

	if statusStr != "" {
		s := todo.Status(statusStr)
		payload.Status = &s
	}

	// Set Priority if provided (payload.Priority is *todo.Priority)
	if priorityStr != "" {
		p := todo.Priority(priorityStr)
		switch p {
		case todo.PriorityLow, todo.PriorityMedium, todo.PriorityHigh:
			payload.Priority = &p
		default:
			return echo.NewHTTPError(http.StatusBadRequest, "invalid priority")
		}
	}

	// 4) Update
	if _, err := h.todoService.UpdateTodo(c, userID, payload); err != nil {
		return err
	}

	// 5) Redirect back (refresh)
	return c.Redirect(http.StatusSeeOther, "/")
}

func (h *TodoHandler) DeleteTodo(c echo.Context) error {
	userID := middleware.GetUserID(c)
	id := c.FormValue("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing id")
	}

	todoID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	if err := h.todoService.DeleteTodo(c, userID, todoID); err != nil {
		return err
	}

	return c.Redirect(http.StatusSeeOther, "/")
}

//API HANDLERS

func (h *TodoHandler) CreateTodoAPI(c echo.Context) error {
	return Handle(
		h.Handler,
		func(c echo.Context, payload *todo.CreateTodoPayload) (*todo.Todo, error) {
			userID := middleware.GetUserID(c)
			return h.todoService.CreateTodo(c, userID, payload)
		},
		http.StatusCreated,
		&todo.CreateTodoPayload{},
	)(c)
}

func (h *TodoHandler) GetTodoByID(c echo.Context) error {
	return Handle(
		h.Handler,
		func(c echo.Context, payload *todo.GetTodoByIDPayload) (*todo.PopulatedTodo, error) {
			userID := middleware.GetUserID(c)
			return h.todoService.GetTodoByID(c, userID, payload.ID)
		},
		http.StatusOK,
		&todo.GetTodoByIDPayload{},
	)(c)
}

func (h *TodoHandler) GetTodos(c echo.Context) error {
	return Handle(
		h.Handler,
		func(c echo.Context, query *todo.GetTodosQuery) (*model.PaginatedResponse[todo.PopulatedTodo], error) {
			// userID := middleware.GetUserID(c)
			return h.todoService.GetTodos(c, query)
		},
		http.StatusOK,
		&todo.GetTodosQuery{},
	)(c)
}

func (h *TodoHandler) UpdateTodoAPI(c echo.Context) error {
	return Handle(
		h.Handler,
		func(c echo.Context, payload *todo.UpdateTodoPayload) (*todo.Todo, error) {
			userID := middleware.GetUserID(c)
			return h.todoService.UpdateTodo(c, userID, payload)
		},
		http.StatusOK,
		&todo.UpdateTodoPayload{},
	)(c)
}

func (h *TodoHandler) DeleteTodoAPI(c echo.Context) error {
	return HandleNoContent(
		h.Handler,
		func(c echo.Context, payload *todo.DeleteTodoPayload) error {
			userID := middleware.GetUserID(c)
			return h.todoService.DeleteTodo(c, userID, payload.ID)
		},
		http.StatusNoContent,
		&todo.DeleteTodoPayload{},
	)(c)
}
