-- name: CreateTask :one
INSERT INTO tasks (type, title, description, status, priority, source, created_by)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = ?;

-- name: ListTasks :many
SELECT * FROM tasks
ORDER BY priority DESC, created_at DESC
LIMIT ?;

-- name: ListTasksByStatus :many
SELECT * FROM tasks
WHERE status = ?
ORDER BY priority DESC, created_at DESC
LIMIT ?;

-- name: ListTasksByType :many
SELECT * FROM tasks
WHERE type = ?
ORDER BY priority DESC, created_at DESC
LIMIT ?;

-- name: ListTasksByStatusAndType :many
SELECT * FROM tasks
WHERE status = ? AND type = ?
ORDER BY priority DESC, created_at DESC
LIMIT ?;

-- name: UpdateTask :one
UPDATE tasks
SET title = ?, description = ?, status = ?, type = ?, priority = ?
WHERE id = ?
RETURNING *;

-- name: CompleteTask :one
UPDATE tasks
SET status = 'done'
WHERE id = ?
RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = ?;

-- name: CountTasksByStatus :many
SELECT status, COUNT(*) as count FROM tasks GROUP BY status;

-- name: CountBugsOpen :one
SELECT COUNT(*) as count FROM tasks WHERE type = 'bug' AND status != 'done';
