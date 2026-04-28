package ui

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/ahoylog/kvik-tasks/internal/ui/templates"
	"github.com/go-chi/chi/v5"
)

type UIHandler struct {
	projectService *core.ProjectService
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
	templates.ProjectPage(project, result.Tasks, stats, pf, categories).Render(r.Context(), w)
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.TaskPage(project, task, categories, commits).Render(r.Context(), w)
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
	templates.TaskList(tasks, slug).Render(r.Context(), w)
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
	templates.TaskRow(task, slug).Render(r.Context(), w)
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.TaskPage(project, task, categories, commits).Render(r.Context(), w)
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
		templates.TaskList(tasks, slug).Render(r.Context(), w)
		return
	}

	tasks, err := taskService.Search(r.Context(), query, 50)
	if err != nil {
		// FTS syntax error — fall back to empty
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.TaskList(nil, slug).Render(r.Context(), w)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.TaskList(tasks, slug).Render(r.Context(), w)
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
	templates.BoardPage(project, todoTasks, doingTasks, doneTasks, stats, typeFilter).Render(r.Context(), w)
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.SettingsPage(project, categories, stats).Render(r.Context(), w)
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
