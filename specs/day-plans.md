# Spec: Day Plans

## Bounded Context

Owns: `day_plans` table, `plan_messages` table, plan CRUD resolvers, block action resolvers (skip, update), plan status lifecycle (draft → accepted), block JSONB read/write logic

Does not own: AI plan generation (owned by `planner`), carry-over/skipped-task review logic (owned by `carry-over`), `actual_minutes` computation on tasks (owned by `carry-over`), task/routine/context CRUD

Depends on: `foundation` (tables + migrations), `tasks` (reads tasks for block linkage validation), `routines` (reads routines for block linkage validation)

Produces:
- Queries: `dayPlan(date)`, `recentPlans(limit)`
- Mutations: `acceptPlan(date)`, `skipBlock(planId, blockId)`, `unskipBlock(planId, blockId)`, `completeBlock(planId, blockId)`, `updateBlock(planId, blockId, input)`
- Note: `sendPlanMessage` is owned by the `planner` spec — this spec owns the plan/message storage that the planner writes to.

## Contracts

### Input

**GraphQL inputs consumed:**

```graphql
input UpdateBlockInput {
  skipped:  Boolean
  done:     Boolean
  time:     String
  duration: Int
  notes:    String
}
```

**Data read from DB:**
- `day_plans` — by `plan_date` (unique) or by `id`
- `plan_messages` — by `plan_id`, ordered by `created_at ASC`

### Output

**GraphQL types returned:**

```graphql
type DayPlan {
  id:        UUID!
  planDate:  Date!
  status:    PlanStatus!    # DRAFT | ACCEPTED
  blocks:    [PlanBlock!]!
  messages:  [PlanMessage!]!
  createdAt: DateTime!
  updatedAt: DateTime!
}

type PlanBlock {
  id:        String!
  time:      String!        # "HH:MM" format
  duration:  Int!           # minutes
  title:     String!
  category:  Category!
  taskId:    UUID
  routineId: UUID
  notes:     String
  skipped:   Boolean!
  done:      Boolean!
}

type PlanMessage {
  id:        UUID!
  role:      String!        # "user" | "assistant"
  content:   String!
  createdAt: DateTime!
}
```

### Data Model

Tables are created by the `foundation` spec. This spec owns the read/write logic.

**`day_plans`** — one row per date (UNIQUE on `plan_date`). `blocks` is a JSONB array of `PlanBlock` objects. `status` is `'draft'` or `'accepted'`.

**`plan_messages`** — ordered conversation history for a plan. Linked via `plan_id` FK with CASCADE delete.

### sqlc Queries

File: `backend/db/queries/day_plans.sql`

```sql
-- name: GetDayPlanByDate :one
SELECT * FROM day_plans WHERE plan_date = $1;

-- name: GetDayPlanByID :one
SELECT * FROM day_plans WHERE id = $1;

-- name: CreateDayPlan :one
INSERT INTO day_plans (plan_date, status, blocks)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateDayPlanBlocks :one
UPDATE day_plans SET blocks = $2, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: UpdateDayPlanStatus :one
UPDATE day_plans SET status = $2, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: RecentPlans :many
SELECT * FROM day_plans ORDER BY plan_date DESC LIMIT $1;

-- name: GetPlanMessages :many
SELECT * FROM plan_messages WHERE plan_id = $1 ORDER BY created_at ASC;

-- name: CreatePlanMessage :one
INSERT INTO plan_messages (plan_id, role, content)
VALUES ($1, $2, $3)
RETURNING *;
```

## Behaviors (EARS syntax)

### Queries

- When `dayPlan(date)` is called, the system shall return the `day_plan` for that date with its blocks (parsed from JSONB) and messages, or `null` if no plan exists.
- When `recentPlans(limit)` is called, the system shall return up to `limit` plans ordered by `plan_date` DESC. If `limit` is not provided, default to 7.

### Plan Status Lifecycle

- When a plan is first created (by the planner), the system shall set `status = 'draft'`.
- When `acceptPlan(date)` is called on a draft plan, the system shall set `status = 'accepted'` and return the updated plan.
- When `acceptPlan(date)` is called and no plan exists for that date, the system shall return an error: "No plan exists for this date."
- When `acceptPlan(date)` is called on an already-accepted plan, the system shall return the plan unchanged (idempotent).

### Block Actions

