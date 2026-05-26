package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tomas-chudjak/kleidi-task/internal/core"
	"github.com/tomas-chudjak/kleidi-task/internal/db"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Install or update Claude Code skills for the current project",
	Long:  `Installs embedded Claude Code skill files into .claude/skills/ in the current project. Safe to run multiple times — existing skills are updated to the latest version.`,
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

		// Detect project root (walks up looking for .tasks/)
		projectPath, err := projectService.DetectProject(cwd)
		if err != nil {
			return fmt.Errorf("no kleidi-task project found — run 'klt init' first")
		}

		if err := projectService.InstallSkills(projectPath); err != nil {
			return fmt.Errorf("installing skills: %w", err)
		}

		fmt.Printf("Claude Code skills installed in %s/.claude/skills/\n", projectPath)
		return nil
	},
}
