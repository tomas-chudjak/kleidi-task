-- name: ListCategories :many
SELECT * FROM categories ORDER BY name;

-- name: CreateCategory :one
INSERT INTO categories (name, color) VALUES (?, ?) RETURNING *;

-- name: DeleteCategory :exec
DELETE FROM categories WHERE name = ?;
