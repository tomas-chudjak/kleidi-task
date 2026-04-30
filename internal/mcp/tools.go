package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ahoylog/kvik-tasks/internal/config"
	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Tool input types

type TaskCreateInput struct {
	Project        string `json:"project" jsonschema:"project slug or 'current'"`
	Title          string `json:"title" jsonschema:"task title"`
	Type           string `json:"type,omitempty" jsonschema:"work item type (built-in: task, bug, feature, hotfix; or any custom type)"`
	Description    string `json:"description,omitempty" jsonschema:"task description"`
	Priority       int64  `json:"priority,omitempty" jsonschema:"higher number means higher priority"`
	Category       string `json:"category,omitempty" jsonschema:"category/area of work (e.g. backend, frontend, design)"`
	ConversationID string `json:"conversation_id,omitempty" jsonschema:"ID of the AI conversation that created this task"`
	SessionID      string `json:"session_id,omitempty" jsonschema:"ID of the MCP session"`
}

type TaskListInput struct {
	Project       string `json:"project,omitempty" jsonschema:"project slug or 'current'"`
	Status        string `json:"status,omitempty" jsonschema:"filter by status,enum=todo,enum=doing,enum=done"`
	Type          string `json:"type,omitempty" jsonschema:"filter by type,enum=task,enum=bug,enum=feature,enum=hotfix (or custom types)"`
	Category      string `json:"category,omitempty" jsonschema:"filter by category (comma-separated for multi-select)"`
	MinPriority   *int64 `json:"min_priority,omitempty" jsonschema:"minimum priority filter"`
	CreatedAfter  string `json:"created_after,omitempty" jsonschema:"filter tasks created after this date (ISO 8601)"`
	CreatedBefore string `json:"created_before,omitempty" jsonschema:"filter tasks created before this date (ISO 8601)"`
	Limit         int64  `json:"limit,omitempty" jsonschema:"max results (default 50)"`
	Offset        int64  `json:"offset,omitempty" jsonschema:"offset for pagination (default 0)"`
}

type TaskGetInput struct {
	Project string `json:"project,omitempty" jsonschema:"project slug or 'current'"`
	ID      int64  `json:"id" jsonschema:"task ID"`
}

type TaskUpdateInput struct {
	Project     string `json:"project,omitempty" jsonschema:"project slug or 'current'"`
	ID          int64  `json:"id" jsonschema:"task ID"`
	Title       string `json:"title,omitempty" jsonschema:"new title"`
	Description string `json:"description,omitempty" jsonschema:"new description"`
	Status      string `json:"status,omitempty" jsonschema:"new status,enum=todo,enum=doing,enum=done"`
	Type        string `json:"type,omitempty" jsonschema:"new type (task, bug, feature, hotfix, or custom)"`
	Priority    *int64 `json:"priority,omitempty" jsonschema:"new priority"`
	Category    string `json:"category,omitempty" jsonschema:"new category"`
}

type TaskCompleteInput struct {
	Project string `json:"project,omitempty" jsonschema:"project slug or 'current'"`
	ID      int64  `json:"id" jsonschema:"task ID"`
}

type TaskDeleteInput struct {
	Project string `json:"project,omitempty" jsonschema:"project slug or 'current'"`
	ID      int64  `json:"id" jsonschema:"task ID"`
}

type TaskArchiveInput struct {
	Project string `json:"project,omitempty" jsonschema:"project slug or 'current'"`
	ID      int64  `json:"id" jsonschema:"task ID"`
}

type TaskSearchInput struct {
	Project string `json:"project,omitempty" jsonschema:"project slug or 'current'"`
	Query   string `json:"query" jsonschema:"search query (FTS5 syntax)"`
	Limit   int64  `json:"limit,omitempty" jsonschema:"max results (default 20)"`
}

type CategoryListInput struct {
	Project string `json:"project,omitempty" jsonschema:"project slug or 'current'"`
}

