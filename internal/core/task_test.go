package core

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/tomas-chudjak/kleidi-task/internal/db"
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

func TestCreateAllTypes(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	ctx := context.Background()

	tests := []struct {
		title    string
		taskType TaskType
	}{
		{"Login broken", TypeBug},
		{"Add dark mode", TypeFeature},
		{"Fix crash on start", TypeHotfix},
		{"Regular work", TypeTask},
	}

	for _, tt := range tests {
		task, err := svc.Create(ctx, CreateTaskInput{
			Title:  tt.title,
			Type:   tt.taskType,
			Source: SourceCLI,
		})
		if err != nil {
			t.Fatalf("creating %s: %v", tt.taskType, err)
		}
		if task.Type != tt.taskType {
			t.Errorf("expected type '%s', got '%s'", tt.taskType, task.Type)
		}
	}
}

func TestDetectTypeFromTitle(t *testing.T) {
	tests := []struct {
		input    string
		wantType TaskType
		wantTitle string
	}{
		{"BUG: login broken", TypeBug, "login broken"},
		{"bug: login broken", TypeBug, "login broken"},
		{"bug login broken", TypeBug, "login broken"},
		{"FEATURE: dark mode", TypeFeature, "dark mode"},
		{"feat: dark mode", TypeFeature, "dark mode"},
		{"feat dark mode", TypeFeature, "dark mode"},
		{"HOTFIX: crash fix", TypeHotfix, "crash fix"},
		{"hotfix: crash fix", TypeHotfix, "crash fix"},
		{"TASK: normal work", TypeTask, "normal work"},
		{"todo: buy milk", TypeTask, "buy milk"},
		{"TODO: buy milk", TypeTask, "buy milk"},
		{"todo buy milk", TypeTask, "buy milk"},
		{"just a normal title", TypeTask, "just a normal title"},
		{"buggy behavior", TypeTask, "buggy behavior"}, // "buggy" != "bug "
		{"todolist app", TypeTask, "todolist app"},       // "todolist" != "todo "
	}

	for _, tt := range tests {
		gotType, gotTitle := DetectTypeFromTitle(tt.input, TypeTask)
		if gotType != tt.wantType {
			t.Errorf("DetectTypeFromTitle(%q): type = %s, want %s", tt.input, gotType, tt.wantType)
		}
		if gotTitle != tt.wantTitle {
			t.Errorf("DetectTypeFromTitle(%q): title = %q, want %q", tt.input, gotTitle, tt.wantTitle)
		}
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

func TestListWithPagination(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		svc.Create(ctx, CreateTaskInput{Title: fmt.Sprintf("Task %d", i+1), Source: SourceCLI})
	}

	// Page 1 of 2
	result, err := svc.ListWithCount(ctx, ListTasksFilter{Limit: 3, Offset: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 5 {
		t.Errorf("expected total 5, got %d", result.Total)
	}
	if len(result.Tasks) != 3 {
		t.Errorf("expected 3 tasks on page 1, got %d", len(result.Tasks))
	}
	if result.TotalPages != 2 {
		t.Errorf("expected 2 total pages, got %d", result.TotalPages)
	}
	if result.Page != 1 {
		t.Errorf("expected page 1, got %d", result.Page)
	}

	// Page 2
	result2, _ := svc.ListWithCount(ctx, ListTasksFilter{Limit: 3, Offset: 3})
	if len(result2.Tasks) != 2 {
		t.Errorf("expected 2 tasks on page 2, got %d", len(result2.Tasks))
	}
	if result2.Page != 2 {
		t.Errorf("expected page 2, got %d", result2.Page)
	}
}

func TestListWithPriorityFilter(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	ctx := context.Background()

	svc.Create(ctx, CreateTaskInput{Title: "Low", Source: SourceCLI, Priority: 1})
	svc.Create(ctx, CreateTaskInput{Title: "Med", Source: SourceCLI, Priority: 5})
	svc.Create(ctx, CreateTaskInput{Title: "High", Source: SourceCLI, Priority: 10})

	minPri := int64(5)
	result, err := svc.ListWithCount(ctx, ListTasksFilter{MinPriority: &minPri})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected 2 tasks with priority >= 5, got %d", result.Total)
	}
	if len(result.Tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(result.Tasks))
	}
}

func TestListWithDateFilter(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	ctx := context.Background()

	svc.Create(ctx, CreateTaskInput{Title: "Task A", Source: SourceCLI})
	svc.Create(ctx, CreateTaskInput{Title: "Task B", Source: SourceCLI})

	// Filter with a future date — should return all tasks
	after := "2020-01-01T00:00:00Z"
	result, err := svc.ListWithCount(ctx, ListTasksFilter{CreatedAfter: &after})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected 2 tasks after 2020, got %d", result.Total)
	}

	// Filter with a past date — should return nothing
	before := "2020-01-01T00:00:00Z"
	result2, _ := svc.ListWithCount(ctx, ListTasksFilter{CreatedBefore: &before})
	if result2.Total != 0 {
		t.Errorf("expected 0 tasks before 2020, got %d", result2.Total)
	}
}

