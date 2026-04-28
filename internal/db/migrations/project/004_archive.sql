-- +goose Up

ALTER TABLE tasks ADD COLUMN is_archived INTEGER NOT NULL DEFAULT 0;

CREATE INDEX idx_tasks_archived ON tasks(is_archived, completed_at DESC)
    WHERE is_archived = 1;

-- +goose Down

DROP INDEX IF EXISTS idx_tasks_archived;
ALTER TABLE tasks DROP COLUMN is_archived;
