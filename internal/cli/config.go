package cli

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
	"github.com/tomas-chudjak/kleidi-task/internal/core"
	"github.com/tomas-chudjak/kleidi-task/internal/db"
)

// configKeys lists the settable configuration keys with their descriptions.
var configKeys = map[string]string{
	"default_priority":     "priority given to new tasks",
	"default_type":         "type given to new tasks",
	"auto_archive_days":    "archive completed tasks after N days (0 = disabled)",
	"template_enforcement": "template validation on create: off, warn or strict",
}

// configServiceForCwd resolves the ConfigService for the project containing cwd.
func configServiceForCwd() (*core.ConfigService, func(), error) {
	manager, err := db.NewManager()
	if err != nil {
		return nil, nil, fmt.Errorf("initializing database: %w", err)
	}

	projectService := core.NewProjectService(manager)

	cwd, err := os.Getwd()
	if err != nil {
		manager.Close()
		return nil, nil, fmt.Errorf("getting current directory: %w", err)
	}

	projectPath, err := projectService.DetectProject(cwd)
	if err != nil {
		manager.Close()
		return nil, nil, err
	}

	configService, err := projectService.ConfigServiceFor(projectPath)
	if err != nil {
		manager.Close()
		return nil, nil, err
	}
	return configService, func() { manager.Close() }, nil
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and change project configuration",
}

var configGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Print the current project configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		configService, closeFn, err := configServiceForCwd()
		if err != nil {
			return err
		}
		defer closeFn()

		cfg, err := configService.Get(cmd.Context())
		if err != nil {
			return err
		}

		fmt.Printf("default_priority      %d\n", cfg.DefaultPriority)
		fmt.Printf("default_type          %s\n", cfg.DefaultType)
		fmt.Printf("auto_archive_days     %d\n", cfg.AutoArchiveDays)
		fmt.Printf("template_enforcement  %s\n", cfg.TemplateEnforcement)
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a project configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]

		if _, ok := configKeys[key]; !ok {
			keys := make([]string, 0, len(configKeys))
			for k := range configKeys {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			return fmt.Errorf("unknown config key '%s' (valid keys: %v)", key, keys)
		}

		if key == "template_enforcement" && core.EnforcementMode(value) != core.ParseEnforcementMode(value) {
			return fmt.Errorf("invalid template_enforcement '%s' (valid: off, warn, strict)", value)
		}

		configService, closeFn, err := configServiceForCwd()
		if err != nil {
			return err
		}
		defer closeFn()

		if err := configService.Set(cmd.Context(), key, value); err != nil {
			return err
		}

		fmt.Printf("%s = %s\n", key, value)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}
