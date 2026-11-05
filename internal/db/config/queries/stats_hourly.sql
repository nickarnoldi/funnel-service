-- name: GetFirstStatsHourlySliceHour :one
SELECT MIN(slice_hour)::timestamp FROM stats_hourly;

-- name: GetLastStatsHourlySliceHour :one
SELECT MAX(slice_hour)::timestamp FROM stats_hourly;

-- name: GetStatsHourlyID :one
SELECT id FROM stats_hourly
WHERE slice_hour = sqlc.arg('slice_hour')::timestamp
AND offer_id = sqlc.arg('offer_id')
AND offer_url_id = sqlc.arg('offer_url_id')
AND category_id = sqlc.arg('category_id')
AND advertiser_id = sqlc.arg('advertiser_id')
AND affiliate_id IS NOT DISTINCT FROM sqlc.arg('affiliate_id')
AND traffic_source = sqlc.arg('traffic_source')
AND is_unique = sqlc.arg('is_unique')
AND is_bot = sqlc.arg('is_bot')
AND is_mobile = sqlc.arg('is_mobile')
AND is_tablet = sqlc.arg('is_tablet')
AND device_type = sqlc.arg('device_type')
AND language_code IS NOT DISTINCT FROM sqlc.arg('language_code')
AND country_code IS NOT DISTINCT FROM sqlc.arg('country_code')
AND region_code IS NOT DISTINCT FROM sqlc.arg('region_code');

-- name: GetStatsHourlyByFunnelIDs :many
SELECT stats_hourly.* FROM stats_hourly
JOIN offer_urls ON stats_hourly.offer_id = offer_urls.offer_id AND stats_hourly.offer_url_id = offer_urls.id
JOIN funnels ON offer_urls.funnel_id = funnels.id
WHERE funnels.id = ANY(sqlc.arg('funnel_ids')::bigint[])
AND (sqlc.narg('start_slice_hour') IS NULL OR stats_hourly.slice_hour >= sqlc.narg('start_slice_hour')::timestamp)
AND (sqlc.narg('end_slice_hour') IS NULL OR stats_hourly.slice_hour <= sqlc.narg('end_slice_hour')::timestamp)
ORDER BY stats_hourly.slice_hour ASC
OFFSET CASE WHEN COALESCE(sqlc.narg('offset')::int, 0) > 0 THEN sqlc.narg('offset')::int ELSE 0 END
LIMIT COALESCE(sqlc.narg('limit')::int, 100);

-- name: CreateStatsHourly :one
INSERT INTO stats_hourly (
    slice_hour,
    offer_id,
    offer_url_id,
    category_id,
    advertiser_id,
    affiliate_id,
    traffic_source,
    is_unique,
    is_bot,
    is_mobile,
    is_tablet,
    device_type,
    language_code,
    country_code,
    region_code,
    clicks,
    conversions
) VALUES (
    sqlc.arg('slice_hour')::timestamp,
    sqlc.arg('offer_id'),
    sqlc.arg('offer_url_id'),
    sqlc.arg('category_id'),
    sqlc.arg('advertiser_id'),
    sqlc.arg('affiliate_id'),
    sqlc.arg('traffic_source'),
    sqlc.arg('is_unique'),
    sqlc.arg('is_bot'),
    sqlc.arg('is_mobile'),
    sqlc.arg('is_tablet'),
    sqlc.arg('device_type'),
    sqlc.arg('language_code'),
    sqlc.arg('country_code'),
    sqlc.arg('region_code'),
    sqlc.arg('clicks'),
    sqlc.arg('conversions')
)
RETURNING *;

