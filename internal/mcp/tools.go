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
	Project     string `json:"project" jsonschema:"project slug or 'current'"`
	Title       string `json:"title" jsonschema:"task title"`
	Type        string `json:"type,omitempty" jsonschema:"work item type,enum=task,enum=bug,enum=feature,enum=hotfix"`
	Description string `json:"description,omitempty" jsonschema:"task description"`
	Priority    int64  `json:"priority,omitempty" jsonschema:"higher number means higher priority"`
}

type TaskListInput struct {
	Project       string `json:"project,omitempty" jsonschema:"project slug or 'current'"`
	Status        string `json:"status,omitempty" jsonschema:"filter by status,enum=todo,enum=doing,enum=done"`
	Type          string `json:"type,omitempty" jsonschema:"filter by type,enum=task,enum=bug,enum=feature,enum=hotfix"`
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
	Type        string `json:"type,omitempty" jsonschema:"new type,enum=task,enum=bug,enum=feature,enum=hotfix"`
	Priority    *int64 `json:"priority,omitempty" jsonschema:"new priority"`
}

type TaskCompleteInput struct {
	Project string `json:"project,omitempty" jsonschema:"project slug or 'current'"`
	ID      int64  `json:"id" jsonschema:"task ID"`
}

type TaskDeleteInput struct {
	Project string `json:"project,omitempty" jsonschema:"project slug or 'current'"`
	ID      int64  `json:"id" jsonschema:"task ID"`
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

	task, err := taskService.Create(ctx, core.CreateTaskInput{
		Title:       title,
		Description: input.Description,
		Type:        taskType,
		Priority:    input.Priority,
		Source:      core.SourceMCP,
	})
	if err != nil {
		return nil, TaskOutput{}, err
	}

	return textResult(fmt.Sprintf("Created %s #%d: %s", task.Type, task.ID, task.Title)), TaskOutput{Task: task}, nil
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

func (s *Server) taskGet(ctx context.Context, req *mcp.CallToolRequest, input TaskGetInput) (*mcp.CallToolResult, TaskOutput, error) {
	taskService, err := s.resolveTaskService(input.Project)
	if err != nil {
		return nil, TaskOutput{}, err
	}

	task, err := taskService.Get(ctx, input.ID)
	if err != nil {
		return nil, TaskOutput{}, err
	}

	return textResult(formatTask(task)), TaskOutput{Task: task}, nil
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

func (s *Server) resolveTaskService(project string) (*core.TaskService, error) {
	var projectPath string

	if project != "" && project != "current" {
		p, err := s.projectService.GetBySlug(project)
		if err != nil {
			return nil, err
		}
		projectPath = p.Path
	} else {
		// Try cwd first
		cwd, err := os.Getwd()
		if err == nil {
			projectPath, err = s.projectService.DetectProject(cwd)
		}
		// Fall back to default project from config
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

	return s.projectService.TaskServiceFor(projectPath)
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
