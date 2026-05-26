package core

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tomas-chudjak/kleidi-task/internal/claude"
	"github.com/tomas-chudjak/kleidi-task/internal/db"
)

// ProjectService handles project registration and detection.
type ProjectService struct {
	manager *db.Manager
}

// NewProjectService creates a new ProjectService.
func NewProjectService(manager *db.Manager) *ProjectService {
	return &ProjectService{manager: manager}
}

// Init initializes a new project in the given directory.
func (s *ProjectService) Init(dir, name string) (Project, error) {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return Project{}, fmt.Errorf("resolving path: %w", err)
	}

	// Check if already initialized
	tasksDir := filepath.Join(absPath, ".tasks")
	if _, err := os.Stat(tasksDir); err == nil {
		return Project{}, fmt.Errorf("%w: .tasks already exists in %s", ErrAlreadyExists, absPath)
	}

	slug := slugFromPath(absPath)
	if name == "" {
		name = filepath.Base(absPath)
	}

	// Initialize the project database (creates .tasks/ dir + runs migrations)
	if err := s.manager.InitProject(absPath, slug, name); err != nil {
		return Project{}, fmt.Errorf("initializing project: %w", err)
	}

	// Register in global registry
	registryDB := s.manager.RegistryDB()
	_, err = registryDB.Exec(
		`INSERT INTO projects (slug, name, path) VALUES (?, ?, ?)`,
		slug, name, absPath,
	)
	if err != nil {
		return Project{}, fmt.Errorf("registering project: %w", err)
	}

	// Install Claude Code skills for this project
	if err := claude.InstallSkills(absPath); err != nil {
		return Project{}, fmt.Errorf("installing claude skills: %w", err)
	}

	return Project{
		Slug: slug,
		Name: name,
		Path: absPath,
	}, nil
}

// DetectProject walks up from the given directory looking for a .tasks/ directory.
// Returns the project path or ErrNoProject if not found.
func (s *ProjectService) DetectProject(startDir string) (string, error) {
	absPath, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	const maxDepth = 20
	dir := absPath
	for i := 0; i < maxDepth; i++ {
		tasksDir := filepath.Join(dir, ".tasks")
		if info, err := os.Stat(tasksDir); err == nil && info.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}

	return "", ErrNoProject
}

// List returns all registered projects.
func (s *ProjectService) List() ([]Project, error) {
	registryDB := s.manager.RegistryDB()
	rows, err := registryDB.Query(
		`SELECT id, slug, name, path, last_seen_at, created_at, cached_todo_count, cached_doing_count, cached_total_count FROM projects ORDER BY last_seen_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Slug, &p.Name, &p.Path, &p.LastSeenAt, &p.CreatedAt, &p.CachedTodoCount, &p.CachedDoingCount, &p.CachedTotalCount); err != nil {
			return nil, fmt.Errorf("scanning project: %w", err)
		}
		projects = append(projects, p)
	}

	return projects, rows.Err()
}

// GetBySlug returns a project by its slug.
func (s *ProjectService) GetBySlug(slug string) (Project, error) {
	registryDB := s.manager.RegistryDB()
	var p Project
	err := registryDB.QueryRow(
		`SELECT id, slug, name, path, last_seen_at, created_at, cached_todo_count, cached_doing_count, cached_total_count FROM projects WHERE slug = ?`,
		slug,
	).Scan(&p.ID, &p.Slug, &p.Name, &p.Path, &p.LastSeenAt, &p.CreatedAt, &p.CachedTodoCount, &p.CachedDoingCount, &p.CachedTotalCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return Project{}, fmt.Errorf("project '%s': %w", slug, ErrProjectNotFound)
		}
		return Project{}, fmt.Errorf("getting project: %w", err)
	}
	return p, nil
}

// TaskServiceFor returns a TaskService for the given project path.
func (s *ProjectService) TaskServiceFor(projectPath string) (*TaskService, error) {
	db, err := s.manager.ProjectDB(projectPath)
	if err != nil {
		return nil, fmt.Errorf("getting project database: %w", err)
	}
	ts := NewTaskService(db)
	ts.SetHooks(NewHookService(projectPath))
	ts.SetTemplates(NewTemplateService(db))
	return ts, nil
}

// UserService returns a UserService using the registry database.
func (s *ProjectService) UserService() *UserService {
	return NewUserService(s.manager.RegistryDB())
}

// HookServiceFor returns a HookService for the given project path.
func (s *ProjectService) HookServiceFor(projectPath string) *HookService {
	return NewHookService(projectPath)
}

// CategoryServiceFor returns a CategoryService for the given project path.
func (s *ProjectService) CategoryServiceFor(projectPath string) (*CategoryService, error) {
	db, err := s.manager.ProjectDB(projectPath)
	if err != nil {
		return nil, fmt.Errorf("getting project database: %w", err)
	}
	return NewCategoryService(db), nil
}

// WorkflowServiceFor returns a WorkflowService for the given project path.
func (s *ProjectService) WorkflowServiceFor(projectPath string) (*WorkflowService, error) {
	db, err := s.manager.ProjectDB(projectPath)
	if err != nil {
		return nil, fmt.Errorf("getting project database: %w", err)
	}
	return NewWorkflowService(db), nil
}

// TemplateServiceFor returns a TemplateService for the given project path.
func (s *ProjectService) TemplateServiceFor(projectPath string) (*TemplateService, error) {
	db, err := s.manager.ProjectDB(projectPath)
	if err != nil {
		return nil, fmt.Errorf("getting project database: %w", err)
	}
	return NewTemplateService(db), nil
}

// ConfigServiceFor returns a ConfigService for the given project path.
func (s *ProjectService) ConfigServiceFor(projectPath string) (*ConfigService, error) {
	db, err := s.manager.ProjectDB(projectPath)
	if err != nil {
		return nil, fmt.Errorf("getting project database: %w", err)
	}
	return NewConfigService(db), nil
}

// Backup creates a consistent backup of the project's database.
// If destPath is empty, it defaults to .tasks/backups/tasks_<timestamp>.db
// Returns the path where the backup was written.
func (s *ProjectService) Backup(projectPath, destPath string) (string, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	if destPath == "" {
		backupDir := filepath.Join(absPath, ".tasks", "backups")
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			return "", fmt.Errorf("creating backup directory: %w", err)
		}
		now := time.Now()
		destPath = filepath.Join(backupDir, fmt.Sprintf("tasks_%s.db", now.Format("20060102_150405")))
	}

	if err := s.manager.BackupProject(absPath, destPath); err != nil {
		return "", err
	}

	return destPath, nil
}

// InstallSkills installs or updates Claude Code skills in the project directory.
// Safe to call on already-initialized projects.
func (s *ProjectService) InstallSkills(dir string) error {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}
	return claude.InstallSkills(absPath)
}

// slugFromPath generates a slug from a directory path.
func slugFromPath(absPath string) string {
	base := filepath.Base(absPath)
	slug := strings.ToLower(base)
	slug = strings.ReplaceAll(slug, " ", "-")
	return slug
}
