package core

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/tomas-chudjak/kleidi-task/internal/db/generated"
)

// EnforcementMode controls how strictly task descriptions must follow the
// template for their type.
type EnforcementMode string

const (
	// EnforcementOff disables template validation entirely.
	EnforcementOff EnforcementMode = "off"
	// EnforcementWarn logs missing sections but creates the task.
	EnforcementWarn EnforcementMode = "warn"
	// EnforcementStrict rejects programmatic creates that skip the template.
	EnforcementStrict EnforcementMode = "strict"
)

// ValidEnforcementModes contains all valid enforcement modes.
var ValidEnforcementModes = []EnforcementMode{EnforcementOff, EnforcementWarn, EnforcementStrict}

// ParseEnforcementMode converts a stored value to an EnforcementMode, falling
// back to EnforcementWarn for anything unrecognised.
func ParseEnforcementMode(v string) EnforcementMode {
	switch EnforcementMode(v) {
	case EnforcementOff:
		return EnforcementOff
	case EnforcementStrict:
		return EnforcementStrict
	default:
		return EnforcementWarn
	}
}

// ProjectConfig holds project-level configuration.
type ProjectConfig struct {
	DefaultPriority     int64           `json:"default_priority"`
	DefaultType         string          `json:"default_type"`
	AutoArchiveDays     int64           `json:"auto_archive_days"`
	TemplateEnforcement EnforcementMode `json:"template_enforcement"`
}

// ConfigService manages project-level configuration.
type ConfigService struct {
	queries *generated.Queries
}

// NewConfigService creates a new ConfigService.
func NewConfigService(db *sql.DB) *ConfigService {
	return &ConfigService{queries: generated.New(db)}
}

// EnforcementMode returns the configured template enforcement mode, falling
// back to EnforcementWarn when the config cannot be read.
func (s *ConfigService) EnforcementMode(ctx context.Context) EnforcementMode {
	cfg, err := s.Get(ctx)
	if err != nil {
		return EnforcementWarn
	}
	return cfg.TemplateEnforcement
}

// Get returns the full project configuration.
func (s *ConfigService) Get(ctx context.Context) (ProjectConfig, error) {
	rows, err := s.queries.ListConfig(ctx)
	if err != nil {
		return ProjectConfig{}, err
	}

	cfg := ProjectConfig{
		DefaultType:         "task",
		TemplateEnforcement: EnforcementWarn,
	}
	for _, r := range rows {
		switch r.Key {
		case "default_priority":
			cfg.DefaultPriority, _ = strconv.ParseInt(r.Value, 10, 64)
		case "default_type":
			cfg.DefaultType = r.Value
		case "auto_archive_days":
			cfg.AutoArchiveDays, _ = strconv.ParseInt(r.Value, 10, 64)
		case "template_enforcement":
			cfg.TemplateEnforcement = ParseEnforcementMode(r.Value)
		}
	}
	return cfg, nil
}

// Set updates a single configuration key.
func (s *ConfigService) Set(ctx context.Context, key, value string) error {
	return s.queries.SetConfig(ctx, generated.SetConfigParams{Key: key, Value: value})
}

// SetAll updates all configuration values at once.
func (s *ConfigService) SetAll(ctx context.Context, cfg ProjectConfig) error {
	if err := s.Set(ctx, "default_priority", strconv.FormatInt(cfg.DefaultPriority, 10)); err != nil {
		return err
	}
	if err := s.Set(ctx, "default_type", cfg.DefaultType); err != nil {
		return err
	}
	if err := s.Set(ctx, "auto_archive_days", strconv.FormatInt(cfg.AutoArchiveDays, 10)); err != nil {
		return err
	}
	return s.Set(ctx, "template_enforcement", string(ParseEnforcementMode(string(cfg.TemplateEnforcement))))
}
