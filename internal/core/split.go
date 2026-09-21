package core

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/tomas-chudjak/kleidi-task/internal/db/generated"
)

// ErrHasOpenChildren is returned when completing a parent whose children are
// not all done.
var ErrHasOpenChildren = fmt.Errorf("%w: task has open children", ErrInvalidInput)

// Split breaks a task into ordered child tasks — the PR-sized chunks the work
// will actually land in. Children inherit the parent's type, priority and
// category; a child with no description of its own gets the type's template
// skeleton, since the parent carries the spec and a child is a slice of it.
//
// Nesting is limited to one level: splitting a task that already has a parent
// is rejected.
func (s *TaskService) Split(ctx context.Context, parentID int64, children []ChildSpec) ([]Task, error) {
	parent, err := s.Get(ctx, parentID)
	if err != nil {
		return nil, err
	}
	if parent.ParentID != nil {
		return nil, fmt.Errorf("%w: task #%d is already a child of #%d — children cannot be split further", ErrInvalidInput, parentID, *parent.ParentID)
	}
	if len(children) == 0 {
		return nil, fmt.Errorf("%w: at least one child is required", ErrInvalidInput)
	}

	for i, c := range children {
		if strings.TrimSpace(c.Title) == "" {
			return nil, fmt.Errorf("%w: child %d has an empty title", ErrInvalidInput, i+1)
		}
	}

	order, err := s.queries.MaxChildOrder(ctx, sql.NullInt64{Int64: parentID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("getting child order: %w", err)
	}

	created := make([]Task, 0, len(children))
	for _, c := range children {
		order++

		description := c.Description
		if strings.TrimSpace(description) == "" {
			description = s.GetTemplateForType(ctx, string(parent.Type))
		}

		child, err := s.Create(ctx, CreateTaskInput{
			Title:       strings.TrimSpace(c.Title),
			Description: description,
			Type:        parent.Type,
			Priority:    parent.Priority,
			Category:    parent.Category,
			Source:      parent.Source,
		})
		if err != nil {
			return created, fmt.Errorf("creating child %q: %w", c.Title, err)
		}

		if err := s.queries.SetTaskParent(ctx, generated.SetTaskParentParams{
			ParentID:   sql.NullInt64{Int64: parentID, Valid: true},
			ChildOrder: order,
			ID:         child.ID,
		}); err != nil {
			return created, fmt.Errorf("linking child #%d: %w", child.ID, err)
		}

		child.ParentID = &parentID
		child.ChildOrder = order
		created = append(created, child)
	}

	return created, nil
}

// Children returns a parent's child tasks in the order they should land.
func (s *TaskService) Children(ctx context.Context, parentID int64) ([]Task, error) {
	rows, err := s.queries.ListChildren(ctx, sql.NullInt64{Int64: parentID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("listing children of #%d: %w", parentID, err)
	}
	tasks := make([]Task, len(rows))
	for i, r := range rows {
		tasks[i] = taskFromRow(r)
	}
	return tasks, nil
}

// OpenChildCount returns how many of a task's children are not yet done.
func (s *TaskService) OpenChildCount(ctx context.Context, parentID int64) (int64, error) {
	return s.queries.CountOpenChildren(ctx, sql.NullInt64{Int64: parentID, Valid: true})
}

// attachChildProgress fills ChildTotal and ChildDone on any task in the slice
// that has children. One query covers the whole page.
func (s *TaskService) attachChildProgress(ctx context.Context, tasks []Task) {
	if len(tasks) == 0 {
		return
	}

	rows, err := s.queries.ListChildProgress(ctx)
	if err != nil || len(rows) == 0 {
		return
	}

	type progress struct{ total, done int64 }
	byParent := make(map[int64]progress, len(rows))
	for _, r := range rows {
		if !r.ParentID.Valid {
			continue
		}
		byParent[r.ParentID.Int64] = progress{total: r.Total, done: r.Done}
	}

	for i := range tasks {
		if p, ok := byParent[tasks[i].ID]; ok {
			tasks[i].ChildTotal = p.total
			tasks[i].ChildDone = p.done
		}
	}
}
