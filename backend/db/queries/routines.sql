-- name: UpsertRoutine :one
INSERT INTO routines (id, title, category, frequency, days_of_week,
  preferred_time_of_day, preferred_duration_min, notes, is_active)
VALUES (
  COALESCE(sqlc.narg('id')::UUID, gen_random_uuid()),
  sqlc.arg('title'),
  sqlc.arg('category'),
  sqlc.arg('frequency'),
  sqlc.narg('days_of_week'),
  sqlc.narg('preferred_time_of_day'),
  sqlc.narg('preferred_duration_min'),
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
  notes = EXCLUDED.notes,
  is_active = EXCLUDED.is_active
RETURNING *;

-- name: GetRoutine :one
SELECT * FROM routines WHERE id = $1;

-- name: ListRoutines :many
SELECT * FROM routines
WHERE (sqlc.narg('active_only')::BOOLEAN IS NULL OR sqlc.narg('active_only') = false OR is_active = true)
ORDER BY created_at;

-- name: DeleteRoutine :exec
DELETE FROM routines WHERE id = $1;

-- name: ListRoutinesForDay :many
-- Used by planner: active routines that apply to a given day-of-week
SELECT * FROM routines
WHERE is_active = true
  AND (
    frequency = 'daily'
    OR (frequency = 'weekdays' AND $1::INT BETWEEN 1 AND 5)
    OR (frequency = 'weekly' AND $1::INT = ANY(days_of_week))
    OR (frequency = 'custom' AND $1::INT = ANY(days_of_week))
  )
ORDER BY preferred_time_of_day, title;
