-- name: CreateTask :one
INSERT INTO tasks (user_id, title, category, priority, parent_id, estimated_minutes,
  deadline_type, deadline_date, deadline_days, notes, is_routine, routine_id)
VALUES (@user_id, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = $1 AND user_id = @user_id;

-- name: ListTasks :many
SELECT * FROM tasks
WHERE user_id = @user_id
  AND (sqlc.narg('category')::TEXT IS NULL OR category = sqlc.narg('category'))
  AND (sqlc.narg('include_completed')::BOOLEAN = true OR is_completed = false)
ORDER BY
  CASE priority WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 END,
  created_at DESC;

-- name: ListSubtasks :many
SELECT * FROM tasks WHERE parent_id = $1 AND user_id = @user_id ORDER BY created_at;

-- name: UpdateTask :one
UPDATE tasks SET
  title = COALESCE(sqlc.narg('title'), title),
  category = COALESCE(sqlc.narg('category'), category),
  priority = COALESCE(sqlc.narg('priority'), priority),
  estimated_minutes = COALESCE(sqlc.narg('estimated_minutes'), estimated_minutes),
  deadline_type = COALESCE(sqlc.narg('deadline_type'), deadline_type),
  deadline_date = COALESCE(sqlc.narg('deadline_date'), deadline_date),
  deadline_days = COALESCE(sqlc.narg('deadline_days'), deadline_days),
  notes = COALESCE(sqlc.narg('notes'), notes),
  routine_id = COALESCE(sqlc.narg('routine_id'), routine_id),
  updated_at = now()
WHERE id = $1 AND user_id = @user_id
RETURNING *;

-- name: CompleteTask :one
UPDATE tasks SET is_completed = true, completed_at = now(), updated_at = now()
WHERE id = $1 AND user_id = @user_id
RETURNING *;

-- name: UncompleteTask :one
UPDATE tasks SET is_completed = false, completed_at = NULL, updated_at = now()
WHERE id = $1 AND user_id = @user_id
RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = $1 AND user_id = @user_id;

-- name: CountIncompleteSubtasks :one
SELECT COUNT(*) FROM tasks WHERE parent_id = $1 AND user_id = @user_id AND is_completed = false;

-- name: ListSchedulableTasks :many
-- Used by planner: subtasks OR standalone tasks, incomplete only
SELECT * FROM tasks
WHERE user_id = @user_id
  AND is_completed = false
  AND (parent_id IS NOT NULL OR (parent_id IS NULL AND estimated_minutes IS NOT NULL))
ORDER BY
  CASE priority WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 END,
  deadline_date ASC NULLS LAST,
  times_deferred DESC,
  created_at;

-- name: IncrementTimesDeferred :one
UPDATE tasks SET times_deferred = times_deferred + 1, last_deferred_at = now(), updated_at = now()
WHERE id = $1 AND user_id = @user_id
RETURNING *;

-- name: UpdateActualMinutes :exec
UPDATE tasks SET actual_minutes = actual_minutes + $2, updated_at = now()
WHERE id = $1 AND user_id = @user_id;
