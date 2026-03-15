-- name: GetGoogleAuth :one
SELECT * FROM google_auth LIMIT 1;

-- name: UpsertGoogleAuth :one
INSERT INTO google_auth (access_token, refresh_token, token_expiry, calendar_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT ((true)) DO UPDATE SET
  access_token = EXCLUDED.access_token,
  refresh_token = EXCLUDED.refresh_token,
  token_expiry = EXCLUDED.token_expiry,
  calendar_id = EXCLUDED.calendar_id,
  updated_at = now()
RETURNING *;

-- name: DeleteGoogleAuth :exec
DELETE FROM google_auth;
