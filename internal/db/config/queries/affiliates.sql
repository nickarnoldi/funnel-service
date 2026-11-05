-- name: ListAffiliates :many
SELECT sqlc.embed(affiliates), sqlc.embed(external_references) FROM affiliates
LEFT JOIN external_references ON affiliates.reference_id = external_references.id;

-- name: GetAffiliate :one
SELECT sqlc.embed(affiliates), sqlc.embed(external_references) FROM affiliates
LEFT JOIN external_references ON affiliates.reference_id = external_references.id
WHERE affiliates.id = $1 LIMIT 1;

-- name: GetAffiliateByReferenceID :one
SELECT sqlc.embed(affiliates), sqlc.embed(external_references) FROM affiliates
LEFT JOIN external_references ON affiliates.reference_id = external_references.id
WHERE affiliates.reference_id = $1 LIMIT 1;

-- name: GetAffiliateIDByExternalReferenceID :one
SELECT affiliates.id FROM affiliates
LEFT JOIN external_references ON affiliates.reference_id = external_references.id
WHERE external_references.platform_id = $1 AND external_references.type = 'affiliate' AND external_references.external_id = $2 LIMIT 1;

-- name: GetAffiliateByExternalReferenceID :one
SELECT sqlc.embed(affiliates), sqlc.embed(external_references) FROM affiliates
LEFT JOIN external_references ON affiliates.reference_id = external_references.id
WHERE external_references.platform_id = $1 AND external_references.type = 'affiliate' AND external_references.external_id = $2 LIMIT 1;

-- name: CreateAffiliate :one
INSERT INTO affiliates (reference_id, network_id, name, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateAffiliate :one
UPDATE affiliates SET
    name = $2,
    status = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;