# Spec: Carry-Over

## Bounded Context

Owns: `resolveSkippedBlock` mutation, skipped-task review logic, `times_deferred` / `last_deferred_at` updates, `actual_minutes` computation, `effectiveDaysRemaining` calculation, priority escalation logic

Does not own: `day_plans` table (owned by day-plans), `tasks` table schema (owned by tasks), block skip/adjust mutations (owned by day-plans), planner prompt construction (owned by planner — but carry-over provides data for the prompt)

Depends on: `day-plans` (reads `day_plans.blocks` to find skipped blocks with `task_id`), `tasks` (reads and updates `tasks` rows)

Produces: `resolveSkippedBlock` mutation, `actual_minutes` batch updates on tasks, `times_deferred` increments, helper functions consumed by the planner context (`effectiveDaysRemaining`, carry-over task list, priority escalation)

## Contracts

### Input

**GraphQL mutation consumed:**

```graphql
resolveSkippedBlock(planId: UUID!, blockId: String!, intentional: Boolean!): Boolean!
```

- `planId` — the previous day's plan
- `blockId` — the specific skipped block being reviewed
- `intentional` — `true` = user chose to skip; `false` = user didn't get to it

**Data read from DB:**

- `day_plans.blocks` (JSONB) — to identify skipped blocks with a `task_id`
- `tasks` — to read/update `times_deferred`, `last_deferred_at`, `actual_minutes`, `deadline_type`, `deadline_days`

### Output

**DB writes:**

When `intentional = false`:
- `tasks.times_deferred` += 1
- `tasks.last_deferred_at` = now()

When `intentional = true`:
- No changes to the task

**`actual_minutes` computation** (separate from resolve flow):
- For each non-skipped block with a `task_id` in an accepted plan, add the block's `duration` to `tasks.actual_minutes`

**Helper functions exported for the planner context:**

```go
// GetCarryOverTasks returns tasks that were skipped (non-intentionally) from
// the most recent past plan, with their times_deferred and effective deadline info.
func GetCarryOverTasks(ctx context.Context, db *pgx.Pool) ([]CarryOverTask, error)

// EffectiveDaysRemaining computes deadline pressure for horizon tasks.
// Returns -1 if the task has no horizon deadline.
func EffectiveDaysRemaining(task Task) int
```

### Data Model

No new tables. This context reads and writes to existing tables owned by other contexts:

- `day_plans` (read-only) — owned by day-plans spec
- `tasks` (read-write: `times_deferred`, `last_deferred_at`, `actual_minutes`) — schema owned by tasks spec

## Behaviors (EARS syntax)

### Skipped block resolution

- When `resolveSkippedBlock` is called with `intentional = true`, the system shall return `true` without modifying the task.
- When `resolveSkippedBlock` is called with `intentional = false`, the system shall increment `tasks.times_deferred` by 1 and set `tasks.last_deferred_at` to the current UTC timestamp.
- When `resolveSkippedBlock` is called with a `blockId` that does not exist in the plan's blocks, the system shall return an error.
- When `resolveSkippedBlock` is called for a block that is not skipped (`skipped = false`), the system shall return an error.
- When `resolveSkippedBlock` is called for a block with no `task_id`, the system shall return `true` without modifying any task (nothing to defer).

### Effective deadline calculation

- Where a task has `deadline_type = 'horizon'`, the system shall compute `effectiveDaysRemaining = deadline_days - times_deferred`.
- Where `effectiveDaysRemaining <= 3`, the system shall treat the task as having a hard deadline for scheduling purposes.
- Where `effectiveDaysRemaining <= 0`, the system shall still return the value (can be negative) — the planner will see extreme urgency.

### Priority escalation

- Where `times_deferred = 1`, the system shall treat the task as one priority level higher than assigned (`low` -> `medium`, `medium` -> `high`, `high` stays `high`).
- Where `times_deferred = 2`, the system shall treat the task as `HIGH` priority regardless of assigned priority.
- Where `times_deferred >= 3`, the system shall flag the task with "OVERDUE" status for the planner to prefix with `⚑ OVERDUE` in block titles and schedule in the first available morning slot.

### `actual_minutes` computation

- When a plan is accepted (status changes to `accepted`), the system shall update `actual_minutes` for all non-skipped blocks with a `task_id` whose block time is in the past.
- When end-of-day processing runs, the system shall update `actual_minutes` for all non-skipped blocks with a `task_id` in today's accepted plan.
- When a block is skipped, the system shall NOT add its duration to `actual_minutes`.
- When a block's duration is adjusted, the system shall use the adjusted duration (not the original) for `actual_minutes`.
- The system shall not double-count: if `actual_minutes` was already updated for a block (e.g., at accept time), end-of-day processing shall not add it again.

### Carry-over task retrieval (for planner prompt)

- When generating a plan for today, the system shall retrieve all skipped blocks from the most recent past accepted plan that have a `task_id` and whose linked task is not completed.
- The system shall include each carry-over task's `times_deferred` and effective deadline in the data passed to the planner.

## Decision Table: Priority Escalation

| `times_deferred` | Assigned Priority | Effective Priority | Scheduling Constraint |
|-------------------|-------------------|--------------------|-----------------------|
| 0 | any | same as assigned | none |
| 1 | LOW | MEDIUM | none |
| 1 | MEDIUM | HIGH | none |
| 1 | HIGH | HIGH | none |
| 2 | any | HIGH | none |
| 3+ | any | HIGH | must schedule in first available morning slot; title prefixed with `⚑ OVERDUE` |

