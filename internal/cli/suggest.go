package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/ahoylog/kvik-tasks/internal/db"
	"github.com/spf13/cobra"
)

var suggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Suggest tasks from TODO/FIXME/HACK/XXX comments in code",
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

		taskService, err := projectService.TaskServiceFor(projectPath)
		if err != nil {
			return err
		}

		suggestService := core.NewSuggestService(projectPath, taskService)
		suggestions, err := suggestService.Scan(context.Background())
		if err != nil {
			return fmt.Errorf("scanning: %w", err)
		}

		if len(suggestions) == 0 {
			fmt.Println("No suggestions found.")
			return nil
		}

		// Count new vs duplicate
		var newCount, dupCount int
		for _, s := range suggestions {
			if s.ExistingTaskID != nil {
				dupCount++
			} else {
				newCount++
			}
		}

		fmt.Printf("Found %d suggestion(s) (%d new, %d matching existing tasks)\n\n", len(suggestions), newCount, dupCount)
		for i, s := range suggestions {
			fmt.Println(core.FormatSuggestion(s, i))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(suggestCmd)
}
