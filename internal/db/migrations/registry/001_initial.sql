-- +goose Up

-- List of projects and their locations
CREATE TABLE projects (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    slug         TEXT NOT NULL UNIQUE,
    name         TEXT NOT NULL,
    path         TEXT NOT NULL UNIQUE,
    last_seen_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,

    -- Cached stats (updated on each access)
    cached_todo_count   INTEGER DEFAULT 0,
    cached_doing_count  INTEGER DEFAULT 0,
    cached_total_count  INTEGER DEFAULT 0,
    stats_updated_at    DATETIME
);

-- Users (multi-user readiness, MVP = only 'local')
CREATE TABLE users (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    username   TEXT NOT NULL UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO users (id, username) VALUES (1, 'local');

-- API tokens for REST/MCP HTTP authentication
CREATE TABLE api_tokens (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER NOT NULL REFERENCES users(id),
    name         TEXT NOT NULL,
    token_hash   TEXT NOT NULL UNIQUE,
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME,
    expires_at   DATETIME
);

CREATE INDEX idx_tokens_hash ON api_tokens(token_hash);

-- +goose Down
DROP TABLE IF EXISTS api_tokens;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS projects;
