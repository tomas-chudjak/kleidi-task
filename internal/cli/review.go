package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tomas-chudjak/kleidi-task/internal/core"
	"github.com/tomas-chudjak/kleidi-task/internal/db"
)

var reviewCmd = &cobra.Command{
	Use:   "review [id]",
	Short: "Review a task description for gaps, conflicts and unmade decisions",
	Long: `Prints the task description alongside the sections its type requires and the
review instruction for that type.

Nothing is written — the review is the checkpoint before implementation, and the
author decides what to accept. The same output is available to AI clients through
the task_review MCP tool.`,
	Args: cobra.ExactArgs(1),
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

		review, err := taskService.Review(cmd.Context(), id)
		if err != nil {
			return err
		}

		printReview(review)
		return nil
	},
}

func printReview(r core.TaskReview) {
	fmt.Printf("Review of %s #%d: %s\n", r.Type, r.TaskID, r.Title)
	if r.Phase != "" {
		fmt.Printf("Phase:            %s\n", r.Phase)
	}
	fmt.Printf("Required sections: %s\n", strings.Join(r.RequiredSections, ", "))

	if len(r.MissingSections) > 0 {
		fmt.Printf("Missing:           %s\n", strings.Join(r.MissingSections, ", "))
	}
	if len(r.EmptySections) > 0 {
		fmt.Printf("Empty:             %s\n", strings.Join(r.EmptySections, ", "))
	}
	if len(r.MissingSections) == 0 && len(r.EmptySections) == 0 {
		fmt.Println("Structure:         all required sections present and non-empty")
	}
	if r.OpenQuestions != "" {
		fmt.Printf("\nAlready open:\n%s\n", r.OpenQuestions)
	}

	fmt.Printf("\n--- Current description ---\n\n%s\n", r.Description)
	if r.Instruction != "" {
		fmt.Printf("\n--- Review instruction ---\n\n%s\n", r.Instruction)
	}
}

func init() {
	rootCmd.AddCommand(reviewCmd)
}
