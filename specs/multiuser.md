# Spec: Multi-User

## Bounded Context

Owns: `users` table, `backend/identity/` package (context key + user struct), `backend/db/scoped.go` (`ScopedQueries` wrapper), `backend/auth/clerk.go` (Clerk JWT middleware), `backend/auth/seed.go` (new user onboarding seeds), `backend/planner/usercap.go` (`WithUserCap` decorator), user-scoped sqlc queries, `user_id` columns on all data tables, frontend Clerk integration

Does not own: Store interfaces in `graph/stores.go` (unchanged), resolver logic in `graph/schema.resolvers.go` (unchanged), planner prompt construction (owned by `planner` — only the hardcoded "Swati" name changes), GraphQL schema types (unchanged), validation rules (owned by `validation`)

Depends on:
- `foundation` — existing migration structure, sqlc + gqlgen setup
- `auth` — replaces the `RequireAuth` middleware (this spec supersedes `specs/auth.md`)
- `planner` — wraps `PlannerService` with `WithUserCap` decorator
- `calendar` — `google_auth` becomes per-user (singleton constraint removed)

Produces:
- Migrations 011–012 (users table, user_id columns)
- `identity.FromContext(ctx)` / `identity.WithUser(ctx, user)` for user propagation
- `db.ScopedQueries` satisfying all store interfaces with automatic user scoping
- `planner.WithUserCap` decorator satisfying `PlannerService` with cap + key fallback
- Clerk-based frontend auth (replaces `AuthGate`)

## Contracts

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `CLERK_SECRET_KEY` | Yes | Clerk backend secret key (`sk_live_...` or `sk_test_...`) |
| `VITE_CLERK_PUBLISHABLE_KEY` | Yes (frontend) | Clerk publishable key (`pk_live_...` or `pk_test_...`) |
| `OWNER_CLERK_ID` | Yes (migration 012 only) | Clerk user ID of the data owner for backfill |
| `DAILY_AI_CAP` | No | Max AI calls per user per day (default: 50) |
| `APP_SECRET` | Removed | No longer used — replaced by Clerk JWT |

### Data Model

#### `users` table (migration 011)

```sql
CREATE TABLE users (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  clerk_id          TEXT NOT NULL UNIQUE,
  email             TEXT NOT NULL,
  display_name      TEXT,
  anthropic_api_key TEXT,           -- nullable; if set, planner uses this key
  daily_ai_cap      INT NOT NULL DEFAULT 50,
  ai_calls_today    INT NOT NULL DEFAULT 0,
  ai_calls_date     DATE,           -- date of last AI call (for daily reset)
  created_at        TIMESTAMPTZ DEFAULT now(),
  updated_at        TIMESTAMPTZ DEFAULT now()
);
```

#### `user_id` columns (migration 012)

Added to: `routines`, `tasks`, `context_entries`, `day_plans`, `task_conversations`, `google_auth`

NOT added to: `plan_messages` (scoped via `plan_id → day_plans.user_id`), `task_messages` (scoped via `conversation_id → task_conversations.user_id`)

Constraint changes:
- `day_plans`: `UNIQUE(plan_date)` → `UNIQUE(user_id, plan_date)`
- `context_entries`: `UNIQUE(category, key)` → `UNIQUE(user_id, category, key)`
- `google_auth`: `UNIQUE INDEX ON ((true))` → `UNIQUE(user_id)`

Backfill strategy:
1. Read `OWNER_CLERK_ID` from a migration config variable or hardcode the owner's Clerk ID
2. Insert owner user row with a fixed UUID (`'00000000-0000-0000-0000-000000000001'`)
3. Add `user_id` columns as nullable
4. Backfill all existing rows with the owner UUID
5. Set `NOT NULL` constraint
6. Add indexes on `user_id` for each table

### sqlc Queries

All existing query files are modified (not duplicated) to add `user_id` scoping:

**Pattern for SELECT queries:**
```sql
-- Before:
SELECT * FROM routines WHERE id = $1;
-- After:
SELECT * FROM routines WHERE id = $1 AND user_id = @user_id;
```

