-- name: GetLastRawConversionTimestamp :one
SELECT MAX(created_at)::timestamp AS last_timestamp FROM raw_conversions;

-- name: GetLastRawConversionTimestampByNetwork :one
SELECT MAX(created_at)::timestamp AS last_timestamp FROM raw_conversions WHERE network_id = $1;

-- name: GetRawConversionIDByNetworkAndConversionID :one
SELECT id FROM raw_conversions WHERE network_id = $1 AND conversion_id = $2 LIMIT 1;

-- name: GetRawConversionByNetworkAndConversionID :one
SELECT * FROM raw_conversions WHERE network_id = $1 AND conversion_id = $2 LIMIT 1;

-- name: GetRawConversionByTransactionID :one
SELECT * FROM raw_conversions WHERE transaction_id = $1 LIMIT 1;

-- name: GetRawConversionsByNetworkAndTransactionID :many
SELECT * FROM raw_conversions WHERE network_id = $1 AND transaction_id = $2;

-- name: GetRawConversionsMissingClickIDByNetworkAndTransactionID :many
SELECT * FROM raw_conversions WHERE network_id = $1 AND transaction_id = $2 AND click_id IS NULL;

-- name: CreateRawConversion :one
INSERT INTO raw_conversions (
    conversion_id,
    offer_id,
    offer_url_id,
    network_id,
    transaction_id,
    click_id,
    status,
    payout_model,
    payout_amount,
    revenue_model,
    revenue_amount,
    ip_address,
    user_agent,
    created_at
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11,
    $12,
    $13,
    $14
)
RETURNING *;

-- name: UpdateRawConversion :one
UPDATE raw_conversions SET
    network_id = $2,
    transaction_id = $3,
    click_id = $4,
    status = $5,
    payout_model = $6,
    payout_amount = $7,
    revenue_model = $8,
    revenue_amount = $9,
    ip_address = $10,
    user_agent = $11,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateRawConversionClickID :one
UPDATE raw_conversions SET
    click_id = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;


-- name: UpdateRawConversionOfferID :one
UPDATE raw_conversions SET
    offer_id = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateRawConversionOfferURLID :one
UPDATE raw_conversions SET
    offer_url_id = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetPaginatedRawConversions :many
SELECT raw_conversions.*, raw_clicks.offer_url_id as click_offer_url_id, raw_clicks.url FROM raw_conversions
LEFT JOIN raw_clicks ON raw_conversions.click_id = raw_clicks.id
WHERE raw_conversions.id > COALESCE(sqlc.narg('after_id')::bigint, 0)
ORDER BY raw_conversions.id ASC
LIMIT COALESCE(sqlc.narg('limit')::int, 100);