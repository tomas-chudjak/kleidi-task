-- name: ListTemplates :many
SELECT * FROM task_templates ORDER BY name;

-- name: GetTemplate :one
SELECT * FROM task_templates WHERE id = ?;

-- name: GetTemplateByType :one
SELECT * FROM task_templates WHERE type = ? LIMIT 1;

-- name: CreateTemplate :one
INSERT INTO task_templates (name, type, priority, description, agent_rules, review_instruction)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateTemplate :one
UPDATE task_templates
SET name = ?, type = ?, priority = ?, description = ?, agent_rules = ?, review_instruction = ?
WHERE id = ?
RETURNING *;

-- name: DeleteTemplate :exec
DELETE FROM task_templates WHERE id = ?;
