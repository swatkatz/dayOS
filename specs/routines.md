# Spec: Routines

## Bounded Context

Owns: Routine CRUD resolvers, day-of-week applicability logic, sqlc queries for routines

Does not own: `routines` table DDL (foundation), routine-to-task linking (tasks context writes `routine_id`), scheduling routines into plan blocks (planner)

Depends on: foundation (table + schema exist)

Produces: GraphQL queries (`routines`) and mutations (`createRoutine`, `updateRoutine`, `deleteRoutine`)

## Contracts

### Input

```graphql
input CreateRoutineInput {
  title:                String!
  category:             Category!
  frequency:            Frequency!       # DAILY | WEEKDAYS | WEEKLY | CUSTOM
  daysOfWeek:           [Int!]           # 0=Sun..6=Sat. Required when frequency=CUSTOM
  preferredTimeOfDay:   TimeOfDay        # MORNING | MIDDAY | AFTERNOON | EVENING | ANY
  preferredDurationMin: Int
  notes:                String
}

input UpdateRoutineInput {
  title:                String
  category:             Category
  frequency:            Frequency
  daysOfWeek:           [Int!]
  preferredTimeOfDay:   TimeOfDay
  preferredDurationMin: Int
  notes:                String
  isActive:             Boolean
}
```

### Output

```graphql
type Routine {
  id, title, category, frequency, daysOfWeek,
  preferredTimeOfDay, preferredDurationMin, notes, isActive
}

# Queries
routines(activeOnly: Boolean): [Routine!]!

# Mutations
createRoutine(input: CreateRoutineInput!): Routine!
updateRoutine(id: UUID!, input: UpdateRoutineInput!): Routine!
deleteRoutine(id: UUID!): Boolean!
```

### sqlc Queries

File: `backend/db/queries/routines.sql`

```sql
-- name: CreateRoutine :one
INSERT INTO routines (title, category, frequency, days_of_week,
  preferred_time_of_day, preferred_duration_min, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetRoutine :one
SELECT * FROM routines WHERE id = $1;

-- name: ListRoutines :many
SELECT * FROM routines
WHERE (sqlc.narg('active_only')::BOOLEAN IS NULL OR sqlc.narg('active_only') = false OR is_active = true)
ORDER BY created_at;

-- name: UpdateRoutine :one
UPDATE routines SET
  title = COALESCE(sqlc.narg('title'), title),
  category = COALESCE(sqlc.narg('category'), category),
  frequency = COALESCE(sqlc.narg('frequency'), frequency),
  days_of_week = COALESCE(sqlc.narg('days_of_week'), days_of_week),
  preferred_time_of_day = COALESCE(sqlc.narg('preferred_time_of_day'), preferred_time_of_day),
  preferred_duration_min = COALESCE(sqlc.narg('preferred_duration_min'), preferred_duration_min),
  notes = COALESCE(sqlc.narg('notes'), notes),
  is_active = COALESCE(sqlc.narg('is_active'), is_active)
WHERE id = $1
RETURNING *;

-- name: DeleteRoutine :exec
DELETE FROM routines WHERE id = $1;

-- name: ListRoutinesForDay :many
-- Used by planner: active routines that apply to a given day-of-week
SELECT * FROM routines
WHERE is_active = true
  AND (
    frequency = 'daily'
    OR (frequency = 'weekdays' AND $1::INT BETWEEN 1 AND 5)
    OR (frequency = 'weekly' AND $1::INT = ANY(days_of_week))
    OR (frequency = 'custom' AND $1::INT = ANY(days_of_week))
  )
ORDER BY preferred_time_of_day, title;
```

Note: `ListRoutinesForDay` is defined here but called by the planner context. The `$1` parameter is the Go `time.Weekday()` value (0=Sunday, 6=Saturday), matching the `days_of_week` convention.

## Day-of-week applicability logic

The `frequency` field combined with `days_of_week` determines which days a routine applies:

| frequency | days_of_week | Applies on |
|---|---|---|
| `daily` | ignored | Every day |
| `weekdays` | ignored | Mon–Fri (1–5) |
| `weekly` | required (at least 1) | Specified days only |
| `custom` | required (at least 1) | Specified days only |

`days_of_week` uses Go's `time.Weekday`: 0=Sunday, 1=Monday, ..., 6=Saturday.

## Behaviors (EARS syntax)

### Input validation

Input validation uses gqlgen directives (see `specs/validation.md`). Fields annotated with `@validate` in `schema.graphqls` are automatically sanitized and length-checked by the directive handler before the resolver runs. Converters do NOT perform text validation.

- When `createRoutine` or `updateRoutine` is called with an annotated field, the `@validate` directive handler shall sanitize and length-check the value before the resolver executes.
- When validation fails, the directive handler shall return the validation error without the resolver or database being reached.

### GraphQL directive annotations

```graphql
input CreateRoutineInput {
  title: String!  @validate(rule: SINGLE_LINE)
  notes: String   @validate(rule: PLAIN_TEXT)
}

input UpdateRoutineInput {
  title: String   @validate(rule: SINGLE_LINE)
  notes: String   @validate(rule: PLAIN_TEXT)
}
```

### Frequency validation

- When `createRoutine` is called with `frequency = WEEKLY` or `CUSTOM` and `daysOfWeek` is empty or null, the system shall return a validation error.
- When `createRoutine` is called with `frequency = DAILY` or `WEEKDAYS`, the system shall ignore any provided `daysOfWeek` value (store it but don't require it).
- When `routines(activeOnly: true)` is called, the system shall return only routines where `is_active = true`.
- When `routines(activeOnly: false)` or `routines()` is called, the system shall return all routines.
- When `updateRoutine` is called with `isActive = false`, the system shall deactivate the routine. Deactivated routines are excluded from plan generation but not deleted.
- When `deleteRoutine` is called, the system shall delete the routine. Any tasks with `routine_id` referencing it will have `routine_id` set to NULL by the DB (ON DELETE SET NULL).
- When `deleteRoutine` is called with a non-existent ID, the system shall return an error.

## Test Anchors

1. Given no routines exist, when `createRoutine({title: "Daily exercise", category: EXERCISE, frequency: DAILY, preferredTimeOfDay: MORNING, preferredDurationMin: 45})` is called, then a routine is returned with `isActive = true` and a generated UUID.

2. Given a routine exists, when `updateRoutine(id, {isActive: false})` is called, then the routine's `isActive` is false, and `routines(activeOnly: true)` no longer returns it.

3. Given no routines exist, when `createRoutine({title: "Weekend prep", category: MEAL, frequency: WEEKLY, daysOfWeek: null})` is called, then a validation error is returned.

4. Given today is Wednesday (day=3) and routines exist with frequencies `daily`, `weekdays`, and `custom(days=[0,6])`, when `ListRoutinesForDay(3)` is called, then the `daily` and `weekdays` routines are returned but not the weekend-only custom routine.

5. Given a routine linked to 2 tasks exists, when `deleteRoutine` is called, then the routine is deleted and both tasks still exist with `routine_id = NULL`.

6. Given no routines exist, when `createRoutine({title: "a" repeated 300 times, ...})` is called (title > 255 chars), then a validation error is returned.

7. Given no routines exist, when `createRoutine({title: "Morning\nexercise", ...})` is called, then the routine is created with title `"Morningexercise"` (newlines stripped).
