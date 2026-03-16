-- name: ListContextEntries :many
SELECT * FROM context_entries
WHERE user_id = @user_id
  AND (sqlc.narg('category')::TEXT IS NULL OR category = sqlc.narg('category'))
ORDER BY category, key;

-- name: GetContextEntry :one
SELECT * FROM context_entries WHERE id = $1 AND user_id = @user_id;

-- name: UpsertContextEntry :one
INSERT INTO context_entries (user_id, category, key, value)
VALUES (@user_id, $1, $2, $3)
ON CONFLICT (user_id, category, key) DO UPDATE SET
  value = EXCLUDED.value,
  updated_at = now()
RETURNING *;

-- name: ToggleContextEntry :one
UPDATE context_entries SET is_active = $2, updated_at = now()
WHERE id = $1 AND user_id = @user_id
RETURNING *;

-- name: DeleteContextEntry :exec
DELETE FROM context_entries WHERE id = $1 AND user_id = @user_id;

-- name: ListActiveContextEntries :many
SELECT * FROM context_entries
WHERE user_id = @user_id AND is_active = true
ORDER BY category, key;
