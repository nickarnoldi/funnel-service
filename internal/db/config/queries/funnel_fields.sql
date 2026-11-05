-- name: GetFunnelFieldsByFunnelID :many
SELECT * FROM funnel_fields
WHERE funnel_id = $1;

-- name: CreateFunnelField :one
INSERT INTO funnel_fields (funnel_id, funnel_step_id, name, do_not_save, is_required, is_honeypot)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateFunnelField :one
UPDATE funnel_fields SET
    funnel_step_id = $2,
    name = $3,
    do_not_save = $4,
    is_required = $5,
    is_honeypot = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING *;