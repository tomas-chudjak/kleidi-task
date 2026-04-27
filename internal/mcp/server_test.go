package mcp

import (
	"context"
	"testing"

	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/ahoylog/kvik-tasks/internal/db"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func setupTestServer(t *testing.T) (*mcp.ClientSession, string, func()) {
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
	slug := project.Slug

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "kvik-tasks-test",
		Version: "test",
	}, nil)

	s := &Server{
		mcpServer:      mcpServer,
		manager:        manager,
		projectService: projectService,
	}
	s.registerTools()
	s.registerResources()

	ctx := context.Background()
	ct, st := mcp.NewInMemoryTransports()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "test",
	}, nil)

	_, err = mcpServer.Connect(ctx, st, nil)
	if err != nil {
		manager.Close()
		t.Fatalf("connecting server: %v", err)
	}

	session, err := client.Connect(ctx, ct, nil)
	if err != nil {
		manager.Close()
		t.Fatalf("connecting client: %v", err)
	}

	cleanup := func() {
		session.Close()
		manager.Close()
	}

	return session, slug, cleanup
}

func TestMCPTaskCreate(t *testing.T) {
	session, slug, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "task_create",
		Arguments: map[string]any{
			"project": slug,
			"title":   "MCP test task",
		},
	})
	if err != nil {
		t.Fatalf("calling task_create: %v", err)
	}

	if res.IsError {
		text := res.Content[0].(*mcp.TextContent).Text
		t.Fatalf("task_create returned error: %s", text)
	}

	text := res.Content[0].(*mcp.TextContent).Text
	if text == "" {
		t.Error("expected non-empty response")
	}
	t.Logf("task_create response: %s", text)
}

func TestMCPTaskListAndComplete(t *testing.T) {
	session, slug, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create a task
	_, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "task_create",
		Arguments: map[string]any{
			"project": slug,
			"title":   "Task to complete",
		},
	})
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}

	// List tasks
	listRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "task_list",
		Arguments: map[string]any{
			"project": slug,
		},
	})
	if err != nil {
		t.Fatalf("listing tasks: %v", err)
	}

	listText := listRes.Content[0].(*mcp.TextContent).Text
	if listText == "No tasks found." {
		t.Error("expected tasks in list")
	}

	// Complete the task (with project)
	completeRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "task_complete",
		Arguments: map[string]any{
			"project": slug,
			"id":      float64(1),
		},
	})
	if err != nil {
		t.Fatalf("completing task: %v", err)
	}

	completeText := completeRes.Content[0].(*mcp.TextContent).Text
	if completeText == "" {
		t.Error("expected non-empty complete response")
	}
	t.Logf("task_complete response: %s", completeText)
}

func TestMCPProjectList(t *testing.T) {
	session, _, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "project_list",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("calling project_list: %v", err)
	}

	text := res.Content[0].(*mcp.TextContent).Text
	if text == "" {
		t.Error("expected non-empty response")
	}
	t.Logf("project_list response: %s", text)
}

func TestMCPProjectStats(t *testing.T) {
	session, slug, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create some tasks first
	session.CallTool(ctx, &mcp.CallToolParams{
		Name: "task_create",
		Arguments: map[string]any{
			"project": slug,
			"title":   "Task 1",
		},
	})
	session.CallTool(ctx, &mcp.CallToolParams{
		Name: "task_create",
		Arguments: map[string]any{
			"project": slug,
			"title":   "Bug 1",
			"type":    "bug",
		},
	})

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "project_stats",
		Arguments: map[string]any{
			"slug": slug,
		},
	})
	if err != nil {
		t.Fatalf("calling project_stats: %v", err)
	}

	text := res.Content[0].(*mcp.TextContent).Text
	t.Logf("project_stats response: %s", text)
}

func TestMCPFullLifecycle(t *testing.T) {
	session, slug, cleanup := setupTestServer(t)
	defer cleanup()
	ctx := context.Background()

	// 1. Create task
	createRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "task_create",
		Arguments: map[string]any{
			"project":  slug,
			"title":    "Lifecycle test",
			"type":     "feature",
			"priority": float64(3),
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if createRes.IsError {
		t.Fatalf("create error: %s", createRes.Content[0].(*mcp.TextContent).Text)
	}
	t.Logf("create: %s", createRes.Content[0].(*mcp.TextContent).Text)

	// 2. Get task (with project)
	getRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "task_get",
		Arguments: map[string]any{
			"project": slug,
			"id":      float64(1),
		},
	})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if getRes.IsError {
		t.Fatalf("get error: %s", getRes.Content[0].(*mcp.TextContent).Text)
	}
	t.Logf("get: %s", getRes.Content[0].(*mcp.TextContent).Text)

	// 3. Update task (with project)
	updateRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "task_update",
		Arguments: map[string]any{
			"project": slug,
			"id":      float64(1),
			"status":  "doing",
			"title":   "Lifecycle test updated",
		},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updateRes.IsError {
		t.Fatalf("update error: %s", updateRes.Content[0].(*mcp.TextContent).Text)
	}
	t.Logf("update: %s", updateRes.Content[0].(*mcp.TextContent).Text)

	// 4. Complete task (with project)
	completeRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "task_complete",
		Arguments: map[string]any{
			"project": slug,
			"id":      float64(1),
		},
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if completeRes.IsError {
		t.Fatalf("complete error: %s", completeRes.Content[0].(*mcp.TextContent).Text)
	}
	t.Logf("complete: %s", completeRes.Content[0].(*mcp.TextContent).Text)

	// 5. Delete task (with project)
	deleteRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "task_delete",
		Arguments: map[string]any{
			"project": slug,
			"id":      float64(1),
		},
	})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if deleteRes.IsError {
		t.Fatalf("delete error: %s", deleteRes.Content[0].(*mcp.TextContent).Text)
	}
	t.Logf("delete: %s", deleteRes.Content[0].(*mcp.TextContent).Text)

	// 6. Verify deleted — list should be empty
	listRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "task_list",
		Arguments: map[string]any{
			"project": slug,
		},
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	listText := listRes.Content[0].(*mcp.TextContent).Text
	if listText != "No tasks found." {
		t.Errorf("expected empty list after delete, got: %s", listText)
	}
}

func TestMCPPrefixDetection(t *testing.T) {
	session, slug, cleanup := setupTestServer(t)
	defer cleanup()
	ctx := context.Background()

	tests := []struct {
		title    string
		wantType string
	}{
		{"BUG: login broken", "bug"},
		{"FEATURE: dark mode", "feature"},
		{"HOTFIX: crash fix", "hotfix"},
		{"Normal task", "task"},
	}

	for i, tt := range tests {
		res, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "task_create",
			Arguments: map[string]any{
				"project": slug,
				"title":   tt.title,
			},
		})
		if err != nil {
			t.Fatalf("test %d: %v", i, err)
		}
		if res.IsError {
			t.Fatalf("test %d error: %s", i, res.Content[0].(*mcp.TextContent).Text)
		}
		text := res.Content[0].(*mcp.TextContent).Text
		if !contains(text, tt.wantType) {
			t.Errorf("test %d: expected type '%s' in response, got: %s", i, tt.wantType, text)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
