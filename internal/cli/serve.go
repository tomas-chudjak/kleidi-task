package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start HTTP server (UI + REST API)",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		fmt.Printf("Starting kvik-tasks server on http://localhost:%d\n", port)
		fmt.Println("REST API and UI server not yet implemented (v0.2)")
		return nil
	},
}

func init() {
	serveCmd.Flags().Int("port", 7842, "Port to listen on")
	rootCmd.AddCommand(serveCmd)
}
