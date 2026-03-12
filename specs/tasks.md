# Spec: Tasks

## Bounded Context

Owns: Task CRUD resolvers, parent/subtask hierarchy logic, completion logic, sqlc queries for tasks, deadline validation

Does not own: `tasks` table DDL (foundation), AI-assisted task creation (planner), carry-over logic that increments `times_deferred` (carry-over), `actual_minutes` computation from blocks (day-plans/carry-over), priority escalation logic (carry-over/planner — priority is stored here but escalation is applied at plan-generation time)

Depends on: foundation (table + schema exist), routines (foreign key `routine_id`)

Produces: GraphQL queries (`tasks`, `task`) and mutations (`createTask`, `updateTask`, `deleteTask`, `completeTask`)

## Contracts

### Input

```graphql
input CreateTaskInput {
  title:            String!
  category:         Category!
  priority:         Priority!
  parentId:         UUID
  estimatedMinutes: Int            # defaults to 60 for standalone/subtask if omitted
  deadlineType:     DeadlineType
  deadlineDate:     Date
  deadlineDays:     Int
  notes:            String
  routineId:        UUID
}

input UpdateTaskInput {
  title:            String
  category:         Category
  priority:         Priority
  estimatedMinutes: Int
  deadlineType:     DeadlineType
  deadlineDate:     Date
  deadlineDays:     Int
  notes:            String
  routineId:        UUID
  isCompleted:      Boolean
}
```

### Output

```graphql
type Task {
  id, title, category, priority, parentId, parent, subtasks,
  estimatedMinutes, actualMinutes, deadlineType, deadlineDate,
  deadlineDays, notes, isRoutine, routine, timesDeferred,
  isCompleted, completedAt, createdAt, updatedAt
}

# Queries
tasks(category: Category, includeCompleted: Boolean): [Task!]!
task(id: UUID!): Task

# Mutations
createTask(input: CreateTaskInput!): Task!
updateTask(id: UUID!, input: UpdateTaskInput!): Task!
deleteTask(id: UUID!): Boolean!
completeTask(id: UUID!): Task!
```

### sqlc Queries

File: `backend/db/queries/tasks.sql`

```sql
-- name: CreateTask :one
INSERT INTO tasks (title, category, priority, parent_id, estimated_minutes,
  deadline_type, deadline_date, deadline_days, notes, is_routine, routine_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = $1;

-- name: ListTasks :many
SELECT * FROM tasks
WHERE (sqlc.narg('category')::TEXT IS NULL OR category = sqlc.narg('category'))
  AND (sqlc.narg('include_completed')::BOOLEAN = true OR is_completed = false)
ORDER BY
  CASE priority WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 END,
  created_at DESC;

-- name: ListSubtasks :many
SELECT * FROM tasks WHERE parent_id = $1 ORDER BY created_at;

-- name: UpdateTask :one
UPDATE tasks SET
  title = COALESCE(sqlc.narg('title'), title),
  category = COALESCE(sqlc.narg('category'), category),
  priority = COALESCE(sqlc.narg('priority'), priority),
  estimated_minutes = COALESCE(sqlc.narg('estimated_minutes'), estimated_minutes),
  deadline_type = COALESCE(sqlc.narg('deadline_type'), deadline_type),
  deadline_date = COALESCE(sqlc.narg('deadline_date'), deadline_date),
  deadline_days = COALESCE(sqlc.narg('deadline_days'), deadline_days),
  notes = COALESCE(sqlc.narg('notes'), notes),
  routine_id = COALESCE(sqlc.narg('routine_id'), routine_id),
  updated_at = now()
WHERE id = $1
RETURNING *;

-- name: CompleteTask :one
UPDATE tasks SET is_completed = true, completed_at = now(), updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UncompleteTask :one
UPDATE tasks SET is_completed = false, completed_at = NULL, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = $1;

-- name: CountIncompleteSubtasks :one
SELECT COUNT(*) FROM tasks WHERE parent_id = $1 AND is_completed = false;

-- name: ListSchedulableTasks :many
-- Used by planner: subtasks OR standalone tasks, incomplete only
SELECT * FROM tasks
WHERE is_completed = false
  AND (parent_id IS NOT NULL OR (parent_id IS NULL AND estimated_minutes IS NOT NULL))
ORDER BY
  CASE priority WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 END,
  deadline_date ASC NULLS LAST,
  times_deferred DESC,
  created_at;

-- name: IncrementTimesDeferred :one
UPDATE tasks SET times_deferred = times_deferred + 1, last_deferred_at = now(), updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateActualMinutes :exec
UPDATE tasks SET actual_minutes = actual_minutes + $2, updated_at = now()
WHERE id = $1;
```

Note: `IncrementTimesDeferred` and `UpdateActualMinutes` are defined here (they write to the `tasks` table) but called by the carry-over and day-plans contexts.

