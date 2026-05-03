package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/tomas-chudjak/kleidi-task/internal/core"
	"github.com/tomas-chudjak/kleidi-task/internal/db"
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

		printTaskDetail(task)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
