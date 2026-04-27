package db

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/project/*.sql
var projectMigrations embed.FS

//go:embed migrations/registry/*.sql
var registryMigrations embed.FS

// Manager handles database connections for both the global registry
// and per-project databases.
type Manager struct {
	registryDB   *sql.DB
	registryPath string

	mu         sync.Mutex
	projectDBs map[string]*sql.DB // keyed by absolute path
}

// NewManager creates a new database manager and initializes the global registry
// at the default location (~/.tasks/registry.db).
func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}
	return NewManagerWithRegistryDir(filepath.Join(home, ".tasks"))
}

// NewManagerWithRegistryDir creates a new database manager with the registry
// in the specified directory. Useful for testing with isolated registries.
func NewManagerWithRegistryDir(registryDir string) (*Manager, error) {
	if err := os.MkdirAll(registryDir, 0755); err != nil {
		return nil, fmt.Errorf("creating registry directory: %w", err)
	}

	registryPath := filepath.Join(registryDir, "registry.db")
	registryDB, err := openDB(registryPath)
	if err != nil {
		return nil, fmt.Errorf("opening registry database: %w", err)
	}

	if err := runMigrations(registryDB, registryMigrations, "migrations/registry"); err != nil {
		registryDB.Close()
		return nil, fmt.Errorf("running registry migrations: %w", err)
	}

	return &Manager{
		registryDB:   registryDB,
		registryPath: registryPath,
		projectDBs:   make(map[string]*sql.DB),
	}, nil
}

// RegistryDB returns the global registry database connection.
func (m *Manager) RegistryDB() *sql.DB {
	return m.registryDB
}

// ProjectDB returns a database connection for the given project path.
// The path should be the absolute path to the project root (containing .tasks/).
// Connections are cached and reused.
func (m *Manager) ProjectDB(projectPath string) (*sql.DB, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	dbPath := filepath.Join(absPath, ".tasks", "tasks.db")

	m.mu.Lock()
	defer m.mu.Unlock()

	if db, ok := m.projectDBs[dbPath]; ok {
		return db, nil
	}

	db, err := openDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening project database at %s: %w", dbPath, err)
	}

	if err := runMigrations(db, projectMigrations, "migrations/project"); err != nil {
		db.Close()
		return nil, fmt.Errorf("running project migrations: %w", err)
	}

	m.projectDBs[dbPath] = db
	return db, nil
}

// InitProject creates the .tasks/ directory and initializes the project database.
func (m *Manager) InitProject(projectPath, slug, name string) error {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	tasksDir := filepath.Join(absPath, ".tasks")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return fmt.Errorf("creating .tasks directory: %w", err)
	}

	db, err := m.ProjectDB(absPath)
	if err != nil {
		return fmt.Errorf("initializing project database: %w", err)
	}

	// Set project metadata
	_, err = db.Exec(`INSERT OR REPLACE INTO meta (key, value) VALUES ('project_slug', ?), ('project_name', ?), ('schema_version', '1')`,
		slug, name)
	if err != nil {
		return fmt.Errorf("setting project metadata: %w", err)
	}

	return nil
}

// Close closes all database connections.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error

	for path, db := range m.projectDBs {
		if err := db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing %s: %w", path, err))
		}
	}

	if err := m.registryDB.Close(); err != nil {
		errs = append(errs, fmt.Errorf("closing registry: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("closing databases: %v", errs)
	}
	return nil
}

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for better concurrent read performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enabling WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	return db, nil
}

func runMigrations(db *sql.DB, fs embed.FS, dir string) error {
	goose.SetBaseFS(fs)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("setting dialect: %w", err)
	}
	if err := goose.Up(db, dir); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}
	return nil
}