type CategoryCreateInput struct {
	Project string `json:"project" jsonschema:"project slug or 'current'"`
	Name    string `json:"name" jsonschema:"category name (e.g. backend, frontend, design)"`
	Color   string `json:"color,omitempty" jsonschema:"hex color (default: #8a8dab)"`
}

type CategoryListOutput struct {
	Categories []core.Category `json:"categories"`
}

type CategoryOutput struct {
	Category core.Category `json:"category"`
}

type TaskBulkUpdateInput struct {
	Project  string `json:"project,omitempty" jsonschema:"project slug or 'current'"`
	IDs      []int64 `json:"ids" jsonschema:"list of task IDs to update"`
	Status   string `json:"status,omitempty" jsonschema:"new status for all,enum=todo,enum=doing,enum=done"`
	Type     string `json:"type,omitempty" jsonschema:"new type for all (task, bug, feature, hotfix, or custom)"`
	Priority *int64 `json:"priority,omitempty" jsonschema:"new priority for all"`
	Category string `json:"category,omitempty" jsonschema:"new category for all"`
}

type TaskBulkCompleteInput struct {
	Project string  `json:"project,omitempty" jsonschema:"project slug or 'current'"`
	IDs     []int64 `json:"ids" jsonschema:"list of task IDs to mark as done"`
}

type BulkOutput struct {
	Updated int      `json:"updated"`
	Errors  []string `json:"errors,omitempty"`
}

type ExtendedStatsOutput struct {
	Stats core.ExtendedStats `json:"stats"`
}

type ProjectBackupInput struct {
	Slug   string `json:"slug,omitempty" jsonschema:"project slug (default: current)"`
	Output string `json:"output,omitempty" jsonschema:"output path (default: .tasks/backups/tasks_<timestamp>.db)"`
}

type ProjectBackupOutput struct {
	Path string `json:"path"`
}

type ProjectStatsInput struct {
	Slug string `json:"slug,omitempty" jsonschema:"project slug (default: current)"`
}

// Tool output types — used for structured content

type TaskOutput struct {
	Task core.Task `json:"task"`
}

type TaskListOutput struct {
	Tasks []core.Task `json:"tasks"`
	Count int         `json:"count"`
}

type DeleteOutput struct {
	Success bool `json:"success"`
}

type ProjectListOutput struct {
	Projects []core.Project `json:"projects"`
}

type ProjectStatsOutput struct {
	Stats core.ProjectStats `json:"stats"`
}

type ProjectCurrentOutput struct {
	Project *core.Project `json:"project,omitempty"`
	Path    string        `json:"path,omitempty"`
}

type TaskAdvanceInput struct {
	Project string `json:"project,omitempty" jsonschema:"project slug or 'current'"`
	ID      int64  `json:"id" jsonschema:"task ID to advance to next phase"`
}

type TaskAdvanceOutput struct {
	Result core.AdvanceResult `json:"result"`
}

type TaskSuggestInput struct {
	Project string `json:"project,omitempty" jsonschema:"project slug or 'current'"`
}

type TaskSuggestOutput struct {
	Suggestions []core.Suggestion `json:"suggestions"`
	Count       int               `json:"count"`
	New         int               `json:"new"`
	Duplicates  int               `json:"duplicates"`
}

