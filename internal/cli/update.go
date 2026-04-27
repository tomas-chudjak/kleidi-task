package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/ahoylog/kvik-tasks/internal/db"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update a task",
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

		input := core.UpdateTaskInput{}
		hasChanges := false

		if cmd.Flags().Changed("title") {
			v, _ := cmd.Flags().GetString("title")
			input.Title = &v
			hasChanges = true
		}
		if cmd.Flags().Changed("description") {
			v, _ := cmd.Flags().GetString("description")
			input.Description = &v
			hasChanges = true
		}
		if cmd.Flags().Changed("status") {
			v, _ := cmd.Flags().GetString("status")
			s := core.TaskStatus(v)
			input.Status = &s
			hasChanges = true
		}
		if cmd.Flags().Changed("type") {
			v, _ := cmd.Flags().GetString("type")
			tt := core.TaskType(v)
			input.Type = &tt
			hasChanges = true
		}
		if cmd.Flags().Changed("priority") {
			v, _ := cmd.Flags().GetInt64("priority")
			input.Priority = &v
			hasChanges = true
		}

		if !hasChanges {
			return fmt.Errorf("no changes specified (use --title, --status, --type, --priority, --description)")
		}

		task, err := taskService.Update(cmd.Context(), id, input)
		if err != nil {
			return err
		}

		fmt.Printf("Task #%d updated: %s [%s]\n", task.ID, task.Title, task.Status)
		return nil
	},
}

func init() {
	updateCmd.Flags().String("title", "", "New title")
	updateCmd.Flags().StringP("description", "d", "", "New description")
	updateCmd.Flags().String("status", "", "New status (todo, doing, done)")
	updateCmd.Flags().String("type", "", "New type (task, bug)")
	updateCmd.Flags().Int64P("priority", "p", 0, "New priority")
	rootCmd.AddCommand(updateCmd)
}
