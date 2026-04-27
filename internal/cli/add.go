package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/ahoylog/kvik-tasks/internal/db"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [title]",
	Short: "Add a new task or bug",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := strings.Join(args, " ")
		description, _ := cmd.Flags().GetString("description")
		isBug, _ := cmd.Flags().GetBool("bug")
		priority, _ := cmd.Flags().GetInt64("priority")

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

		taskType := core.TypeTask
		if isBug {
			taskType = core.TypeBug
		}

		task, err := taskService.Create(cmd.Context(), core.CreateTaskInput{
			Title:       title,
			Description: description,
			Type:        taskType,
			Priority:    priority,
			Source:      core.SourceCLI,
		})
		if err != nil {
			return err
		}

		typeLabel := "Task"
		if task.Type == core.TypeBug {
			typeLabel = "Bug"
		}
		fmt.Printf("%s #%d created: %s\n", typeLabel, task.ID, task.Title)
		return nil
	},
}

func init() {
	addCmd.Flags().Bool("bug", false, "Create a bug instead of a task")
	addCmd.Flags().StringP("description", "d", "", "Task description")
	addCmd.Flags().Int64P("priority", "p", 0, "Priority (higher = more important)")
	rootCmd.AddCommand(addCmd)
}
