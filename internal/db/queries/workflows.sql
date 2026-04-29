-- name: GetWorkflow :one
SELECT * FROM workflows WHERE task_type = ?;

-- name: ListWorkflows :many
SELECT * FROM workflows ORDER BY task_type;

-- name: SetTaskPhase :exec
UPDATE tasks SET phase = ?, status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;
