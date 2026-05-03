package cli

import (
	"fmt"
	"os"

	"github.com/tomas-chudjak/kleidi-task/internal/core"
	"github.com/tomas-chudjak/kleidi-task/internal/db"
	"github.com/spf13/cobra"
)

var backupOutput string

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create a backup of the current project's database",
	Long:  `Creates a consistent snapshot of the project's tasks.db using SQLite VACUUM INTO. Backups are stored in .tasks/backups/ by default.`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		backupPath, err := projectService.Backup(projectPath, backupOutput)
		if err != nil {
			return err
		}

		fmt.Printf("Backup created: %s\n", backupPath)
		return nil
	},
}

func init() {
	backupCmd.Flags().StringVarP(&backupOutput, "output", "o", "", "output path (default: .tasks/backups/tasks_<timestamp>.db)")
	rootCmd.AddCommand(backupCmd)
}