- When `skipBlock(planId, blockId)` is called, the system shall set `skipped: true` on the matching block in the JSONB array and return the updated plan.
- When `skipBlock` is called on a plan that is not `accepted`, the system shall return an error: "Can only skip blocks on an accepted plan."
- When `skipBlock` is called with a `blockId` that does not exist in the plan's blocks, the system shall return an error: "Block not found."
- When `unskipBlock(planId, blockId)` is called, the system shall set `skipped: false` on the matching block and return the updated plan.
- When `unskipBlock` is called on a plan that is not `accepted`, the system shall return an error: "Can only unskip blocks on an accepted plan."
- When `unskipBlock` is called with a `blockId` that does not exist, the system shall return an error: "Block not found."
- When `completeBlock(planId, blockId)` is called, the system shall set `done: true` on the matching block and return the updated plan.
- When `completeBlock` is called on a plan that is not `accepted`, the system shall return an error: "Can only complete blocks on an accepted plan."
- When `completeBlock` is called with a `blockId` that does not exist, the system shall return an error: "Block not found."
- When `updateBlock(planId, blockId, input)` is called, the system shall update the specified fields on the matching block. Only provided (non-null) fields are updated (including `done`).
- When `updateBlock` is called on a plan that is not `accepted`, the system shall return an error: "Can only update blocks on an accepted plan."
- When `updateBlock` is called with a `blockId` that does not exist, the system shall return an error: "Block not found."
- When `updateBlock` sets `duration` to a value <= 0, the system shall return an error: "Duration must be positive."

### Messages

- The system shall store all plan chat messages in `plan_messages` with `role` = `'user'` or `'assistant'`.
- When resolving `DayPlan.messages`, the system shall return all `plan_messages` for that plan ordered by `created_at ASC`.

### JSONB Block Handling

- The system shall marshal `PlanBlock` structs to/from JSONB using Go's `encoding/json`.
- When reading blocks from the database, the system shall unmarshal the JSONB array into `[]PlanBlock` and validate that each block has a non-empty `id`, `time`, `title`, and `category`.
- If a block in JSONB has an unrecognized category, the system shall default it to `"admin"` rather than failing.

## Decision Table: Block Action Validation

| Plan Status | Block Exists | Action        | Result                                    |
|-------------|-------------|---------------|-------------------------------------------|
| draft       | -           | skipBlock     | Error: "Can only skip blocks on an accepted plan" |
| draft       | -           | unskipBlock   | Error: "Can only unskip blocks on an accepted plan" |
| draft       | -           | completeBlock | Error: "Can only complete blocks on an accepted plan" |
| draft       | -           | updateBlock   | Error: "Can only update blocks on an accepted plan" |
| accepted    | no          | skipBlock     | Error: "Block not found"                  |
| accepted    | no          | unskipBlock   | Error: "Block not found"                  |
| accepted    | no          | completeBlock | Error: "Block not found"                  |
| accepted    | no          | updateBlock   | Error: "Block not found"                  |
| accepted    | yes         | skipBlock     | Set `skipped: true`, return updated plan  |
| accepted    | yes         | unskipBlock   | Set `skipped: false`, return updated plan |
| accepted    | yes         | completeBlock | Set `done: true`, return updated plan     |
| accepted    | yes         | updateBlock   | Update provided fields, return updated plan |

## Test Anchors

1. Given no plan exists for `2026-03-05`, when `dayPlan(date: "2026-03-05")` is called, then `null` is returned.

2. Given a draft plan exists for `2026-03-05` with 3 blocks, when `acceptPlan(date: "2026-03-05")` is called, then the plan status changes to `accepted` and all 3 blocks are preserved in the response.

3. Given an accepted plan with block `"block-1"`, when `skipBlock(planId, "block-1")` is called, then the block's `skipped` field is `true` in the returned plan and all other blocks remain unchanged.

4. Given an accepted plan with block `"block-2"` having `duration: 60`, when `updateBlock(planId, "block-2", { duration: 45 })` is called, then the block's duration is `45` and all other fields on that block are unchanged.

5. Given a draft plan exists, when `skipBlock` is called on it, then an error is returned indicating blocks can only be modified on accepted plans.

6. Given an accepted plan, when `updateBlock` is called with a non-existent `blockId`, then an error "Block not found" is returned.

7. Given 3 plans exist for different dates, when `recentPlans(limit: 2)` is called, then exactly 2 plans are returned, ordered by `plan_date` DESC.

8. Given a plan with 2 messages (user + assistant), when `dayPlan` is queried with the `messages` field, then both messages are returned in chronological order.

9. Given an accepted plan with block `"block-3"`, when `updateBlock(planId, "block-3", { duration: 0 })` is called, then an error "Duration must be positive" is returned.

10. Given an accepted plan with a skipped block `"block-4"`, when `unskipBlock(planId, "block-4")` is called, then the block's `skipped` field is `false` in the returned plan.

11. Given an accepted plan with block `"block-5"`, when `completeBlock(planId, "block-5")` is called, then the block's `done` field is `true` in the returned plan and all other blocks remain unchanged.

12. Given a draft plan, when `completeBlock` is called on it, then an error is returned indicating blocks can only be completed on accepted plans.
