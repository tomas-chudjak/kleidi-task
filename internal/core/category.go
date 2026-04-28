package core

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ahoylog/kvik-tasks/internal/db/generated"
)

// CategoryService handles category business logic.
type CategoryService struct {
	queries *generated.Queries
}

// NewCategoryService creates a new CategoryService.
func NewCategoryService(db *sql.DB) *CategoryService {
	return &CategoryService{queries: generated.New(db)}
}

// List returns all categories for the project.
func (s *CategoryService) List(ctx context.Context) ([]Category, error) {
	rows, err := s.queries.ListCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing categories: %w", err)
	}

	categories := make([]Category, len(rows))
	for i, row := range rows {
		categories[i] = Category{
			ID:    row.ID,
			Name:  row.Name,
			Color: row.Color,
		}
	}
	return categories, nil
}

// Create adds a new category.
func (s *CategoryService) Create(ctx context.Context, name, color string) (Category, error) {
	if name == "" {
		return Category{}, fmt.Errorf("%w: category name is required", ErrInvalidInput)
	}
	if color == "" {
		color = "#8a8dab"
	}

	row, err := s.queries.CreateCategory(ctx, generated.CreateCategoryParams{
		Name:  name,
		Color: color,
	})
	if err != nil {
		return Category{}, fmt.Errorf("creating category: %w", err)
	}

	return Category{ID: row.ID, Name: row.Name, Color: row.Color}, nil
}

// Delete removes a category.
func (s *CategoryService) Delete(ctx context.Context, name string) error {
	if err := s.queries.DeleteCategory(ctx, name); err != nil {
		return fmt.Errorf("deleting category: %w", err)
	}
	return nil
}
