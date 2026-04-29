-- name: CreateTask :one
INSERT INTO tasks (type, title, description, status, priority, source, created_by, category, metadata)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = ?;

-- name: ListTasksFiltered :many
SELECT * FROM tasks
WHERE is_archived = 0
  AND (sqlc.narg('status') IS NULL OR instr(',' || sqlc.narg('status') || ',', ',' || status || ',') > 0)
  AND (sqlc.narg('type') IS NULL OR instr(',' || sqlc.narg('type') || ',', ',' || type || ',') > 0)
  AND (sqlc.narg('category') IS NULL OR instr(',' || sqlc.narg('category') || ',', ',' || category || ',') > 0)
  AND (sqlc.narg('min_priority') IS NULL OR priority >= sqlc.narg('min_priority'))
  AND (sqlc.narg('created_after') IS NULL OR created_at >= sqlc.narg('created_after'))
  AND (sqlc.narg('created_before') IS NULL OR created_at <= sqlc.narg('created_before'))
ORDER BY priority DESC, created_at DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: CountTasksFiltered :one
SELECT count(*) FROM tasks
WHERE is_archived = 0
  AND (sqlc.narg('status') IS NULL OR instr(',' || sqlc.narg('status') || ',', ',' || status || ',') > 0)
  AND (sqlc.narg('type') IS NULL OR instr(',' || sqlc.narg('type') || ',', ',' || type || ',') > 0)
  AND (sqlc.narg('category') IS NULL OR instr(',' || sqlc.narg('category') || ',', ',' || category || ',') > 0)
  AND (sqlc.narg('min_priority') IS NULL OR priority >= sqlc.narg('min_priority'))
  AND (sqlc.narg('created_after') IS NULL OR created_at >= sqlc.narg('created_after'))
  AND (sqlc.narg('created_before') IS NULL OR created_at <= sqlc.narg('created_before'));

-- name: UpdateTask :one
UPDATE tasks
SET title = ?, description = ?, status = ?, type = ?, priority = ?, category = ?, metadata = ?
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
SELECT status, COUNT(*) as count FROM tasks WHERE is_archived = 0 GROUP BY status;

-- name: CountBugsOpen :one
SELECT COUNT(*) as count FROM tasks WHERE is_archived = 0 AND type = 'bug' AND status != 'done';

-- name: CountCompletedSince :one
SELECT COUNT(*) as count FROM tasks WHERE is_archived = 0 AND status = 'done' AND completed_at >= ?;

-- name: CountByType :many
SELECT type, COUNT(*) as count FROM tasks WHERE is_archived = 0 GROUP BY type;

-- name: RecentCompleted :many
SELECT * FROM tasks WHERE is_archived = 0 AND status = 'done' ORDER BY completed_at DESC LIMIT ?;

-- name: ArchiveTask :one
UPDATE tasks SET is_archived = 1 WHERE id = ? AND status = 'done' RETURNING *;

-- name: UnarchiveTask :one
UPDATE tasks SET is_archived = 0 WHERE id = ? AND is_archived = 1 RETURNING *;

-- name: ListArchivedFiltered :many
SELECT * FROM tasks
WHERE is_archived = 1
  AND (sqlc.narg('type') IS NULL OR instr(',' || sqlc.narg('type') || ',', ',' || type || ',') > 0)
  AND (sqlc.narg('category') IS NULL OR instr(',' || sqlc.narg('category') || ',', ',' || category || ',') > 0)
  AND (sqlc.narg('created_after') IS NULL OR completed_at >= sqlc.narg('created_after'))
  AND (sqlc.narg('created_before') IS NULL OR completed_at <= sqlc.narg('created_before'))
ORDER BY completed_at DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: CountArchivedFiltered :one
SELECT count(*) FROM tasks
WHERE is_archived = 1
  AND (sqlc.narg('type') IS NULL OR instr(',' || sqlc.narg('type') || ',', ',' || type || ',') > 0)
  AND (sqlc.narg('category') IS NULL OR instr(',' || sqlc.narg('category') || ',', ',' || category || ',') > 0)
  AND (sqlc.narg('created_after') IS NULL OR completed_at >= sqlc.narg('created_after'))
  AND (sqlc.narg('created_before') IS NULL OR completed_at <= sqlc.narg('created_before'));

-- name: CountArchived :one
SELECT COUNT(*) as count FROM tasks WHERE is_archived = 1;

-- name: ArchiveCompletedBefore :execresult
UPDATE tasks SET is_archived = 1
WHERE status = 'done' AND is_archived = 0
AND completed_at IS NOT NULL AND completed_at < ?;
