-- name: ListOffers :many
SELECT sqlc.embed(offers), sqlc.embed(external_references) FROM offers
LEFT JOIN external_references ON offers.reference_id = external_references.id;

-- name: GetOffer :one
SELECT sqlc.embed(offers), sqlc.embed(external_references) FROM offers
LEFT JOIN external_references ON offers.reference_id = external_references.id
WHERE offers.id = $1 LIMIT 1;

-- name: GetOfferIDByReferenceID :one
SELECT id FROM offers WHERE reference_id = $1 LIMIT 1;

-- name: GetOfferByReferenceID :one
SELECT sqlc.embed(offers), sqlc.embed(external_references) FROM offers
LEFT JOIN external_references ON offers.reference_id = external_references.id
WHERE offers.reference_id = $1 LIMIT 1;

-- name: GetOfferIDByExternalReferenceID :one
SELECT offers.id FROM offers
LEFT JOIN external_references ON offers.reference_id = external_references.id
WHERE external_references.platform_id = $1 AND external_references.type = 'offer' AND external_references.external_id = $2 LIMIT 1;

-- name: GetOfferByExternalReferenceID :one
SELECT sqlc.embed(offers), sqlc.embed(external_references) FROM offers
LEFT JOIN external_references ON offers.reference_id = external_references.id
WHERE external_references.platform_id = $1 AND external_references.type = 'offer' AND external_references.external_id = $2 LIMIT 1;

-- name: GetOffersByPageURL :many
SELECT sqlc.embed(offers), sqlc.embed(offer_urls), sqlc.embed(categories), external_references.external_id AS external_offer_id, offer_urls_ref.external_id AS external_offer_url_id FROM offers
LEFT JOIN offer_urls ON offers.id = offer_urls.offer_id
LEFT JOIN categories ON offers.category_id = categories.id
LEFT JOIN external_references ON offers.reference_id = external_references.id
LEFT JOIN external_references AS offer_urls_ref ON offer_urls.reference_id = offer_urls_ref.id
WHERE (
    REGEXP_LIKE(offer_urls.destination_url, CONCAT_WS('', sqlc.arg(page_url)::text, '\??.*'), 'i')
    OR (offer_urls.preview_url IS NOT NULL AND REGEXP_LIKE(offer_urls.preview_url, CONCAT_WS('', sqlc.arg(page_url)::text, '\??.*'), 'i'))
)
ORDER BY offers.status ASC, offer_urls.status ASC;

-- name: GetNetworkIDByOfferID :one
SELECT network_id FROM offers WHERE id = $1 LIMIT 1;

-- name: CreateOffer :one
INSERT INTO offers (
    reference_id,
    network_id,
    category_id,
    advertiser_id,
    name,
    status,
    thumbnail_url,
    visibility
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateOffer :one
UPDATE offers SET
    category_id = $2,
    advertiser_id = $3,
    name = $4,
    status = $5,
    thumbnail_url = $6,
    visibility = $7,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: FindBestOffers :many
SELECT
COALESCE((
    SELECT id FROM raw_clicks
    WHERE raw_clicks.offer_id = offers.id
    AND raw_clicks.offer_url_id = offer_urls.id
    AND raw_clicks.transaction_id = sqlc.narg(transaction_id)::text
    ORDER BY raw_clicks.created_at ASC
    LIMIT 1
), 0)::bigint AS click_id,
sqlc.embed(offers),
sqlc.embed(offer_urls),
sqlc.embed(categories),
ex_ref_offer.external_id AS external_offer_id,
ex_ref_offer_urls.external_id AS external_offer_url_id
FROM offer_urls
JOIN offers ON offers.id = offer_urls.offer_id
LEFT JOIN categories ON offers.category_id = categories.id
LEFT JOIN external_references AS ex_ref_offer ON offers.reference_id = ex_ref_offer.id
LEFT JOIN external_references AS ex_ref_offer_urls ON offer_urls.reference_id = ex_ref_offer_urls.id
WHERE (
    offer_urls.destination_url ILIKE REGEXP_REPLACE(sqlc.arg(page_url)::text, '^(https?://)?([^/]+)/?([^/\?]*)(?:\??.*)?$', '%\2/\3%')
    OR (offer_urls.preview_url IS NOT NULL AND offer_urls.preview_url ILIKE REGEXP_REPLACE(sqlc.arg(page_url)::text, '^(https?://)?([^/]+)/?([^/\?]*)(?:\??.*)?$', '%\2/\3%'))
) AND  (
    sqlc.narg(external_offer_id)::bigint IS NULL OR (ex_ref_offer.external_id = sqlc.narg(external_offer_id)::bigint)
) AND (
    sqlc.narg(external_offer_url_id)::bigint IS NULL OR (ex_ref_offer_urls.external_id = sqlc.narg(external_offer_url_id)::bigint)
)
ORDER BY offers.status ASC, offer_urls.status ASC;