func (s *Server) registerTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "task_create",
		Description: "Create a new task or bug in a project",
	}, s.taskCreate)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "task_list",
		Description: "List tasks with optional filters by project, status, and type",
	}, s.taskList)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "task_search",
		Description: "Search tasks by title or description (full-text search)",
	}, s.taskSearch)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "task_get",
		Description: "Get detailed information about a specific task",
	}, s.taskGet)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "task_update",
		Description: "Update an existing task (title, description, status, type, priority)",
	}, s.taskUpdate)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "task_complete",
		Description: "Mark a task as done",
	}, s.taskComplete)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "task_delete",
		Description: "Permanently delete a task",
	}, s.taskDelete)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "task_archive",
		Description: "Archive a completed task (removes it from active views)",
	}, s.taskArchive)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "task_unarchive",
		Description: "Unarchive a task (restore it back to done status)",
	}, s.taskUnarchive)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "category_list",
		Description: "List all categories for a project",
	}, s.categoryList)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "category_create",
		Description: "Create a new category (area of work like backend, frontend, design)",
	}, s.categoryCreate)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "task_bulk_update",
		Description: "Update multiple tasks at once (change status, type, priority, or category for a list of task IDs)",
	}, s.taskBulkUpdate)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "task_bulk_complete",
		Description: "Mark multiple tasks as done at once",
	}, s.taskBulkComplete)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "project_stats_extended",
		Description: "Get extended project statistics: velocity (completed this week), type breakdown, recent completions",
	}, s.projectStatsExtended)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "project_backup",
		Description: "Create a consistent backup of the project's task database",
	}, s.projectBackup)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "project_list",
		Description: "List all registered projects",
	}, s.projectList)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "project_current",
		Description: "Get the current project based on working directory",
	}, s.projectCurrent)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "project_stats",
		Description: "Get task statistics for a project (todo/doing/done counts, open bugs)",
	}, s.projectStats)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "task_advance",
		Description: "Advance a task to the next workflow phase. Returns suggested skills to execute for the new phase.",
	}, s.taskAdvance)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "task_suggest",
		Description: "Scan source code for TODO/FIXME/HACK/XXX comments and suggest new tasks. Checks for duplicates against existing tasks.",
	}, s.taskSuggest)
}

func (s *Server) taskCreate(ctx context.Context, req *mcp.CallToolRequest, input TaskCreateInput) (*mcp.CallToolResult, TaskOutput, error) {
	taskService, err := s.resolveTaskService(input.Project)
	if err != nil {
		return nil, TaskOutput{}, err
	}

	title := input.Title
	taskType := core.TypeTask
	if input.Type != "" {
		taskType = core.TaskType(input.Type)
	} else {
		// No explicit type — try prefix detection
		taskType, title = core.DetectTypeFromTitle(title, core.TypeTask)
	}

	// Look up template for this type to provide as AI instruction
	var templateHint string
	projectPath, _ := s.resolveProjectPath(input.Project)
	if projectPath != "" && input.Description == "" {
		tplService, err := s.projectService.TemplateServiceFor(projectPath)
		if err == nil {
			tpl, err := tplService.GetByType(ctx, string(taskType))
			if err == nil && tpl.Description != "" {
				templateHint = tpl.Description
			}
		}
	}

	task, err := taskService.Create(ctx, core.CreateTaskInput{
		Title:          title,
		Description:    input.Description,
		Type:           taskType,
		Priority:       input.Priority,
		Category:       input.Category,
		ConversationID: input.ConversationID,
		SessionID:      input.SessionID,
		Source:         core.SourceMCP,
	})
	if err != nil {
		return nil, TaskOutput{}, err
	}

	text := fmt.Sprintf("Created %s #%d: %s", task.Type, task.ID, task.Title)
	if templateHint != "" {
		text += fmt.Sprintf("\n\nTemplate for %s — please generate a description following this structure and update the task:\n%s", taskType, templateHint)
	}

	return textResult(text), TaskOutput{Task: task}, nil
}

func (s *Server) taskList(ctx context.Context, req *mcp.CallToolRequest, input TaskListInput) (*mcp.CallToolResult, TaskListOutput, error) {
	taskService, err := s.resolveTaskService(input.Project)
	if err != nil {
		return nil, TaskListOutput{}, err
	}

	filter := core.ListTasksFilter{Limit: input.Limit, Offset: input.Offset}
	if input.Status != "" {
		filter.Status = input.Status
	}
	if input.Type != "" {
		filter.Type = input.Type
	}
	if input.Category != "" {
		filter.Category = input.Category
	}
	if input.MinPriority != nil {
		filter.MinPriority = input.MinPriority
	}
	if input.CreatedAfter != "" {
		filter.CreatedAfter = &input.CreatedAfter
	}
	if input.CreatedBefore != "" {
		filter.CreatedBefore = &input.CreatedBefore
	}

	result, err := taskService.ListWithCount(ctx, filter)
	if err != nil {
		return nil, TaskListOutput{}, err
	}

	text := formatTaskList(result.Tasks)
	if result.TotalPages > 1 {
		text += fmt.Sprintf("\nPage %d/%d (total: %d)", result.Page, result.TotalPages, result.Total)
	}
	return textResult(text), TaskListOutput{Tasks: result.Tasks, Count: len(result.Tasks)}, nil
}

