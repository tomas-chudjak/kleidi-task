-- +goose Up

-- FTS5 external content table synced with tasks
CREATE VIRTUAL TABLE tasks_fts USING fts5(title, description, content=tasks, content_rowid=id);

-- Populate FTS index from existing data
INSERT INTO tasks_fts(tasks_fts) VALUES ('rebuild');

-- Keep FTS in sync via external content triggers
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

-- +goose Down
DROP TRIGGER IF EXISTS tasks_fts_delete;
DROP TRIGGER IF EXISTS tasks_fts_update;
DROP TRIGGER IF EXISTS tasks_fts_insert;
DROP TABLE IF EXISTS tasks_fts;
