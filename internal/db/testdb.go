package db

import (
	"database/sql"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

// NewTestProjectDB creates an in-memory SQLite database with project migrations
// applied. Intended for use in tests only.
func NewTestProjectDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}

	goose.SetBaseFS(projectMigrations)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("setting dialect: %v", err)
	}
	if err := goose.Up(db, "migrations/project"); err != nil {
		t.Fatalf("running migrations: %v", err)
	}

	t.Cleanup(func() { db.Close() })
	return db
}
