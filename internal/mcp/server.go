package mcp

import (
	"context"
	"fmt"

	"github.com/tomas-chudjak/kleidi-task/internal/core"
	"github.com/tomas-chudjak/kleidi-task/internal/db"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps the MCP server with kleidi-task services.
type Server struct {
	mcpServer      *mcp.Server
	manager        *db.Manager
	projectService *core.ProjectService
}

// NewServer creates a new MCP server with all tools and resources registered.
func NewServer(version string) (*Server, error) {
	manager, err := db.NewManager()
	if err != nil {
		return nil, fmt.Errorf("initializing database: %w", err)
	}

	projectService := core.NewProjectService(manager)

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "kleidi-task",
		Title:   "Kleidi Task — local task tracker",
		Version: version,
	}, nil)

	s := &Server{
		mcpServer:      mcpServer,
		manager:        manager,
		projectService: projectService,
	}

	s.registerTools()
	s.registerResources()

	return s, nil
}

// RunStdio starts the MCP server on stdio transport.
func (s *Server) RunStdio(ctx context.Context) error {
	defer s.manager.Close()
	return s.mcpServer.Run(ctx, &mcp.StdioTransport{})
}