**Pattern for INSERT queries:**
```sql
-- Before:
INSERT INTO routines (title, ...) VALUES ($1, ...);
-- After:
INSERT INTO routines (user_id, title, ...) VALUES (@user_id, $1, ...);
```

**Pattern for DELETE queries:**
```sql
-- Before:
DELETE FROM routines WHERE id = $1;
-- After:
DELETE FROM routines WHERE id = $1 AND user_id = @user_id;
```

**New file: `backend/db/queries/users.sql`**

```sql
-- name: UpsertUserByClerkID :one
INSERT INTO users (clerk_id, email, display_name)
VALUES (@clerk_id, @email, @display_name)
ON CONFLICT (clerk_id) DO UPDATE SET
  email = EXCLUDED.email,
  display_name = COALESCE(EXCLUDED.display_name, users.display_name),
  updated_at = now()
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: IncrementAICallsToday :one
UPDATE users SET
  ai_calls_today = CASE
    WHEN ai_calls_date = CURRENT_DATE THEN ai_calls_today + 1
    ELSE 1
  END,
  ai_calls_date = CURRENT_DATE,
  updated_at = now()
WHERE id = $1
RETURNING ai_calls_today;

-- name: GetAICallsToday :one
SELECT CASE WHEN ai_calls_date = CURRENT_DATE THEN ai_calls_today ELSE 0 END AS calls_today
FROM users WHERE id = $1;
```

**Files modified:**
- `backend/db/queries/routines.sql` — add `user_id` to all queries
- `backend/db/queries/tasks.sql` — add `user_id` to all queries
- `backend/db/queries/context_entries.sql` — add `user_id`, update conflict target to `(user_id, category, key)`
- `backend/db/queries/day_plans.sql` — add `user_id`, `GetDayPlanByDate` becomes `WHERE user_id = @user_id AND plan_date = $1`
- `backend/db/queries/task_conversations.sql` — add `user_id` to create + get queries
- `backend/db/queries/google_auth.sql` — replace singleton queries with user-scoped ones, `ON CONFLICT (user_id)`
- `backend/db/queries/carryover.sql` — add `user_id` to both queries

### Identity Package

File: `backend/identity/identity.go`

```go
type User struct {
    ID        pgtype.UUID
    ClerkID   string
    Email     string
    DisplayName string
    AnthropicAPIKey *string
}

func WithUser(ctx context.Context, u User) context.Context
func FromContext(ctx context.Context) (User, bool)
func MustFromContext(ctx context.Context) User  // panics if missing — use only after middleware
```

Follows the same `context.Value` pattern as `backend/tz/tz.go`.

### ScopedQueries

File: `backend/db/scoped.go`

```go
type ScopedQueries struct {
    q *Queries
}

func NewScopedQueries(q *Queries) *ScopedQueries
```

- Constructed once at startup, wired into `Resolver` as all store fields
- Each method reads `identity.MustFromContext(ctx)` to get the user ID
- Delegates to the underlying `*Queries` methods (which now include `user_id` params)
- Satisfies: `RoutineStore`, `TaskStore`, `ContextStore`, `DayPlanStore`, `TaskConversationStore`
- Also satisfies the calendar `GoogleAuthStore` (user-scoped google_auth queries)

### Auth Middleware

File: `backend/auth/clerk.go` (replaces `backend/auth/middleware.go`)

```go
func RequireClerk(queries *db.Queries) func(http.Handler) http.Handler
```

Flow:
1. Extract `Authorization: Bearer <jwt>` from request
2. Verify JWT via Clerk Go SDK (`clerk.VerifyToken()`) using `CLERK_SECRET_KEY`
3. Extract `sub` (Clerk user ID), `email` from JWT claims
4. Call `queries.UpsertUserByClerkID(ctx, ...)` — returns user row
5. Detect new user: `created_at ≈ updated_at` (within 1 second)
6. If new: call `SeedDefaultContextEntries(ctx, queries, user.ID)`
7. Call `identity.WithUser(ctx, user)` to inject user into context
8. Call `next.ServeHTTP(w, r.WithContext(ctx))`

