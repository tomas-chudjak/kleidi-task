-- +goose Up

-- Step 1: Extend workflows table with color, prefix, and is_builtin flag
ALTER TABLE workflows ADD COLUMN color TEXT NOT NULL DEFAULT '';
ALTER TABLE workflows ADD COLUMN prefix TEXT NOT NULL DEFAULT '';
ALTER TABLE workflows ADD COLUMN is_builtin INTEGER NOT NULL DEFAULT 0;

-- Seed built-in type metadata
UPDATE workflows SET color = '#e8eef5', prefix = '', is_builtin = 1 WHERE task_type = 'task';
UPDATE workflows SET color = '#fde8ee', prefix = 'BUG', is_builtin = 1 WHERE task_type = 'bug';
UPDATE workflows SET color = '#f0e8ff', prefix = 'FEAT', is_builtin = 1 WHERE task_type = 'feature';
UPDATE workflows SET color = '#fff1e6', prefix = 'HOTFIX', is_builtin = 1 WHERE task_type = 'hotfix';

-- Step 2: Recreate tasks table without CHECK constraint on type
-- SQLite cannot ALTER CHECK constraints, so we must recreate the table.

-- Drop all triggers that reference tasks
DROP TRIGGER IF EXISTS tasks_after_update;
DROP TRIGGER IF EXISTS tasks_fts_insert;
DROP TRIGGER IF EXISTS tasks_fts_update;
DROP TRIGGER IF EXISTS tasks_fts_delete;

-- Drop all indexes on tasks
DROP INDEX IF EXISTS idx_tasks_status;
DROP INDEX IF EXISTS idx_tasks_type;
DROP INDEX IF EXISTS idx_tasks_assigned;
DROP INDEX IF EXISTS idx_tasks_priority;
DROP INDEX IF EXISTS idx_tasks_category;
DROP INDEX IF EXISTS idx_tasks_archived;

-- Create new table without CHECK on type
CREATE TABLE tasks_new (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    type         TEXT NOT NULL DEFAULT 'task',
    title        TEXT NOT NULL,
    description  TEXT,
    status       TEXT NOT NULL DEFAULT 'todo'
                 CHECK(status IN ('todo', 'doing', 'done')),
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    created_by   INTEGER NOT NULL DEFAULT 1,
    assigned_to  INTEGER,
    priority     INTEGER NOT NULL DEFAULT 0,
    source       TEXT NOT NULL DEFAULT 'cli'
                 CHECK(source IN ('cli', 'mcp', 'ui', 'api')),
    metadata     TEXT,
    category     TEXT,
    is_archived  INTEGER NOT NULL DEFAULT 0,
    phase        TEXT
);

-- Copy all data
INSERT INTO tasks_new SELECT id, type, title, description, status, created_at, updated_at, completed_at, created_by, assigned_to, priority, source, metadata, category, is_archived, phase FROM tasks;

-- Drop old table and rename
DROP TABLE tasks;
ALTER TABLE tasks_new RENAME TO tasks;

-- Recreate indexes (from migrations 001, 003, 004)
CREATE INDEX idx_tasks_status ON tasks(status) WHERE status != 'done';
CREATE INDEX idx_tasks_type ON tasks(type);
CREATE INDEX idx_tasks_assigned ON tasks(assigned_to) WHERE assigned_to IS NOT NULL;
CREATE INDEX idx_tasks_priority ON tasks(priority) WHERE status != 'done';
CREATE INDEX idx_tasks_category ON tasks(category) WHERE category IS NOT NULL;
CREATE INDEX idx_tasks_archived ON tasks(is_archived, completed_at DESC) WHERE is_archived = 1;

-- Recreate tasks_after_update trigger (from migration 001)
-- +goose StatementBegin
CREATE TRIGGER tasks_after_update AFTER UPDATE ON tasks
BEGIN
    UPDATE tasks SET
        updated_at = CURRENT_TIMESTAMP,
        completed_at = CASE
            WHEN NEW.status = 'done' AND OLD.status != 'done' THEN CURRENT_TIMESTAMP
            WHEN NEW.status != 'done' THEN NULL
            ELSE completed_at
        END
    WHERE id = NEW.id;
END;
-- +goose StatementEnd

-- Recreate FTS triggers (from migration 002)
-- +goose StatementBegin
CREATE TRIGGER tasks_fts_insert AFTER INSERT ON tasks
BEGIN
    INSERT INTO tasks_fts(rowid, title, description)
    VALUES (NEW.id, NEW.title, COALESCE(NEW.description, ''));
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER tasks_fts_update AFTER UPDATE OF title, description ON tasks
BEGIN
    INSERT INTO tasks_fts(tasks_fts, rowid, title, description)
    VALUES ('delete', OLD.id, OLD.title, COALESCE(OLD.description, ''));
    INSERT INTO tasks_fts(rowid, title, description)
    VALUES (NEW.id, NEW.title, COALESCE(NEW.description, ''));
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER tasks_fts_delete AFTER DELETE ON tasks
BEGIN
    INSERT INTO tasks_fts(tasks_fts, rowid, title, description)
    VALUES ('delete', OLD.id, OLD.title, COALESCE(OLD.description, ''));
END;
-- +goose StatementEnd

-- Rebuild FTS index to ensure consistency after table recreation
INSERT INTO tasks_fts(tasks_fts) VALUES ('rebuild');

-- +goose Down

-- Remove custom type columns from workflows
ALTER TABLE workflows DROP COLUMN color;
ALTER TABLE workflows DROP COLUMN prefix;
ALTER TABLE workflows DROP COLUMN is_builtin;

-- Note: we do NOT restore the CHECK constraint on tasks.type in the down migration.
-- The built-in types remain valid and the constraint was only limiting flexibility.