func (s *Server) taskSearch(ctx context.Context, req *mcp.CallToolRequest, input TaskSearchInput) (*mcp.CallToolResult, TaskListOutput, error) {
	taskService, err := s.resolveTaskService(input.Project)
	if err != nil {
		return nil, TaskListOutput{}, err
	}

	tasks, err := taskService.Search(ctx, input.Query, input.Limit)
	if err != nil {
		return nil, TaskListOutput{}, err
	}

	text := formatTaskList(tasks)
	return textResult(text), TaskListOutput{Tasks: tasks, Count: len(tasks)}, nil
}

func (s *Server) taskGet(ctx context.Context, req *mcp.CallToolRequest, input TaskGetInput) (*mcp.CallToolResult, TaskOutput, error) {
	taskService, err := s.resolveTaskService(input.Project)
	if err != nil {
		return nil, TaskOutput{}, err
	}

	task, err := taskService.Get(ctx, input.ID)
	if err != nil {
		return nil, TaskOutput{}, err
	}

	// Include workflow context
	text := formatTask(task)
	projectPath, _ := s.resolveProjectPath(input.Project)
	if projectPath != "" {
		wfService, err := s.projectService.WorkflowServiceFor(projectPath)
		if err == nil {
			wc, err := wfService.GetContext(ctx, task)
			if err == nil && len(wc.Phases) > 1 {
				text += fmt.Sprintf("\n\nWorkflow: %s (phase %d/%d)", wc.CurrentPhase, wc.PhaseIndex+1, len(wc.Phases))
				if wc.CurrentPrompt != "" {
					text += fmt.Sprintf("\nPhase instruction: %s", wc.CurrentPrompt)
				}
				if wc.NextPhase != "" {
					text += fmt.Sprintf("\nNext phase: %s", wc.NextPhase)
				}
			}
		}
	}

	return textResult(text), TaskOutput{Task: task}, nil
}

func (s *Server) taskUpdate(ctx context.Context, req *mcp.CallToolRequest, input TaskUpdateInput) (*mcp.CallToolResult, TaskOutput, error) {
	taskService, err := s.resolveTaskService(input.Project)
	if err != nil {
		return nil, TaskOutput{}, err
	}

	updateInput := core.UpdateTaskInput{}
	if input.Title != "" {
		updateInput.Title = &input.Title
	}
	if input.Description != "" {
		updateInput.Description = &input.Description
	}
	if input.Status != "" {
		s := core.TaskStatus(input.Status)
		updateInput.Status = &s
	}
	if input.Type != "" {
		t := core.TaskType(input.Type)
		updateInput.Type = &t
	}
	if input.Priority != nil {
		updateInput.Priority = input.Priority
	}
	if input.Category != "" {
		updateInput.Category = &input.Category
	}

	task, err := taskService.Update(ctx, input.ID, updateInput)
	if err != nil {
		return nil, TaskOutput{}, err
	}

	return textResult(fmt.Sprintf("Updated task #%d: %s [%s]", task.ID, task.Title, task.Status)), TaskOutput{Task: task}, nil
}

func (s *Server) taskComplete(ctx context.Context, req *mcp.CallToolRequest, input TaskCompleteInput) (*mcp.CallToolResult, TaskOutput, error) {
	taskService, err := s.resolveTaskService(input.Project)
	if err != nil {
		return nil, TaskOutput{}, err
	}

	task, err := taskService.Complete(ctx, input.ID)
	if err != nil {
		return nil, TaskOutput{}, err
	}

	return textResult(fmt.Sprintf("Completed task #%d: %s", task.ID, task.Title)), TaskOutput{Task: task}, nil
}