On failure: return 401 `{"error": "unauthorized"}` (same as current behavior).

### New User Onboarding

File: `backend/auth/seed.go`

Seeds example context entries for new users. These are generic examples that encourage
customization, not Swati-specific data.

```go
var defaultContextEntries = []struct {
    Category string
    Key      string
    Value    string
}{
    {"constraints", "energy", "Set your energy and scheduling constraints here. Example: Cap deep cognitive work at 5h/day."},
    {"constraints", "work_window", "Set your available hours here. Example: Deep focus work 9am-5pm weekdays."},
    {"preferences", "planning_style", "Describe your preferred planning style. Example: Time-blocked days with buffers between sessions."},
}
```

### Planner Decorator

File: `backend/planner/usercap.go`

```go
type WithUserCap struct {
    inner     PlannerService
    store     UserCapStore
    systemKey string
}

func NewWithUserCap(inner PlannerService, store UserCapStore, systemKey string) *WithUserCap
```

Satisfies `PlannerService` (same interface as `planner.Service`).

For each `PlanChat` / `TaskChat` call:
1. Read user from `identity.MustFromContext(ctx)`
2. Check `store.GetAICallsToday(ctx, user.ID)` — if `>= user.DailyAICap`, return error
3. Resolve API key: if `user.AnthropicAPIKey != nil`, create a one-off client; else use system client
4. Call `inner.PlanChat(ctx, input)` or `inner.TaskChat(ctx, ...)`
5. Call `store.IncrementAICallsToday(ctx, user.ID)`
6. Return result

The `UserCapStore` interface:
```go
type UserCapStore interface {
    GetAICallsToday(ctx context.Context, userID pgtype.UUID) (int32, error)
    IncrementAICallsToday(ctx context.Context, userID pgtype.UUID) (int32, error)
    GetUserByID(ctx context.Context, id pgtype.UUID) (db.User, error)
}
```

### Planner Prompt Changes

The system prompts in `backend/planner/planner.go` currently hardcode "Swati". Change to use
the user's display name (or "the user" as fallback):

```
// Before:
"You are a daily planning assistant for Swati."
// After:
fmt.Sprintf("You are a daily planning assistant for %s.", userName)
```

Pass `userName` through `PlanChatInput` and `TaskChatInput` (new field). The resolver reads
`identity.FromContext(ctx).DisplayName` and passes it.

### Frontend Changes

**`frontend/src/main.tsx`** — wrap app with `<ClerkProvider>`:
```tsx
import { ClerkProvider } from '@clerk/clerk-react'
<ClerkProvider publishableKey={import.meta.env.VITE_CLERK_PUBLISHABLE_KEY}>
  <App />
</ClerkProvider>
```

**`frontend/src/App.tsx`** — replace manual auth state with Clerk:
```tsx
import { SignedIn, SignedOut, useAuth } from '@clerk/clerk-react'

export default function App() {
  const { getToken } = useAuth()
  useEffect(() => {
    setTokenGetter(() => getToken())
  }, [getToken])

  return (
    <>
      <SignedOut><SignInPage /></SignedOut>
      <SignedIn>
        <ApolloProvider client={client}>
          <BrowserRouter>...</BrowserRouter>
        </ApolloProvider>
      </SignedIn>
    </>
  )
}
```

**`frontend/src/apollo.ts`** — replace localStorage token with Clerk async getter:
```ts
let tokenGetter: (() => Promise<string | null>) | null = null
export function setTokenGetter(fn: () => Promise<string | null>) { tokenGetter = fn }

const authLink = new SetContextLink(async ({ headers }) => {
  const token = tokenGetter ? await tokenGetter() : null
  return {
    headers: {
      ...headers,
      ...(token ? { authorization: `Bearer ${token}` } : {}),
      'X-Timezone': Intl.DateTimeFormat().resolvedOptions().timeZone,
    },
  }
})
```

Remove: `getToken()`, `setToken()`, `clearToken()`, `TOKEN_KEY` constant.

