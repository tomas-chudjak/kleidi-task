-- name: ListCategories :many
SELECT * FROM categories ORDER BY name;

-- name: CreateCategory :one
INSERT INTO categories (name, color) VALUES (?, ?) RETURNING *;

-- name: UpdateCategory :one
UPDATE categories SET name = ?, color = ? WHERE id = ? RETURNING *;

-- name: GetCategory :one
SELECT * FROM categories WHERE id = ?;

-- name: DeleteCategoryByID :exec
DELETE FROM categories WHERE id = ?;

-- name: DeleteCategory :exec
DELETE FROM categories WHERE name = ?;
