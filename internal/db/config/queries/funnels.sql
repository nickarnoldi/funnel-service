-- name: CreateFunnel :one
INSERT INTO funnels (name)
VALUES ($1)
RETURNING *;

-- name: GetAllFunnels :many
SELECT * FROM funnels;

-- name: GetFunnelByID :one
SELECT * FROM funnels
WHERE id = $1 LIMIT 1;

-- name: GetFunnelsByID :many
SELECT * FROM funnels
WHERE id = ANY($1::bigint[]);

-- name: ListFunnelsWithOfferCount :many
SELECT sqlc.embed(funnels), STRING_AGG(DISTINCT offers.name, ', ') offer_names, COUNT(offer_urls) offer_count FROM funnels
LEFT JOIN offer_urls ON offer_urls.funnel_id = funnels.id
LEFT JOIN offers ON offer_urls.offer_id = offers.id
GROUP BY funnels.id, offers.id;
