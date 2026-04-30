package ui

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/ahoylog/kvik-tasks/internal/ui/templates"
	"github.com/go-chi/chi/v5"
)

type UIHandler struct {
	projectService *core.ProjectService
}

// workflows returns the workflow definitions for a project path.
func (h *UIHandler) workflows(r *http.Request, projectPath string) []core.WorkflowDef {
	wfService, _ := h.projectService.WorkflowServiceFor(projectPath)
	if wfService == nil {
		return nil
	}
	wfs, _ := wfService.ListWorkflows(r.Context())
	return wfs
}

// Dashboard renders the main page with all projects and live stats.
func (h *UIHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	projects, err := h.projectService.List()
	if err != nil {
		slog.Error("listing projects", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Compute extended stats for each project + global aggregates
	var items []templates.DashboardProject
	var global templates.GlobalStats
	for _, p := range projects {
		dp := templates.DashboardProject{Project: p}
		taskService, err := h.projectService.TaskServiceFor(p.Path)
		if err == nil {
			ext, err := taskService.ExtendedStats(r.Context())
			if err == nil {
				dp.ExtStats = ext
				dp.Stats = ext.ProjectStats
				global.Todo += ext.Todo
				global.Doing += ext.Doing
				global.Done += ext.Done
				global.BugsOpen += ext.BugsOpen
				global.CompletedThisWeek += ext.CompletedThisWeek
				global.Total += ext.Total
			}
		}
		items = append(items, dp)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.Dashboard(items, global).Render(r.Context(), w)
}

// Project renders the project page with tasks.
func (h *UIHandler) Project(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		slog.Error("getting project", "slug", slug, "err", err)
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	taskService, err := h.projectService.TaskServiceFor(project.Path)
	if err != nil {
		slog.Error("getting task service", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	filter := core.ListTasksFilter{Limit: 20}
	if r.URL.Query().Has("status") {
		filter.Status = r.URL.Query().Get("status") // explicit empty = all
	} else {
		filter.Status = "todo" // default view
	}
	if t := r.URL.Query().Get("type"); t != "" {
		filter.Type = t
	}
	if p := r.URL.Query().Get("min_priority"); p != "" {
		if pri, err := strconv.ParseInt(p, 10, 64); err == nil {
			filter.MinPriority = &pri
		}
	}
	if v := r.URL.Query().Get("created_after"); v != "" {
		filter.CreatedAfter = &v
	}
	if v := r.URL.Query().Get("created_before"); v != "" {
		filter.CreatedBefore = &v
	}
	if pg := r.URL.Query().Get("page"); pg != "" {
		if page, err := strconv.ParseInt(pg, 10, 64); err == nil && page > 1 {
			filter.Offset = (page - 1) * filter.Limit
		}
	}

	result, err := taskService.ListWithCount(r.Context(), filter)
	if err != nil {
		slog.Error("listing tasks", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	stats, err := taskService.Stats(r.Context())
	if err != nil {
		slog.Error("getting stats", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	pf := templates.ProjectFilter{
		Status:        filter.Status,
		Type:          r.URL.Query().Get("type"),
		MinPriority:   r.URL.Query().Get("min_priority"),
		CreatedAfter:  r.URL.Query().Get("created_after"),
		CreatedBefore: r.URL.Query().Get("created_before"),
		Page:          result.Page,
		TotalPages:    result.TotalPages,
		Total:         result.Total,
	}

	// Fetch categories for the sidebar
	catService, _ := h.projectService.CategoryServiceFor(project.Path)
	var categories []core.Category
	if catService != nil {
		categories, _ = catService.List(r.Context())
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.ProjectPage(project, result.Tasks, stats, pf, categories, h.workflows(r, project.Path)).Render(r.Context(), w)
}

// TaskDetail renders a single task page.
func (h *UIHandler) TaskDetail(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	taskService, err := h.projectService.TaskServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	task, err := taskService.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	catService, _ := h.projectService.CategoryServiceFor(project.Path)
	var categories []core.Category
	if catService != nil {
		categories, _ = catService.List(r.Context())
	}

	gitService := &core.GitService{}
	commits, _ := gitService.CommitsForTask(r.Context(), project.Path, id)

	wfService, _ := h.projectService.WorkflowServiceFor(project.Path)
	var workflow core.WorkflowContext
	if wfService != nil {
		workflow, _ = wfService.GetContext(r.Context(), task)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var history []core.HistoryEntry
	if wfService != nil {
		history, _ = wfService.GetHistory(r.Context(), task.ID)
	}
	templates.TaskPage(project, task, categories, commits, workflow, history, h.workflows(r, project.Path)).Render(r.Context(), w)
}

// TaskNewPage renders the detailed task creation page.
func (h *UIHandler) TaskNewPage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	catService, _ := h.projectService.CategoryServiceFor(project.Path)
	var categories []core.Category
	if catService != nil {
		categories, _ = catService.List(r.Context())
	}

	configService, _ := h.projectService.ConfigServiceFor(project.Path)
	var config core.ProjectConfig
	if configService != nil {
		config, _ = configService.Get(r.Context())
	}

	tplService, _ := h.projectService.TemplateServiceFor(project.Path)
	var taskTemplates []core.TaskTemplate
	if tplService != nil {
		taskTemplates, _ = tplService.List(r.Context())
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.TaskNewPage(project, categories, config, taskTemplates, h.workflows(r, project.Path)).Render(r.Context(), w)
}

// CreateDetailedTask handles the detailed task creation form — creates task and redirects to detail.
func (h *UIHandler) CreateDetailedTask(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	taskService, err := h.projectService.TaskServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Type        string `json:"type"`
		Priority    int64  `json:"priority"`
		Category    string `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	taskType := core.TaskType(input.Type)
	if input.Type == "" {
		taskType = core.TypeTask
	}
	// Auto-detect type from title prefix
	detectedType, cleanTitle := core.DetectTypeFromTitle(input.Title, taskType)

	task, err := taskService.Create(r.Context(), core.CreateTaskInput{
		Title:       cleanTitle,
		Description: input.Description,
		Type:        detectedType,
		Priority:    input.Priority,
		Category:    input.Category,
		Source:      core.SourceUI,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect to the new task's detail page
	w.Header().Set("HX-Redirect", fmt.Sprintf("/p/%s/t/%d", slug, task.ID))
	w.WriteHeader(http.StatusOK)
}

// CreateTask handles HTMX task creation — accepts JSON, returns HTML task list.
func (h *UIHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	taskService, err := h.projectService.TaskServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Category    string `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	taskType, title := core.DetectTypeFromTitle(input.Title, core.TypeTask)

	_, err = taskService.Create(r.Context(), core.CreateTaskInput{
		Title:       title,
		Description: input.Description,
		Category:    input.Category,
		Type:        taskType,
		Source:      core.SourceUI,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	// Return updated task list + OOB stats
	tasks, _ := taskService.List(r.Context(), core.ListTasksFilter{Limit: 100})
	stats, _ := taskService.Stats(r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.TaskList(tasks, slug, h.workflows(r, project.Path)).Render(r.Context(), w)
	templates.StatsBarOOB(stats).Render(r.Context(), w)
}

// CompleteTask handles HTMX task completion — returns HTML for the completed task row.
func (h *UIHandler) CompleteTask(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	taskService, err := h.projectService.TaskServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	task, err := taskService.Complete(r.Context(), id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	// Return updated task row + OOB stats
	stats, _ := taskService.Stats(r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.TaskRow(task, slug, h.workflows(r, project.Path)).Render(r.Context(), w)
	templates.StatsBarOOB(stats).Render(r.Context(), w)
}

// DeleteTask handles HTMX task deletion — returns empty string to remove the row + OOB stats.
func (h *UIHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	taskService, err := h.projectService.TaskServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := taskService.Delete(r.Context(), id); err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	// Return OOB stats only — empty response removes the target element
	stats, _ := taskService.Stats(r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.StatsBarOOB(stats).Render(r.Context(), w)
}

// UpdateTaskField handles inline field updates from the task detail page.
func (h *UIHandler) UpdateTaskField(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	taskService, err := h.projectService.TaskServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var input map[string]any
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	updateInput := core.UpdateTaskInput{}
	if v, ok := input["status"].(string); ok {
		s := core.TaskStatus(v)
		updateInput.Status = &s
	}
	if v, ok := input["type"].(string); ok {
		t := core.TaskType(v)
		updateInput.Type = &t
	}
	if v, ok := input["priority"].(float64); ok {
		p := int64(v)
		updateInput.Priority = &p
	}
	if v, ok := input["title"].(string); ok {
		updateInput.Title = &v
	}
	if v, ok := input["description"].(string); ok {
		updateInput.Description = &v
	}
	if v, ok := input["category"].(string); ok {
		updateInput.Category = &v
	}

	task, err := taskService.Update(r.Context(), id, updateInput)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	catService, _ := h.projectService.CategoryServiceFor(project.Path)
	var categories []core.Category
	if catService != nil {
		categories, _ = catService.List(r.Context())
	}

	gitService := &core.GitService{}
	commits, _ := gitService.CommitsForTask(r.Context(), project.Path, id)

	wfService, _ := h.projectService.WorkflowServiceFor(project.Path)
	var workflow core.WorkflowContext
	if wfService != nil {
		workflow, _ = wfService.GetContext(r.Context(), task)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var history []core.HistoryEntry
	if wfService != nil {
		history, _ = wfService.GetHistory(r.Context(), task.ID)
	}
	templates.TaskPage(project, task, categories, commits, workflow, history, h.workflows(r, project.Path)).Render(r.Context(), w)
}

// AdvanceTask advances a task to the next workflow phase and redirects back to detail.
func (h *UIHandler) AdvanceTask(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	wfService, err := h.projectService.WorkflowServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	result, err := wfService.Advance(r.Context(), id)
	if err != nil {
		slog.Error("advancing task", "id", id, "err", err)
	}

	// Show advance result summary via query param
	summary := fmt.Sprintf("%s→%s", result.PreviousPhase, result.CurrentPhase)
	if len(result.ActionResults) > 0 {
		ok := 0
		for _, ar := range result.ActionResults {
			if ar.Success {
				ok++
			}
		}
		summary += fmt.Sprintf(" (%d/%d triggers OK)", ok, len(result.ActionResults))
	}
	http.Redirect(w, r, fmt.Sprintf("/p/%s/t/%d?advanced=%s", slug, id, summary), http.StatusSeeOther)
}

// TaskHistory renders the full workflow history page for a task.
func (h *UIHandler) TaskHistory(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	taskService, err := h.projectService.TaskServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	task, err := taskService.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	wfService, _ := h.projectService.WorkflowServiceFor(project.Path)
	var history []core.HistoryEntry
	if wfService != nil {
		history, _ = wfService.GetHistory(r.Context(), id)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.HistoryPage(project, task, history, h.workflows(r, project.Path)).Render(r.Context(), w)
}

// BulkAction handles bulk operations on multiple tasks.
func (h *UIHandler) BulkAction(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	taskService, err := h.projectService.TaskServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var input struct {
		Action string  `json:"action"` // complete, delete, archive, status, type
		IDs    []int64 `json:"ids"`
		Value  string  `json:"value"` // for status/type actions
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if len(input.IDs) == 0 {
		http.Error(w, "No tasks selected", http.StatusBadRequest)
		return
	}

	for _, id := range input.IDs {
		switch input.Action {
		case "complete":
			taskService.Complete(r.Context(), id)
		case "delete":
			taskService.Delete(r.Context(), id)
		case "archive":
			taskService.Archive(r.Context(), id)
		case "status":
			s := core.TaskStatus(input.Value)
			taskService.Update(r.Context(), id, core.UpdateTaskInput{Status: &s})
		case "type":
			t := core.TaskType(input.Value)
			taskService.Update(r.Context(), id, core.UpdateTaskInput{Type: &t})
		}
	}

	// Return updated task list + OOB stats
	tasks, _ := taskService.List(r.Context(), core.ListTasksFilter{Limit: 100})
	stats, _ := taskService.Stats(r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.TaskList(tasks, slug, h.workflows(r, project.Path)).Render(r.Context(), w)
	templates.StatsBarOOB(stats).Render(r.Context(), w)
}

// SearchTasks handles search via HTMX — returns task list HTML fragment.
func (h *UIHandler) SearchTasks(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	taskService, err := h.projectService.TaskServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		// Empty query — return full list
		tasks, _ := taskService.List(r.Context(), core.ListTasksFilter{Limit: 20})
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.TaskList(tasks, slug, h.workflows(r, project.Path)).Render(r.Context(), w)
		return
	}

	tasks, err := taskService.Search(r.Context(), query, 50)
	if err != nil {
		// FTS syntax error — fall back to empty
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.TaskList(nil, slug, h.workflows(r, project.Path)).Render(r.Context(), w)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.TaskList(tasks, slug, h.workflows(r, project.Path)).Render(r.Context(), w)
}

// ExportTasks handles task export as JSON or Markdown file download.
func (h *UIHandler) ExportTasks(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	taskService, err := h.projectService.TaskServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	filter := core.ListTasksFilter{Limit: 10000}
	if s := r.URL.Query().Get("status"); s != "" {
		filter.Status = s
	}
	if t := r.URL.Query().Get("type"); t != "" {
		filter.Type = t
	}
	archived := r.URL.Query().Get("archived") == "1"

	var tasks []core.Task
	if archived {
		result, err := taskService.ListArchived(r.Context(), filter)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		tasks = result.Tasks
	} else {
		tasks, err = taskService.List(r.Context(), filter)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	var data []byte
	var contentType, ext string
	switch format {
	case "md", "markdown":
		data = core.ExportMarkdown(project.Name, tasks)
		contentType = "text/markdown"
		ext = "md"
	default:
		data, err = core.ExportJSON(project.Name, tasks)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		contentType = "application/json"
		ext = "json"
	}

	filename := fmt.Sprintf("%s-tasks.%s", project.Slug, ext)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Write(data)
}

// Board renders the kanban board view.
func (h *UIHandler) Board(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		slog.Error("getting project", "slug", slug, "err", err)
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	taskService, err := h.projectService.TaskServiceFor(project.Path)
	if err != nil {
		slog.Error("getting task service", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Fetch tasks for each column with optional type filter
	typeFilter := r.URL.Query().Get("type")
	todoTasks, _ := taskService.List(r.Context(), core.ListTasksFilter{Status: string(core.StatusTodo), Type: typeFilter, Limit: 100})
	doingTasks, _ := taskService.List(r.Context(), core.ListTasksFilter{Status: string(core.StatusDoing), Type: typeFilter, Limit: 100})
	doneTasks, _ := taskService.List(r.Context(), core.ListTasksFilter{Status: string(core.StatusDone), Type: typeFilter, Limit: 100})
	stats, _ := taskService.Stats(r.Context())

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.BoardPage(project, todoTasks, doingTasks, doneTasks, stats, typeFilter, h.workflows(r, project.Path)).Render(r.Context(), w)
}

// MoveTask handles drag & drop status change from kanban board.
func (h *UIHandler) MoveTask(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	taskService, err := h.projectService.TaskServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var input struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	status := core.TaskStatus(input.Status)
	_, err = taskService.Update(r.Context(), id, core.UpdateTaskInput{Status: &status})
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	// Return updated stats OOB
	stats, _ := taskService.Stats(r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.StatsBarOOB(stats).Render(r.Context(), w)
}

// Settings renders the project settings page.
func (h *UIHandler) Settings(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	catService, err := h.projectService.CategoryServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	categories, err := catService.List(r.Context())
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	taskService, err := h.projectService.TaskServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	stats, _ := taskService.Stats(r.Context())

	configService, _ := h.projectService.ConfigServiceFor(project.Path)
	var config core.ProjectConfig
	if configService != nil {
		config, _ = configService.Get(r.Context())
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tplService, _ := h.projectService.TemplateServiceFor(project.Path)
	var taskTemplates []core.TaskTemplate
	if tplService != nil {
		taskTemplates, _ = tplService.List(r.Context())
	}

	wfService, _ := h.projectService.WorkflowServiceFor(project.Path)
	var workflows []core.WorkflowDef
	if wfService != nil {
		workflows, _ = wfService.ListWorkflows(r.Context())
	}

	hookService := h.projectService.HookServiceFor(project.Path)
	var hooks []core.Hook
	if hookService != nil {
		hooks, _ = hookService.List()
	}

	templates.SettingsPage(project, categories, stats, config, taskTemplates, workflows, hooks).Render(r.Context(), w)
}

// CreateHook handles POST /p/{slug}/hooks — adds a new hook.
func (h *UIHandler) CreateHook(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	var input struct {
		Event   string `json:"event"`
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	hookService := h.projectService.HookServiceFor(project.Path)
	hookService.Add(core.Hook{
		Event:   core.HookEvent(input.Event),
		Command: input.Command,
	})

	hooks, _ := hookService.List()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.HookList(hooks, slug).Render(r.Context(), w)
}

// DeleteHook handles DELETE /p/{slug}/hooks/{hookID} — removes a hook.
func (h *UIHandler) DeleteHook(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	hookIDStr := chi.URLParam(r, "hookID")
	hookID, err := strconv.Atoi(hookIDStr)
	if err != nil {
		http.Error(w, "Invalid hook ID", http.StatusBadRequest)
		return
	}

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	hookService := h.projectService.HookServiceFor(project.Path)
	hookService.Remove(hookID)

	hooks, _ := hookService.List()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.HookList(hooks, slug).Render(r.Context(), w)
}

// SaveConfig handles HTMX config save from settings page.
func (h *UIHandler) SaveConfig(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	configService, err := h.projectService.ConfigServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var input struct {
		DefaultPriority int64  `json:"default_priority"`
		DefaultType     string `json:"default_type"`
		AutoArchiveDays int64  `json:"auto_archive_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	cfg := core.ProjectConfig{
		DefaultPriority: input.DefaultPriority,
		DefaultType:     input.DefaultType,
		AutoArchiveDays: input.AutoArchiveDays,
	}
	if err := configService.SetAll(r.Context(), cfg); err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<span style="color:var(--kvt-success);font-size:0.8rem;">Saved</span>`)
}

// ListCategories returns the category management HTML fragment.
func (h *UIHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	catService, err := h.projectService.CategoryServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	categories, err := catService.List(r.Context())
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.CategoryList(categories, slug).Render(r.Context(), w)
}

// CreateCategory handles HTMX category creation.
func (h *UIHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	catService, err := h.projectService.CategoryServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var input struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	_, err = catService.Create(r.Context(), input.Name, input.Color)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	categories, _ := catService.List(r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.CategoryList(categories, slug).Render(r.Context(), w)
}

// UpdateCategory handles HTMX category update.
func (h *UIHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	idStr := chi.URLParam(r, "catID")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	catService, err := h.projectService.CategoryServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var input struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	_, err = catService.Update(r.Context(), id, input.Name, input.Color)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	categories, _ := catService.List(r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.CategoryList(categories, slug).Render(r.Context(), w)
}

// DeleteCategory handles HTMX category deletion.
func (h *UIHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	idStr := chi.URLParam(r, "catID")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	catService, err := h.projectService.CategoryServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := catService.DeleteByID(r.Context(), id); err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	categories, _ := catService.List(r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.CategoryList(categories, slug).Render(r.Context(), w)
}

// ArchivePage renders the archive page for a project.
func (h *UIHandler) ArchivePage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	taskService, err := h.projectService.TaskServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	filter := core.ListTasksFilter{Limit: 20}
	if t := r.URL.Query().Get("type"); t != "" {
		filter.Type = t
	}
	if v := r.URL.Query().Get("category"); v != "" {
		filter.Category = v
	}
	if v := r.URL.Query().Get("created_after"); v != "" {
		filter.CreatedAfter = &v
	}
	if v := r.URL.Query().Get("created_before"); v != "" {
		filter.CreatedBefore = &v
	}
	if pg := r.URL.Query().Get("page"); pg != "" {
		if page, err := strconv.ParseInt(pg, 10, 64); err == nil && page > 1 {
			filter.Offset = (page - 1) * filter.Limit
		}
	}

	result, err := taskService.ListArchived(r.Context(), filter)
	if err != nil {
		slog.Error("listing archived tasks", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	af := templates.ArchiveFilter{
		Type:          r.URL.Query().Get("type"),
		Category:      r.URL.Query().Get("category"),
		CreatedAfter:  r.URL.Query().Get("created_after"),
		CreatedBefore: r.URL.Query().Get("created_before"),
		Page:          result.Page,
		TotalPages:    result.TotalPages,
		Total:         result.Total,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.ArchivePage(project, result.Tasks, af, h.workflows(r, project.Path)).Render(r.Context(), w)
}

// ArchiveTaskRedirect archives a task and redirects back.
func (h *UIHandler) ArchiveTaskRedirect(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	taskService, err := h.projectService.TaskServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	taskService.Archive(r.Context(), id)

	// Redirect to referer or project page
	ref := r.Referer()
	if ref == "" {
		ref = "/p/" + slug
	}
	http.Redirect(w, r, ref, http.StatusSeeOther)
}

// UnarchiveTaskRedirect unarchives a task and redirects back to archive page.
func (h *UIHandler) UnarchiveTaskRedirect(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	taskService, err := h.projectService.TaskServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	taskService.Unarchive(r.Context(), id)
	http.Redirect(w, r, "/p/"+slug+"/archive", http.StatusSeeOther)
}

// DeleteTaskRedirect handles task deletion from detail page — deletes and redirects to project.
func (h *UIHandler) DeleteTaskRedirect(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	taskService, err := h.projectService.TaskServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	taskService.Delete(r.Context(), id)
	http.Redirect(w, r, "/p/"+slug, http.StatusSeeOther)
}

// CreateTemplate handles HTMX template creation.
func (h *UIHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	tplService, err := h.projectService.TemplateServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var input struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	tplService.Create(r.Context(), input.Name, input.Type, 0, "")

	h.renderTemplateList(w, r, tplService, slug)
}

// DeleteTemplate handles HTMX template deletion.
func (h *UIHandler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	idStr := chi.URLParam(r, "tplID")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	tplService, err := h.projectService.TemplateServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	tplService.Delete(r.Context(), id)

	h.renderTemplateList(w, r, tplService, slug)
}

// TemplateDetail renders the template edit page.
func (h *UIHandler) TemplateDetail(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	idStr := chi.URLParam(r, "tplID")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	tplService, err := h.projectService.TemplateServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	tpl, err := tplService.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.TemplateDetailPage(project, tpl, h.workflows(r, project.Path)).Render(r.Context(), w)
}

// UpdateTemplate handles template update from detail page.
func (h *UIHandler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	idStr := chi.URLParam(r, "tplID")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	tplService, err := h.projectService.TemplateServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var input map[string]any
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	name, _ := input["name"].(string)
	typ, _ := input["type"].(string)
	desc, _ := input["description"].(string)
	var priority int64
	switch v := input["priority"].(type) {
	case float64:
		priority = int64(v)
	case string:
		priority, _ = strconv.ParseInt(v, 10, 64)
	}

	_, err = tplService.Update(r.Context(), id, name, typ, priority, desc)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "showSaved")
	w.WriteHeader(http.StatusOK)
}

// DeleteTemplateRedirect deletes a template and redirects to settings.
func (h *UIHandler) DeleteTemplateRedirect(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	idStr := chi.URLParam(r, "tplID")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	tplService, err := h.projectService.TemplateServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	tplService.Delete(r.Context(), id)
	http.Redirect(w, r, "/p/"+slug+"/settings#templates", http.StatusSeeOther)
}

// WorkflowsPage renders the standalone workflows listing page.
func (h *UIHandler) WorkflowsPage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	wfService, err := h.projectService.WorkflowServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	workflows, err := wfService.ListWorkflows(r.Context())
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.WorkflowsPage(project, workflows).Render(r.Context(), w)
}

// WorkflowEditor renders the workflow editor page for a task type.
func (h *UIHandler) WorkflowEditor(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	taskType := chi.URLParam(r, "taskType")

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	wfService, err := h.projectService.WorkflowServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	wf, err := wfService.GetWorkflow(r.Context(), taskType)
	if err != nil {
		http.Error(w, "Workflow not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.WorkflowEditorPage(project, wf, h.workflows(r, project.Path)).Render(r.Context(), w)
}

// UpdateWorkflow handles saving workflow changes.
func (h *UIHandler) UpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	taskType := chi.URLParam(r, "taskType")

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	wfService, err := h.projectService.WorkflowServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var input map[string]any
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Parse phases, prompts, and triggers from numbered fields
	var phases []string
	prompts := map[string]string{}
	triggers := map[string]core.Triggers{}
	for i := 0; ; i++ {
		nameKey := fmt.Sprintf("phase_%d_name", i)
		name, ok := input[nameKey].(string)
		if !ok || name == "" {
			break
		}
		phases = append(phases, name)
		promptKey := fmt.Sprintf("phase_%d_prompt", i)
		if prompt, ok := input[promptKey].(string); ok && prompt != "" {
			prompts[name] = prompt
		}
		// Parse triggers
		var t core.Triggers
		if before, ok := input[fmt.Sprintf("phase_%d_trigger_before", i)].(string); ok && before != "" {
			t.Before = splitTriggers(before)
		}
		if after, ok := input[fmt.Sprintf("phase_%d_trigger_after", i)].(string); ok && after != "" {
			t.After = splitTriggers(after)
		}
		if len(t.Before) > 0 || len(t.After) > 0 {
			triggers[name] = t
		}
	}

	if len(phases) < 2 {
		http.Error(w, "Workflow must have at least 2 phases", http.StatusBadRequest)
		return
	}

	wf := core.WorkflowDef{
		TaskType:     taskType,
		Phases:       phases,
		Triggers:     triggers,
		PhasePrompts: prompts,
	}

	if err := wfService.UpdateWorkflow(r.Context(), wf); err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "showSaved")
	w.WriteHeader(http.StatusOK)
}

// CreateCustomType handles POST /p/{slug}/workflows — creates a new custom task type with workflow.
func (h *UIHandler) CreateCustomType(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	wfService, err := h.projectService.WorkflowServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var input struct {
		TaskType string `json:"task_type"`
		Prefix   string `json:"prefix"`
		Color    string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if input.TaskType == "" {
		http.Error(w, "Type name is required", http.StatusBadRequest)
		return
	}

	wf := core.WorkflowDef{
		TaskType: input.TaskType,
		Prefix:   input.Prefix,
		Color:    input.Color,
	}

	if _, err := wfService.CreateWorkflow(r.Context(), wf); err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect back to workflows page
	w.Header().Set("HX-Redirect", fmt.Sprintf("/p/%s/workflows", slug))
	w.WriteHeader(http.StatusCreated)
}

// DeleteCustomType handles DELETE /p/{slug}/workflows/{taskType} — deletes a custom task type.
func (h *UIHandler) DeleteCustomType(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	taskType := chi.URLParam(r, "taskType")

	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	wfService, err := h.projectService.WorkflowServiceFor(project.Path)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := wfService.DeleteWorkflow(r.Context(), taskType); err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect back to workflows page
	w.Header().Set("HX-Redirect", fmt.Sprintf("/p/%s/workflows", slug))
	w.WriteHeader(http.StatusOK)
}

func (h *UIHandler) renderTemplateList(w http.ResponseWriter, r *http.Request, tplService *core.TemplateService, slug string) {
	tpls, _ := tplService.List(r.Context())

	project, _ := h.projectService.GetBySlug(slug)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.TemplateList(tpls, project.Slug, h.workflows(r, project.Path)).Render(r.Context(), w)
}

// splitTriggers splits a comma-separated trigger string into a cleaned slice.
func splitTriggers(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		t := strings.TrimSpace(part)
		if t != "" {
			result = append(result, t)
		}
	}
	return result
}
