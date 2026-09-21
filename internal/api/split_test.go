package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/tomas-chudjak/kleidi-task/internal/core"
)

// createTask posts a task and returns it.
func createTask(t *testing.T, base, body string) core.Task {
	t.Helper()
	resp, err := http.Post(base+"/tasks", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("creating task: expected 201, got %d", resp.StatusCode)
	}
	var task core.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		t.Fatalf("decoding task: %v", err)
	}
	return task
}

// splitTask posts a split request and returns the response status and children.
func splitTask(t *testing.T, base string, id int64, body string) (int, []core.Task) {
	t.Helper()
	url := fmt.Sprintf("%s/tasks/%d/split", base, id)
	resp, err := http.Post(url, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("splitting task: %v", err)
	}
	defer resp.Body.Close()

	var children []core.Task
	if resp.StatusCode == http.StatusCreated {
		if err := json.NewDecoder(resp.Body).Decode(&children); err != nil {
			t.Fatalf("decoding children: %v", err)
		}
	}
	return resp.StatusCode, children
}

func TestAPISplitAndChildren(t *testing.T) {
	server, slug := setupTestAPI(t)
	base := server.URL + "/api/v1/projects/" + slug

	parent := createTask(t, base, `{"title":"MCP HTTP transport","type":"feature","priority":5}`)

	status, children := splitTask(t, base, parent.ID,
		`{"children":[{"title":"Schema and migration"},{"title":"Service layer"},{"title":"MCP and CLI wiring"}]}`)
	if status != http.StatusCreated {
		t.Fatalf("split: expected 201, got %d", status)
	}
	if len(children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(children))
	}
	for i, c := range children {
		if c.ChildOrder != int64(i+1) {
			t.Errorf("child %d has order %d", i, c.ChildOrder)
		}
		if c.ParentID == nil || *c.ParentID != parent.ID {
			t.Errorf("child %d is not linked to parent #%d", i, parent.ID)
		}
	}

	// GET children returns them in order.
	resp, err := http.Get(fmt.Sprintf("%s/tasks/%d/children", base, parent.ID))
	if err != nil {
		t.Fatalf("getting children: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("children: expected 200, got %d", resp.StatusCode)
	}
	var listed []core.Task
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		t.Fatalf("decoding children: %v", err)
	}
	if len(listed) != 3 || listed[0].Title != "Schema and migration" || listed[2].Title != "MCP and CLI wiring" {
		t.Errorf("children out of order: %v", listed)
	}
}

func TestAPIGetReportsChildProgress(t *testing.T) {
	server, slug := setupTestAPI(t)
	base := server.URL + "/api/v1/projects/" + slug

	parent := createTask(t, base, `{"title":"Parent","type":"feature"}`)
	_, children := splitTask(t, base, parent.ID, `{"children":[{"title":"a"},{"title":"b"}]}`)

	// Complete one child.
	resp, err := http.Post(fmt.Sprintf("%s/tasks/%d/complete", base, children[0].ID), "application/json", nil)
	if err != nil {
		t.Fatalf("completing child: %v", err)
	}
	resp.Body.Close()

	resp, err = http.Get(fmt.Sprintf("%s/tasks/%d", base, parent.ID))
	if err != nil {
		t.Fatalf("getting parent: %v", err)
	}
	defer resp.Body.Close()
	var got core.Task
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decoding parent: %v", err)
	}
	if got.ChildTotal != 2 || got.ChildDone != 1 {
		t.Errorf("expected 1/2, got %d/%d", got.ChildDone, got.ChildTotal)
	}
}

func TestAPIParentCannotCompleteWithOpenChildren(t *testing.T) {
	server, slug := setupTestAPI(t)
	base := server.URL + "/api/v1/projects/" + slug

	parent := createTask(t, base, `{"title":"Parent","type":"feature"}`)
	splitTask(t, base, parent.ID, `{"children":[{"title":"a"}]}`)

	resp, err := http.Post(fmt.Sprintf("%s/tasks/%d/complete", base, parent.ID), "application/json", nil)
	if err != nil {
		t.Fatalf("completing parent: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for open children, got %d", resp.StatusCode)
	}
}

func TestAPISplitRejectsNesting(t *testing.T) {
	server, slug := setupTestAPI(t)
	base := server.URL + "/api/v1/projects/" + slug

	parent := createTask(t, base, `{"title":"Parent","type":"feature"}`)
	_, children := splitTask(t, base, parent.ID, `{"children":[{"title":"child"}]}`)

	status, _ := splitTask(t, base, children[0].ID, `{"children":[{"title":"grandchild"}]}`)
	if status != http.StatusBadRequest {
		t.Errorf("expected 400 for nesting, got %d", status)
	}
}

func TestAPISplitRejectsBadInput(t *testing.T) {
	server, slug := setupTestAPI(t)
	base := server.URL + "/api/v1/projects/" + slug

	parent := createTask(t, base, `{"title":"Parent","type":"feature"}`)

	if status, _ := splitTask(t, base, parent.ID, `{"children":[]}`); status != http.StatusBadRequest {
		t.Errorf("expected 400 for no children, got %d", status)
	}
	if status, _ := splitTask(t, base, parent.ID, `{"children":[{"title":"  "}]}`); status != http.StatusBadRequest {
		t.Errorf("expected 400 for blank title, got %d", status)
	}
	if status, _ := splitTask(t, base, parent.ID, `not json`); status != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed body, got %d", status)
	}
}

func TestAPISplitUnknownTask(t *testing.T) {
	server, slug := setupTestAPI(t)
	base := server.URL + "/api/v1/projects/" + slug

	if status, _ := splitTask(t, base, 9999, `{"children":[{"title":"x"}]}`); status != http.StatusNotFound {
		t.Errorf("expected 404, got %d", status)
	}
}
