package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/goku-m/starter/internal/errs"
	"github.com/goku-m/starter/internal/model"
	"github.com/goku-m/starter/internal/model/todo"
	"github.com/goku-m/starter/internal/server"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type TodoRepository struct {
	server *server.Server
}

func NewTodoRepository(server *server.Server) *TodoRepository {
	return &TodoRepository{server: server}
}

func (r *TodoRepository) CreateTodo(ctx context.Context, userID string, payload *todo.CreateTodoPayload) (*todo.Todo, error) {
	stmt := `
		INSERT INTO
			todos (
				user_id,
				title,
				description,
				priority,
				due_date
							)
		VALUES
			(
				@user_id,
				@title,
				@description,
				@priority,
				@due_date
				
			)
		RETURNING
		*
	`
	priority := todo.PriorityMedium
	if payload.Priority != nil {
		priority = *payload.Priority
	}

	rows, err := r.server.DB.Pool.Query(ctx, stmt, pgx.NamedArgs{
		"user_id":     userID,
		"title":       payload.Title,
		"description": payload.Description,
		"priority":    priority,
		"due_date":    payload.DueDate,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute create todo query for user_id=%s title=%s: %w", userID, payload.Title, err)
	}

	todoItem, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[todo.Todo])
	if err != nil {
		return nil, fmt.Errorf("failed to collect row from table:todos for user_id=%s title=%s: %w", userID, payload.Title, err)
	}

	return &todoItem, nil
}

func (r *TodoRepository) GetTodoByID(ctx context.Context, userID string, todoID uuid.UUID) (*todo.PopulatedTodo, error) {
	stmt := `
	SELECT
		t.*
		
	FROM
		todos t
	
	WHERE
		t.id=@id
		AND t.user_id=@user_id
	GROUP BY
		t.id
		
`

	rows, err := r.server.DB.Pool.Query(ctx, stmt, pgx.NamedArgs{
		"id":      todoID,
		"user_id": userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute get todo by id query for todo_id=%s user_id=%s: %w", todoID.String(), userID, err)
	}

	todoItem, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[todo.PopulatedTodo])
	if err != nil {
		return nil, fmt.Errorf("failed to collect row from table:todos for todo_id=%s user_id=%s: %w", todoID.String(), userID, err)
	}

	return &todoItem, nil
}

func (r *TodoRepository) CheckTodoExists(ctx context.Context, userID string, todoID uuid.UUID) (*todo.Todo, error) {
	stmt := `
		SELECT
			*
		FROM
			todos
		WHERE
			id=@id
			AND user_id=@user_id
	`

	rows, err := r.server.DB.Pool.Query(ctx, stmt, pgx.NamedArgs{
		"id":      todoID,
		"user_id": userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check if todo exists for todo_id=%s user_id=%s: %w", todoID.String(), userID, err)
	}

	todoItem, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[todo.Todo])
	if err != nil {
		return nil, fmt.Errorf("failed to collect row from table:todos for todo_id=%s user_id=%s: %w", todoID.String(), userID, err)
	}

	return &todoItem, nil
}

func (r *TodoRepository) GetTodos(
	ctx context.Context,
	query *todo.GetTodosQuery,
) (*model.PaginatedResponse[todo.PopulatedTodo], error) {

	stmt := `
	SELECT
		t.*
	FROM
		todos t
	`

	args := pgx.NamedArgs{}
	conditions := []string{}

	if query != nil {
		if query.Status != nil {
			conditions = append(conditions, "t.status = @status")
			args["status"] = *query.Status
		}

		if query.Priority != nil {
			conditions = append(conditions, "t.priority = @priority")
			args["priority"] = *query.Priority
		}

		if query.Completed != nil {
			if *query.Completed {
				conditions = append(conditions, "t.status = 'completed'")
			} else {
				conditions = append(conditions, "t.status != 'completed'")
			}
		}

		if query.Search != nil {
			conditions = append(conditions, "(t.title ILIKE @search OR t.description ILIKE @search)")
			args["search"] = "%" + *query.Search + "%"
		}
	}

	if len(conditions) > 0 {
		stmt += " WHERE " + strings.Join(conditions, " AND ")
	}

	// ----- count query -----
	countStmt := "SELECT COUNT(*) FROM todos t"
	if len(conditions) > 0 {
		countStmt += " WHERE " + strings.Join(conditions, " AND ")
	}

	// If server/DB/Pool can be nil in your wiring, you may also want to guard here:
	// if r == nil || r.server == nil || r.server.DB == nil || r.server.DB.Pool == nil { ... }

	var total int
	err := r.server.DB.Pool.QueryRow(ctx, countStmt, args).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count for todos: %w", err)
	}

	// You don't need GROUP BY for "SELECT t.*" from a single table.
	// But keeping your intent: remove it to avoid unnecessary work.
	// stmt += " GROUP BY t.id"

	// ----- safe defaults for pagination -----
	page := 1
	limit := 10
	if query != nil {
		if query.Page != nil && *query.Page > 0 {
			page = *query.Page
		}
		if query.Limit != nil && *query.Limit > 0 {
			limit = *query.Limit
		}
	}

	// ----- safe sorting (whitelist to prevent SQL injection) -----
	sortCol := "created_at"
	orderDesc := true

	allowedSort := map[string]bool{
		"created_at": true,
		"title":      true,
		"priority":   true,
		"status":     true,
		"updated_at": true,
	}

	if query != nil && query.Sort != nil && allowedSort[*query.Sort] {
		sortCol = *query.Sort
	}

	if query != nil && query.Order != nil && strings.EqualFold(*query.Order, "desc") {
		orderDesc = true
	}

	stmt += " ORDER BY t." + sortCol
	if orderDesc {
		stmt += " DESC"
	} else {
		stmt += " ASC"
	}

	// ----- pagination -----
	stmt += " LIMIT @limit OFFSET @offset"
	args["limit"] = limit
	args["offset"] = (page - 1) * limit

	rows, err := r.server.DB.Pool.Query(ctx, stmt, args)
	if err != nil {
		return nil, fmt.Errorf("failed to execute get todos query: %w", err)
	}
	defer rows.Close()

	todos, err := pgx.CollectRows(rows, pgx.RowToStructByName[todo.PopulatedTodo])
	if err != nil {
		// NOTE: CollectRows typically returns nil error with empty slice if no rows,
		// but keeping your fallback logic is fine.
		if errors.Is(err, pgx.ErrNoRows) {
			return &model.PaginatedResponse[todo.PopulatedTodo]{
				Data:       []todo.PopulatedTodo{},
				Page:       page,
				Limit:      limit,
				Total:      0,
				TotalPages: 0,
			}, nil
		}
		return nil, fmt.Errorf("failed to collect rows from table:todos: %w", err)
	}

	return &model.PaginatedResponse[todo.PopulatedTodo]{
		Data:       todos,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: (total + limit - 1) / limit,
	}, nil
}

