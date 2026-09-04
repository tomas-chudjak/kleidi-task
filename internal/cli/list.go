package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tomas-chudjak/kleidi-task/internal/core"
	"github.com/tomas-chudjak/kleidi-task/internal/db"
	"github.com/tomas-chudjak/kleidi-task/internal/render"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		status, _ := cmd.Flags().GetString("status")
		taskType, _ := cmd.Flags().GetString("type")
		limit, _ := cmd.Flags().GetInt64("limit")
		format, _ := cmd.Flags().GetString("format")
		switch format {
		case "table", "md", "json":
		default:
			return fmt.Errorf("unknown format %q (want table, md or json)", format)
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

		filter := core.ListTasksFilter{Limit: limit}
		if status != "" {
			filter.Status = status
		}
		if taskType != "" {
			filter.Type = taskType
		}

		result, err := taskService.ListWithCount(cmd.Context(), filter)
		if err != nil {
			return err
		}

		switch format {
		case "json":
			return render.JSON(os.Stdout, result.Tasks)
		case "md":
			meta, err := listMeta(cmd.Context(), projectService, taskService, projectPath, result)
			if err != nil {
				return err
			}
			fmt.Println(render.Markdown(result.Tasks, meta))
			return nil
		}

		if len(result.Tasks) == 0 {
			fmt.Println("No tasks found.")
			return nil
		}

		printTaskTable(result.Tasks)
		if result.TotalPages > 1 {
			fmt.Printf("\nPage %d/%d (total: %d)\n", result.Page, result.TotalPages, result.Total)
		}
		return nil
	},
}

// listMeta builds the canonical header context (project name + open/done
// counts + pagination) shared by the CLI and MCP renderers.
func listMeta(ctx context.Context, projectService *core.ProjectService, taskService *core.TaskService, projectPath string, result core.ListResult) (render.ListMeta, error) {
	name := filepath.Base(projectPath)
	if p, err := projectService.GetByPath(projectPath); err == nil {
		name = p.Name
	}

	stats, err := taskService.Stats(ctx)
	if err != nil {
		return render.ListMeta{}, err
	}

	return render.MetaFor(name, stats, result), nil
}

func init() {
	listCmd.Flags().String("status", "", "Filter by status (todo, doing, done)")
	listCmd.Flags().String("type", "", "Filter by type (task, bug)")
	listCmd.Flags().Int64("limit", 50, "Maximum number of tasks to show")
	listCmd.Flags().String("format", "table", "Output format: table (terminal), md (canonical markdown, same as MCP), json")
	rootCmd.AddCommand(listCmd)
}
