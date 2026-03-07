# Spec: Foundation

## Bounded Context

Owns: All database migrations (7 files), Go module initialization, gqlgen setup + config, sqlc setup + config, project skeleton (directory structure, Makefile, tooling config), `main.go` entry point

Does not own: Resolver implementations (owned by tasks, routines, context, day-plans, planner, carry-over specs), planner service logic, frontend code, Railway deployment config (see `deployment` spec)

Depends on: Nothing — this is the root context

Produces: Database schema (all tables), Go module with gqlgen scaffolding, sqlc query infrastructure, Makefile with dev commands, runnable server with GraphQL endpoint

## Contracts

### Data Model

Seven migration files in `backend/db/migrations/`. Order matters — foreign keys reference earlier tables.

**001_create_routines.up.sql**

```sql
CREATE TABLE routines (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title       TEXT NOT NULL,
  category    TEXT NOT NULL,
  frequency   TEXT NOT NULL,
  days_of_week  INT[],
  preferred_time_of_day  TEXT,
  preferred_duration_min INT,
  notes       TEXT,
  is_active   BOOLEAN DEFAULT true,
  created_at  TIMESTAMPTZ DEFAULT now()
);
```

```sql
-- 001_create_routines.down.sql
DROP TABLE IF EXISTS routines;
```

**002_create_tasks.up.sql**

```sql
CREATE TABLE tasks (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title       TEXT NOT NULL,
  category    TEXT NOT NULL,
  priority    TEXT NOT NULL,
  parent_id   UUID REFERENCES tasks(id) ON DELETE CASCADE,
  estimated_minutes  INT,
  actual_minutes     INT DEFAULT 0,
  deadline_type   TEXT,
  deadline_date   DATE,
  deadline_days   INT,
  notes       TEXT,
  is_routine  BOOLEAN DEFAULT false,
  routine_id  UUID REFERENCES routines(id) ON DELETE SET NULL,
  times_deferred   INT DEFAULT 0,
  last_deferred_at TIMESTAMPTZ,
  is_completed   BOOLEAN DEFAULT false,
  completed_at   TIMESTAMPTZ,
  created_at     TIMESTAMPTZ DEFAULT now(),
  updated_at     TIMESTAMPTZ DEFAULT now()
);
```

```sql
-- 002_create_tasks.down.sql
DROP TABLE IF EXISTS tasks;
```

**003_create_context_entries.up.sql**

```sql
CREATE TABLE context_entries (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  category    TEXT NOT NULL,
  key         TEXT NOT NULL,
  value       TEXT NOT NULL,
  is_active   BOOLEAN DEFAULT true,
  created_at  TIMESTAMPTZ DEFAULT now(),
  updated_at  TIMESTAMPTZ DEFAULT now()
);
```

```sql
-- 003_create_context_entries.down.sql
DROP TABLE IF EXISTS context_entries;
```

**004_create_day_plans.up.sql**

```sql
CREATE TABLE day_plans (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  plan_date   DATE NOT NULL UNIQUE,
  status      TEXT NOT NULL DEFAULT 'draft',
  blocks      JSONB NOT NULL,
  created_at  TIMESTAMPTZ DEFAULT now(),
  updated_at  TIMESTAMPTZ DEFAULT now()
);
```

```sql
-- 004_create_day_plans.down.sql
DROP TABLE IF EXISTS day_plans;
```

**005_create_plan_messages.up.sql**

```sql
CREATE TABLE plan_messages (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  plan_id     UUID NOT NULL REFERENCES day_plans(id) ON DELETE CASCADE,
  role        TEXT NOT NULL,
  content     TEXT NOT NULL,
  created_at  TIMESTAMPTZ DEFAULT now()
);
```

```sql
-- 005_create_plan_messages.down.sql
DROP TABLE IF EXISTS plan_messages;
```

**006_create_task_conversations.up.sql**

```sql
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
```

```sql
-- 006_create_task_conversations.down.sql
DROP TABLE IF EXISTS task_messages;
DROP TABLE IF EXISTS task_conversations;
```

**007_seed_context.up.sql**

Optional convenience seed. Context entries can also be added via the `/context` UI page.

```sql
INSERT INTO context_entries (category, key, value) VALUES
  ('life', 'baby', '6-month-old daughter. Nanny present 9am-4pm Mon-Fri. Best quality time after naps (~10am, ~2pm).'),
  ('life', 'family', 'Partner Elijah. Family of household. Cooking Indian food regularly.'),
  ('constraints', 'work_window', 'Deep focus work: 9am-4pm only (nanny present). Before 9 and after 4 = family time.'),
  ('constraints', 'location', 'Toronto, Canada.'),
  ('constraints', 'energy', 'New parent. Cap deep cognitive work at 5h/day max. Hardest tasks go first in the morning.'),
  ('constraints', 'dinner_prep', 'Dinner is always prepped after 16:00. This is a fixed daily block, not optional. Duration ~45-60 min.'),
  ('constraints', 'evening_window', 'Baby typically asleep by ~8pm. After cleanup, there is a light evening window (~20:00-22:00). This time is primarily for rest and unwinding — do NOT schedule deep focus work here. Light tasks are acceptable: reading, light job search browsing, low-effort admin, meal planning, watching a talk. Max one light task per evening; relaxation comes first.'),
  ('equipment', 'kitchen', 'Full Indian kitchen setup. All standard utensils. Pressure cooker, tawa, etc.'),
  ('preferences', 'planning_style', 'Time-blocked days. Buffers between intense sessions. Interview prep is non-negotiable daily.');
```

