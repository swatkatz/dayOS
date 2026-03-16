-- Restore old unique constraints
ALTER TABLE day_plans DROP CONSTRAINT IF EXISTS day_plans_user_plan_date_key;
ALTER TABLE day_plans ADD CONSTRAINT day_plans_plan_date_key UNIQUE (plan_date);

DROP INDEX IF EXISTS idx_context_entries_user_category_key;
CREATE UNIQUE INDEX idx_context_entries_category_key ON context_entries(category, key);

ALTER TABLE google_auth DROP CONSTRAINT IF EXISTS google_auth_user_id_key;
CREATE UNIQUE INDEX idx_google_auth_singleton ON google_auth ((true));

-- Drop indexes
DROP INDEX IF EXISTS idx_routines_user_id;
DROP INDEX IF EXISTS idx_tasks_user_id;
DROP INDEX IF EXISTS idx_context_entries_user_id;
DROP INDEX IF EXISTS idx_day_plans_user_id;
DROP INDEX IF EXISTS idx_task_conversations_user_id;

-- Drop user_id columns
ALTER TABLE routines DROP COLUMN IF EXISTS user_id;
ALTER TABLE tasks DROP COLUMN IF EXISTS user_id;
ALTER TABLE context_entries DROP COLUMN IF EXISTS user_id;
ALTER TABLE day_plans DROP COLUMN IF EXISTS user_id;
ALTER TABLE task_conversations DROP COLUMN IF EXISTS user_id;
ALTER TABLE google_auth DROP COLUMN IF EXISTS user_id;

-- Remove owner user
DELETE FROM users WHERE id = '00000000-0000-0000-0000-000000000001';
