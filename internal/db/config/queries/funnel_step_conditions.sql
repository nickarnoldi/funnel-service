-- name: GetFunnelStepConditionsByFunnelID :many
SELECT funnel_step_conditions.* FROM funnel_step_conditions
JOIN funnel_steps ON funnel_step_conditions.funnel_step_id = funnel_steps.id
WHERE funnel_steps.funnel_id = $1;

-- name: CreateFunnelStepCondition :one
INSERT INTO funnel_step_conditions (funnel_step_id, field, operator, condition)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateFunnelStepCondition :one
UPDATE funnel_step_conditions
SET
    funnel_step_id = $2,
    field = $3,
    operator = $4,
    condition = $5,
    updated_at = NOW()
WHERE id = $1
RETURNING *;
