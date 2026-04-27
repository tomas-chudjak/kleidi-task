package handlers

import (
	"net/http"

	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/go-chi/chi/v5"
)

type ProjectHandler struct {
	projectService *core.ProjectService
}

func NewProjectHandler(ps *core.ProjectService) *ProjectHandler {
	return &ProjectHandler{projectService: ps}
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	projects, err := h.projectService.List()
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, projects)
}

func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, project)
}

func (h *ProjectHandler) Stats(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		respondError(w, err)
		return
	}

	taskService, err := h.projectService.TaskServiceFor(project.Path)
	if err != nil {
		respondError(w, err)
		return
	}

	stats, err := taskService.Stats(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, stats)
}
