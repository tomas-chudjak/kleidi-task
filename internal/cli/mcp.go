package cli

import (
	"fmt"

	mcpserver "github.com/tomas-chudjak/kleidi-task/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server (stdio transport for Claude Desktop, Cursor, etc.)",
	RunE: func(cmd *cobra.Command, args []string) error {
		server, err := mcpserver.NewServer(version)
		if err != nil {
			return fmt.Errorf("creating MCP server: %w", err)
		}

		return server.RunStdio(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
