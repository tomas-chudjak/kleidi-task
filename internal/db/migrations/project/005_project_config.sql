-- +goose Up

-- Project-level configuration (key-value store)
CREATE TABLE project_config (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Insert defaults
INSERT INTO project_config (key, value) VALUES ('default_priority', '0');
INSERT INTO project_config (key, value) VALUES ('default_type', 'task');
INSERT INTO project_config (key, value) VALUES ('auto_archive_days', '0');

-- +goose Down
DROP TABLE IF EXISTS project_config;