func (r *TodoRepository) UpdateTodo(ctx context.Context, userID string, payload *todo.UpdateTodoPayload) (*todo.Todo, error) {
	stmt := "UPDATE todos SET "
	args := pgx.NamedArgs{
		"todo_id": payload.ID,
		"user_id": userID,
	}
	setClauses := []string{}

	if payload.Title != nil {
		setClauses = append(setClauses, "title = @title")
		args["title"] = *payload.Title
	}

	if payload.Description != nil {
		setClauses = append(setClauses, "description = @description")
		args["description"] = *payload.Description
	}

	if payload.Status != nil {
		setClauses = append(setClauses, "status = @status")
		args["status"] = *payload.Status

		// Auto-set completed_at when status changes to completed
		if *payload.Status == todo.StatusCompleted {
			setClauses = append(setClauses, "completed_at = @completed_at")
			args["completed_at"] = time.Now()
		} else if *payload.Status != todo.StatusCompleted {
			setClauses = append(setClauses, "completed_at = NULL")
		}
	}

	if payload.Priority != nil {
		setClauses = append(setClauses, "priority = @priority")
		args["priority"] = *payload.Priority
	}

	if len(setClauses) == 0 {
		return nil, errs.NewBadRequestError("no fields to update", false, nil, nil, nil)
	}

	stmt += strings.Join(setClauses, ", ")
	stmt += " WHERE id = @todo_id AND user_id = @user_id RETURNING *"

	rows, err := r.server.DB.Pool.Query(ctx, stmt, args)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	updatedTodo, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[todo.Todo])
	if err != nil {
		return nil, fmt.Errorf("failed to collect row from table:todos: %w", err)
	}

	return &updatedTodo, nil
}

func (r *TodoRepository) DeleteTodo(ctx context.Context, userID string, todoID uuid.UUID) error {
	stmt := `
		DELETE FROM todos
		WHERE
			id=@todo_id
			AND user_id=@user_id
	`

	result, err := r.server.DB.Pool.Exec(ctx, stmt, pgx.NamedArgs{
		"todo_id": todoID,
		"user_id": userID,
	})
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	if result.RowsAffected() == 0 {
		code := "TODO_NOT_FOUND"
		return errs.NewNotFoundError("todo not found", false, &code)
	}

	return nil
}

func (r *TodoRepository) GetTodoStats(ctx context.Context, userID string) (*todo.TodoStats, error) {
	stmt := `
		SELECT
			COUNT(*) AS total,
			COUNT(
				CASE
					WHEN status='draft' THEN 1
				END
			) AS draft,
			COUNT(
				CASE
					WHEN status='active' THEN 1
				END
			) AS active,
			COUNT(
				CASE
					WHEN status='completed' THEN 1
				END
			) AS completed,
			COUNT(
				CASE
					WHEN status='archived' THEN 1
				END
			) AS archived,
			COUNT(
				CASE
					WHEN due_date<NOW()
					AND status!='completed' THEN 1
				END
			) AS overdue
		FROM
			todos
		WHERE
			user_id=@user_id
	`

	rows, err := r.server.DB.Pool.Query(ctx, stmt, pgx.NamedArgs{
		"user_id": userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	stats, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[todo.TodoStats])
	if err != nil {
		return nil, fmt.Errorf("failed to collect row from table:todos: %w", err)
	}

	return &stats, nil
}
