package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kvt",
	Short: "kvik-tasks — local-first task tracker for developers",
	Long:  `kvik-tasks (kvt) is a local-first, single-binary task tracker with MCP-first AI integration.`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
}
