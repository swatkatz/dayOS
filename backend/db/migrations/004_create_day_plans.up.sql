CREATE TABLE day_plans (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  plan_date   DATE NOT NULL UNIQUE,
  status      TEXT NOT NULL DEFAULT 'draft',
  blocks      JSONB NOT NULL,
  created_at  TIMESTAMPTZ DEFAULT now(),
  updated_at  TIMESTAMPTZ DEFAULT now()
);
