package todo

import (
	"time"

	"github.com/goku-m/starter/internal/model"
)

type Status string

const (
	StatusDraft     Status = "draft"
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
	StatusArchived  Status = "archived"
)

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

type Todo struct {
	model.Base
	UserID      string     `json:"userId" db:"user_id"`
	Title       string     `json:"title" db:"title"`
	Description *string    `json:"description" db:"description"`
	Status      Status     `json:"status" db:"status"`
	Priority    Priority   `json:"priority" db:"priority"`
	DueDate     *time.Time `json:"dueDate" db:"due_date"`
	CompletedAt *time.Time `json:"completedAt" db:"completed_at"`
	SortOrder   int        `json:"sortOrder" db:"sort_order"`
}


type PopulatedTodo struct {
	Todo
	
}

type TodoStats struct {
	Total     int `json:"total"`
	Draft     int `json:"draft"`
	Active    int `json:"active"`
	Completed int `json:"completed"`
	Archived  int `json:"archived"`
	Overdue   int `json:"overdue"`
}

type UserWeeklyStats struct {
	UserID         string `json:"userId" db:"user_id"`
	CreatedCount   int    `json:"createdCount" db:"created_count"`
	CompletedCount int    `json:"completedCount" db:"completed_count"`
	ActiveCount    int    `json:"activeCount" db:"active_count"`
	OverdueCount   int    `json:"overdueCount" db:"overdue_count"`
}

func (t *Todo) IsOverdue() bool {
	return t.DueDate != nil && t.DueDate.Before(time.Now()) && t.Status != StatusCompleted
}