-- name: UpdateStatsHourly :one
UPDATE stats_hourly SET clicks = $1, conversions = $2
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: UpdateStatsHourlyClicks :one
UPDATE stats_hourly SET clicks = $1
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: UpdateStatsHourlyConversions :one
UPDATE stats_hourly SET conversions = $1
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: GenerateHourlyStats :many
SELECT
    slice_hour,
    offer_id,
    offer_url_id,
    category_id,
    advertiser_id,
    affiliate_id,
    traffic_source,
    is_unique,
    is_bot,
    is_mobile,
    is_tablet,
    device_type,
    language_code,
    country_code,
    region_code,
    SUM(clicks)::int AS clicks,
    SUM(conversions)::int AS conversions
FROM (
    (
        SELECT
            date_trunc('hour', created_at)::timestamp AS slice_hour,
            offer_id,
            offer_url_id,
            category_id,
            advertiser_id,
            affiliate_id,
            traffic_source,
            is_unique,
            is_bot,
            is_mobile,
            is_tablet,
            device_type,
            language_code,
            country_code,
            region_code,
            COUNT(*)::int AS clicks,
            0::int AS conversions
        FROM raw_clicks
        WHERE raw_clicks.created_at BETWEEN sqlc.arg('date')::date + time '00:00:00' AND sqlc.arg('date')::date + time '23:59:59'
        GROUP BY
            slice_hour,
            offer_id,
            offer_url_id,
            category_id,
            advertiser_id,
            affiliate_id,
            traffic_source,
            is_unique,
            is_bot,
            is_mobile,
            is_tablet,
            device_type,
            language_code,
            country_code,
            region_code
        ORDER BY slice_hour, offer_id, offer_url_id ASC
    ) UNION ALL (
        SELECT
            date_trunc('hour', raw_conversions.created_at)::timestamp AS slice_hour,
            raw_conversions.offer_id,
            raw_conversions.offer_url_id,
            raw_clicks.category_id,
            raw_clicks.advertiser_id,
            raw_clicks.affiliate_id,
            raw_clicks.traffic_source,
            raw_clicks.is_unique,
            raw_clicks.is_bot,
            raw_clicks.is_mobile,
            raw_clicks.is_tablet,
            raw_clicks.device_type,
            raw_clicks.language_code,
            raw_clicks.country_code,
            raw_clicks.region_code,
            0::int AS clicks,
            COUNT(*)::int AS conversions
        FROM raw_conversions
        JOIN raw_clicks ON raw_conversions.click_id = raw_clicks.id
        WHERE raw_conversions.created_at BETWEEN sqlc.arg('date')::date + time '00:00:00' AND sqlc.arg('date')::date + time '23:59:59'
        GROUP BY
            slice_hour,
            raw_conversions.offer_id,
            raw_conversions.offer_url_id,
            raw_clicks.category_id,
            raw_clicks.advertiser_id,
            raw_clicks.affiliate_id,
            raw_clicks.traffic_source,
            raw_clicks.is_unique,
            raw_clicks.is_bot,
            raw_clicks.is_mobile,
            raw_clicks.is_tablet,
            raw_clicks.device_type,
            raw_clicks.language_code,
            raw_clicks.country_code,
            raw_clicks.region_code
        ORDER BY slice_hour, raw_conversions.offer_id, raw_conversions.offer_url_id ASC
    )
) as combined
GROUP BY slice_hour, offer_id, offer_url_id, category_id, advertiser_id, affiliate_id, traffic_source, is_unique, is_bot, is_mobile, is_tablet, device_type, language_code, country_code, region_code
ORDER BY slice_hour, offer_id, offer_url_id ASC;

-- name: GenerateHourlyStatsByFunnelIDs :many
SELECT
    slice_hour,
    offer_id,
    offer_url_id,
    category_id,
    advertiser_id,
    affiliate_id,
    traffic_source,
    is_unique,
    is_bot,
    is_mobile,
    is_tablet,
    device_type,
    language_code,
    country_code,
    region_code,
    SUM(clicks)::int AS clicks,
    SUM(conversions)::int AS conversions