**`frontend/src/components/SignInPage.tsx`** — new file, replaces `AuthGate.tsx`:
```tsx
import { SignIn } from '@clerk/clerk-react'

export default function SignInPage() {
  return (
    <div className="flex items-center justify-center min-h-screen bg-bg-primary">
      <div className="flex flex-col items-center gap-6">
        <div className="text-center">
          <div className="text-4xl mb-3">🗓️</div>
          <h1 className="text-xl font-semibold text-accent tracking-tight">DayOS</h1>
        </div>
        <SignIn routing="hash" />
      </div>
    </div>
  )
}
```

Clerk appearance customization to match dark theme:
```ts
appearance={{
  variables: {
    colorBackground: '#0f0f11',
    colorText: '#e8e6e1',
    colorPrimary: '#c5a55a',
    colorInputBackground: '#1a1a1e',
    borderRadius: '0.75rem',
  }
}}
```

**`frontend/src/components/AuthGate.tsx`** — deleted.

### main.go Changes

```go
// Remove:
appSecret := os.Getenv("APP_SECRET")

// Add:
clerkSecretKey := os.Getenv("CLERK_SECRET_KEY")
clerk.SetKey(clerkSecretKey)

// Stores: ScopedQueries instead of raw *db.Queries
scoped := db.NewScopedQueries(queries)

// Planner: WithUserCap decorator
basePlanner := planner.New(aiClient)
plannerSvc := planner.NewWithUserCap(basePlanner, queries, os.Getenv("ANTHROPIC_API_KEY"))

// Resolver wiring
cfg := graph.Config{
    Resolvers: &graph.Resolver{
        RoutineStore:          scoped,
        TaskStore:             scoped,
        ContextStore:          scoped,
        DayPlanStore:          scoped,
        TaskConversationStore: scoped,
        Planner:               plannerSvc,
        Calendar:              calendarSvc, // calendarSvc.Store also uses scoped
    },
}

// Middleware
authMiddleware := auth.RequireClerk(queries)
http.Handle("/graphql", corsMiddleware(authMiddleware(tz.Middleware(srv))))
```

### Calendar Service Changes

- `calendar.Service.Store` receives `ScopedQueries` (instead of raw `*db.Queries`)
- `GetGoogleAuth(ctx)` automatically scopes to user via `ScopedQueries`
- Cache key includes user ID: `fmt.Sprintf("cal:events:%s:%s", userID, date)`
- Remove `idx_google_auth_singleton` constraint, replace with `UNIQUE(user_id)`

## Files

### New files

| File | Description |
|------|-------------|
| `backend/db/migrations/011_create_users.up.sql` | Create `users` table |
| `backend/db/migrations/011_create_users.down.sql` | Drop `users` table |
| `backend/db/migrations/012_add_user_id.up.sql` | Add `user_id` to all tables, backfill, update constraints |
| `backend/db/migrations/012_add_user_id.down.sql` | Reverse user_id changes |
| `backend/db/queries/users.sql` | User CRUD + AI cap queries |
| `backend/db/scoped.go` | `ScopedQueries` — user-scoped store wrapper |
| `backend/db/scoped_test.go` | Tests for `ScopedQueries` |
| `backend/identity/identity.go` | User context key, `FromContext`, `WithUser` |
| `backend/identity/identity_test.go` | Context round-trip tests |
| `backend/auth/clerk.go` | Clerk JWT middleware |
| `backend/auth/clerk_test.go` | Clerk middleware tests |
| `backend/auth/seed.go` | Default context entry seeder for new users |
| `backend/planner/usercap.go` | `WithUserCap` planner decorator |
| `backend/planner/usercap_test.go` | Cap enforcement + key fallback tests |
| `frontend/src/components/SignInPage.tsx` | Clerk sign-in page |

### Modified files

