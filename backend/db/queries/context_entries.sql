-- name: ListContextEntries :many
SELECT * FROM context_entries
WHERE (sqlc.narg('category')::TEXT IS NULL OR category = sqlc.narg('category'))
ORDER BY category, key;

-- name: GetContextEntry :one
SELECT * FROM context_entries WHERE id = $1;

-- name: UpsertContextEntry :one
INSERT INTO context_entries (category, key, value)
VALUES ($1, $2, $3)
ON CONFLICT (category, key) DO UPDATE SET
  value = EXCLUDED.value,
  updated_at = now()
RETURNING *;

-- name: ToggleContextEntry :one
UPDATE context_entries SET is_active = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteContextEntry :exec
DELETE FROM context_entries WHERE id = $1;

-- name: ListActiveContextEntries :many
SELECT * FROM context_entries
WHERE is_active = true
ORDER BY category, key;
