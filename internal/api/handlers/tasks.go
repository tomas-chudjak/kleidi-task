package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/go-chi/chi/v5"
)

type TaskHandler struct {
	projectService *core.ProjectService
}

func NewTaskHandler(ps *core.ProjectService) *TaskHandler {
	return &TaskHandler{projectService: ps}
}

func (h *TaskHandler) taskServiceForSlug(slug string) (*core.TaskService, error) {
	project, err := h.projectService.GetBySlug(slug)
	if err != nil {
		return nil, err
	}
	return h.projectService.TaskServiceFor(project.Path)
}

func (h *TaskHandler) parseID(r *http.Request) (int64, error) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid task ID: %s", core.ErrInvalidInput, idStr)
	}
	return id, nil
}

// ListByProject returns tasks for a project with optional filters.
func (h *TaskHandler) ListByProject(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	svc, err := h.taskServiceForSlug(slug)
	if err != nil {
		respondError(w, err)
		return
	}

	filter := core.ListTasksFilter{Limit: 50}

	if s := r.URL.Query().Get("status"); s != "" {
		status := core.TaskStatus(s)
		filter.Status = &status
	}
	if t := r.URL.Query().Get("type"); t != "" {
		taskType := core.TaskType(t)
		filter.Type = &taskType
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if limit, err := strconv.ParseInt(l, 10, 64); err == nil {
			filter.Limit = limit
		}
	}

	tasks, err := svc.List(r.Context(), filter)
	if err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, tasks)
}

// Create adds a new task to a project.
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	svc, err := h.taskServiceForSlug(slug)
	if err != nil {
		respondError(w, err)
		return
	}

	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Type        string `json:"type"`
		Priority    int64  `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, fmt.Errorf("%w: invalid JSON body", core.ErrInvalidInput))
		return
	}

	taskType := core.TaskType(input.Type)
	title := input.Title
	if input.Type == "" {
		taskType, title = core.DetectTypeFromTitle(title, core.TypeTask)
	}

	task, err := svc.Create(r.Context(), core.CreateTaskInput{
		Title:       title,
		Description: input.Description,
		Type:        taskType,
		Priority:    input.Priority,
		Source:      core.SourceAPI,
	})
	if err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, task)
}

// Get returns a single task.
func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	svc, err := h.taskServiceForSlug(slug)
	if err != nil {
		respondError(w, err)
		return
	}

	id, err := h.parseID(r)
	if err != nil {
		respondError(w, err)
		return
	}

	task, err := svc.Get(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, task)
}

// Update partially updates a task.
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	svc, err := h.taskServiceForSlug(slug)
	if err != nil {
		respondError(w, err)
		return
	}

	id, err := h.parseID(r)
	if err != nil {
		respondError(w, err)
		return
	}

	var input struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Status      *string `json:"status"`
		Type        *string `json:"type"`
		Priority    *int64  `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, fmt.Errorf("%w: invalid JSON body", core.ErrInvalidInput))
		return
	}

	updateInput := core.UpdateTaskInput{}
	if input.Title != nil {
		updateInput.Title = input.Title
	}
	if input.Description != nil {
		updateInput.Description = input.Description
	}
	if input.Status != nil {
		s := core.TaskStatus(*input.Status)
		updateInput.Status = &s
	}
	if input.Type != nil {
		t := core.TaskType(*input.Type)
		updateInput.Type = &t
	}
	if input.Priority != nil {
		updateInput.Priority = input.Priority
	}

	task, err := svc.Update(r.Context(), id, updateInput)
	if err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, task)
}

// Delete permanently removes a task.
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	svc, err := h.taskServiceForSlug(slug)
	if err != nil {
		respondError(w, err)
		return
	}

	id, err := h.parseID(r)
	if err != nil {
		respondError(w, err)
		return
	}

	if err := svc.Delete(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// Complete marks a task as done.
func (h *TaskHandler) Complete(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	svc, err := h.taskServiceForSlug(slug)
	if err != nil {
		respondError(w, err)
		return
	}

	id, err := h.parseID(r)
	if err != nil {
		respondError(w, err)
		return
	}

	task, err := svc.Complete(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, task)
}
