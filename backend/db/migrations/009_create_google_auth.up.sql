CREATE TABLE google_auth (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  access_token   TEXT NOT NULL,
  refresh_token  TEXT NOT NULL,
  token_expiry   TIMESTAMPTZ NOT NULL,
  calendar_id    TEXT NOT NULL DEFAULT 'primary',
  created_at     TIMESTAMPTZ DEFAULT now(),
  updated_at     TIMESTAMPTZ DEFAULT now()
);

CREATE UNIQUE INDEX idx_google_auth_singleton ON google_auth ((true));
