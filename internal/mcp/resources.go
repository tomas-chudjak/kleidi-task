package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/tomas-chudjak/kleidi-task/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerResources() {
	s.mcpServer.AddResource(&mcp.Resource{
		URI:         "tasks://projects",
		Name:        "All projects",
		Description: "List of all registered kleidi-task projects",
		MIMEType:    "application/json",
	}, s.resourceProjects)

	s.mcpServer.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "tasks://project/{slug}",
		Name:        "Project overview",
		Description: "Overview of a project including stats and recent tasks",
		MIMEType:    "application/json",
	}, s.resourceProject)

	s.mcpServer.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "tasks://project/{slug}/tasks",
		Name:        "Project tasks",
		Description: "All tasks for a specific project",
		MIMEType:    "application/json",
	}, s.resourceProjectTasks)

	s.mcpServer.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "tasks://task/{id}",
		Name:        "Task detail",
		Description: "Detailed information about a specific task",
		MIMEType:    "application/json",
	}, s.resourceTask)
}

func (s *Server) resourceProjects(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	projects, err := s.projectService.List()
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}

	data, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling projects: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:  req.Params.URI,
			Text: string(data),
		}},
	}, nil
}

func (s *Server) resourceProject(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	slug := extractSlugFromURI(req.Params.URI, "tasks://project/")
	if slug == "" {
		return nil, fmt.Errorf("invalid project URI: %s", req.Params.URI)
	}

	// Remove trailing /tasks if present
	if len(slug) > 6 && slug[len(slug)-6:] == "/tasks" {
		slug = slug[:len(slug)-6]
	}

	project, err := s.projectService.GetBySlug(slug)
	if err != nil {
		return nil, err
	}

	taskService, err := s.projectService.TaskServiceFor(project.Path)
	if err != nil {
		return nil, err
	}

	stats, err := taskService.Stats(ctx)
	if err != nil {
		return nil, err
	}

	overview := map[string]any{
		"project": project,
		"stats":   stats,
	}

	data, err := json.MarshalIndent(overview, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling project overview: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:  req.Params.URI,
			Text: string(data),
		}},
	}, nil
}

func (s *Server) resourceProjectTasks(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	// Extract slug from tasks://project/{slug}/tasks
	slug := extractSlugFromURI(req.Params.URI, "tasks://project/")
	if slug == "" {
		return nil, fmt.Errorf("invalid project tasks URI: %s", req.Params.URI)
	}
	if len(slug) > 6 && slug[len(slug)-6:] == "/tasks" {
		slug = slug[:len(slug)-6]
	}

	project, err := s.projectService.GetBySlug(slug)
	if err != nil {
		return nil, err
	}

	taskService, err := s.projectService.TaskServiceFor(project.Path)
	if err != nil {
		return nil, err
	}

	tasks, err := taskService.List(ctx, core.ListTasksFilter{Limit: 100})
	if err != nil {
		return nil, err
	}

	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling tasks: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:  req.Params.URI,
			Text: string(data),
		}},
	}, nil
}

func (s *Server) resourceTask(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	// For task resource, we need to find the task across projects
	// For now, use current project
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}

	projectPath, err := s.projectService.DetectProject(cwd)
	if err != nil {
		return nil, err
	}

	taskService, err := s.projectService.TaskServiceFor(projectPath)
	if err != nil {
		return nil, err
	}

	// Extract ID from tasks://task/{id}
	idStr := extractSlugFromURI(req.Params.URI, "tasks://task/")
	var id int64
	fmt.Sscanf(idStr, "%d", &id)
	if id == 0 {
		return nil, fmt.Errorf("invalid task ID in URI: %s", req.Params.URI)
	}

	task, err := taskService.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling task: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:  req.Params.URI,
			Text: string(data),
		}},
	}, nil
}

func extractSlugFromURI(uri, prefix string) string {
	if len(uri) <= len(prefix) {
		return ""
	}
	return uri[len(prefix):]
}
