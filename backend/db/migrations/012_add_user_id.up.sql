-- Insert owner user row for data backfill.
-- OWNER_CLERK_ID should be set to the real Clerk user ID after first login.
-- Using a fixed UUID so all existing data can be assigned to this user.
INSERT INTO users (id, clerk_id, email)
VALUES ('00000000-0000-0000-0000-000000000001', 'pending_owner', 'owner@placeholder.local');

-- Add user_id columns (nullable initially for backfill)
ALTER TABLE routines ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE tasks ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE context_entries ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE day_plans ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE task_conversations ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE google_auth ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;

-- Backfill all existing rows to owner
UPDATE routines SET user_id = '00000000-0000-0000-0000-000000000001';
UPDATE tasks SET user_id = '00000000-0000-0000-0000-000000000001';
UPDATE context_entries SET user_id = '00000000-0000-0000-0000-000000000001';
UPDATE day_plans SET user_id = '00000000-0000-0000-0000-000000000001';
UPDATE task_conversations SET user_id = '00000000-0000-0000-0000-000000000001';
UPDATE google_auth SET user_id = '00000000-0000-0000-0000-000000000001';

-- Set NOT NULL after backfill
ALTER TABLE routines ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE tasks ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE context_entries ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE day_plans ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE task_conversations ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE google_auth ALTER COLUMN user_id SET NOT NULL;

-- Drop old unique constraints and create user-scoped ones
ALTER TABLE day_plans DROP CONSTRAINT day_plans_plan_date_key;
ALTER TABLE day_plans ADD CONSTRAINT day_plans_user_plan_date_key UNIQUE (user_id, plan_date);

DROP INDEX IF EXISTS idx_context_entries_category_key;
CREATE UNIQUE INDEX idx_context_entries_user_category_key ON context_entries(user_id, category, key);

DROP INDEX IF EXISTS idx_google_auth_singleton;
ALTER TABLE google_auth ADD CONSTRAINT google_auth_user_id_key UNIQUE (user_id);

-- Indexes for common query patterns
CREATE INDEX idx_routines_user_id ON routines(user_id);
CREATE INDEX idx_tasks_user_id ON tasks(user_id);
CREATE INDEX idx_context_entries_user_id ON context_entries(user_id);
CREATE INDEX idx_day_plans_user_id ON day_plans(user_id);
CREATE INDEX idx_task_conversations_user_id ON task_conversations(user_id);
