# Implement: $ARGUMENTS

You are implementing a bounded context for DayOS from its spec. Follow these instructions exactly.

## Before you start

Read these files in order:
1. `CLAUDE.md` — project overview, conventions, current state
2. The assigned spec from `specs/` (identified by `$ARGUMENTS`, e.g., "foundation", "tasks", "frontend-shell")
3. `docs/DESIGN.md` — for domain context and GraphQL schema referenced by the spec
4. Any specs listed in the "Depends on" section of your spec

## File placement

### Backend contexts

- Main code: `backend/` following the directory structure in DESIGN.md
- GraphQL schema: `backend/graph/schema.graphqls`
- GraphQL resolvers: `backend/graph/*.resolvers.go`
- Generated gqlgen code: `backend/graph/generated.go`, `backend/graph/model/models_gen.go`
- sqlc queries: `backend/db/queries/{context}.sql` (e.g., `tasks.sql`, `routines.sql`)
- sqlc generated code: `backend/db/`
- Migrations: `backend/db/migrations/` (numbered, already defined in foundation spec)
- Planner service: `backend/planner/planner.go`
- Tests: `*_test.go` in the same package as the code being tested

### Frontend contexts

- All code in `frontend/src/`
- Components: `frontend/src/components/`
- Pages: `frontend/src/pages/`
- GraphQL operations: `frontend/src/graphql/`
- Follow existing Vite + React + TypeScript project structure

## TDD workflow (backend contexts)

For each test anchor in the spec:

1. **Write the failing test first.** Translate the test anchor into a Go test. Use table-driven tests when there are 3+ scenarios.
2. **Write the minimum implementation to make it pass.**
3. **Verify the test passes:** `cd backend && go test ./...`
4. **Repeat** for the next test anchor.

Do NOT write all tests at once then implement. Do NOT write implementation without a test.

## Frontend workflow

1. Read the spec for components, routes, interactions, and visual requirements
2. Implement in the spec's section order
3. Verify the build passes: `cd frontend && npm run build`

## Implementation conventions

### Backend (Go)

- `context.Context` as the first parameter on all public functions
- Errors: return `error`, don't panic. Wrap with `fmt.Errorf("doing x: %w", err)`
- Use sqlc for all database queries — no raw SQL in resolvers
- Resolvers are thin: validate input, call sqlc queries or service functions, return result
- Business logic goes in service layer (e.g., `backend/planner/planner.go`), not resolvers
- Use `pgx/v5` for the database driver
- UUIDs everywhere, timestamps in UTC
- Never hardcode the Anthropic model — use `ANTHROPIC_MODEL` env var

### Frontend (React + TypeScript)

- No component library — Tailwind CSS only
- Apollo Client for all data fetching (no REST calls)
- Use codegen'd types from the GraphQL schema
- Dark theme: background `#0f0f11`, text `#e8e6e1`, accent `#c5a55a`
- Category colors: job=`#6366f1`, interview=`#0ea5e9`, project=`#8b5cf6`, meal=`#10b981`, baby=`#f59e0b`, exercise=`#ef4444`, admin=`#6b7280`

## Codegen

After any change to `schema.graphqls` or sqlc query files, run:

```
make generate
```

This regenerates both gqlgen Go code and sqlc Go code.

## What NOT to do

- Don't modify other bounded contexts' code or files they own
- Don't add features beyond what the spec defines
- Don't add Docker, CI, or integration test infrastructure
- Don't create README or documentation files unless the spec calls for it
- Don't over-engineer — keep it simple, this is a personal app
- Don't add error handling for scenarios that can't happen
- Don't add comments, docstrings, or type annotations to code you didn't write

## When you're done

### Backend contexts
1. Run all tests: `cd backend && go test ./...`
2. Run vet: `cd backend && go vet ./...`
3. Print a summary of what was implemented and any decisions made

### Frontend contexts
1. Run build: `cd frontend && npm run build`
2. Print a summary of what was implemented and any decisions made

### All contexts
- Do NOT commit or push. The user handles git operations.