func TestSearch(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	ctx := context.Background()

	svc.Create(ctx, CreateTaskInput{Title: "Fix login page", Description: "The login form crashes on Safari", Source: SourceCLI})
	svc.Create(ctx, CreateTaskInput{Title: "Add dark mode", Description: "User requested dark theme support", Source: SourceCLI})
	svc.Create(ctx, CreateTaskInput{Title: "Refactor auth module", Source: SourceCLI})

	// Search by title
	results, err := svc.Search(ctx, "login", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'login', got %d", len(results))
	}

	// Search by description
	results2, _ := svc.Search(ctx, "Safari", 10)
	if len(results2) != 1 {
		t.Errorf("expected 1 result for 'Safari', got %d", len(results2))
	}

	// Search matching multiple
	results3, _ := svc.Search(ctx, "dark OR auth", 10)
	if len(results3) != 2 {
		t.Errorf("expected 2 results for 'dark OR auth', got %d", len(results3))
	}

	// Empty query
	_, err = svc.Search(ctx, "", 10)
	if err == nil {
		t.Error("expected error for empty query")
	}
}

func TestListWithMultiSelectStatus(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	ctx := context.Background()

	t1, _ := svc.Create(ctx, CreateTaskInput{Title: "Todo task", Source: SourceCLI})
	t2, _ := svc.Create(ctx, CreateTaskInput{Title: "Doing task", Source: SourceCLI})
	svc.Update(ctx, t2.ID, UpdateTaskInput{Status: func() *TaskStatus { s := StatusDoing; return &s }()})
	t3, _ := svc.Create(ctx, CreateTaskInput{Title: "Done task", Source: SourceCLI})
	svc.Complete(ctx, t3.ID)

	// Multi-select: todo + doing
	tasks, err := svc.List(ctx, ListTasksFilter{Status: "todo,doing"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks (todo+doing), got %d", len(tasks))
	}

	// Single select still works
	tasks2, _ := svc.List(ctx, ListTasksFilter{Status: "done"})
	if len(tasks2) != 1 {
		t.Errorf("expected 1 done task, got %d", len(tasks2))
	}

	_ = t1
}

func TestListWithMultiSelectType(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	ctx := context.Background()

	svc.Create(ctx, CreateTaskInput{Title: "Task 1", Source: SourceCLI})
	svc.Create(ctx, CreateTaskInput{Title: "Bug 1", Type: TypeBug, Source: SourceCLI})
	svc.Create(ctx, CreateTaskInput{Title: "Feature 1", Type: TypeFeature, Source: SourceCLI})

	// Multi-select: bug + feature
	tasks, err := svc.List(ctx, ListTasksFilter{Type: "bug,feature"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks (bug+feature), got %d", len(tasks))
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

	tasks, err := svc.List(ctx, ListTasksFilter{Type: "bug"})
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

	tasks, err := svc.List(ctx, ListTasksFilter{Status: "todo"})
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

func TestGetTemplateForType(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	tplService := NewTemplateService(testDB)
	svc.SetTemplates(tplService)
	ctx := context.Background()

	// Custom type with known template
	_, err := tplService.Create(ctx, "Spike Template", "spike", 3, "## Goal\n\n## Timebox\n\n## Findings")
	if err != nil {
		t.Fatalf("creating template: %v", err)
	}

	// Should return template description
	got := svc.GetTemplateForType(ctx, "spike")
	if got != "## Goal\n\n## Timebox\n\n## Findings" {
		t.Errorf("expected spike template, got: %q", got)
	}

	// Unknown type — should return empty
	got2 := svc.GetTemplateForType(ctx, "epic")
	if got2 != "" {
		t.Errorf("expected empty for unknown type, got: %q", got2)
	}

	// Pre-seeded bug template should exist
	got3 := svc.GetTemplateForType(ctx, "bug")
	if got3 == "" {
		t.Error("expected pre-seeded bug template, got empty")
	}
}

func TestGetTemplateForTypeWithoutTemplates(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	// No SetTemplates called
	ctx := context.Background()

	got := svc.GetTemplateForType(ctx, "bug")
	if got != "" {
		t.Errorf("expected empty when templates not configured, got: %q", got)
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
