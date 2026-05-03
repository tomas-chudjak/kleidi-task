package core

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/tomas-chudjak/kleidi-task/internal/db/generated"
)

// TaskTemplate represents a reusable task template.
type TaskTemplate struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Priority    int64     `json:"priority"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// TemplateService manages task templates.
type TemplateService struct {
	queries *generated.Queries
}

// NewTemplateService creates a new TemplateService.
func NewTemplateService(db *sql.DB) *TemplateService {
	return &TemplateService{queries: generated.New(db)}
}

// List returns all templates.
func (s *TemplateService) List(ctx context.Context) ([]TaskTemplate, error) {
	rows, err := s.queries.ListTemplates(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing templates: %w", err)
	}
	templates := make([]TaskTemplate, len(rows))
	for i, r := range rows {
		templates[i] = templateFromRow(r)
	}
	return templates, nil
}

// Get returns a template by ID.
func (s *TemplateService) Get(ctx context.Context, id int64) (TaskTemplate, error) {
	row, err := s.queries.GetTemplate(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return TaskTemplate{}, fmt.Errorf("template %d: %w", id, ErrNotFound)
		}
		return TaskTemplate{}, fmt.Errorf("getting template %d: %w", id, err)
	}
	return templateFromRow(row), nil
}

// GetByType returns the first template matching a task type.
func (s *TemplateService) GetByType(ctx context.Context, taskType string) (TaskTemplate, error) {
	row, err := s.queries.GetTemplateByType(ctx, taskType)
	if err != nil {
		if err == sql.ErrNoRows {
			return TaskTemplate{}, fmt.Errorf("template for type %s: %w", taskType, ErrNotFound)
		}
		return TaskTemplate{}, fmt.Errorf("getting template by type: %w", err)
	}
	return templateFromRow(row), nil
}

// Create creates a new template.
func (s *TemplateService) Create(ctx context.Context, name, taskType string, priority int64, description string) (TaskTemplate, error) {
	row, err := s.queries.CreateTemplate(ctx, generated.CreateTemplateParams{
		Name:        name,
		Type:        taskType,
		Priority:    priority,
		Description: description,
	})
	if err != nil {
		return TaskTemplate{}, fmt.Errorf("creating template: %w", err)
	}
	return templateFromRow(row), nil
}

// Update updates an existing template.
func (s *TemplateService) Update(ctx context.Context, id int64, name, taskType string, priority int64, description string) (TaskTemplate, error) {
	row, err := s.queries.UpdateTemplate(ctx, generated.UpdateTemplateParams{
		ID:          id,
		Name:        name,
		Type:        taskType,
		Priority:    priority,
		Description: description,
	})
	if err != nil {
		return TaskTemplate{}, fmt.Errorf("updating template %d: %w", id, err)
	}
	return templateFromRow(row), nil
}

// Delete removes a template.
func (s *TemplateService) Delete(ctx context.Context, id int64) error {
	return s.queries.DeleteTemplate(ctx, id)
}

func templateFromRow(r generated.TaskTemplate) TaskTemplate {
	return TaskTemplate{
		ID:          r.ID,
		Name:        r.Name,
		Type:        r.Type,
		Priority:    r.Priority,
		Description: r.Description,
		CreatedAt:   r.CreatedAt,
	}
}
