package core

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tomas-chudjak/kleidi-task/internal/db"
)

func splitFixture(t *testing.T) (*TaskService, context.Context, Task) {
	t.Helper()
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	svc.SetTemplates(NewTemplateService(testDB))
	ctx := context.Background()

	parent, err := svc.Create(ctx, CreateTaskInput{
		Title:    "MCP HTTP transport",
		Type:     TypeFeature,
		Priority: 5,
		Category: "backend",
		Source:   SourceCLI,
	})
	if err != nil {
		t.Fatalf("creating parent: %v", err)
	}
	return svc, ctx, parent
}

func TestSplitCreatesOrderedChildren(t *testing.T) {
	svc, ctx, parent := splitFixture(t)

	children, err := svc.Split(ctx, parent.ID, []ChildSpec{
		{Title: "Schema and migration"},
		{Title: "Service layer"},
		{Title: "MCP and CLI wiring"},
	})
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	if len(children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(children))
	}

	for i, c := range children {
		if c.ChildOrder != int64(i+1) {
			t.Errorf("child %d has order %d", i, c.ChildOrder)
		}
		if c.ParentID == nil || *c.ParentID != parent.ID {
			t.Errorf("child %d is not linked to the parent", i)
		}
		// Children inherit the parent's classification.
		if c.Type != parent.Type || c.Priority != parent.Priority || c.Category != parent.Category {
			t.Errorf("child %d did not inherit type/priority/category", i)
		}
	}

	// Reading them back preserves the order the work should land in.
	listed, err := svc.Children(ctx, parent.ID)
	if err != nil {
		t.Fatalf("listing children: %v", err)
	}
	if len(listed) != 3 || listed[0].Title != "Schema and migration" || listed[2].Title != "MCP and CLI wiring" {
		t.Errorf("children out of order: %v", listed)
	}
}

func TestSplitScaffoldsChildDescriptions(t *testing.T) {
	svc, ctx, parent := splitFixture(t)

	children, err := svc.Split(ctx, parent.ID, []ChildSpec{
		{Title: "no description of its own"},
		{Title: "has one", Description: "## Problem\n\nspecific\n"},
	})
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	if !strings.Contains(children[0].Description, "## Problem") {
		t.Errorf("expected template skeleton, got %q", children[0].Description)
	}
	if children[1].Description != "## Problem\n\nspecific\n" {
		t.Errorf("explicit description was overwritten: %q", children[1].Description)
	}
}

func TestSplitRejectsNesting(t *testing.T) {
	svc, ctx, parent := splitFixture(t)

	children, err := svc.Split(ctx, parent.ID, []ChildSpec{{Title: "child"}})
	if err != nil {
		t.Fatalf("split: %v", err)
	}

	_, err = svc.Split(ctx, children[0].ID, []ChildSpec{{Title: "grandchild"}})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for nesting, got %v", err)
	}
}

func TestSplitRejectsEmptyInput(t *testing.T) {
	svc, ctx, parent := splitFixture(t)

	if _, err := svc.Split(ctx, parent.ID, nil); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected error for no children, got %v", err)
	}
	if _, err := svc.Split(ctx, parent.ID, []ChildSpec{{Title: "  "}}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected error for blank title, got %v", err)
	}
}

func TestSplitAppendsToExistingChildren(t *testing.T) {
	svc, ctx, parent := splitFixture(t)

	if _, err := svc.Split(ctx, parent.ID, []ChildSpec{{Title: "first"}}); err != nil {
		t.Fatalf("first split: %v", err)
	}
	more, err := svc.Split(ctx, parent.ID, []ChildSpec{{Title: "second"}})
	if err != nil {
		t.Fatalf("second split: %v", err)
	}
	if more[0].ChildOrder != 2 {
		t.Errorf("expected order 2, got %d", more[0].ChildOrder)
	}
}

func TestParentCannotCompleteWithOpenChildren(t *testing.T) {
	svc, ctx, parent := splitFixture(t)

	children, err := svc.Split(ctx, parent.ID, []ChildSpec{{Title: "a"}, {Title: "b"}})
	if err != nil {
		t.Fatalf("split: %v", err)
	}

	_, err = svc.Complete(ctx, parent.ID)
	if !errors.Is(err, ErrHasOpenChildren) {
		t.Fatalf("expected ErrHasOpenChildren, got %v", err)
	}
	if !strings.Contains(err.Error(), "2 open") {
		t.Errorf("error should count open children: %v", err)
	}

	// Finishing the children unblocks the parent.
	for _, c := range children {
		if _, err := svc.Complete(ctx, c.ID); err != nil {
			t.Fatalf("completing child #%d: %v", c.ID, err)
		}
	}
	if _, err := svc.Complete(ctx, parent.ID); err != nil {
		t.Fatalf("parent should complete once children are done: %v", err)
	}
}

func TestChildProgressReportedOnGetAndList(t *testing.T) {
	svc, ctx, parent := splitFixture(t)

	children, err := svc.Split(ctx, parent.ID, []ChildSpec{{Title: "a"}, {Title: "b"}, {Title: "c"}})
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	if _, err := svc.Complete(ctx, children[0].ID); err != nil {
		t.Fatalf("completing child: %v", err)
	}

	got, err := svc.Get(ctx, parent.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ChildTotal != 3 || got.ChildDone != 1 {
		t.Errorf("expected 1/3, got %d/%d", got.ChildDone, got.ChildTotal)
	}
	if !got.IsParent() {
		t.Error("parent should report IsParent")
	}

	tasks, err := svc.List(ctx, ListTasksFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, task := range tasks {
		switch {
		case task.ID == parent.ID:
			if task.ChildTotal != 3 || task.ChildDone != 1 {
				t.Errorf("list: expected 1/3, got %d/%d", task.ChildDone, task.ChildTotal)
			}
		case task.IsChild():
			if task.ChildTotal != 0 {
				t.Errorf("a child should report no children of its own")
			}
		}
	}
}

func TestTaskWithoutChildrenReportsNoProgress(t *testing.T) {
	svc, ctx, parent := splitFixture(t)

	got, err := svc.Get(ctx, parent.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.IsParent() || got.ChildTotal != 0 {
		t.Errorf("unsplit task should report no children, got %d", got.ChildTotal)
	}
	if _, err := svc.Complete(ctx, parent.ID); err != nil {
		t.Errorf("unsplit task should complete freely: %v", err)
	}
}
