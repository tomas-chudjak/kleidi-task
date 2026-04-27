package cli

import (
	"fmt"
	"os"

	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/ahoylog/kvik-tasks/internal/db"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new project in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")

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

		project, err := projectService.Init(cwd, name)
		if err != nil {
			return err
		}

		fmt.Printf("Initialized project '%s' (%s) in %s\n", project.Name, project.Slug, project.Path)
		return nil
	},
}

func init() {
	initCmd.Flags().String("name", "", "Project name (defaults to directory name)")
}
