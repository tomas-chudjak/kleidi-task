package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExportJSON(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "Test task", Type: TypeTask, Status: StatusTodo, Priority: 3},
		{ID: 2, Title: "Fix bug", Type: TypeBug, Status: StatusDone, Priority: 5, Category: "backend"},
	}

	data, err := ExportJSON("test-project", tasks)
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	var export ExportData
	if err := json.Unmarshal(data, &export); err != nil {
		t.Fatalf("Failed to parse exported JSON: %v", err)
	}

	if export.Project != "test-project" {
		t.Errorf("expected project 'test-project', got %q", export.Project)
	}
	if export.Count != 2 {
		t.Errorf("expected count 2, got %d", export.Count)
	}
	if len(export.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(export.Tasks))
	}
	if export.Tasks[0].Title != "Test task" {
		t.Errorf("expected title 'Test task', got %q", export.Tasks[0].Title)
	}
	if export.Tasks[1].Category != "backend" {
		t.Errorf("expected category 'backend', got %q", export.Tasks[1].Category)
	}
}

func TestExportMarkdown(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "Test task", Type: TypeTask, Status: StatusTodo, Priority: 3},
		{ID: 2, Title: "Fix bug", Type: TypeBug, Status: StatusDone, Priority: 5, Description: "Steps to reproduce"},
	}

	md := string(ExportMarkdown("test-project", tasks))

	if !strings.Contains(md, "# test-project") {
		t.Error("missing project header")
	}
	if !strings.Contains(md, "## Todo") {
		t.Error("missing Todo section")
	}
	if !strings.Contains(md, "## Done") {
		t.Error("missing Done section")
	}
	if !strings.Contains(md, "- [ ] **#1** Test task") {
		t.Error("missing todo task checkbox")
	}
	if !strings.Contains(md, "- [x] **#2** Fix bug") {
		t.Error("missing done task checkbox")
	}
	if !strings.Contains(md, "> Steps to reproduce") {
		t.Error("missing description quote")
	}
}

func TestImportJSON_Roundtrip(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "Task one", Type: TypeTask, Status: StatusTodo, Priority: 3},
		{ID: 2, Title: "Task two", Type: TypeFeature, Status: StatusDoing, Priority: 7, Description: "Feature desc", Category: "frontend"},
	}

	data, err := ExportJSON("test", tasks)
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	imported, err := ImportJSON(data)
	if err != nil {
		t.Fatalf("ImportJSON failed: %v", err)
	}

	if imported.Count != 2 {
		t.Errorf("expected count 2, got %d", imported.Count)
	}
	if len(imported.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(imported.Tasks))
	}
	if imported.Tasks[0].Title != "Task one" {
		t.Errorf("expected 'Task one', got %q", imported.Tasks[0].Title)
	}
	if imported.Tasks[1].Type != TypeFeature {
		t.Errorf("expected type feature, got %q", imported.Tasks[1].Type)
	}
	if imported.Tasks[1].Description != "Feature desc" {
		t.Errorf("expected description 'Feature desc', got %q", imported.Tasks[1].Description)
	}
	if imported.Tasks[1].Category != "frontend" {
		t.Errorf("expected category 'frontend', got %q", imported.Tasks[1].Category)
	}
}

func TestImportJSON_InvalidData(t *testing.T) {
	_, err := ImportJSON([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
