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

	// Compute live stats for each project
	var items []templates.DashboardProject
	for _, p := range projects {
		dp := templates.DashboardProject{Project: p}
		taskService, err := h.projectService.TaskServiceFor(p.Path)
		if err == nil {
			stats, err := taskService.Stats(r.Context())
			if err == nil {
				dp.Stats = stats
			}
		}
		items = append(items, dp)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.Dashboard(items).Render(r.Context(), w)
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

	filter := core.ListTasksFilter{Limit: 100}
	if s := r.URL.Query().Get("status"); s != "" {
		status := core.TaskStatus(s)
		filter.Status = &status
	}
	if t := r.URL.Query().Get("type"); t != "" {
		taskType := core.TaskType(t)
		filter.Type = &taskType
	}

	tasks, err := taskService.List(r.Context(), filter)
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.ProjectPage(project, tasks, stats).Render(r.Context(), w)
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.TaskPage(project, task).Render(r.Context(), w)
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
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	taskType, title := core.DetectTypeFromTitle(input.Title, core.TypeTask)

	_, err = taskService.Create(r.Context(), core.CreateTaskInput{
		Title:  title,
		Type:   taskType,
		Source: core.SourceUI,
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

	task, err := taskService.Update(r.Context(), id, updateInput)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.TaskPage(project, task).Render(r.Context(), w)
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
