-- name: GetWorkflow :one
SELECT * FROM workflows WHERE task_type = ?;

-- name: ListWorkflows :many
SELECT * FROM workflows ORDER BY task_type;

-- name: SetTaskPhase :exec
UPDATE tasks SET phase = ?, status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: InsertWorkflowHistory :one
INSERT INTO workflow_history (task_id, phase, action, action_type, output, success, duration_ms)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListWorkflowHistory :many
SELECT * FROM workflow_history WHERE task_id = ? ORDER BY created_at ASC;
