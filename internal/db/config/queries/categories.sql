-- name: ListCategories :many
SELECT sqlc.embed(categories) FROM categories;

-- name: GetCategory :one
SELECT sqlc.embed(categories) FROM categories
WHERE id = $1 LIMIT 1;

-- name: GetCategoryByName :one
SELECT sqlc.embed(categories) FROM categories
WHERE name = $1 LIMIT 1;

-- name: GetCategoryByNetworkAndName :one
SELECT sqlc.embed(categories) FROM categories
WHERE network_id = $1 AND name = $2 LIMIT 1;

-- name: GetCategoryByReferenceID :one
SELECT sqlc.embed(categories), sqlc.embed(external_references) FROM categories
LEFT JOIN external_references ON categories.reference_id = external_references.id
WHERE categories.reference_id = $1 LIMIT 1;

-- name: GetCategoryIDByExternalReferenceID :one
SELECT categories.id FROM categories
LEFT JOIN external_references ON categories.reference_id = external_references.id
WHERE external_references.platform_id = $1 AND external_references.type = 'category' AND external_references.external_id = $2 LIMIT 1;

-- name: GetCategoryByExternalReferenceID :one
SELECT sqlc.embed(categories), sqlc.embed(external_references) FROM categories
LEFT JOIN external_references ON categories.reference_id = external_references.id
WHERE external_references.platform_id = $1 AND external_references.type = 'category' AND external_references.external_id = $2 LIMIT 1;

-- name: CreateCategory :one
INSERT INTO categories (reference_id, network_id, name, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateCategory :one
UPDATE categories
SET
    name = $2,
    status = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;