func (s *Server) taskDelete(ctx context.Context, req *mcp.CallToolRequest, input TaskDeleteInput) (*mcp.CallToolResult, DeleteOutput, error) {
	taskService, err := s.resolveTaskService(input.Project)
	if err != nil {
		return nil, DeleteOutput{}, err
	}

	if err := taskService.Delete(ctx, input.ID); err != nil {
		return nil, DeleteOutput{}, err
	}

	return textResult(fmt.Sprintf("Deleted task #%d", input.ID)), DeleteOutput{Success: true}, nil
}

func (s *Server) taskArchive(ctx context.Context, req *mcp.CallToolRequest, input TaskArchiveInput) (*mcp.CallToolResult, TaskOutput, error) {
	taskService, err := s.resolveTaskService(input.Project)
	if err != nil {
		return nil, TaskOutput{}, err
	}

	task, err := taskService.Archive(ctx, input.ID)
	if err != nil {
		return nil, TaskOutput{}, err
	}

	return textResult(fmt.Sprintf("Archived task #%d: %s", task.ID, task.Title)), TaskOutput{Task: task}, nil
}

func (s *Server) taskUnarchive(ctx context.Context, req *mcp.CallToolRequest, input TaskArchiveInput) (*mcp.CallToolResult, TaskOutput, error) {
	taskService, err := s.resolveTaskService(input.Project)
	if err != nil {
		return nil, TaskOutput{}, err
	}

	task, err := taskService.Unarchive(ctx, input.ID)
	if err != nil {
		return nil, TaskOutput{}, err
	}

	return textResult(fmt.Sprintf("Unarchived task #%d: %s (back to done)", task.ID, task.Title)), TaskOutput{Task: task}, nil
}

func (s *Server) projectBackup(ctx context.Context, req *mcp.CallToolRequest, input ProjectBackupInput) (*mcp.CallToolResult, ProjectBackupOutput, error) {
	var projectPath string
	if input.Slug != "" && input.Slug != "current" {
		project, err := s.projectService.GetBySlug(input.Slug)
		if err != nil {
			return nil, ProjectBackupOutput{}, err
		}
		projectPath = project.Path
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, ProjectBackupOutput{}, fmt.Errorf("getting working directory: %w", err)
		}
		projectPath, err = s.projectService.DetectProject(cwd)
		if err != nil {
			return nil, ProjectBackupOutput{}, err
		}
	}

	backupPath, err := s.projectService.Backup(projectPath, input.Output)
	if err != nil {
		return nil, ProjectBackupOutput{}, err
	}

	return textResult(fmt.Sprintf("Backup created: %s", backupPath)), ProjectBackupOutput{Path: backupPath}, nil
}

func (s *Server) projectList(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, ProjectListOutput, error) {
	projects, err := s.projectService.List()
	if err != nil {
		return nil, ProjectListOutput{}, err
	}

	text := fmt.Sprintf("Found %d project(s):\n", len(projects))
	for _, p := range projects {
		text += fmt.Sprintf("  %s — %s (%s)\n", p.Slug, p.Name, p.Path)
	}

	return textResult(text), ProjectListOutput{Projects: projects}, nil
}

func (s *Server) projectCurrent(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, ProjectCurrentOutput, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, ProjectCurrentOutput{}, fmt.Errorf("getting working directory: %w", err)
	}

	projectPath, err := s.projectService.DetectProject(cwd)
	if err != nil {
		return textResult("No project found in current directory. Run 'kvt init' first."), ProjectCurrentOutput{}, nil
	}

	// Try to find project in registry
	projects, _ := s.projectService.List()
	for _, p := range projects {
		if p.Path == projectPath {
			return textResult(fmt.Sprintf("Current project: %s (%s) at %s", p.Name, p.Slug, p.Path)),
				ProjectCurrentOutput{Project: &p, Path: projectPath}, nil
		}
	}

	return textResult(fmt.Sprintf("Project found at %s (not in registry)", projectPath)),
		ProjectCurrentOutput{Path: projectPath}, nil
}

