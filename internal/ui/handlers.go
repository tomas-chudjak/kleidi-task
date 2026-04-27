package ui

import (
	"log/slog"
	"net/http"

	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/ahoylog/kvik-tasks/internal/ui/templates"
	"github.com/go-chi/chi/v5"
)

type UIHandler struct {
	projectService *core.ProjectService
}

func (h *UIHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	projects, err := h.projectService.List()
	if err != nil {
		slog.Error("listing projects", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.Dashboard(projects).Render(r.Context(), w)
}

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

	tasks, err := taskService.List(r.Context(), core.ListTasksFilter{Limit: 100})
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