## Decision Table: Resolve Skipped Block

| Block exists? | Block skipped? | Has task_id? | intentional? | Action |
|---------------|----------------|--------------|--------------|--------|
| No | - | - | - | Return error |
| Yes | No | - | - | Return error |
| Yes | Yes | No | - | Return `true`, no DB writes |
| Yes | Yes | Yes | true | Return `true`, no task modifications |
| Yes | Yes | Yes | false | Increment `times_deferred`, set `last_deferred_at`, return `true` |

## Implementation Notes

### `actual_minutes` tracking strategy

To avoid double-counting, use a pragmatic approach: compute `actual_minutes` as a derived value rather than incrementally updating it. When `actual_minutes` needs to be current:

```go
// ComputeActualMinutes sums duration of all non-skipped blocks across all
// accepted plans that reference this task_id.
func ComputeActualMinutes(ctx context.Context, db *pgx.Pool, taskID uuid.UUID) (int, error)
```

**sqlc query:**

```sql
-- name: ComputeActualMinutesForTask :one
-- Sums duration of non-skipped blocks referencing a given task_id across all accepted plans.
-- Blocks are JSONB arrays; use jsonb_array_elements to unnest.
SELECT COALESCE(SUM((elem->>'duration')::int), 0)::int AS total_minutes
FROM day_plans, jsonb_array_elements(blocks) AS elem
WHERE status = 'accepted'
  AND elem->>'task_id' = @task_id::text
  AND (elem->>'skipped')::boolean = false;
```

Call this query and write the result to `tasks.actual_minutes` at these trigger points:
1. When `acceptPlan` is called (update all tasks referenced in the plan)
2. When `skipBlock` is called (recompute for the affected task)
3. When `updateBlock` is called with a duration change (recompute for the affected task)
4. On a periodic/on-demand basis for the backlog view

### `GetCarryOverTasks` query

```sql
-- name: GetSkippedBlocksFromLastPlan :many
-- Returns skipped blocks with task_ids from the most recent accepted plan before today.
SELECT elem->>'id' AS block_id,
       elem->>'title' AS title,
       elem->>'category' AS category,
       (elem->>'duration')::int AS duration,
       (elem->>'task_id')::uuid AS task_id
FROM day_plans, jsonb_array_elements(blocks) AS elem
WHERE status = 'accepted'
  AND plan_date < @today::date
  AND (elem->>'skipped')::boolean = true
  AND elem->>'task_id' IS NOT NULL
  AND elem->>'task_id' != 'null'
  AND elem->>'task_id' != ''
ORDER BY plan_date DESC;
```

Filter results in Go to only include the most recent plan_date and tasks where `is_completed = false`.

### `EffectiveDaysRemaining` (pure function)

```go
func EffectiveDaysRemaining(deadlineDays *int, timesDeferred int) int {
    if deadlineDays == nil {
        return -1 // no horizon deadline
    }
    return *deadlineDays - timesDeferred
}

func EffectivePriority(assignedPriority string, timesDeferred int) string {
    switch {
    case timesDeferred >= 2:
        return "high"
    case timesDeferred == 1:
        switch assignedPriority {
        case "low":
            return "medium"
        case "medium":
            return "high"
        default:
            return assignedPriority
        }
    default:
        return assignedPriority
    }
}

func IsOverdue(timesDeferred int) bool {
    return timesDeferred >= 3
}
```

## Test Anchors

1. Given a skipped block with `task_id` pointing to a task with `times_deferred = 0`, when `resolveSkippedBlock` is called with `intentional = false`, then `tasks.times_deferred` becomes 1 and `last_deferred_at` is set to approximately now.

2. Given a skipped block with `task_id`, when `resolveSkippedBlock` is called with `intentional = true`, then `tasks.times_deferred` remains unchanged and `last_deferred_at` remains unchanged.

3. Given a block that is not skipped (`skipped = false`), when `resolveSkippedBlock` is called, then the system returns an error.

4. Given a skipped block with no `task_id` (e.g., a routine block), when `resolveSkippedBlock` is called, then it returns `true` without any task modification.

5. Given a task with `deadline_type = 'horizon'`, `deadline_days = 14`, and `times_deferred = 3`, when `EffectiveDaysRemaining` is computed, then the result is 11.

6. Given a task with `times_deferred = 1` and `priority = 'low'`, when `EffectivePriority` is computed, then the result is `'medium'`.

7. Given a task with `times_deferred = 2` and `priority = 'low'`, when `EffectivePriority` is computed, then the result is `'high'`.

8. Given a task with `times_deferred = 3`, when `IsOverdue` is checked, then it returns `true`.

9. Given an accepted plan with 3 blocks (2 non-skipped with task_ids, 1 skipped), when `ComputeActualMinutesForTask` is called for one of the non-skipped blocks' task, then it returns the sum of that task's non-skipped block durations across all plans.

10. Given yesterday's accepted plan has 2 skipped blocks with task_ids (one task completed, one not), when `GetCarryOverTasks` is called, then only the non-completed task is returned.

11. Given a task with `deadline_type = 'horizon'`, `deadline_days = 5`, and `times_deferred = 3` (effective = 2), when the planner queries carry-over data, then the task is treated as having a hard deadline (effective days remaining <= 3).
