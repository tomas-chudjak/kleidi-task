package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/ahoylog/kvik-tasks/internal/db"
	"github.com/spf13/cobra"
)

var advanceCmd = &cobra.Command{
	Use:   "advance [id]",
	Short: "Advance a task to the next workflow phase",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid task ID: %w", err)
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

		wfService, err := projectService.WorkflowServiceFor(projectPath)
		if err != nil {
			return err
		}

		result, err := wfService.Advance(cmd.Context(), id)
		if err != nil {
			return err
		}

		fmt.Printf("Task #%d: %s → %s\n", id, result.PreviousPhase, result.CurrentPhase)
		if result.IsComplete {
			fmt.Println("Task completed!")
		}
		if len(result.SuggestedSkills) > 0 {
			fmt.Printf("Suggested skills: %s\n", strings.Join(result.SuggestedSkills, ", "))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(advanceCmd)
}