func (s *Server) projectStats(ctx context.Context, req *mcp.CallToolRequest, input ProjectStatsInput) (*mcp.CallToolResult, ProjectStatsOutput, error) {
	var projectPath string
	if input.Slug != "" {
		project, err := s.projectService.GetBySlug(input.Slug)
		if err != nil {
			return nil, ProjectStatsOutput{}, err
		}
		projectPath = project.Path
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, ProjectStatsOutput{}, fmt.Errorf("getting working directory: %w", err)
		}
		projectPath, err = s.projectService.DetectProject(cwd)
		if err != nil {
			return nil, ProjectStatsOutput{}, err
		}
	}

	taskService, err := s.projectService.TaskServiceFor(projectPath)
	if err != nil {
		return nil, ProjectStatsOutput{}, err
	}

	stats, err := taskService.Stats(ctx)
	if err != nil {
		return nil, ProjectStatsOutput{}, err
	}

	text := fmt.Sprintf("Todo: %d | Doing: %d | Done: %d | Bugs open: %d",
		stats.Todo, stats.Doing, stats.Done, stats.BugsOpen)

	return textResult(text), ProjectStatsOutput{Stats: stats}, nil
}

// Helper methods

func (s *Server) resolveProjectPath(project string) (string, error) {
	if project != "" && project != "current" {
		p, err := s.projectService.GetBySlug(project)
		if err != nil {
			return "", err
		}
		return p.Path, nil
	}
	// Try cwd first
	cwd, err := os.Getwd()
	if err == nil {
		projectPath, err := s.projectService.DetectProject(cwd)
		if err == nil && projectPath != "" {
			return projectPath, nil
		}
	}
	// Fall back to default project from config
	cfg := config.LoadGlobal()
	if cfg.DefaultProject != "" {
		p, slugErr := s.projectService.GetBySlug(cfg.DefaultProject)
		if slugErr == nil {
			return p.Path, nil
		}
	}
	return "", core.ErrNoProject
}

func (s *Server) resolveTaskService(project string) (*core.TaskService, error) {
	projectPath, err := s.resolveProjectPath(project)
	if err != nil {
		return nil, err
	}

	return s.projectService.TaskServiceFor(projectPath)
}

func (s *Server) taskBulkUpdate(ctx context.Context, req *mcp.CallToolRequest, input TaskBulkUpdateInput) (*mcp.CallToolResult, BulkOutput, error) {
	taskService, err := s.resolveTaskService(input.Project)
	if err != nil {
		return nil, BulkOutput{}, err
	}

	var updated int
	var errors []string
	for _, id := range input.IDs {
		upd := core.UpdateTaskInput{}
		if input.Status != "" {
			s := core.TaskStatus(input.Status)
			upd.Status = &s
		}
		if input.Type != "" {
			t := core.TaskType(input.Type)
			upd.Type = &t
		}
		if input.Priority != nil {
			upd.Priority = input.Priority
		}
		if input.Category != "" {
			upd.Category = &input.Category
		}
		_, err := taskService.Update(ctx, id, upd)
		if err != nil {
			errors = append(errors, fmt.Sprintf("#%d: %v", id, err))
		} else {
			updated++
		}
	}

	text := fmt.Sprintf("Updated %d/%d tasks.", updated, len(input.IDs))
	if len(errors) > 0 {
		text += fmt.Sprintf(" Errors: %d", len(errors))
	}
	return textResult(text), BulkOutput{Updated: updated, Errors: errors}, nil
}

