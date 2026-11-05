-- name: CreateAbandonedLead :one
INSERT INTO abandoned_leads (
    offer_id,
    offer_url_id,
    click_id,
    step_number,
    ip_address,
    transaction_id,
    affiliate_id,
    campaign_id,
    phone_number,
    email,
    jornaya_id,
    trustedform_id,
    is_bot,
    is_mobile,
    country_code,
    region_code,
    city,
    postal_code,
    date_of_birth,
    employment_status,
    gross_monthly_income,
    anura_score,
    honeypot_filled,
    session_length,
    landing_page_url,
    referrer,
    cookies,
    body
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28)
RETURNING *;

-- name: UpdateAbandonedLead :one
UPDATE abandoned_leads
SET
    offer_id = $2,
    offer_url_id = $3,
    click_id = $4,
    step_number = $5,
    ip_address = $6,
    transaction_id = $7,
    affiliate_id = $8,
    campaign_id = $9,
    phone_number = $10,
    email = $11,
    jornaya_id = $12,
    trustedform_id = $13,
    is_bot = $14,
    is_mobile = $15,
    country_code = $16,
    region_code = $17,
    city = $18,
    postal_code = $19,
    date_of_birth = $20,
    employment_status = $21,
    gross_monthly_income = $22,
    anura_score = $23,
    honeypot_filled = $24,
    session_length = $25,
    landing_page_url = $26,
    referrer = $27,
    cookies = $28,
    body = $29
WHERE id = $1
RETURNING *;

-- name: ListAbandonedLeads :many
SELECT * FROM abandoned_leads;

-- name: GetAbandonedLeadByID :one
SELECT * FROM abandoned_leads WHERE id = $1 LIMIT 1;

-- name: CountAbandonedLeadsByFunnelID :one
SELECT COUNT(*)::int FROM abandoned_leads
JOIN offer_urls ON abandoned_leads.offer_url_id = offer_urls.id
JOIN funnels ON offer_urls.funnel_id = funnels.id
WHERE funnels.id = ANY(sqlc.arg('funnel_ids')::bigint[]);

-- name: GetAbandonedLeadsByFunnelID :many
SELECT abandoned_leads.* FROM abandoned_leads
JOIN offer_urls ON abandoned_leads.offer_url_id = offer_urls.id
JOIN funnels ON offer_urls.funnel_id = funnels.id
WHERE funnels.id = ANY(sqlc.arg('funnel_ids')::bigint[])
AND abandoned_leads.id > COALESCE(sqlc.narg('after_id')::bigint, 0)
ORDER BY abandoned_leads.id ASC
LIMIT COALESCE(sqlc.narg('limit')::int, 100);

-- name: GetRecentAbandonedLeadBodysByFunnelID :many
SELECT abandoned_leads.body
FROM abandoned_leads
JOIN offer_urls ON abandoned_leads.offer_url_id = offer_urls.id
JOIN funnels ON offer_urls.funnel_id = funnels.id
WHERE funnels.id = $1
AND abandoned_leads.created_at <= COALESCE(sqlc.narg('created_at'), NOW())
ORDER BY abandoned_leads.created_at DESC
LIMIT 10;

-- name: GetAbandonedLeadsByDateRange :many
SELECT * FROM abandoned_leads
WHERE created_at BETWEEN sqlc.arg('from') AND sqlc.arg('to')
ORDER BY created_at ASC
OFFSET CASE WHEN COALESCE(sqlc.narg('offset')::int, 0) > 0 THEN sqlc.narg('offset')::int ELSE 0 END
LIMIT COALESCE(sqlc.narg('limit')::int, 100);

-- name: GetAbandonedLeadsMissingClicksByDateRange :many
SELECT abandoned_leads.*, offers.network_id FROM abandoned_leads
LEFT JOIN offers ON abandoned_leads.offer_id = offers.id
WHERE abandoned_leads.click_id IS NULL
AND abandoned_leads.created_at BETWEEN sqlc.arg('from')::timestamp AND sqlc.arg('to')::timestamp
ORDER BY abandoned_leads.created_at ASC
OFFSET CASE WHEN COALESCE(sqlc.narg('offset')::int, 0) > 0 THEN sqlc.narg('offset')::int ELSE 0 END
LIMIT COALESCE(sqlc.narg('limit')::int, 100);

-- name: UpdateAbandonedLeadClickID :one
UPDATE abandoned_leads
SET click_id = sqlc.arg('click_id')::bigint
WHERE id = $1
RETURNING *;

-- name: UpdateAbandonedLeadClickOfferAndOfferURL :one
UPDATE abandoned_leads
SET
    click_id = sqlc.arg('click_id')::bigint,
    offer_id = sqlc.arg('offer_id')::bigint,
    offer_url_id = sqlc.arg('offer_url_id')
WHERE id = $1
RETURNING *;

-- name: CountAbandonedLeadsMissingClicksByDateRange :one
SELECT COUNT(*)::int FROM abandoned_leads
WHERE click_id IS NULL
AND created_at BETWEEN sqlc.arg('from')::timestamp AND sqlc.arg('to')::timestamp;