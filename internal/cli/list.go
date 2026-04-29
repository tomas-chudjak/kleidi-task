package cli

import (
	"fmt"
	"os"

	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/ahoylog/kvik-tasks/internal/db"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		status, _ := cmd.Flags().GetString("status")
		taskType, _ := cmd.Flags().GetString("type")
		limit, _ := cmd.Flags().GetInt64("limit")

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

		filter := core.ListTasksFilter{Limit: limit}
		if status != "" {
			filter.Status = status
		}
		if taskType != "" {
			filter.Type = taskType
		}

		tasks, err := taskService.List(cmd.Context(), filter)
		if err != nil {
			return err
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found.")
			return nil
		}

		printTaskTable(tasks)
		return nil
	},
}

func init() {
	listCmd.Flags().String("status", "", "Filter by status (todo, doing, done)")
	listCmd.Flags().String("type", "", "Filter by type (task, bug)")
	listCmd.Flags().Int64("limit", 50, "Maximum number of tasks to show")
	rootCmd.AddCommand(listCmd)
}
