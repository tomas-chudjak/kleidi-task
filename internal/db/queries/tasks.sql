-- name: CreateTask :one
INSERT INTO tasks (type, title, description, status, priority, source, created_by)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = ?;

-- name: ListTasksFiltered :many
SELECT * FROM tasks
WHERE (sqlc.narg('status') IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('type') IS NULL OR type = sqlc.narg('type'))
  AND (sqlc.narg('min_priority') IS NULL OR priority >= sqlc.narg('min_priority'))
  AND (sqlc.narg('created_after') IS NULL OR created_at >= sqlc.narg('created_after'))
  AND (sqlc.narg('created_before') IS NULL OR created_at <= sqlc.narg('created_before'))
ORDER BY priority DESC, created_at DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: CountTasksFiltered :one
SELECT count(*) FROM tasks
WHERE (sqlc.narg('status') IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('type') IS NULL OR type = sqlc.narg('type'))
  AND (sqlc.narg('min_priority') IS NULL OR priority >= sqlc.narg('min_priority'))
  AND (sqlc.narg('created_after') IS NULL OR created_at >= sqlc.narg('created_after'))
  AND (sqlc.narg('created_before') IS NULL OR created_at <= sqlc.narg('created_before'));

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