| File | Change |
|------|--------|
| `backend/auth/middleware.go` | Delete `RequireAuth` (replaced by `clerk.go`) |
| `backend/main.go` | New env vars, `ScopedQueries`, `WithUserCap`, `RequireClerk` |
| `backend/db/queries/routines.sql` | Add `user_id` to all queries |
| `backend/db/queries/tasks.sql` | Add `user_id` to all queries |
| `backend/db/queries/context_entries.sql` | Add `user_id`, update conflict target |
| `backend/db/queries/day_plans.sql` | Add `user_id` to all queries |
| `backend/db/queries/task_conversations.sql` | Add `user_id` to create + get |
| `backend/db/queries/google_auth.sql` | User-scoped, remove singleton |
| `backend/db/queries/carryover.sql` | Add `user_id` to both queries |
| `backend/planner/planner.go` | Replace hardcoded "Swati" with user display name, add `UserName` to input structs |
| `backend/calendar/calendar.go` | Per-user cache key |
| `backend/graph/resolver.go` | No changes (stores remain same interfaces) |
| `backend/graph/stores.go` | No changes |
| `backend/graph/testutil_test.go` | Add test user context helper, update factories |
| `frontend/src/main.tsx` | Add `<ClerkProvider>` |
| `frontend/src/App.tsx` | Replace manual auth with Clerk hooks |
| `frontend/src/apollo.ts` | Async token getter, remove localStorage functions |

### Deleted files

| File | Reason |
|------|--------|
| `frontend/src/components/AuthGate.tsx` | Replaced by `SignInPage.tsx` with Clerk `<SignIn />` |

## Behaviors (EARS syntax)

### Authentication

- When a request arrives at `/graphql` without a valid Clerk JWT, the system shall respond with HTTP 401 `{"error": "unauthorized"}`.
- When a request arrives with a valid Clerk JWT, the system shall extract the user's Clerk ID and email, upsert the user row, inject the user into context, and forward to the GraphQL handler.
- When a request arrives with an expired Clerk JWT, the system shall respond with HTTP 401.
- The system shall use `CLERK_SECRET_KEY` to verify JWT signatures via the Clerk Go SDK.
- When `CLERK_SECRET_KEY` is not set, the system shall exit on startup with an error.

### New user onboarding

- When a user logs in for the first time (user row just created), the system shall seed default context entries for that user.
- When a user logs in subsequently, the system shall NOT re-seed context entries.
- The system shall detect new users by comparing `created_at` and `updated_at` on the upserted row (difference < 1 second = new).

### User-scoped data access

- When any store method is called via `ScopedQueries`, the system shall read the user ID from `context.Context` and include it in the sqlc query.
- When `ScopedQueries` is called without a user in context (should never happen after middleware), the system shall return an error.
- When `GetRoutine(ctx, id)` is called, the system shall only return the routine if it belongs to the authenticated user (defense in depth — UUIDs are unguessable but we scope anyway).
- When `GetDayPlanByDate(ctx, date)` is called, the system shall return only the authenticated user's plan for that date.
- When data is created (insert), the system shall set `user_id` to the authenticated user's ID.

### AI usage cap

- When a user calls `PlanChat` or `TaskChat` and their `ai_calls_today >= daily_ai_cap`, the system shall return an error `"Daily AI usage limit reached. Try again tomorrow."`.
- When a user's `ai_calls_date` is before today, the system shall reset `ai_calls_today` to 0 before checking the cap (lazy reset on next call, not a cron job).
- When a user has `anthropic_api_key` set, the system shall use that key for their AI calls.
- When a user has no `anthropic_api_key`, the system shall use the system `ANTHROPIC_API_KEY`.
- After a successful AI call, the system shall increment `ai_calls_today` and set `ai_calls_date` to today.

### Data migration

- Migration 012 shall insert the owner user row using `OWNER_CLERK_ID` env var (or a hardcoded constant) with UUID `'00000000-0000-0000-0000-000000000001'`.
- Migration 012 shall add `user_id` as nullable, backfill all existing rows with the owner UUID, then set `NOT NULL`.
- Migration 012 shall drop old unique constraints and create new user-scoped ones.
- Migration 012 shall add indexes on `user_id` for each table.

### Frontend auth

