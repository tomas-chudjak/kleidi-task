package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/ahoylog/kvik-tasks/internal/db"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a task permanently",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid task ID: %s", args[0])
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

		// Show the task before deleting
		task, err := taskService.Get(cmd.Context(), id)
		if err != nil {
			return err
		}

		if err := taskService.Delete(cmd.Context(), id); err != nil {
			return err
		}

		fmt.Printf("Deleted task #%d: %s\n", task.ID, task.Title)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
