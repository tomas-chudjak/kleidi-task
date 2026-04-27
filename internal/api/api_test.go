package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/ahoylog/kvik-tasks/internal/db"
)

func setupTestAPI(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	tmpDir := t.TempDir()
	registryDir := t.TempDir()

	manager, err := db.NewManagerWithRegistryDir(registryDir)
	if err != nil {
		t.Fatalf("creating manager: %v", err)
	}

	projectService := core.NewProjectService(manager)
	project, err := projectService.Init(tmpDir, "test")
	if err != nil {
		manager.Close()
		t.Fatalf("initializing project: %v", err)
	}

	router := NewRouter(projectService)
	server := httptest.NewServer(router)

	t.Cleanup(func() {
		server.Close()
		manager.Close()
	})

	return server, project.Slug
}

func TestHealthAndVersion(t *testing.T) {
	server, _ := setupTestAPI(t)

	resp, err := http.Get(server.URL + "/api/v1/health")
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("health: expected 200, got %d", resp.StatusCode)
	}

	resp, err = http.Get(server.URL + "/api/v1/version")
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("version: expected 200, got %d", resp.StatusCode)
	}
}

func TestAPIFullLifecycle(t *testing.T) {
	server, slug := setupTestAPI(t)
	base := server.URL + "/api/v1/projects/" + slug

	// 1. Create task
	body := `{"title":"API lifecycle test","type":"feature","priority":5}`
	resp, err := http.Post(base+"/tasks", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", resp.StatusCode)
	}
	var created core.Task
	json.NewDecoder(resp.Body).Decode(&created)
	if created.Type != core.TypeFeature {
		t.Errorf("expected type 'feature', got '%s'", created.Type)
	}
	if created.Priority != 5 {
		t.Errorf("expected priority 5, got %d", created.Priority)
	}
	if created.Source != core.SourceAPI {
		t.Errorf("expected source 'api', got '%s'", created.Source)
	}

	// 2. Get task
	resp, err = http.Get(base + "/tasks/1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", resp.StatusCode)
	}

	// 3. Update task
	updateBody := `{"status":"doing","title":"Updated title"}`
	req, _ := http.NewRequest(http.MethodPatch, base+"/tasks/1", bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update: expected 200, got %d", resp.StatusCode)
	}
	var updated core.Task
	json.NewDecoder(resp.Body).Decode(&updated)
	if updated.Status != core.StatusDoing {
		t.Errorf("expected status 'doing', got '%s'", updated.Status)
	}
	if updated.Title != "Updated title" {
		t.Errorf("expected 'Updated title', got '%s'", updated.Title)
	}

	// 4. Complete
	resp, err = http.Post(base+"/tasks/1/complete", "", nil)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("complete: expected 200, got %d", resp.StatusCode)
	}

	// 5. Stats
	resp, err = http.Get(server.URL + "/api/v1/projects/" + slug + "/stats")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	defer resp.Body.Close()
	var stats core.ProjectStats
	json.NewDecoder(resp.Body).Decode(&stats)
	if stats.Done != 1 {
		t.Errorf("expected 1 done, got %d", stats.Done)
	}

	// 6. Delete
	req, _ = http.NewRequest(http.MethodDelete, base+"/tasks/1", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d", resp.StatusCode)
	}

	// 7. Verify deleted
	resp, err = http.Get(base + "/tasks/1")
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", resp.StatusCode)
	}
}

func TestAPIPrefixDetection(t *testing.T) {
	server, slug := setupTestAPI(t)
	base := server.URL + "/api/v1/projects/" + slug

	body := `{"title":"BUG: auto detected"}`
	resp, err := http.Post(base+"/tasks", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", resp.StatusCode)
	}
	var task core.Task
	json.NewDecoder(resp.Body).Decode(&task)
	if task.Type != core.TypeBug {
		t.Errorf("expected type 'bug', got '%s'", task.Type)
	}
	if task.Title != "auto detected" {
		t.Errorf("expected title 'auto detected', got '%s'", task.Title)
	}
}

func TestAPINotFound(t *testing.T) {
	server, slug := setupTestAPI(t)

	// Non-existent task
	resp, err := http.Get(server.URL + "/api/v1/projects/" + slug + "/tasks/999")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}

	// Non-existent project
	resp, err = http.Get(server.URL + "/api/v1/projects/nonexistent")
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent project, got %d", resp.StatusCode)
	}
}
