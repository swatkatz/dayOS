-- name: UpsertUserByClerkID :one
INSERT INTO users (clerk_id, email, display_name)
VALUES (@clerk_id, @email, @display_name)
ON CONFLICT (clerk_id) DO UPDATE SET
  email = COALESCE(NULLIF(EXCLUDED.email, ''), users.email),
  display_name = COALESCE(NULLIF(EXCLUDED.display_name, ''), users.display_name),
  updated_at = now()
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: IncrementAICallsToday :one
UPDATE users SET
  ai_calls_today = CASE
    WHEN ai_calls_date = CURRENT_DATE THEN ai_calls_today + 1
    ELSE 1
  END,
  ai_calls_date = CURRENT_DATE,
  updated_at = now()
WHERE id = $1
RETURNING ai_calls_today;

-- name: GetAICallsToday :one
SELECT CASE WHEN ai_calls_date = CURRENT_DATE THEN ai_calls_today ELSE 0 END AS calls_today
FROM users WHERE id = $1;
