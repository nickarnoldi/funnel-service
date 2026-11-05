-- name: GetAbandonedLeadFunnelStep :one
SELECT * FROM abandoned_lead_funnel_step
WHERE abandoned_lead_id = $1 AND funnel_step_id = $2
LIMIT 1;

-- name: GetAbandonedLeadFunnelStepByAbandonedLeadID :one
SELECT * FROM abandoned_lead_funnel_step
WHERE abandoned_lead_id = $1;

-- name: GetAbandonedLeadFunnelStepByID :one
SELECT * FROM abandoned_lead_funnel_step
WHERE id = $1
LIMIT 1;

-- name: CreateAbandonedLeadFunnelStep :one
INSERT INTO abandoned_lead_funnel_step (abandoned_lead_id, funnel_step_id)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateAbandonedLeadFunnelStep :one
UPDATE abandoned_lead_funnel_step
SET
    abandoned_lead_id = $2,
    funnel_step_id = $3
WHERE id = $1
RETURNING *;

-- name: DeleteAbandonedLeadFunnelStep :exec
DELETE FROM abandoned_lead_funnel_step
WHERE id = $1;
