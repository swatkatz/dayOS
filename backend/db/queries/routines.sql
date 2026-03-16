-- name: UpsertRoutine :one
INSERT INTO routines (id, user_id, title, category, frequency, days_of_week,
  preferred_time_of_day, preferred_duration_min, preferred_exact_time, notes, is_active)
VALUES (
  COALESCE(sqlc.narg('id')::UUID, gen_random_uuid()),
  @user_id,
  sqlc.arg('title'),
  sqlc.arg('category'),
  sqlc.arg('frequency'),
  sqlc.narg('days_of_week'),
  sqlc.narg('preferred_time_of_day'),
  sqlc.narg('preferred_duration_min'),
  sqlc.narg('preferred_exact_time'),
  sqlc.narg('notes'),
  COALESCE(sqlc.narg('is_active')::BOOLEAN, true)
)
ON CONFLICT (id) DO UPDATE SET
  title = EXCLUDED.title,
  category = EXCLUDED.category,
  frequency = EXCLUDED.frequency,
  days_of_week = EXCLUDED.days_of_week,
  preferred_time_of_day = EXCLUDED.preferred_time_of_day,
  preferred_duration_min = EXCLUDED.preferred_duration_min,
  preferred_exact_time = EXCLUDED.preferred_exact_time,
  notes = EXCLUDED.notes,
  is_active = EXCLUDED.is_active
RETURNING *;

-- name: GetRoutine :one
SELECT * FROM routines WHERE id = $1 AND user_id = @user_id;

-- name: ListRoutines :many
SELECT * FROM routines
WHERE user_id = @user_id
  AND (sqlc.narg('active_only')::BOOLEAN IS NULL OR sqlc.narg('active_only') = false OR is_active = true)
ORDER BY created_at;

-- name: DeleteRoutine :exec
DELETE FROM routines WHERE id = $1 AND user_id = @user_id;

-- name: ListRoutinesForDay :many
-- Used by planner: active routines that apply to a given day-of-week
SELECT * FROM routines
WHERE user_id = @user_id
  AND is_active = true
  AND (
    LOWER(frequency) = 'daily'
    OR (LOWER(frequency) = 'weekdays' AND @day_of_week::INT BETWEEN 1 AND 5)
    OR (LOWER(frequency) = 'weekly' AND @day_of_week::INT = ANY(days_of_week))
    OR (LOWER(frequency) = 'custom' AND @day_of_week::INT = ANY(days_of_week))
  )
ORDER BY preferred_exact_time NULLS LAST, preferred_time_of_day, title;
