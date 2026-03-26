ALTER TABLE day_plans
  ADD COLUMN previous_blocks  JSONB,
  ADD COLUMN previous_status  TEXT;