```sql
-- 007_seed_context.down.sql
DELETE FROM context_entries WHERE key IN (
  'baby', 'family', 'work_window', 'location', 'energy',
  'dinner_prep', 'evening_window', 'kitchen', 'planning_style'
);
```

## Project Skeleton

### Go Module

Initialize at `backend/`:

```
go mod init dayos
```

Key dependencies:
- `github.com/99designs/gqlgen` — GraphQL code generation
- `github.com/vektah/gqlparser/v2` — GraphQL parsing (gqlgen dependency)
- `github.com/jackc/pgx/v5` — PostgreSQL driver
- `github.com/golang-migrate/migrate/v4` — database migrations
- `github.com/google/uuid` — UUID handling

### gqlgen Config

File: `backend/gqlgen.yml`

```yaml
schema:
  - graph/schema.graphqls

exec:
  filename: graph/generated.go
  package: graph

model:
  filename: graph/model/models_gen.go
  package: model

resolver:
  layout: follow-schema
  dir: graph
  package: graph
  filename_template: "{name}.resolvers.go"

models:
  UUID:
    model:
      - github.com/google/uuid.UUID
  Date:
    model:
      - dayos/graph/model.Date
  DateTime:
    model:
      - dayos/graph/model.DateTime
```

Custom scalar types defined in `backend/graph/model/scalars.go`:
- `Date` — marshals to/from `"YYYY-MM-DD"` string, backed by `time.Time`
- `DateTime` — marshals to/from RFC3339 string, backed by `time.Time`

### sqlc Config

File: `backend/sqlc.yaml`

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "db/queries/"
    schema: "db/migrations/"
    gen:
      go:
        package: "db"
        out: "db"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: false
        emit_exact_table_names: false
```

Query files are stubs at this stage — each context spec defines its own queries.

### GraphQL Schema

File: `backend/graph/schema.graphqls` — the full schema from DESIGN.md (scalars, enums, types, inputs, Query, Mutation). Placed during foundation setup; resolvers are stubbed by gqlgen and implemented by later specs.

### Makefile

```makefile
.PHONY: dev generate migrate build

dev:
	cd backend && go run ./main.go

generate:
	cd backend && go run github.com/99designs/gqlgen generate
	cd backend && sqlc generate

migrate:
	cd backend && migrate -path db/migrations -database "$$DATABASE_URL" up

build:
	cd frontend && npm run build
	cd backend && go build -o dayos ./main.go
```

### main.go

File: `backend/main.go`

Responsibilities:
1. Read `DATABASE_URL` and `PORT` from environment (`PORT` defaults to `8080`)
2. Run migrations on startup via golang-migrate
3. Initialize pgx connection pool
4. Initialize gqlgen handler at `/graphql`
5. Serve GraphQL Playground at `/` in dev mode
6. Start HTTP server on `PORT`

Frontend embedding and auth middleware are NOT part of this spec (see `deployment` spec).

## Behaviors (EARS syntax)

- When the application starts, the system shall run all pending database migrations before accepting HTTP requests.
- When a migration fails, the system shall log the error and exit with a non-zero status code.
- When `make generate` is run, the system shall regenerate both gqlgen Go code and sqlc Go code.
- When `make migrate` is run, the system shall apply all pending migrations to the database at `DATABASE_URL`.
- The system shall use UUIDs (generated by `gen_random_uuid()`) as primary keys for all tables.
- The system shall store all timestamps as `TIMESTAMPTZ` (UTC).
- When inserting a `day_plan` with a `plan_date` that already exists, the system shall reject the insert with a unique constraint violation.
- When a parent task is deleted, the system shall cascade-delete all its subtasks (via `ON DELETE CASCADE` on `parent_id`).
- When a routine is deleted, the system shall set `routine_id` to NULL on any linked tasks (via `ON DELETE SET NULL`).
- When a `day_plan` is deleted, the system shall cascade-delete all associated `plan_messages`.
- When a `task_conversation` is deleted, the system shall cascade-delete all associated `task_messages`.
- When `DATABASE_URL` is not set, the system shall exit immediately with a clear error message.

## Test Anchors

1. Given a fresh database, when all 7 migrations are applied, then all tables (`routines`, `tasks`, `context_entries`, `day_plans`, `plan_messages`, `task_conversations`, `task_messages`) exist and accept inserts.

2. Given migration 007 has run, when querying `context_entries`, then exactly 9 seed rows are returned with the correct category/key/value combinations.

3. Given a task with subtasks exists, when the parent task is deleted, then all subtasks are also deleted (CASCADE).

4. Given a routine linked to tasks exists, when the routine is deleted, then the tasks' `routine_id` is set to NULL and the tasks still exist.

5. Given a `day_plan` exists for `2026-03-05`, when inserting another `day_plan` for `2026-03-05`, then the insert fails with a unique constraint violation.

6. Given the Go module is initialized, when `make generate` is run, then `graph/generated.go` and `graph/model/models_gen.go` are produced without errors.

7. Given the server starts with a valid `DATABASE_URL`, when a POST to `/graphql` with an introspection query is sent, then a valid GraphQL schema response is returned.
