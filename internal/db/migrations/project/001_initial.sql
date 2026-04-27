-- +goose Up

-- Meta table for versioning and project metadata
CREATE TABLE meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Main table for tasks and bugs
CREATE TABLE tasks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    type         TEXT NOT NULL DEFAULT 'task'
                 CHECK(type IN ('task', 'bug')),
    title        TEXT NOT NULL,
    description  TEXT,
    status       TEXT NOT NULL DEFAULT 'todo'
                 CHECK(status IN ('todo', 'doing', 'done')),
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,

    -- Multi-user readiness (default = 1 = local user)
    created_by   INTEGER NOT NULL DEFAULT 1,
    assigned_to  INTEGER,

    -- Priority (higher number = higher priority, default 0 = normal)
    priority     INTEGER NOT NULL DEFAULT 0,

    -- Audit trail — set automatically by entry point
    source       TEXT NOT NULL DEFAULT 'cli'
                 CHECK(source IN ('cli', 'mcp', 'ui', 'api')),

    -- Future extensions (stored as JSON)
    metadata     TEXT
);

-- Indexes
CREATE INDEX idx_tasks_status ON tasks(status) WHERE status != 'done';
CREATE INDEX idx_tasks_type ON tasks(type);
CREATE INDEX idx_tasks_assigned ON tasks(assigned_to) WHERE assigned_to IS NOT NULL;
CREATE INDEX idx_tasks_priority ON tasks(priority) WHERE status != 'done';

-- Single trigger: updated_at always, completed_at on transition to 'done'
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

-- +goose Down
DROP TRIGGER IF EXISTS tasks_after_update;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS meta;
