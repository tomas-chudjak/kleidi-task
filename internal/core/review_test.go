package core

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tomas-chudjak/kleidi-task/internal/db"
)

func reviewService(t *testing.T) (*TaskService, context.Context) {
	t.Helper()
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	svc.SetTemplates(NewTemplateService(testDB))
	return svc, context.Background()
}

func TestReviewReportsMissingAndEmptySections(t *testing.T) {
	svc, ctx := reviewService(t)

	task, err := svc.Create(ctx, CreateTaskInput{
		Title:       "Add dark mode",
		Type:        TypeFeature,
		Description: "## Problem\n\nno dark mode\n\n## Proposed solution\n",
		Source:      SourceCLI,
	})
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}

	review, err := svc.Review(ctx, task.ID)
	if err != nil {
		t.Fatalf("review: %v", err)
	}

	if !contains(review.MissingSections, "Acceptance criteria") {
		t.Errorf("expected Acceptance criteria missing, got %v", review.MissingSections)
	}
	// Present but empty is a gap too, and invisible to MissingSections.
	if !contains(review.EmptySections, "Proposed solution") {
		t.Errorf("expected Proposed solution empty, got %v", review.EmptySections)
	}
	// A section with content is neither.
	if contains(review.EmptySections, "Problem") || contains(review.MissingSections, "Problem") {
		t.Error("Problem has content and should be reported as neither")
	}
	if review.Instruction == "" {
		t.Error("expected a seeded review instruction")
	}
}

func TestReviewEmptyOpenQuestionsIsNotAGap(t *testing.T) {
	svc, ctx := reviewService(t)

	task, err := svc.Create(ctx, CreateTaskInput{
		Title:  "scaffolded",
		Type:   TypeFeature,
		Source: SourceCLI,
	})
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}

	review, err := svc.Review(ctx, task.ID)
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if contains(review.EmptySections, OpenQuestionsSection) {
		t.Error("an empty Open questions is the good case, not a gap")
	}
}

func TestReviewSurfacesOpenQuestions(t *testing.T) {
	svc, ctx := reviewService(t)

	task, err := svc.Create(ctx, CreateTaskInput{
		Title:       "Add dark mode",
		Type:        TypeFeature,
		Description: "## Problem\n\nx\n\n## Open questions\n\n- Which token set?\n",
		Source:      SourceCLI,
	})
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}

	review, err := svc.Review(ctx, task.ID)
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if !strings.Contains(review.OpenQuestions, "Which token set?") {
		t.Errorf("expected open questions surfaced, got %q", review.OpenQuestions)
	}
}

func TestReviewPerformsNoWrites(t *testing.T) {
	svc, ctx := reviewService(t)

	const desc = "## Problem\n\nbroken\n"
	task, err := svc.Create(ctx, CreateTaskInput{
		Title:       "Add dark mode",
		Type:        TypeFeature,
		Description: desc,
		Source:      SourceCLI,
	})
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}

	if _, err := svc.Review(ctx, task.ID); err != nil {
		t.Fatalf("review: %v", err)
	}

	after, err := svc.Get(ctx, task.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if after.Description != desc {
		t.Errorf("review modified the description: %q", after.Description)
	}
	if !after.UpdatedAt.Equal(task.UpdatedAt) {
		t.Error("review touched updated_at")
	}
}

func TestReviewUnknownTaskReturnsNotFound(t *testing.T) {
	svc, ctx := reviewService(t)

	if _, err := svc.Review(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestReviewTypeWithoutTemplateErrorsClearly(t *testing.T) {
	svc, ctx := reviewService(t)

	task, err := svc.Create(ctx, CreateTaskInput{
		Title:       "Spike",
		Type:        TaskType("spike"),
		Description: "whatever",
		Source:      SourceCLI,
	})
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}

	_, err = svc.Review(ctx, task.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if !strings.Contains(err.Error(), "spike") {
		t.Errorf("error should name the type: %v", err)
	}
}

func TestReviewWithoutTemplateServiceErrorsClearly(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB) // no SetTemplates
	ctx := context.Background()

	task, err := svc.Create(ctx, CreateTaskInput{Title: "x", Source: SourceCLI})
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}
	if _, err := svc.Review(ctx, task.ID); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}
