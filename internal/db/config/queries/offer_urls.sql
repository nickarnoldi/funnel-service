-- name: GetOfferLinksMatchingURL :many
SELECT offer_urls.* FROM offer_urls
WHERE sqlc.arg('url') ILIKE REGEXP_REPLACE(offer_urls.destination_url, '\={[^}]+}', '=%', 'g')
AND (sqlc.narg('offer_id')::bigint IS NULL OR offer_urls.offer_id = sqlc.narg('offer_id')::bigint);

-- name: GetOfferLinkIDMatchingURL :one
SELECT offer_urls.id FROM offer_urls
WHERE sqlc.arg('url') ILIKE REGEXP_REPLACE(offer_urls.destination_url, '\={[^}]+}', '=%', 'g')
AND (sqlc.narg('offer_id')::bigint IS NULL OR offer_urls.offer_id = sqlc.narg('offer_id')::bigint)
ORDER BY LENGTH(offer_urls.destination_url) DESC
LIMIT 1;

-- name: GetOfferURLs :many
SELECT offer_urls.*, sqlc.embed(offers), sqlc.embed(external_references) FROM offer_urls
LEFT JOIN offers ON offer_urls.offer_id = offers.id
LEFT JOIN external_references ON offer_urls.reference_id = external_references.id;

-- name: GetOfferURLsByOfferID :many
SELECT offer_urls.*, sqlc.embed(offers), sqlc.embed(external_references) FROM offer_urls
LEFT JOIN offers ON offer_urls.offer_id = offers.id
LEFT JOIN external_references ON offer_urls.reference_id = external_references.id
WHERE offer_urls.offer_id = $1;

-- name: GetOfferURL :one
SELECT offer_urls.*, sqlc.embed(offers), sqlc.embed(external_references) FROM offer_urls
LEFT JOIN offers ON offer_urls.offer_id = offers.id
LEFT JOIN external_references ON offer_urls.reference_id = external_references.id
WHERE offer_urls.id = $1 LIMIT 1;

-- name: GetDefaultOfferURLIDByOfferID :one
SELECT offer_urls.id FROM offer_urls
WHERE offer_urls.offer_id = $1 AND offer_urls.name = 'Default' LIMIT 1;

-- name: GetDefaultOfferURLByOfferID :one
SELECT offer_urls.* FROM offer_urls
WHERE offer_urls.offer_id = $1 AND offer_urls.name = 'Default' LIMIT 1;

-- name: GetOfferURLByReferenceID :one
SELECT offer_urls.*, sqlc.embed(offers), sqlc.embed(external_references) FROM offer_urls
LEFT JOIN offers ON offer_urls.offer_id = offers.id
LEFT JOIN external_references ON offer_urls.reference_id = external_references.id
WHERE offer_urls.reference_id = $1 LIMIT 1;

-- name: GetOfferURLByExternalReferenceID :one
SELECT offer_urls.*, sqlc.embed(offers), sqlc.embed(external_references) FROM offer_urls
LEFT JOIN offers ON offer_urls.offer_id = offers.id
LEFT JOIN external_references ON offer_urls.reference_id = external_references.id
WHERE external_references.platform_id = $1 AND external_references.type = 'offer_url' AND external_references.external_id = $2 LIMIT 1;

-- name: GetOfferURLIDByExternalReferenceID :one
SELECT offer_urls.id FROM offer_urls
LEFT JOIN external_references ON offer_urls.reference_id = external_references.id
WHERE external_references.platform_id = $1 AND external_references.type = 'offer_url' AND external_references.external_id = $2 LIMIT 1;

-- name: GetOfferURLIDLikeURL :one
SELECT id FROM offer_urls WHERE REGEXP_LIKE(offer_urls.destination_url, CONCAT_WS('', sqlc.arg(page_url)::text, '\??.*'), 'i') LIMIT 1;

-- name: GetOfferURLIDByURL :one
SELECT id FROM offer_urls WHERE offer_urls.destination_url = $1 LIMIT 1;

-- name: CreateOfferURL :one
INSERT INTO offer_urls (
    reference_id,
    funnel_id,
    network_id,
    offer_id,
    name,
    status,
    destination_url,
    preview_url
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateOfferURL :one
UPDATE offer_urls SET
    offer_id = $2,
    funnel_id = $3,
    name = $4,
    status = $5,
    destination_url = $6,
    preview_url = $7,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateOfferURLFunnelID :one
UPDATE offer_urls SET
    funnel_id = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;
