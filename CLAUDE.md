# DayOS — Claude Code Instructions

## What this is
Personal daily planning app. Single user (Swati). Go + GraphQL + PostgreSQL backend,
React + TypeScript frontend, deployed on Railway.

## Key commands
- `make dev` — start backend + frontend in dev mode
- `make generate` — run gqlgen code generation
- `make migrate` — run database migrations
- `make build` — build frontend + embed into Go binary

## Architecture rules
- All AI calls go through `backend/planner/planner.go` — nowhere else
- Use sqlc for database queries (not raw SQL in resolvers)
- GraphQL resolvers should be thin — business logic in service layer
- Frontend uses Apollo Client with codegen types
- Never hardcode the Anthropic model — use ANTHROPIC_MODEL env var (default: claude-sonnet-4-6)

## Data model rules
- UUIDs everywhere (gen_random_uuid())
- All timestamps in UTC
- Soft-delete not used — just delete (plans are not precious)
- `blocks` in day_plans stored as JSONB (flexibility for schema evolution)

## Planner rules
- Always pull ALL active context_entries into every prompt
- Routines take priority over backlog tasks in scheduling
- Never schedule anything in a past time slot when replanning
- If Claude API returns unparseable JSON, retry once with stricter prompt before erroring

## Frontend rules
- No component library — Tailwind only
- Apollo Client for all data fetching (no REST calls)
- Dark theme: background #0f0f11, text #e8e6e1, accent #c5a55a
- Category colors: job=#6366f1, interview=#0ea5e9, project=#8b5cf6,
  meal=#10b981, baby=#f59e0b, exercise=#ef4444, admin=#6b7280

## Specs

All specs live in `specs/`. Each is a bounded context spec following the template in
`docs/DESIGN.md`. Read the relevant spec before implementing any context.

Use `/write-spec <context-name>` to generate a spec.

### Spec list (build order)

**Wave 1 — Foundation**
1. `specs/foundation.md` — DB migrations (7 total), Go module + gqlgen + sqlc setup

**Wave 2 — Backend Core**
2. `specs/tasks.md` — Task CRUD, parent/subtask hierarchy, completion, deferred tracking
3. `specs/routines.md` — Routine CRUD, day-of-week applicability
4. `specs/context.md` — Context entry CRUD, active/inactive toggle

**Wave 3 — Plans + AI**
5. `specs/day-plans.md` — Plan storage, block skip/adjust, plan status (draft/accepted)
6. `specs/planner.md` — AI plan chat, task scoping chat, prompt construction, JSON parsing
7. `specs/carry-over.md` — Skipped task review, deferred counting, actual_minutes computation

**Wave 4 — Frontend**
8. `specs/frontend-shell.md` — App shell, routing, Apollo + Tailwind setup, dark theme, auth
9. `specs/frontend-today.md` — Today page, plan chat interface, block view, skip/adjust, replan
10. `specs/frontend-backlog.md` — Task backlog, parent/subtask grouping, "Scope with AI" chat
11. `specs/frontend-manage.md` — Routines + Context + History pages

**Wave 5 — Deployment**
12. `specs/deployment.md` — Railway config, embedded frontend, env vars, simple auth middleware

### Spec workflow
- Write specs first, then implement. Never implement without a spec.
- Read `docs/DESIGN.md` + the relevant spec before coding any context.
- Cross-reference dependent specs when a context depends on another.

