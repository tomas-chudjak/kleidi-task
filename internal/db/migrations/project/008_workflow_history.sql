-- +goose Up

-- Execution history for workflow phase transitions
CREATE TABLE workflow_history (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id    INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    phase      TEXT NOT NULL,
    action     TEXT NOT NULL DEFAULT '',
    action_type TEXT NOT NULL DEFAULT 'none',
    output     TEXT NOT NULL DEFAULT '',
    success    INTEGER NOT NULL DEFAULT 1,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_workflow_history_task ON workflow_history(task_id);

-- +goose Down
DROP INDEX IF EXISTS idx_workflow_history_task;
DROP TABLE IF EXISTS workflow_history;