- When the user is not signed in, the system shall display the Clerk `<SignIn />` component (embedded, not redirect).
- When the user is signed in, the system shall mount the Apollo provider and render the app.
- The system shall inject the Clerk session token as `Authorization: Bearer <token>` on every Apollo request.
- When the backend returns 401, the system shall let Clerk handle re-authentication.

### Calendar per-user

- When `GetGoogleAuth(ctx)` is called via `ScopedQueries`, the system shall return only the authenticated user's Google auth credentials.
- When calendar events are cached, the cache key shall include the user ID.

## Decision Table: API Key Resolution

| User has `anthropic_api_key`? | System `ANTHROPIC_API_KEY` set? | Behavior |
|---|----|---|
| Yes | Any | Use user's key |
| No | Yes | Use system key |
| No | No | Error: no API key available |

## Decision Table: AI Cap Check

| `ai_calls_date` | `ai_calls_today` vs `daily_ai_cap` | Behavior |
|---|---|---|
| Today | Below cap | Allow, increment |
| Today | At or above cap | Reject with error |
| Before today | Any | Reset to 0, allow, increment |
| NULL | Any | Treat as new day, allow, increment |

## Build Sequence

### Phase 1 — Identity Package
- [ ] Create `backend/identity/identity.go`
- [ ] Create `backend/identity/identity_test.go`

### Phase 2 — Database Migrations
- [ ] Create migration 011 (users table)
- [ ] Create migration 012 (add user_id to all tables)
- [ ] Run `make migrate` to verify

### Phase 3 — sqlc Query Updates
- [ ] Create `backend/db/queries/users.sql`
- [ ] Update all existing query files to add `user_id`
- [ ] Run `make generate` to regenerate Go code

### Phase 4 — ScopedQueries
- [ ] Create `backend/db/scoped.go`
- [ ] Create `backend/db/scoped_test.go`
- [ ] Verify compile-time interface satisfaction

### Phase 5 — Auth Middleware
- [ ] Add `github.com/clerk/clerk-sdk-go/v2` dependency
- [ ] Create `backend/auth/clerk.go`
- [ ] Create `backend/auth/seed.go`
- [ ] Create `backend/auth/clerk_test.go`
- [ ] Delete `backend/auth/middleware.go` content (or replace entirely)

### Phase 6 — Planner Decorator
- [ ] Create `backend/planner/usercap.go`
- [ ] Create `backend/planner/usercap_test.go`
- [ ] Update `backend/planner/planner.go` — parameterize user name in prompts

### Phase 7 — Wire main.go
- [ ] Update `backend/main.go` — new env vars, ScopedQueries, WithUserCap, RequireClerk
- [ ] Update calendar service to use ScopedQueries as Store
- [ ] Update calendar cache key to include user ID

### Phase 8 — Frontend
- [ ] `npm install @clerk/clerk-react` in frontend
- [ ] Create `frontend/src/components/SignInPage.tsx`
- [ ] Update `frontend/src/main.tsx` — ClerkProvider
- [ ] Update `frontend/src/App.tsx` — Clerk auth state
- [ ] Update `frontend/src/apollo.ts` — async token getter
- [ ] Delete `frontend/src/components/AuthGate.tsx`

### Phase 9 — Test Updates
- [ ] Update `backend/graph/testutil_test.go` — add `testUserCtx()` helper
- [ ] Update all resolver tests to use user context
- [ ] Verify `make lint` passes

## Test Anchors

### Identity package

1. **Given** a context with a user set via `identity.WithUser`, **when** `identity.FromContext` is called, **then** it returns the user and `true`.

2. **Given** a bare `context.Background()`, **when** `identity.FromContext` is called, **then** it returns zero value and `false`.

3. **Given** a bare `context.Background()`, **when** `identity.MustFromContext` is called, **then** it panics.

### Auth middleware

4. **Given** `CLERK_SECRET_KEY` is configured, **when** a request arrives with a valid Clerk JWT for a new user, **then** the user row is created, default context entries are seeded, and the request proceeds with the user in context.

