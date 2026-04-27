package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds all configuration options.
// Project config overrides global config.
type Config struct {
	// Server
	Port int `json:"port,omitempty"` // default: 7842

	// Defaults
	DefaultPriority int    `json:"default_priority,omitempty"` // default priority for new tasks
	DefaultProject  string `json:"default_project,omitempty"`  // slug for MCP when not in a project dir

	// Task types (extend built-in: task, bug, feature, hotfix)
	// Note: custom types require a DB migration to update the CHECK constraint.
	// For now, use the 4 built-in types. Custom types will be fully supported
	// when we add dynamic type registration (v0.4+).
	CustomTypes []CustomType `json:"custom_types,omitempty"`
}

// CustomType defines a user-defined task type.
type CustomType struct {
	Name   string `json:"name"`   // type name (e.g., "spike")
	Prefix string `json:"prefix"` // title prefix for auto-detection (e.g., "SPIKE")
}

// Defaults returns a Config with default values.
func Defaults() Config {
	return Config{
		Port: 7842,
	}
}

// Load reads the global config, then merges project config on top.
// Missing files are silently ignored.
func Load(projectPath string) Config {
	cfg := Defaults()

	// Global config: ~/.tasks/config.json
	home, err := os.UserHomeDir()
	if err == nil {
		globalPath := filepath.Join(home, ".tasks", "config.json")
		mergeFromFile(&cfg, globalPath)
	}

	// Project config: <project>/.tasks/config.json
	if projectPath != "" {
		projectCfgPath := filepath.Join(projectPath, ".tasks", "config.json")
		mergeFromFile(&cfg, projectCfgPath)
	}

	return cfg
}

// LoadGlobal reads only the global config.
func LoadGlobal() Config {
	cfg := Defaults()
	home, err := os.UserHomeDir()
	if err == nil {
		globalPath := filepath.Join(home, ".tasks", "config.json")
		mergeFromFile(&cfg, globalPath)
	}
	return cfg
}

func mergeFromFile(cfg *Config, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return // file doesn't exist, skip
	}

	var fileCfg Config
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return // invalid JSON, skip
	}

	// Merge non-zero values
	if fileCfg.Port != 0 {
		cfg.Port = fileCfg.Port
	}
	if fileCfg.DefaultPriority != 0 {
		cfg.DefaultPriority = fileCfg.DefaultPriority
	}
	if fileCfg.DefaultProject != "" {
		cfg.DefaultProject = fileCfg.DefaultProject
	}
	if len(fileCfg.CustomTypes) > 0 {
		cfg.CustomTypes = append(cfg.CustomTypes, fileCfg.CustomTypes...)
	}
}
