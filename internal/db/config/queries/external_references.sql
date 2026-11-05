-- name: GetReference :one
SELECT * FROM external_references
WHERE id = $1 LIMIT 1;

-- name: GetReferenceID :one
SELECT id FROM external_references
WHERE platform_id = $1 AND type = $2 AND external_id = $3 LIMIT 1;

-- name: CreateReferenceID :one
INSERT INTO external_references (platform_id, type, external_id)
VALUES ($1, $2, $3)
RETURNING id;
