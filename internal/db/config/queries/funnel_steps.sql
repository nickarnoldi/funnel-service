-- name: GetFunnelStepsByFunnelID :many
SELECT * FROM funnel_steps
WHERE funnel_id = $1
ORDER BY step_index ASC;

-- name: CreateFunnelStep :one
INSERT INTO funnel_steps (funnel_id, name, step_index)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateFunnelStep :one
UPDATE funnel_steps
SET
    name = $2,
    step_index = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;