FROM (
    (
        SELECT
            date_trunc('hour', raw_clicks.created_at)::timestamp AS slice_hour,
            raw_clicks.offer_id,
            raw_clicks.offer_url_id,
            raw_clicks.category_id,
            raw_clicks.advertiser_id,
            raw_clicks.affiliate_id,
            raw_clicks.traffic_source,
            raw_clicks.is_unique,
            raw_clicks.is_bot,
            raw_clicks.is_mobile,
            raw_clicks.is_tablet,
            raw_clicks.device_type,
            raw_clicks.language_code,
            raw_clicks.country_code,
            raw_clicks.region_code,
            COUNT(raw_clicks.*)::int AS clicks,
            0::int AS conversions
        FROM raw_clicks
        JOIN offer_urls ON raw_clicks.offer_id = offer_urls.offer_id AND raw_clicks.offer_url_id = offer_urls.id
        JOIN funnels ON offer_urls.funnel_id = funnels.id
        WHERE funnels.id = ANY(sqlc.arg('funnel_ids')::bigint[])
        AND raw_clicks.created_at BETWEEN sqlc.arg('date')::date + time '00:00:00' AND sqlc.arg('date')::date + time '23:59:59'
        GROUP BY
            slice_hour,
            raw_clicks.offer_id,
            raw_clicks.offer_url_id,
            raw_clicks.category_id,
            raw_clicks.advertiser_id,
            raw_clicks.affiliate_id,
            raw_clicks.traffic_source,
            raw_clicks.is_unique,
            raw_clicks.is_bot,
            raw_clicks.is_mobile,
            raw_clicks.is_tablet,
            raw_clicks.device_type,
            raw_clicks.language_code,
            raw_clicks.country_code,
            raw_clicks.region_code
        ORDER BY slice_hour, raw_clicks.offer_id, raw_clicks.offer_url_id ASC
    ) UNION ALL (
        SELECT
            date_trunc('hour', raw_conversions.created_at)::timestamp AS slice_hour,
            raw_conversions.offer_id,
            raw_conversions.offer_url_id,
            raw_clicks.category_id,
            raw_clicks.advertiser_id,
            raw_clicks.affiliate_id,
            raw_clicks.traffic_source,
            raw_clicks.is_unique,
            raw_clicks.is_bot,
            raw_clicks.is_mobile,
            raw_clicks.is_tablet,
            raw_clicks.device_type,
            raw_clicks.language_code,
            raw_clicks.country_code,
            raw_clicks.region_code,
            0::int AS clicks,
            COUNT(*)::int AS conversions
        FROM raw_conversions
        JOIN raw_clicks ON raw_conversions.click_id = raw_clicks.id
        JOIN offer_urls ON raw_clicks.offer_id = offer_urls.offer_id AND raw_clicks.offer_url_id = offer_urls.id
        JOIN funnels ON offer_urls.funnel_id = funnels.id
        WHERE funnels.id = ANY(sqlc.arg('funnel_ids')::bigint[])
        AND raw_conversions.created_at BETWEEN sqlc.arg('date')::date + time '00:00:00' AND sqlc.arg('date')::date + time '23:59:59'
        GROUP BY
            slice_hour,
            raw_conversions.offer_id,
            raw_conversions.offer_url_id,
            raw_clicks.category_id,
            raw_clicks.advertiser_id,
            raw_clicks.affiliate_id,
            raw_clicks.traffic_source,
            raw_clicks.is_unique,
            raw_clicks.is_bot,
            raw_clicks.is_mobile,
            raw_clicks.is_tablet,
            raw_clicks.device_type,
            raw_clicks.language_code,
            raw_clicks.country_code,
            raw_clicks.region_code
        ORDER BY slice_hour, raw_conversions.offer_id, raw_conversions.offer_url_id ASC
    )
) as combined
GROUP BY slice_hour, offer_id, offer_url_id, category_id, advertiser_id, affiliate_id, traffic_source, is_unique, is_bot, is_mobile, is_tablet, device_type, language_code, country_code, region_code
ORDER BY slice_hour, offer_id, offer_url_id ASC;