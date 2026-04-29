package core

import "time"

// Source represents the entry point that created or modified a task.
type Source string

const (
	SourceCLI Source = "cli"
	SourceMCP Source = "mcp"
	SourceUI  Source = "ui"
	SourceAPI Source = "api"
)

// TaskType differentiates between tasks, bugs, features, and hotfixes.
type TaskType string

const (
	TypeTask    TaskType = "task"
	TypeBug     TaskType = "bug"
	TypeFeature TaskType = "feature"
	TypeHotfix  TaskType = "hotfix"
)

// ValidTaskTypes contains all valid task types.
var ValidTaskTypes = []TaskType{TypeTask, TypeBug, TypeFeature, TypeHotfix}

// TaskStatus represents the lifecycle state of a task.
type TaskStatus string

const (
	StatusTodo  TaskStatus = "todo"
	StatusDoing TaskStatus = "doing"
	StatusDone  TaskStatus = "done"
)

// TaskMetadata holds optional structured data stored in the metadata JSON column.
type TaskMetadata struct {
	ConversationID string `json:"conversation_id,omitempty"`
	SessionID      string `json:"session_id,omitempty"`
}

// Task is the domain representation of a task or bug.
type Task struct {
	ID          int64      `json:"id"`
	Type        TaskType   `json:"type"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Status      TaskStatus `json:"status"`
	Priority    int64      `json:"priority"`
	Category    string     `json:"category,omitempty"`
	IsArchived  bool          `json:"is_archived"`
	Metadata    *TaskMetadata `json:"metadata,omitempty"`
	Source      Source        `json:"source"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedBy   int64      `json:"created_by"`
	AssignedTo  *int64     `json:"assigned_to,omitempty"`
}

// CreateTaskInput holds the parameters for creating a new task.
type CreateTaskInput struct {
	Title          string   `json:"title"`
	Description    string   `json:"description,omitempty"`
	Type           TaskType `json:"type"`
	Priority       int64    `json:"priority"`
	Category       string   `json:"category,omitempty"`
	ConversationID string   `json:"conversation_id,omitempty"`
	SessionID      string   `json:"session_id,omitempty"`
	Source         Source   `json:"source"` // required, set by entry point
}

// UpdateTaskInput holds the parameters for updating a task.
// Nil fields are not updated.
type UpdateTaskInput struct {
	Title       *string     `json:"title,omitempty"`
	Description *string     `json:"description,omitempty"`
	Status      *TaskStatus `json:"status,omitempty"`
	Type        *TaskType   `json:"type,omitempty"`
	Priority    *int64      `json:"priority,omitempty"`
	Category    *string     `json:"category,omitempty"`
}

// ListTasksFilter holds filter parameters for listing tasks.
// Status and Type support comma-separated multi-select (e.g. "todo,doing").
type ListTasksFilter struct {
	Status        string  `json:"status,omitempty"`         // comma-separated: "todo", "todo,doing"
	Type          string  `json:"type,omitempty"`           // comma-separated: "bug", "bug,feature"
	Category      string  `json:"category,omitempty"`       // comma-separated: "backend", "backend,frontend"
	MinPriority   *int64  `json:"min_priority,omitempty"`
	CreatedAfter  *string `json:"created_after,omitempty"`  // ISO 8601 datetime
	CreatedBefore *string `json:"created_before,omitempty"` // ISO 8601 datetime
	Limit         int64   `json:"limit"`
	Offset        int64   `json:"offset"`
}

// ListResult holds paginated task results.
type ListResult struct {
	Tasks      []Task `json:"tasks"`
	Total      int64  `json:"total"`
	Limit      int64  `json:"limit"`
	Offset     int64  `json:"offset"`
	TotalPages int64  `json:"total_pages"`
	Page       int64  `json:"page"`
}

// Category represents a custom label/area of work for tasks.
type Category struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// Commit represents a git commit linked to a task.
type Commit struct {
	Hash      string    `json:"hash"`
	ShortHash string    `json:"short_hash"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Date      time.Time `json:"date"`
}

// TypeCount holds a count per task type.
type TypeCount struct {
	Type  TaskType `json:"type"`
	Count int64    `json:"count"`
}

// ExtendedStats holds detailed statistics for a project.
type ExtendedStats struct {
	ProjectStats
	CompletedThisWeek int64      `json:"completed_this_week"`
	TypeBreakdown     []TypeCount `json:"type_breakdown"`
	RecentCompleted   []Task     `json:"recent_completed"`
	Total             int64      `json:"total"`
}

// Project represents a registered project in the global registry.
type Project struct {
	ID               int64     `json:"id"`
	Slug             string    `json:"slug"`
	Name             string    `json:"name"`
	Path             string    `json:"path"`
	LastSeenAt       time.Time `json:"last_seen_at"`
	CreatedAt        time.Time `json:"created_at"`
	CachedTodoCount  int64     `json:"cached_todo_count"`
	CachedDoingCount int64     `json:"cached_doing_count"`
	CachedTotalCount int64     `json:"cached_total_count"`
}

// ProjectStats holds aggregate statistics for a project.
type ProjectStats struct {
	Todo     int64 `json:"todo"`
	Doing    int64 `json:"doing"`
	Done     int64 `json:"done"`
	BugsOpen int64 `json:"bugs_open"`
}
