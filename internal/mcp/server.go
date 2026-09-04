package mcp

import (
	"context"
	"fmt"

	"github.com/tomas-chudjak/kleidi-task/internal/core"
	"github.com/tomas-chudjak/kleidi-task/internal/db"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// serverInstructions is sent to the client on initialize. It exists to keep
// task output stable across sessions: task_list and task_search already return
// the canonical rendering, so the client's job is to pass it through unchanged.
const serverInstructions = `kleidi-task is a local task tracker.

Output contract — task_list and task_search return a pre-rendered markdown
table in their text content. Print that block verbatim. Do not rebuild it as
bullets, do not add or drop columns, do not re-sort rows, and do not summarize
it in prose instead of showing it. Rows are already ordered by priority DESC,
then created_at DESC. Add your own commentary after the table, never in place
of it.

Use the structured content (tasks array) when you need field values for
further tool calls; use the text block when showing the list to the user.`

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
	}, &mcp.ServerOptions{Instructions: serverInstructions})

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
