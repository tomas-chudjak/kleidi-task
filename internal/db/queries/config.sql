-- name: GetConfig :one
SELECT value FROM project_config WHERE key = ?;

-- name: SetConfig :exec
INSERT INTO project_config (key, value) VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value;

-- name: ListConfig :many
SELECT key, value FROM project_config ORDER BY key;