func (s *Server) taskBulkComplete(ctx context.Context, req *mcp.CallToolRequest, input TaskBulkCompleteInput) (*mcp.CallToolResult, BulkOutput, error) {
	taskService, err := s.resolveTaskService(input.Project)
	if err != nil {
		return nil, BulkOutput{}, err
	}

	var updated int
	var errors []string
	for _, id := range input.IDs {
		_, err := taskService.Complete(ctx, id)
		if err != nil {
			errors = append(errors, fmt.Sprintf("#%d: %v", id, err))
		} else {
			updated++
		}
	}

	text := fmt.Sprintf("Completed %d/%d tasks.", updated, len(input.IDs))
	return textResult(text), BulkOutput{Updated: updated, Errors: errors}, nil
}

func (s *Server) projectStatsExtended(ctx context.Context, req *mcp.CallToolRequest, input ProjectStatsInput) (*mcp.CallToolResult, ExtendedStatsOutput, error) {
	var projectPath string
	if input.Slug != "" && input.Slug != "current" {
		project, err := s.projectService.GetBySlug(input.Slug)
		if err != nil {
			return nil, ExtendedStatsOutput{}, err
		}
		projectPath = project.Path
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, ExtendedStatsOutput{}, fmt.Errorf("getting working directory: %w", err)
		}
		projectPath, err = s.projectService.DetectProject(cwd)
		if err != nil {
			return nil, ExtendedStatsOutput{}, err
		}
	}

	taskService, err := s.projectService.TaskServiceFor(projectPath)
	if err != nil {
		return nil, ExtendedStatsOutput{}, err
	}

	stats, err := taskService.ExtendedStats(ctx)
	if err != nil {
		return nil, ExtendedStatsOutput{}, err
	}

	text := fmt.Sprintf("Todo: %d | Doing: %d | Done: %d | Bugs: %d\n", stats.Todo, stats.Doing, stats.Done, stats.BugsOpen)
	text += fmt.Sprintf("Completed this week: %d | Total: %d\n", stats.CompletedThisWeek, stats.Total)
	if len(stats.TypeBreakdown) > 0 {
		text += "Type breakdown: "
		for i, tc := range stats.TypeBreakdown {
			if i > 0 {
				text += ", "
			}
			text += fmt.Sprintf("%s=%d", tc.Type, tc.Count)
		}
		text += "\n"
	}
	if len(stats.RecentCompleted) > 0 {
		text += "Recent completions:\n"
		for _, t := range stats.RecentCompleted {
			text += fmt.Sprintf("  #%d %s", t.ID, t.Title)
			if t.CompletedAt != nil {
				text += fmt.Sprintf(" (%s)", t.CompletedAt.Format("Jan 2 15:04"))
			}
			text += "\n"
		}
	}

	return textResult(text), ExtendedStatsOutput{Stats: stats}, nil
}

func (s *Server) categoryList(ctx context.Context, req *mcp.CallToolRequest, input CategoryListInput) (*mcp.CallToolResult, CategoryListOutput, error) {
	catService, err := s.resolveCategoryService(input.Project)
	if err != nil {
		return nil, CategoryListOutput{}, err
	}

	categories, err := catService.List(ctx)
	if err != nil {
		return nil, CategoryListOutput{}, err
	}

	text := fmt.Sprintf("Found %d category(ies):\n", len(categories))
	for _, c := range categories {
		text += fmt.Sprintf("  %s (%s)\n", c.Name, c.Color)
	}
	return textResult(text), CategoryListOutput{Categories: categories}, nil
}

func (s *Server) categoryCreate(ctx context.Context, req *mcp.CallToolRequest, input CategoryCreateInput) (*mcp.CallToolResult, CategoryOutput, error) {
	catService, err := s.resolveCategoryService(input.Project)
	if err != nil {
		return nil, CategoryOutput{}, err
	}

	cat, err := catService.Create(ctx, input.Name, input.Color)
	if err != nil {
		return nil, CategoryOutput{}, err
	}

	return textResult(fmt.Sprintf("Created category: %s (%s)", cat.Name, cat.Color)), CategoryOutput{Category: cat}, nil
}

