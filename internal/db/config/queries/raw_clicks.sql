-- name: GetFirstRawClickTimestamp :one
SELECT MIN(created_at)::timestamp AS first_timestamp FROM raw_clicks;

-- name: GetNextRawClickTimestamp :one
SELECT MIN(created_at)::timestamp AS next_timestamp FROM raw_clicks WHERE created_at >= $1;

-- name: GetLastRawClickTimestamp :one
SELECT MAX(created_at)::timestamp AS last_timestamp FROM raw_clicks;

-- name: GetLastRawClickTimestampByNetwork :one
SELECT MAX(created_at)::timestamp AS last_timestamp FROM raw_clicks WHERE network_id = $1;

-- name: GetRawClickByID :one
SELECT * FROM raw_clicks WHERE id = $1 LIMIT 1;

-- name: GetRawClicksByTransactionID :many
SELECT * FROM raw_clicks WHERE transaction_id = $1;

-- name: GetRawClickByNetworkAndTransactionID :one
SELECT * FROM raw_clicks WHERE network_id = $1 AND transaction_id = $2 LIMIT 1;

-- name: GetRawClickIDByNetworkAndTransactionID :one
SELECT id FROM raw_clicks WHERE network_id = $1 AND transaction_id = $2 LIMIT 1;

-- name: CreateRawClick :one
INSERT INTO raw_clicks (
    offer_id,
    offer_url_id,
    advertiser_id,
    affiliate_id,
    category_id,
    network_id,
    transaction_id,
    is_unique,
    tracking_url,
    source_id,
    sub1,
    sub2,
    sub3,
    sub4,
    sub5,
    payout_amount,
    revenue_amount,
    referer,
    error_code,
    error_filter_id,
    url,
    is_view_through,
    is_async,
    server_side_url,
    custom_landing_page_id,
    is_test_mode,
    idfa,
    google_ad_id,
    creative_id,
    coupon_code,
    redirect_method,
    is_sdk_click,
    ip_address,
    user_agent,
    geolocation,
    device_info,
    traffic_source,
    is_bot,
    is_mobile,
    is_tablet,
    device_type,
    language_code,
    country_code,
    region_code,
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
    $14,
    $15,
    $16,
    $17,
    $18,
    $19,
    $20,
    $21,
    $22,
    $23,
    $24,
    $25,
    $26,
    $27,
    $28,
    $29,
    $30,
    $31,
    $32,
    $33,
    $34,
    $35,
    $36,
    $37,
    $38,
    $39,
    $40,
    $41,
    $42,
    $43,
    $44,
    $45
)
RETURNING *;

-- name: UpdateRawClick :one
UPDATE raw_clicks
SET
    offer_id = $2,
    offer_url_id = $3,
    advertiser_id = $4,
    affiliate_id = $5,
    category_id = $6,
    network_id = $7,
    transaction_id = $8,
    is_unique = $9,
    tracking_url = $10,
    source_id = $11,
    sub1 = $12,
    sub2 = $13,
    sub3 = $14,
    sub4 = $15,
    sub5 = $16,
    payout_amount = $17,
    revenue_amount = $18,
    referer = $19,
    error_code = $20,
    error_filter_id = $21,
    url = $22,
    is_view_through = $23,
    is_async = $24,
    server_side_url = $25,
    custom_landing_page_id = $26,
    is_test_mode = $27,
    idfa = $28,
    google_ad_id = $29,
    creative_id = $30,
    coupon_code = $31,
    redirect_method = $32,
    is_sdk_click = $33,
    ip_address = $34,
    user_agent = $35,
    geolocation = $36,
    traffic_source = $37,
    device_info = $38,
    is_bot = $39,
    is_mobile = $40,
    is_tablet = $41,
    device_type = $42,
    language_code = $43,
    country_code = $44,
    region_code = $45
WHERE id = $1
RETURNING *;

-- name: UpdateRawClickStatsAggregates :one
UPDATE raw_clicks
SET traffic_source = $2,
    is_bot = $3,
    is_mobile = $4,
    is_tablet = $5,
    device_type = $6,
    language_code = $7,
    country_code = $8,
    region_code = $9
WHERE id = $1
RETURNING *;

-- name: UpdateRawClickMetadata :one
UPDATE raw_clicks
SET geolocation = $2,
    device_info = $3
WHERE id = $1
RETURNING *;

-- name: UpdateRawClickOfferURLID :one
UPDATE raw_clicks
SET offer_url_id = $2
WHERE id = $1
RETURNING *;

-- name: GetPaginatedRawClicks :many
SELECT * FROM raw_clicks
WHERE id > COALESCE(sqlc.narg('after_id')::bigint, 0)
ORDER BY id ASC
LIMIT COALESCE(sqlc.narg('limit')::int, 100);