-- name: ListAdvertisers :many
SELECT sqlc.embed(advertisers), sqlc.embed(external_references) FROM advertisers
LEFT JOIN external_references ON advertisers.reference_id = external_references.id;

-- name: GetAdvertiser :one
SELECT sqlc.embed(advertisers), sqlc.embed(external_references) FROM advertisers
LEFT JOIN external_references ON advertisers.reference_id = external_references.id
WHERE advertisers.id = $1 LIMIT 1;

-- name: GetAdvertiserByReferenceID :one
SELECT sqlc.embed(advertisers), sqlc.embed(external_references) FROM advertisers
LEFT JOIN external_references ON advertisers.reference_id = external_references.id
WHERE advertisers.reference_id = $1 LIMIT 1;

-- name: GetAdvertiserIDByExternalReferenceID :one
SELECT advertisers.id FROM advertisers
LEFT JOIN external_references ON advertisers.reference_id = external_references.id
WHERE external_references.platform_id = $1 AND external_references.type = 'advertiser' AND external_references.external_id = $2 LIMIT 1;

-- name: GetAdvertiserByExternalReferenceID :one
SELECT sqlc.embed(advertisers), sqlc.embed(external_references) FROM advertisers
LEFT JOIN external_references ON advertisers.reference_id = external_references.id
WHERE external_references.platform_id = $1 AND external_references.type = 'advertiser' AND external_references.external_id = $2 LIMIT 1;

-- name: CreateAdvertiser :one
INSERT INTO advertisers (reference_id, network_id, name, status, default_currency, attribution_method, attribution_priority, email_attibution_method)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateAdvertiser :one
UPDATE advertisers
SET
    name = $2,
    status = $3,
    default_currency = $4,
    attribution_method = $5,
    attribution_priority = $6,
    email_attibution_method = $7,
    updated_at = NOW()
WHERE id = $1
RETURNING *;