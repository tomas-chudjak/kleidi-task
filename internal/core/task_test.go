package core

import (
	"context"
	"errors"
	"testing"

	"github.com/ahoylog/kvik-tasks/internal/db"
)

func TestCreateTask(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	ctx := context.Background()

	task, err := svc.Create(ctx, CreateTaskInput{
		Title:  "Test task",
		Source: SourceCLI,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.ID != 1 {
		t.Errorf("expected ID 1, got %d", task.ID)
	}
	if task.Title != "Test task" {
		t.Errorf("expected title 'Test task', got '%s'", task.Title)
	}
	if task.Type != TypeTask {
		t.Errorf("expected type 'task', got '%s'", task.Type)
	}
	if task.Status != StatusTodo {
		t.Errorf("expected status 'todo', got '%s'", task.Status)
	}
	if task.Source != SourceCLI {
		t.Errorf("expected source 'cli', got '%s'", task.Source)
	}
}

func TestCreateBug(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	ctx := context.Background()

	task, err := svc.Create(ctx, CreateTaskInput{
		Title:  "Login broken",
		Type:   TypeBug,
		Source: SourceMCP,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.Type != TypeBug {
		t.Errorf("expected type 'bug', got '%s'", task.Type)
	}
	if task.Source != SourceMCP {
		t.Errorf("expected source 'mcp', got '%s'", task.Source)
	}
}

func TestCreateTaskValidation(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateTaskInput{Title: "", Source: SourceCLI})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for empty title, got: %v", err)
	}

	_, err = svc.Create(ctx, CreateTaskInput{Title: "test", Source: ""})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for empty source, got: %v", err)
	}
}

func TestGetTask(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	ctx := context.Background()

	created, _ := svc.Create(ctx, CreateTaskInput{Title: "Find me", Source: SourceCLI})

	found, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.Title != "Find me" {
		t.Errorf("expected 'Find me', got '%s'", found.Title)
	}
}

func TestGetTaskNotFound(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	ctx := context.Background()

	_, err := svc.Get(ctx, 999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestListTasks(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	ctx := context.Background()

	svc.Create(ctx, CreateTaskInput{Title: "Low pri", Source: SourceCLI, Priority: 0})
	svc.Create(ctx, CreateTaskInput{Title: "High pri", Source: SourceCLI, Priority: 10})
	svc.Create(ctx, CreateTaskInput{Title: "Med pri", Source: SourceCLI, Priority: 5})

	tasks, err := svc.List(ctx, ListTasksFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}

	// Should be ordered by priority DESC
	if tasks[0].Title != "High pri" {
		t.Errorf("expected first task 'High pri', got '%s'", tasks[0].Title)
	}
	if tasks[1].Title != "Med pri" {
		t.Errorf("expected second task 'Med pri', got '%s'", tasks[1].Title)
	}
}

func TestListTasksFilterByType(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	ctx := context.Background()

	svc.Create(ctx, CreateTaskInput{Title: "Task 1", Source: SourceCLI})
	svc.Create(ctx, CreateTaskInput{Title: "Bug 1", Type: TypeBug, Source: SourceCLI})

	bugType := TypeBug
	tasks, err := svc.List(ctx, ListTasksFilter{Type: &bugType})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("expected 1 bug, got %d", len(tasks))
	}
	if tasks[0].Title != "Bug 1" {
		t.Errorf("expected 'Bug 1', got '%s'", tasks[0].Title)
	}
}

func TestListTasksFilterByStatus(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	ctx := context.Background()

	t1, _ := svc.Create(ctx, CreateTaskInput{Title: "Todo", Source: SourceCLI})
	svc.Create(ctx, CreateTaskInput{Title: "Also todo", Source: SourceCLI})
	svc.Complete(ctx, t1.ID)

	todoStatus := StatusTodo
	tasks, err := svc.List(ctx, ListTasksFilter{Status: &todoStatus})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("expected 1 todo task, got %d", len(tasks))
	}
}

func TestCompleteTask(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	ctx := context.Background()

	created, _ := svc.Create(ctx, CreateTaskInput{Title: "Complete me", Source: SourceCLI})

	completed, err := svc.Complete(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if completed.Status != StatusDone {
		t.Errorf("expected status 'done', got '%s'", completed.Status)
	}

	// Verify via Get that completed_at is set (trigger should fire)
	fetched, _ := svc.Get(ctx, created.ID)
	if fetched.CompletedAt == nil {
		t.Error("expected completed_at to be set after completion")
	}
}

func TestUpdateTask(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	ctx := context.Background()

	created, _ := svc.Create(ctx, CreateTaskInput{Title: "Original", Source: SourceCLI})

	newTitle := "Updated"
	newPri := int64(5)
	updated, err := svc.Update(ctx, created.ID, UpdateTaskInput{
		Title:    &newTitle,
		Priority: &newPri,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.Title != "Updated" {
		t.Errorf("expected title 'Updated', got '%s'", updated.Title)
	}
	if updated.Priority != 5 {
		t.Errorf("expected priority 5, got %d", updated.Priority)
	}
}

func TestUpdateTaskPartial(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	ctx := context.Background()

	created, _ := svc.Create(ctx, CreateTaskInput{
		Title:    "Original",
		Source:   SourceCLI,
		Priority: 3,
	})

	// Update only title, priority should remain
	newTitle := "New title"
	updated, err := svc.Update(ctx, created.ID, UpdateTaskInput{Title: &newTitle})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.Title != "New title" {
		t.Errorf("expected 'New title', got '%s'", updated.Title)
	}
	if updated.Priority != 3 {
		t.Errorf("expected priority to remain 3, got %d", updated.Priority)
	}
}

func TestDeleteTask(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	ctx := context.Background()

	created, _ := svc.Create(ctx, CreateTaskInput{Title: "Delete me", Source: SourceCLI})

	err := svc.Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.Get(ctx, created.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got: %v", err)
	}
}

func TestDeleteTaskNotFound(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	ctx := context.Background()

	err := svc.Delete(ctx, 999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestStats(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	ctx := context.Background()

	svc.Create(ctx, CreateTaskInput{Title: "Todo 1", Source: SourceCLI})
	svc.Create(ctx, CreateTaskInput{Title: "Todo 2", Source: SourceCLI})
	svc.Create(ctx, CreateTaskInput{Title: "Bug", Type: TypeBug, Source: SourceCLI})

	task3, _ := svc.Create(ctx, CreateTaskInput{Title: "Done", Source: SourceCLI})
	svc.Complete(ctx, task3.ID)

	stats, err := svc.Stats(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.Todo != 3 {
		t.Errorf("expected 3 todo, got %d", stats.Todo)
	}
	if stats.Done != 1 {
		t.Errorf("expected 1 done, got %d", stats.Done)
	}
	if stats.BugsOpen != 1 {
		t.Errorf("expected 1 open bug, got %d", stats.BugsOpen)
	}
}
