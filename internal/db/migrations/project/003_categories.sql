-- +goose Up

-- Categories defined per-project
CREATE TABLE categories (
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    name  TEXT NOT NULL UNIQUE,
    color TEXT NOT NULL DEFAULT '#8a8dab'
);

-- Add category column to tasks
ALTER TABLE tasks ADD COLUMN category TEXT;
CREATE INDEX idx_tasks_category ON tasks(category) WHERE category IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_tasks_category;

-- SQLite doesn't support DROP COLUMN before 3.35.0, but modernc.org/sqlite supports it
ALTER TABLE tasks DROP COLUMN category;
DROP TABLE IF EXISTS categories;
