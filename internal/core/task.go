package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ahoylog/kvik-tasks/internal/db/generated"
)

// TaskService handles task business logic.
type TaskService struct {
	db      *sql.DB
	queries *generated.Queries
}

// NewTaskService creates a new TaskService with the given database connection.
func NewTaskService(db *sql.DB) *TaskService {
	return &TaskService{
		db:      db,
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
		Category:    toNullString(input.Category),
		Metadata:    serializeMetadata(input.ConversationID, input.SessionID),
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

// List returns tasks matching the given filter (backward-compatible, no pagination info).
func (s *TaskService) List(ctx context.Context, filter ListTasksFilter) ([]Task, error) {
	result, err := s.ListWithCount(ctx, filter)
	if err != nil {
		return nil, err
	}
	return result.Tasks, nil
}

// ListWithCount returns tasks with pagination metadata.
func (s *TaskService) ListWithCount(ctx context.Context, filter ListTasksFilter) (ListResult, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	params := s.buildFilterParams(filter)

	countParams := generated.CountTasksFilteredParams{
		Status:        params.Status,
		Type:          params.Type,
		MinPriority:   params.MinPriority,
		CreatedAfter:  params.CreatedAfter,
		CreatedBefore: params.CreatedBefore,
	}

	total, err := s.queries.CountTasksFiltered(ctx, countParams)
	if err != nil {
		return ListResult{}, fmt.Errorf("counting tasks: %w", err)
	}

	rows, err := s.queries.ListTasksFiltered(ctx, params)
	if err != nil {
		return ListResult{}, fmt.Errorf("listing tasks: %w", err)
	}

	tasks := make([]Task, len(rows))
	for i, row := range rows {
		tasks[i] = taskFromRow(row)
	}

	totalPages := (total + filter.Limit - 1) / filter.Limit
	page := filter.Offset/filter.Limit + 1

	return ListResult{
		Tasks:      tasks,
		Total:      total,
		Limit:      filter.Limit,
		Offset:     filter.Offset,
		TotalPages: totalPages,
		Page:       page,
	}, nil
}

func (s *TaskService) buildFilterParams(filter ListTasksFilter) generated.ListTasksFilteredParams {
	params := generated.ListTasksFilteredParams{
		Lim: filter.Limit,
		Off: filter.Offset,
	}
	if filter.Status != "" {
		params.Status = filter.Status
	}
	if filter.Type != "" {
		params.Type = filter.Type
	}
	if filter.Category != "" {
		params.Category = filter.Category
	}
	if filter.MinPriority != nil {
		params.MinPriority = *filter.MinPriority
	}
	if filter.CreatedAfter != nil {
		params.CreatedAfter = *filter.CreatedAfter
	}
	if filter.CreatedBefore != nil {
		params.CreatedBefore = *filter.CreatedBefore
	}
	return params
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
		Category:    existing.Category,
		Metadata:    existing.Metadata,
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
	if input.Category != nil {
		params.Category = toNullString(*input.Category)
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

// ExtendedStats returns detailed project statistics including velocity and type breakdown.
func (s *TaskService) ExtendedStats(ctx context.Context) (ExtendedStats, error) {
	basic, err := s.Stats(ctx)
	if err != nil {
		return ExtendedStats{}, err
	}

	weekAgo := sql.NullTime{Time: time.Now().AddDate(0, 0, -7), Valid: true}
	completedWeek, _ := s.queries.CountCompletedSince(ctx, weekAgo)

	typeRows, _ := s.queries.CountByType(ctx)
	var types []TypeCount
	var total int64
	for _, r := range typeRows {
		types = append(types, TypeCount{Type: TaskType(r.Type), Count: r.Count})
		total += r.Count
	}

	recentRows, _ := s.queries.RecentCompleted(ctx, 5)
	recent := make([]Task, len(recentRows))
	for i, r := range recentRows {
		recent[i] = taskFromRow(r)
	}

	return ExtendedStats{
		ProjectStats:      basic,
		CompletedThisWeek: completedWeek,
		TypeBreakdown:     types,
		RecentCompleted:   recent,
		Total:             total,
	}, nil
}

// Search performs full-text search across task titles and descriptions.
func (s *TaskService) Search(ctx context.Context, query string, limit int64) ([]Task, error) {
	if query == "" {
		return nil, fmt.Errorf("%w: search query is required", ErrInvalidInput)
	}
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT tasks.id, tasks.type, tasks.title, tasks.description, tasks.status,
		        tasks.created_at, tasks.updated_at, tasks.completed_at,
		        tasks.created_by, tasks.assigned_to, tasks.priority, tasks.source, tasks.metadata,
		        tasks.category, tasks.is_archived
		 FROM tasks
		 JOIN tasks_fts ON tasks.id = tasks_fts.rowid
		 WHERE tasks_fts MATCH ? AND tasks.is_archived = 0
		 ORDER BY rank
		 LIMIT ?`, query, limit)
	if err != nil {
		return nil, fmt.Errorf("searching tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var row generated.Task
		if err := rows.Scan(
			&row.ID, &row.Type, &row.Title, &row.Description, &row.Status,
			&row.CreatedAt, &row.UpdatedAt, &row.CompletedAt,
			&row.CreatedBy, &row.AssignedTo, &row.Priority, &row.Source, &row.Metadata,
			&row.Category, &row.IsArchived,
		); err != nil {
			return nil, fmt.Errorf("scanning search result: %w", err)
		}
		tasks = append(tasks, taskFromRow(row))
	}
	return tasks, nil
}

// Archive marks a completed task as archived.
func (s *TaskService) Archive(ctx context.Context, id int64) (Task, error) {
	row, err := s.queries.ArchiveTask(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return Task{}, fmt.Errorf("task %d: %w (must be done to archive)", id, ErrNotFound)
		}
		return Task{}, fmt.Errorf("archiving task %d: %w", id, err)
	}
	return taskFromRow(row), nil
}

// Unarchive restores an archived task back to done status.
func (s *TaskService) Unarchive(ctx context.Context, id int64) (Task, error) {
	row, err := s.queries.UnarchiveTask(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return Task{}, fmt.Errorf("task %d: %w (not archived)", id, ErrNotFound)
		}
		return Task{}, fmt.Errorf("unarchiving task %d: %w", id, err)
	}
	return taskFromRow(row), nil
}

// ListArchived returns archived tasks with pagination.
func (s *TaskService) ListArchived(ctx context.Context, filter ListTasksFilter) (ListResult, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	params := generated.ListArchivedFilteredParams{
		Lim: filter.Limit,
		Off: filter.Offset,
	}
	countParams := generated.CountArchivedFilteredParams{}

	if filter.Type != "" {
		params.Type = filter.Type
		countParams.Type = filter.Type
	}
	if filter.Category != "" {
		params.Category = filter.Category
		countParams.Category = filter.Category
	}
	if filter.CreatedAfter != nil {
		params.CreatedAfter = *filter.CreatedAfter
		countParams.CreatedAfter = *filter.CreatedAfter
	}
	if filter.CreatedBefore != nil {
		params.CreatedBefore = *filter.CreatedBefore
		countParams.CreatedBefore = *filter.CreatedBefore
	}

	total, err := s.queries.CountArchivedFiltered(ctx, countParams)
	if err != nil {
		return ListResult{}, fmt.Errorf("counting archived tasks: %w", err)
	}

	rows, err := s.queries.ListArchivedFiltered(ctx, params)
	if err != nil {
		return ListResult{}, fmt.Errorf("listing archived tasks: %w", err)
	}

	tasks := make([]Task, len(rows))
	for i, row := range rows {
		tasks[i] = taskFromRow(row)
	}

	totalPages := (total + filter.Limit - 1) / filter.Limit
	page := filter.Offset/filter.Limit + 1

	return ListResult{
		Tasks:      tasks,
		Total:      total,
		Limit:      filter.Limit,
		Offset:     filter.Offset,
		TotalPages: totalPages,
		Page:       page,
	}, nil
}

// ArchiveCompletedBefore archives all done tasks completed before the given time.
func (s *TaskService) ArchiveCompletedBefore(ctx context.Context, before time.Time) (int64, error) {
	result, err := s.queries.ArchiveCompletedBefore(ctx, sql.NullTime{Time: before, Valid: true})
	if err != nil {
		return 0, fmt.Errorf("auto-archiving tasks: %w", err)
	}
	count, _ := result.RowsAffected()
	return count, nil
}

// CountArchived returns the total number of archived tasks.
func (s *TaskService) CountArchived(ctx context.Context) (int64, error) {
	return s.queries.CountArchived(ctx)
}

// taskFromRow converts a generated.Task to a domain Task.
func taskFromRow(row generated.Task) Task {
	t := Task{
		ID:         row.ID,
		Type:       TaskType(row.Type),
		Title:      row.Title,
		Status:     TaskStatus(row.Status),
		Priority:   row.Priority,
		IsArchived: row.IsArchived != 0,
		Source:     Source(row.Source),
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
		CreatedBy:  row.CreatedBy,
	}

	if row.Description.Valid {
		t.Description = row.Description.String
	}
	if row.Category.Valid {
		t.Category = row.Category.String
	}
	if row.CompletedAt.Valid {
		t.CompletedAt = &row.CompletedAt.Time
	}
	if row.AssignedTo.Valid {
		t.AssignedTo = &row.AssignedTo.Int64
	}
	if row.Phase.Valid {
		t.Phase = row.Phase.String
	}
	if row.Metadata.Valid {
		var meta TaskMetadata
		if json.Unmarshal([]byte(row.Metadata.String), &meta) == nil {
			if meta.ConversationID != "" || meta.SessionID != "" {
				t.Metadata = &meta
			}
		}
	}

	return t
}

func serializeMetadata(conversationID, sessionID string) sql.NullString {
	if conversationID == "" && sessionID == "" {
		return sql.NullString{}
	}
	meta := TaskMetadata{
		ConversationID: conversationID,
		SessionID:      sessionID,
	}
	b, err := json.Marshal(meta)
	if err != nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(b), Valid: true}
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
