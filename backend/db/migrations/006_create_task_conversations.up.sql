CREATE TABLE task_conversations (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  parent_task_id  UUID REFERENCES tasks(id) ON DELETE CASCADE,
  status      TEXT NOT NULL DEFAULT 'active',
  created_at  TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE task_messages (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID NOT NULL REFERENCES task_conversations(id) ON DELETE CASCADE,
  role            TEXT NOT NULL,
  content         TEXT NOT NULL,
  created_at      TIMESTAMPTZ DEFAULT now()
);
