package cli

import (
	"fmt"
	"os"

	"github.com/tomas-chudjak/kleidi-task/internal/core"
	"github.com/tomas-chudjak/kleidi-task/internal/db"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export tasks to JSON or Markdown file",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		output, _ := cmd.Flags().GetString("output")
		status, _ := cmd.Flags().GetString("status")
		taskType, _ := cmd.Flags().GetString("type")
		archived, _ := cmd.Flags().GetBool("archived")

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

		// Get project info for naming
		projects, _ := projectService.List()
		var project core.Project
		for _, p := range projects {
			if p.Path == projectPath {
				project = p
				break
			}
		}

		var tasks []core.Task
		if archived {
			result, err := taskService.ListArchived(cmd.Context(), core.ListTasksFilter{Limit: 10000})
			if err != nil {
				return err
			}
			tasks = result.Tasks
		} else {
			tasks, err = taskService.List(cmd.Context(), core.ListTasksFilter{
				Status: status,
				Type:   taskType,
				Limit:  10000,
			})
			if err != nil {
				return err
			}
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks to export.")
			return nil
		}

		var data []byte
		switch format {
		case "md", "markdown":
			data = core.ExportMarkdown(project.Name, tasks)
		default:
			data, err = core.ExportJSON(project.Name, tasks)
			if err != nil {
				return err
			}
		}

		if output == "" {
			output = project.Slug + "-tasks." + format
			if format == "markdown" {
				output = project.Slug + "-tasks.md"
			}
		}

		if err := os.WriteFile(output, data, 0644); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}

		fmt.Printf("Exported %d tasks to %s\n", len(tasks), output)
		return nil
	},
}

func init() {
	exportCmd.Flags().StringP("format", "f", "json", "Export format (json, md)")
	exportCmd.Flags().StringP("output", "o", "", "Output file path (default: <project>-tasks.<format>)")
	exportCmd.Flags().String("status", "", "Filter by status (todo, doing, done)")
	exportCmd.Flags().String("type", "", "Filter by type (task, bug, feature, hotfix)")
	exportCmd.Flags().Bool("archived", false, "Export archived tasks instead")
	rootCmd.AddCommand(exportCmd)
}