5. **Given** `CLERK_SECRET_KEY` is configured, **when** a request arrives with a valid Clerk JWT for an existing user, **then** the user row is updated (email sync), no seeding occurs, and the request proceeds.

6. **Given** `CLERK_SECRET_KEY` is configured, **when** a request arrives with no `Authorization` header, **then** the response is HTTP 401 `{"error": "unauthorized"}`.

7. **Given** `CLERK_SECRET_KEY` is configured, **when** a request arrives with an invalid JWT, **then** the response is HTTP 401.

### ScopedQueries

8. **Given** a context with user A, **when** `ScopedQueries.ListRoutines(ctx, nil)` is called, **then** the underlying query includes `user_id = <user_A_id>` and only user A's routines are returned.

9. **Given** a context with user A, **when** `ScopedQueries.UpsertRoutine(ctx, params)` is called, **then** the created routine has `user_id = <user_A_id>`.

10. **Given** a context with user A and a routine owned by user B, **when** `ScopedQueries.GetRoutine(ctx, routineID)` is called, **then** no result is returned (scoped by user_id).

11. **Given** a context with no user, **when** any `ScopedQueries` method is called, **then** an error is returned.

12. **Given** a context with user A, **when** `ScopedQueries.GetDayPlanByDate(ctx, today)` is called, **then** only user A's plan for today is returned (not user B's plan for the same date).

### Planner decorator (WithUserCap)

13. **Given** a user with `ai_calls_today = 49` and `daily_ai_cap = 50`, **when** `WithUserCap.PlanChat` is called, **then** the inner planner is called, `ai_calls_today` is incremented to 50, and the result is returned.

14. **Given** a user with `ai_calls_today = 50` and `daily_ai_cap = 50`, **when** `WithUserCap.PlanChat` is called, **then** the inner planner is NOT called and an error `"Daily AI usage limit reached"` is returned.

15. **Given** a user with `ai_calls_date` = yesterday and `ai_calls_today = 50`, **when** `WithUserCap.PlanChat` is called, **then** the counter resets to 0, the inner planner is called, and `ai_calls_today` becomes 1.

16. **Given** a user with `anthropic_api_key = "sk-user-key"`, **when** `WithUserCap.PlanChat` is called, **then** the inner planner uses the user's API key (not the system key).

17. **Given** a user with `anthropic_api_key = NULL`, **when** `WithUserCap.PlanChat` is called, **then** the inner planner uses the system `ANTHROPIC_API_KEY`.

### New user seeding

18. **Given** a new user just created via `UpsertUserByClerkID`, **when** `SeedDefaultContextEntries` is called, **then** example context entries are created for that user.

19. **Given** `SeedDefaultContextEntries` is called for a user who already has context entries, **then** it upserts (no duplicates), and existing entries are NOT overwritten.

### Data migration

20. **Given** the database has existing routines, tasks, context entries, and day plans with no `user_id`, **when** migration 012 runs, **then** all rows have `user_id = '00000000-0000-0000-0000-000000000001'` and `NOT NULL` constraints are in place.

21. **Given** migration 012 has run, **when** two users each create a day plan for the same date, **then** both plans are stored (unique constraint is on `(user_id, plan_date)`, not just `plan_date`).

22. **Given** migration 012 has run, **when** two users each create a context entry with the same `(category, key)`, **then** both entries are stored.

### Frontend

23. **Given** no Clerk session exists, **when** the app loads, **then** the `<SignIn />` component is displayed with DayOS branding and dark theme.

24. **Given** a valid Clerk session exists, **when** the app loads, **then** Apollo is initialized with the Clerk token and the app renders normally.

25. **Given** a signed-in user, **when** a GraphQL request is made, **then** the `Authorization: Bearer <clerk_jwt>` header is included.

### Cross-user isolation (integration)

26. **Given** user A creates a routine and user B is signed in, **when** user B calls `routines(activeOnly: false)`, **then** user A's routine is NOT returned.

27. **Given** user A has a day plan for today, **when** user B calls `dayPlan(date: today)`, **then** `null` is returned (no plan for user B).
