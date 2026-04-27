package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/ahoylog/kvik-tasks/internal/db"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show task details",
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

		task, err := taskService.Get(cmd.Context(), id)
		if err != nil {
			return err
		}

		fmt.Printf("ID:          %d\n", task.ID)
		fmt.Printf("Type:        %s\n", task.Type)
		fmt.Printf("Title:       %s\n", task.Title)
		fmt.Printf("Status:      %s\n", task.Status)
		fmt.Printf("Priority:    %d\n", task.Priority)
		fmt.Printf("Source:      %s\n", task.Source)
		fmt.Printf("Created:     %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Updated:     %s\n", task.UpdatedAt.Format("2006-01-02 15:04:05"))
		if task.Description != "" {
			fmt.Printf("Description: %s\n", task.Description)
		}
		if task.CompletedAt != nil {
			fmt.Printf("Completed:   %s\n", task.CompletedAt.Format("2006-01-02 15:04:05"))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
