CREATE TABLE users (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  clerk_id          TEXT NOT NULL UNIQUE,
  email             TEXT NOT NULL,
  display_name      TEXT,
  anthropic_api_key TEXT,
  daily_ai_cap      INT NOT NULL DEFAULT 50,
  ai_calls_today    INT NOT NULL DEFAULT 0,
  ai_calls_date     DATE,
  created_at        TIMESTAMPTZ DEFAULT now(),
  updated_at        TIMESTAMPTZ DEFAULT now()
);
