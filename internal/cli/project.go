package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/tomas-chudjak/kleidi-task/internal/core"
	"github.com/tomas-chudjak/kleidi-task/internal/db"
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := db.NewManager()
		if err != nil {
			return fmt.Errorf("initializing database: %w", err)
		}
		defer manager.Close()

		projectService := core.NewProjectService(manager)

		projects, err := projectService.List()
		if err != nil {
			return err
		}

		if len(projects) == 0 {
			fmt.Println("No projects registered. Run 'klt init' in a project directory.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "SLUG\tNAME\tPATH\tTODO\tDOING")
		for _, p := range projects {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\n", p.Slug, p.Name, p.Path, p.CachedTodoCount, p.CachedDoingCount)
		}
		w.Flush()

		return nil
	},
}

var projectStatsCmd = &cobra.Command{
	Use:   "stats [slug]",
	Short: "Show project statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := db.NewManager()
		if err != nil {
			return fmt.Errorf("initializing database: %w", err)
		}
		defer manager.Close()

		projectService := core.NewProjectService(manager)

		var projectPath string
		if len(args) > 0 {
			project, err := projectService.GetBySlug(args[0])
			if err != nil {
				return err
			}
			projectPath = project.Path
		} else {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}
			projectPath, err = projectService.DetectProject(cwd)
			if err != nil {
				return err
			}
		}

		taskService, err := projectService.TaskServiceFor(projectPath)
		if err != nil {
			return err
		}

		stats, err := taskService.Stats(cmd.Context())
		if err != nil {
			return err
		}

		fmt.Printf("Todo:      %d\n", stats.Todo)
		fmt.Printf("Doing:     %d\n", stats.Doing)
		fmt.Printf("Done:      %d\n", stats.Done)
		fmt.Printf("Bugs open: %d\n", stats.BugsOpen)

		return nil
	},
}

func init() {
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectStatsCmd)
	rootCmd.AddCommand(projectCmd)
}