func (s *Server) resolveCategoryService(project string) (*core.CategoryService, error) {
	var projectPath string

	if project != "" && project != "current" {
		p, err := s.projectService.GetBySlug(project)
		if err != nil {
			return nil, err
		}
		projectPath = p.Path
	} else {
		cwd, err := os.Getwd()
		if err == nil {
			projectPath, err = s.projectService.DetectProject(cwd)
		}
		if err != nil || projectPath == "" {
			cfg := config.LoadGlobal()
			if cfg.DefaultProject != "" {
				p, slugErr := s.projectService.GetBySlug(cfg.DefaultProject)
				if slugErr == nil {
					projectPath = p.Path
				}
			}
			if projectPath == "" {
				return nil, core.ErrNoProject
			}
		}
	}

	return s.projectService.CategoryServiceFor(projectPath)
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

func formatTaskList(tasks []core.Task) string {
	if len(tasks) == 0 {
		return "No tasks found."
	}

	text := fmt.Sprintf("Found %d task(s):\n", len(tasks))
	for _, t := range tasks {
		pri := ""
		if t.Priority > 0 {
			pri = fmt.Sprintf(" [P%d]", t.Priority)
		}
		text += fmt.Sprintf("  #%d [%s] %s — %s%s\n", t.ID, t.Status, t.Type, t.Title, pri)
	}
	return text
}

func formatTask(t core.Task) string {
	data, _ := json.MarshalIndent(t, "", "  ")
	return string(data)
}

func (s *Server) taskSuggest(ctx context.Context, req *mcp.CallToolRequest, input TaskSuggestInput) (*mcp.CallToolResult, TaskSuggestOutput, error) {
	projectPath, err := s.resolveProjectPath(input.Project)
	if err != nil {
		return nil, TaskSuggestOutput{}, err
	}

	taskService, err := s.projectService.TaskServiceFor(projectPath)
	if err != nil {
		return nil, TaskSuggestOutput{}, err
	}

	suggestService := core.NewSuggestService(projectPath, taskService)
	suggestions, err := suggestService.Scan(ctx)
	if err != nil {
		return nil, TaskSuggestOutput{}, fmt.Errorf("scanning: %w", err)
	}

	var newCount, dupCount int
	for _, sg := range suggestions {
		if sg.ExistingTaskID != nil {
			dupCount++
		} else {
			newCount++
		}
	}

	var text string
	if len(suggestions) == 0 {
		text = "No suggestions found."
	} else {
		text = fmt.Sprintf("Found %d suggestion(s) (%d new, %d matching existing tasks)\n\n", len(suggestions), newCount, dupCount)
		for i, sg := range suggestions {
			text += core.FormatSuggestion(sg, i) + "\n"
		}
	}

	return textResult(text), TaskSuggestOutput{
		Suggestions: suggestions,
		Count:       len(suggestions),
		New:         newCount,
		Duplicates:  dupCount,
	}, nil
}

func (s *Server) taskAdvance(ctx context.Context, req *mcp.CallToolRequest, input TaskAdvanceInput) (*mcp.CallToolResult, TaskAdvanceOutput, error) {
	projectPath, err := s.resolveProjectPath(input.Project)
	if err != nil {
		return nil, TaskAdvanceOutput{}, err
	}

	wfService, err := s.projectService.WorkflowServiceFor(projectPath)
	if err != nil {
		return nil, TaskAdvanceOutput{}, err
	}

	result, err := wfService.Advance(ctx, input.ID)
	if err != nil {
		return nil, TaskAdvanceOutput{}, err
	}

	text := fmt.Sprintf("Task #%d advanced: %s → %s", input.ID, result.PreviousPhase, result.CurrentPhase)
	if result.IsComplete {
		text += " (completed)"
	}
	if len(result.SuggestedSkills) > 0 {
		text += fmt.Sprintf("\n\nSuggested skills for phase %q:\n", result.CurrentPhase)
		for _, sk := range result.SuggestedSkills {
			text += fmt.Sprintf("  - %s\n", sk)
		}
	}

	return textResult(text), TaskAdvanceOutput{Result: result}, nil
}
