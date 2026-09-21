package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/tomas-chudjak/kleidi-task/internal/core"
	"github.com/tomas-chudjak/kleidi-task/internal/db"
)

var splitCmd = &cobra.Command{
	Use:   "split [id] [child title]...",
	Short: "Split a task into ordered child tasks",
	Long: `Breaks a task into the PR-sized chunks the work will actually land in.

Children inherit the parent's type, priority and category, and get the type's
template skeleton as their description. Nesting is one level: a child cannot be
split further, and the parent cannot be completed while any child is open.

  klt split 42 "Schema and migration" "Service layer" "MCP and CLI wiring"`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid task ID: %s", args[0])
		}

		children := make([]core.ChildSpec, 0, len(args)-1)
		for _, title := range args[1:] {
			children = append(children, core.ChildSpec{Title: title})
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

		created, err := taskService.Split(cmd.Context(), id, children)
		if err != nil {
			return err
		}

		fmt.Printf("Split #%d into %d child task(s):\n", id, len(created))
		for _, c := range created {
			fmt.Printf("  %d. #%d %s\n", c.ChildOrder, c.ID, c.Title)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(splitCmd)
}