## Behaviors (EARS syntax)

### Input validation

Input validation uses gqlgen directives (see `specs/validation.md`). Fields annotated with `@validate` in `schema.graphqls` are automatically sanitized and length-checked by the directive handler before the resolver runs. Converters do NOT perform text validation.

- When `createTask` or `updateTask` is called with an annotated field, the `@validate` directive handler shall sanitize and length-check the value before the resolver executes.
- When validation fails, the directive handler shall return the validation error without the resolver or database being reached.

### GraphQL directive annotations

```graphql
input CreateTaskInput {
  title: String!  @validate(rule: SINGLE_LINE)
  notes: String   @validate(rule: PLAIN_TEXT)
  # remaining fields: enums, ints, UUIDs — no validation directives needed
}

input UpdateTaskInput {
  title: String   @validate(rule: SINGLE_LINE)
  notes: String   @validate(rule: PLAIN_TEXT)
}
```

### Creation

- When `createTask` is called without `estimatedMinutes` and the task is standalone (no `parentId`) or a subtask, the system shall default `estimatedMinutes` to 60.
- When `createTask` is called with a `parentId`, the system shall verify the parent exists and return an error if not.
- When `createTask` is called with a `parentId` that itself has a `parentId` (nested subtask), the system shall return an error — only one level of nesting is allowed.
- When `createTask` is called with `deadlineType = HARD` and `deadlineDate` is null, the system shall return a validation error.
- When `createTask` is called with `deadlineType = HORIZON` and `deadlineDays` is null, the system shall return a validation error.
- When `createTask` is called with a `routineId`, the system shall set `isRoutine = true`.

### Hierarchy & querying

- When the `tasks` query is called, the system shall return top-level tasks only (`parent_id IS NULL`). Subtasks are accessed via the `subtasks` field resolver on each parent.
- When the `subtasks` field is resolved, the system shall return all tasks where `parent_id` matches the parent's `id`, ordered by `created_at`.
- When the `parent` field is resolved on a task with a non-null `parentId`, the system shall return that parent task.

### Completion

- When `completeTask` is called on a standalone task, the system shall mark it completed directly.
- When `completeTask` is called on a subtask, the system shall mark it completed, then check if all sibling subtasks are also completed. If so, the system shall auto-complete the parent task.
- When `completeTask` is called on a parent task (has subtasks), the system shall return an error — parent completion is automatic only.
- When `updateTask` sets `isCompleted = true`, the system shall apply the same completion logic as `completeTask`.
- When `updateTask` sets `isCompleted = false` on a completed task, the system shall uncomplete it (`is_completed = false`, `completed_at = NULL`).

### Deletion

- When `deleteTask` is called on a parent task, all subtasks are cascade-deleted by the DB constraint.
- When `deleteTask` is called with a non-existent ID, the system shall return an error.

### Filtering

- When `tasks` query is called with `includeCompleted` omitted or `false`, the system shall exclude completed tasks.
- When `tasks` query is called with `includeCompleted = true`, the system shall include completed tasks.

## Decision Table: Deadline validation

| deadlineType | deadlineDate | deadlineDays | Result |
|---|---|---|---|
| NULL | any | any | Valid (no deadline) |
| HARD | set | any | Valid |
| HARD | NULL | any | Error: `deadlineDate required for HARD deadline` |
| HORIZON | any | set | Valid |
| HORIZON | any | NULL | Error: `deadlineDays required for HORIZON deadline` |

## Test Anchors

1. Given no tasks exist, when `createTask({title: "Apply to Stripe", category: JOB, priority: HIGH})` is called, then a task is returned with `estimatedMinutes = 60`, `actualMinutes = 0`, `timesDeferred = 0`, `isCompleted = false`.

2. Given a parent task with 2 subtasks (one completed, one incomplete), when `completeTask` is called on the incomplete subtask, then both the subtask and the parent are marked completed.

3. Given a parent task with 2 subtasks (one completed, one incomplete), when `completeTask` is called on the parent task directly, then an error is returned.

4. Given tasks in multiple categories, when `tasks(category: INTERVIEW, includeCompleted: false)` is called, then only incomplete interview tasks are returned.

5. Given a parent task with 3 subtasks, when `deleteTask` is called on the parent, then all 4 rows are deleted.

6. Given no tasks exist, when `createTask({..., deadlineType: HARD, deadlineDate: null})` is called, then a validation error is returned.

7. Given a subtask exists under a parent, when `createTask({..., parentId: subtaskId})` is called (nesting), then an error is returned.

8. Given no tasks exist, when `createTask({title: "a]repeated 300 times", ...})` is called (title > 255 chars), then a validation error is returned.

9. Given no tasks exist, when `createTask({title: "Test\ninjection\nattempt", ...})` is called, then the task is created with title `"Testinjectionattempt"` (newlines stripped).
