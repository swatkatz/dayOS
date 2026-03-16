-- name: GetDayPlanByDate :one
SELECT * FROM day_plans WHERE user_id = @user_id AND plan_date = $1;

-- name: GetDayPlanByID :one
SELECT * FROM day_plans WHERE id = $1 AND user_id = @user_id;

-- name: CreateDayPlan :one
INSERT INTO day_plans (user_id, plan_date, status, blocks)
VALUES (@user_id, $1, $2, $3)
RETURNING *;

-- name: UpdateDayPlanBlocks :one
UPDATE day_plans SET blocks = $2, updated_at = now()
WHERE id = $1 AND user_id = @user_id RETURNING *;

-- name: UpdateDayPlanStatus :one
UPDATE day_plans SET status = $2, updated_at = now()
WHERE id = $1 AND user_id = @user_id RETURNING *;

-- name: RecentPlans :many
SELECT * FROM day_plans WHERE user_id = @user_id ORDER BY plan_date DESC LIMIT $1;

-- name: GetPlanMessages :many
SELECT * FROM plan_messages WHERE plan_id = $1 ORDER BY created_at ASC;

-- name: CreatePlanMessage :one
INSERT INTO plan_messages (plan_id, role, content)
VALUES ($1, $2, $3)
RETURNING *;
