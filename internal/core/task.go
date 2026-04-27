package core

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ahoylog/kvik-tasks/internal/db/generated"
)

// TaskService handles task business logic.
type TaskService struct {
	queries *generated.Queries
}

// NewTaskService creates a new TaskService with the given database connection.
func NewTaskService(db *sql.DB) *TaskService {
	return &TaskService{
		queries: generated.New(db),
	}
}

// Create creates a new task.
func (s *TaskService) Create(ctx context.Context, input CreateTaskInput) (Task, error) {
	if input.Title == "" {
		return Task{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if input.Source == "" {
		return Task{}, fmt.Errorf("%w: source is required", ErrInvalidInput)
	}

	if input.Type == "" {
		input.Type = TypeTask
	}

	row, err := s.queries.CreateTask(ctx, generated.CreateTaskParams{
		Type:        string(input.Type),
		Title:       input.Title,
		Description: toNullString(input.Description),
		Status:      string(StatusTodo),
		Priority:    input.Priority,
		Source:      string(input.Source),
		CreatedBy:   1, // default local user
	})
	if err != nil {
		return Task{}, fmt.Errorf("creating task: %w", err)
	}

	return taskFromRow(row), nil
}

// Get returns a task by ID.
func (s *TaskService) Get(ctx context.Context, id int64) (Task, error) {
	row, err := s.queries.GetTask(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return Task{}, fmt.Errorf("task %d: %w", id, ErrNotFound)
		}
		return Task{}, fmt.Errorf("getting task %d: %w", id, err)
	}
	return taskFromRow(row), nil
}

// List returns tasks matching the given filter.
func (s *TaskService) List(ctx context.Context, filter ListTasksFilter) ([]Task, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}

	var rows []generated.Task
	var err error

	switch {
	case filter.Status != nil && filter.Type != nil:
		rows, err = s.queries.ListTasksByStatusAndType(ctx, generated.ListTasksByStatusAndTypeParams{
			Status: string(*filter.Status),
			Type:   string(*filter.Type),
			Limit:  filter.Limit,
		})
	case filter.Status != nil:
		rows, err = s.queries.ListTasksByStatus(ctx, generated.ListTasksByStatusParams{
			Status: string(*filter.Status),
			Limit:  filter.Limit,
		})
	case filter.Type != nil:
		rows, err = s.queries.ListTasksByType(ctx, generated.ListTasksByTypeParams{
			Type:  string(*filter.Type),
			Limit: filter.Limit,
		})
	default:
		rows, err = s.queries.ListTasks(ctx, filter.Limit)
	}

	if err != nil {
		return nil, fmt.Errorf("listing tasks: %w", err)
	}

	tasks := make([]Task, len(rows))
	for i, row := range rows {
		tasks[i] = taskFromRow(row)
	}
	return tasks, nil
}

// Update updates an existing task with partial input.
func (s *TaskService) Update(ctx context.Context, id int64, input UpdateTaskInput) (Task, error) {
	existing, err := s.queries.GetTask(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return Task{}, fmt.Errorf("task %d: %w", id, ErrNotFound)
		}
		return Task{}, fmt.Errorf("getting task %d: %w", id, err)
	}

	params := generated.UpdateTaskParams{
		ID:          id,
		Title:       existing.Title,
		Description: existing.Description,
		Status:      existing.Status,
		Type:        existing.Type,
		Priority:    existing.Priority,
	}

	if input.Title != nil {
		params.Title = *input.Title
	}
	if input.Description != nil {
		params.Description = toNullString(*input.Description)
	}
	if input.Status != nil {
		params.Status = string(*input.Status)
	}
	if input.Type != nil {
		params.Type = string(*input.Type)
	}
	if input.Priority != nil {
		params.Priority = *input.Priority
	}

	row, err := s.queries.UpdateTask(ctx, params)
	if err != nil {
		return Task{}, fmt.Errorf("updating task %d: %w", id, err)
	}

	return taskFromRow(row), nil
}

// Complete marks a task as done.
func (s *TaskService) Complete(ctx context.Context, id int64) (Task, error) {
	row, err := s.queries.CompleteTask(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return Task{}, fmt.Errorf("task %d: %w", id, ErrNotFound)
		}
		return Task{}, fmt.Errorf("completing task %d: %w", id, err)
	}
	return taskFromRow(row), nil
}

// Delete permanently removes a task.
func (s *TaskService) Delete(ctx context.Context, id int64) error {
	// Verify the task exists first
	_, err := s.queries.GetTask(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("task %d: %w", id, ErrNotFound)
		}
		return fmt.Errorf("getting task %d: %w", id, err)
	}

	if err := s.queries.DeleteTask(ctx, id); err != nil {
		return fmt.Errorf("deleting task %d: %w", id, err)
	}
	return nil
}

// Stats returns aggregate statistics.
func (s *TaskService) Stats(ctx context.Context) (ProjectStats, error) {
	counts, err := s.queries.CountTasksByStatus(ctx)
	if err != nil {
		return ProjectStats{}, fmt.Errorf("counting tasks: %w", err)
	}

	bugsOpen, err := s.queries.CountBugsOpen(ctx)
	if err != nil {
		return ProjectStats{}, fmt.Errorf("counting open bugs: %w", err)
	}

	stats := ProjectStats{BugsOpen: bugsOpen}
	for _, c := range counts {
		switch TaskStatus(c.Status) {
		case StatusTodo:
			stats.Todo = c.Count
		case StatusDoing:
			stats.Doing = c.Count
		case StatusDone:
			stats.Done = c.Count
		}
	}

	return stats, nil
}

// taskFromRow converts a generated.Task to a domain Task.
func taskFromRow(row generated.Task) Task {
	t := Task{
		ID:        row.ID,
		Type:      TaskType(row.Type),
		Title:     row.Title,
		Status:    TaskStatus(row.Status),
		Priority:  row.Priority,
		Source:    Source(row.Source),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		CreatedBy: row.CreatedBy,
	}

	if row.Description.Valid {
		t.Description = row.Description.String
	}
	if row.CompletedAt.Valid {
		t.CompletedAt = &row.CompletedAt.Time
	}
	if row.AssignedTo.Valid {
		t.AssignedTo = &row.AssignedTo.Int64
	}

	return t
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
