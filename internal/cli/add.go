package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/ahoylog/kvik-tasks/internal/config"
	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/ahoylog/kvik-tasks/internal/db"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [title]",
	Short: "Add a new task, bug, feature, or hotfix",
	Long: `Add a new work item. Type is detected from flags or title prefix:
  kvt add "Implement auth"                → task
  kvt add --bug "Login broken"            → bug
  kvt add --feature "Dark mode"           → feature
  kvt add --hotfix "Fix crash on start"   → hotfix
  kvt add "BUG: Login broken"             → bug (auto-detected, prefix stripped)
  kvt add "FEATURE: Dark mode"            → feature (auto-detected)
  kvt add "FEAT: Dark mode"               → feature (shorthand)`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := strings.Join(args, " ")
		description, _ := cmd.Flags().GetString("description")
		priority, _ := cmd.Flags().GetInt64("priority")

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

		// Load project config for defaults and custom types
		cfg := config.Load(projectPath)

		// Apply default priority if not set via flag
		if !flagChanged(cmd, "priority") && cfg.DefaultPriority != 0 {
			priority = int64(cfg.DefaultPriority)
		}

		// Build custom type prefixes from config
		var extraPrefixes []core.TypePrefix
		for _, ct := range cfg.CustomTypes {
			extraPrefixes = append(extraPrefixes, core.TypePrefix{
				Prefix:   ct.Prefix,
				TaskType: core.TaskType(ct.Name),
			})
		}

		// Determine type from flags or prefix detection
		taskType := core.TypeTask
		switch {
		case flagChanged(cmd, "bug"):
			taskType = core.TypeBug
		case flagChanged(cmd, "feature"):
			taskType = core.TypeFeature
		case flagChanged(cmd, "hotfix"):
			taskType = core.TypeHotfix
		default:
			taskType, title = core.DetectTypeFromTitle(title, core.TypeTask, extraPrefixes...)
		}

		taskService, err := projectService.TaskServiceFor(projectPath)
		if err != nil {
			return err
		}

		task, err := taskService.Create(cmd.Context(), core.CreateTaskInput{
			Title:       title,
			Description: description,
			Type:        taskType,
			Priority:    priority,
			Source:      core.SourceCLI,
		})
		if err != nil {
			return err
		}

		printTaskRow("Created", task)
		return nil
	},
}

func flagChanged(cmd *cobra.Command, name string) bool {
	f := cmd.Flags().Lookup(name)
	return f != nil && f.Changed
}

func init() {
	addCmd.Flags().Bool("bug", false, "Create a bug")
	addCmd.Flags().Bool("feature", false, "Create a feature")
	addCmd.Flags().Bool("hotfix", false, "Create a hotfix")
	addCmd.Flags().StringP("description", "d", "", "Description")
	addCmd.Flags().Int64P("priority", "p", 0, "Priority (higher = more important)")
	rootCmd.AddCommand(addCmd)
}
