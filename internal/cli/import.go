package cli

import (
	"fmt"
	"os"

	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/ahoylog/kvik-tasks/internal/db"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import tasks from a JSON export file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputFile := args[0]

		data, err := os.ReadFile(inputFile)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}

		export, err := core.ImportJSON(data)
		if err != nil {
			return err
		}

		manager, err := db.NewManager()
		if err != nil {
			return fmt.Errorf("initializing database: %w", err)
		}
		defer manager.Close()

		projectService := core.NewProjectService(manager)

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}

		projectPath, err := projectService.DetectProject(cwd)
		if err != nil {
			return err
		}

		taskService, err := projectService.TaskServiceFor(projectPath)
		if err != nil {
			return err
		}

		created, skipped, err := core.ImportTasks(cmd.Context(), taskService, export.Tasks, core.SourceCLI)
		if err != nil {
			return err
		}

		fmt.Printf("Imported %d tasks (%d skipped as duplicates)\n", created, skipped)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
}
