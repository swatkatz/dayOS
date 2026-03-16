# DayOS — Daily Planning App
## Claude Code Spec

---

## Overview

A daily planning web app. It uses the Anthropic API to generate
intelligent, time-blocked day plans from a persistent task backlog, routines, and
user context. Plans are created and refined through a **chat interface** — the user
describes their day, Claude generates a plan, and they go back and forth until the plan
feels right. Tasks are also scoped through AI-assisted chat: describe a goal, Claude
asks clarifying questions, then proposes subtasks with time estimates.

Hosted on Railway with PostgreSQL. Multi-user with Clerk authentication (invite-only).
Shared with friends and family.

---

## Tech Stack

| Layer | Choice | Rationale |
|---|---|---|
| Backend | Go + gqlgen | Consistent with AdvisorHub, BabyBaton |
| Database | PostgreSQL on Railway | Already in use |
| Frontend | React + TypeScript + Vite | Type safety, fast dev loop |
| Styling | Tailwind CSS | Utility-first, no component library needed |
| Deployment | Railway (monorepo) | Single platform for DB + backend + frontend |
| AI | Anthropic Claude API (claude-sonnet-4-6) | Plan generation, replanning, task scoping |

---

## Repository Structure

```
dayos/
├── backend/
│   ├── graph/
│   │   ├── schema.graphqls
│   │   ├── resolver.go          # Resolver struct with per-context store interfaces
│   │   ├── stores.go            # Store interfaces (RoutineStore, TaskStore, etc.)
│   │   ├── {context}_convert.go # Per-context converter: FromDB, ToDB, MergeParams
│   │   ├── schema.resolvers.go  # Resolver implementations (generated scaffold)
│   │   ├── testutil_test.go     # Mock stores + factories for tests
│   │   └── model/
│   ├── db/
│   │   ├── migrations/
│   │   └── queries/
│   ├── planner/          # Claude API integration (plans + task scoping)
│   │   └── planner.go
│   ├── main.go
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   ├── graphql/
│   │   └── App.tsx
│   ├── package.json
│   └── vite.config.ts
├── specs/                # Bounded context specs
├── docs/
│   └── DESIGN.md
├── railway.toml
├── CLAUDE.md             # Persistent Claude Code instructions
└── README.md
```

---

## Data Models

### `tasks`

The core backlog. Tasks come in two forms:
- **Parent tasks** — containers that hold a deadline and group subtasks. Created through
  AI-assisted chat. Not directly schedulable into plan blocks.
- **Subtasks** — concrete units of work with time estimates. These get scheduled into
  day plan blocks. Have a `parent_id` linking to their parent.
- **Standalone tasks** — quick one-off tasks with no parent. Created manually. Also
  schedulable.

```sql
CREATE TABLE tasks (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title       TEXT NOT NULL,
  category    TEXT NOT NULL,         -- job|meal|baby|exercise|project|interview|admin
  priority    TEXT NOT NULL,         -- high|medium|low

  -- Hierarchy
  parent_id   UUID REFERENCES tasks(id) ON DELETE CASCADE,  -- NULL = parent or standalone

  -- Time estimates (required for subtasks and standalone; NULL for parents)
  estimated_minutes  INT,            -- how long this task should take total
  actual_minutes     INT DEFAULT 0,  -- computed: sum of non-skipped block durations

  -- Deadline: either a hard date OR a horizon ("within N days")
  -- On parent tasks, deadline applies to all subtasks
  deadline_type   TEXT,              -- 'hard' | 'horizon' | NULL
  deadline_date   DATE,              -- set when deadline_type = 'hard'
  deadline_days   INT,               -- set when deadline_type = 'horizon' (e.g., 14 = within 2 weeks)

  notes       TEXT,
  is_routine  BOOLEAN DEFAULT false, -- marks tasks that repeat
  routine_id  UUID REFERENCES routines(id) ON DELETE SET NULL,

  -- Carry-over tracking
  times_deferred   INT DEFAULT 0,    -- incremented each time task is scheduled but skipped
  last_deferred_at TIMESTAMPTZ,      -- when it was last missed

  is_completed   BOOLEAN DEFAULT false,
  completed_at   TIMESTAMPTZ,
  created_at     TIMESTAMPTZ DEFAULT now(),
  updated_at     TIMESTAMPTZ DEFAULT now()
);
```

**Task creation modes:**

1. **AI-assisted (chat)** — User describes a high-level goal. Claude asks clarifying
   questions (scope, deliverables, timeline). Claude proposes subtasks with time estimates.
   User reviews/adjusts, then they're created as a parent + subtasks.

2. **Manual (quick add)** — User creates a standalone task directly. `estimated_minutes`
   defaults to 60, editable.

**Parent task completion:** A parent task is considered complete when all its subtasks
are completed. No manual check-off on parents.

**Categories:**
- `job` — Job search, applications, LinkedIn
- `interview` — Interview prep, practice problems (has hard deadlines)
- `project` — Side projects (BabyBaton, graphql-jit, emailresponder)
- `meal` — Meal planning, shopping, cooking
- `baby` — Quality time with daughter
- `exercise` — Daily movement
- `admin` — Everything else

---

### `routines`

Reusable templates for tasks that repeat on a schedule. A routine can generate
task instances automatically, or just inform the planner that this happens daily.

```sql
CREATE TABLE routines (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title       TEXT NOT NULL,
  category    TEXT NOT NULL,

  -- Frequency
  frequency   TEXT NOT NULL,         -- 'daily' | 'weekdays' | 'weekly' | 'custom'
  days_of_week  INT[],               -- [1,2,3,4,5] = Mon–Fri; [0,6] = weekend

  -- Time preference (hint for the planner, not a hard constraint)
  preferred_time_of_day  TEXT,       -- 'morning' | 'midday' | 'afternoon' | 'evening' | 'any'
  preferred_duration_min INT,        -- e.g. 45 (minutes)

  notes       TEXT,
  is_active   BOOLEAN DEFAULT true,
  created_at  TIMESTAMPTZ DEFAULT now()
);
```

**Example routines to seed:**
- Daily exercise — daily, morning, 45 min
- Baby quality time — daily, any (best post-nap), 45 min
- Meal planning — weekly (Sunday), 30 min
- Meta interview prep — weekdays, morning, 60 min (until deadline)

---

### `context_entries`

Persistent facts about the user's life that should always inform plan generation.
Think of these as "things the planner always knows." Categorized and editable.

```sql
CREATE TABLE context_entries (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  category    TEXT NOT NULL,         -- 'life' | 'constraints' | 'equipment' | 'preferences' | 'custom'
  key         TEXT NOT NULL,         -- short label, e.g. "nanny_hours"
  value       TEXT NOT NULL,         -- the actual content
  is_active   BOOLEAN DEFAULT true,
  created_at  TIMESTAMPTZ DEFAULT now(),
  updated_at  TIMESTAMPTZ DEFAULT now()
);
```

**Seed data (pre-populate on first run):**

| category | key | value |
|---|---|---|
| life | baby | 6-month-old daughter. Nanny present 9am–4pm Mon–Fri. Best quality time after naps (~10am, ~2pm). |
| life | family | Partner Elijah. Family of household. Cooking Indian food regularly. |
| constraints | work_window | Deep focus work: 9am–4pm only (nanny present). Before 9 and after 4 = family time. |
| constraints | location | Toronto, Canada. |
| constraints | energy | New parent. Cap deep cognitive work at 5h/day max. Hardest tasks go first in the morning. |
| constraints | dinner_prep | Dinner is always prepped after 16:00. This is a fixed daily block, not optional. Duration ~45–60 min. |
| constraints | evening_window | Baby typically asleep by ~8pm. After cleanup, there is a light evening window (~20:00–22:00). This time is primarily for rest and unwinding — do NOT schedule deep focus work here. Light tasks are acceptable: reading, light job search browsing, low-effort admin, meal planning, watching a talk. Max one light task per evening; relaxation comes first. |
| equipment | kitchen | Full Indian kitchen setup. All standard utensils. Pressure cooker, tawa, etc. |
| preferences | planning_style | Time-blocked days. Buffers between intense sessions. Interview prep is non-negotiable daily. |

---

### `day_plans`

Stores generated plans so they persist and can be resumed across sessions.

```sql
CREATE TABLE day_plans (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  plan_date   DATE NOT NULL UNIQUE,
  status      TEXT NOT NULL DEFAULT 'draft',  -- 'draft' | 'accepted'
  blocks      JSONB NOT NULL,        -- array of time blocks (see Block schema below)
  created_at  TIMESTAMPTZ DEFAULT now(),
  updated_at  TIMESTAMPTZ DEFAULT now()
);
```

**Plan status:**
- `draft` — plan is being shaped via chat. Blocks are not yet actionable.
- `accepted` — plan is locked in for execution. Blocks can be skipped or adjusted.

**Block JSON schema:**
```json
{
  "id": "uuid",
  "time": "09:00",
  "duration": 60,
  "title": "Meta DP problem — longest increasing subsequence variants",
  "category": "interview",
  "task_id": "uuid-or-null",
  "routine_id": "uuid-or-null",
  "notes": "Focus on O(n log n) optimization",
  "skipped": false
}
```

**Block actions (after plan is accepted):**

1. **Skip** — "I didn't do this / not going to do this." Sets `skipped: true`. Skipped
   blocks don't contribute to `actual_minutes` on the linked task. Skipped blocks with
   a `task_id` trigger carry-over review the next day.

2. **Adjust duration** — "I spent more/less time than planned." User edits `duration`
   to reflect reality. The updated duration is what gets added to `actual_minutes`.

Blocks that are not skipped are assumed done when their time passes. No explicit
"mark as done" action needed.

---

### `plan_messages`

Stores the chat conversation for plan generation and refinement.

```sql
CREATE TABLE plan_messages (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  plan_id     UUID NOT NULL REFERENCES day_plans(id) ON DELETE CASCADE,
  role        TEXT NOT NULL,          -- 'user' | 'assistant'
  content     TEXT NOT NULL,          -- user message or Claude's response
  created_at  TIMESTAMPTZ DEFAULT now()
);
```

The first user message replaces the old `morningNote` concept. Subsequent messages
are refinements ("move interview prep to after lunch", "make it a lighter day").

---

### `task_conversations`

Stores the chat conversation for AI-assisted task scoping.

```sql
CREATE TABLE task_conversations (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  parent_task_id  UUID REFERENCES tasks(id) ON DELETE CASCADE,  -- linked after tasks are created
  status      TEXT NOT NULL DEFAULT 'active',  -- 'active' | 'completed'
  created_at  TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE task_messages (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID NOT NULL REFERENCES task_conversations(id) ON DELETE CASCADE,
  role            TEXT NOT NULL,      -- 'user' | 'assistant'
  content         TEXT NOT NULL,
  created_at      TIMESTAMPTZ DEFAULT now()
);
```

**Task scoping flow:**
1. User starts a conversation describing a goal
2. Claude asks clarifying questions (scope, deliverables, timeline)
3. User answers
4. Claude proposes subtasks with time estimates as a structured response
5. User reviews, adjusts, and confirms
6. System creates parent task + subtasks, links conversation to parent

---

## GraphQL Schema

```graphql
# schema.graphqls

scalar UUID
scalar Date
scalar DateTime

# ─── Enums ───────────────────────────────────────────────

enum Category {
  JOB
  INTERVIEW
  PROJECT
  MEAL
  BABY
  EXERCISE
  ADMIN
}

enum Priority {
  HIGH
  MEDIUM
  LOW
}

enum DeadlineType {
  HARD     # specific date
  HORIZON  # within N days
}

enum Frequency {
  DAILY
  WEEKDAYS
  WEEKLY
  CUSTOM
}

enum TimeOfDay {
  MORNING
  MIDDAY
  AFTERNOON
  EVENING
  ANY
}

enum ContextCategory {
  LIFE
  CONSTRAINTS
  EQUIPMENT
  PREFERENCES
  CUSTOM
}

enum PlanStatus {
  DRAFT
  ACCEPTED
}

# ─── Types ───────────────────────────────────────────────

type Task {
  id:               UUID!
  title:            String!
  category:         Category!
  priority:         Priority!
  parentId:         UUID
  parent:           Task
  subtasks:         [Task!]!
  estimatedMinutes: Int
  actualMinutes:    Int!
  deadlineType:     DeadlineType
  deadlineDate:     Date
  deadlineDays:     Int
  notes:            String
  isRoutine:        Boolean!
  routine:          Routine
  timesDeferred:    Int!
  isCompleted:      Boolean!
  completedAt:      DateTime
  createdAt:        DateTime!
  updatedAt:        DateTime!
}

type Routine {
  id:                    UUID!
  title:                 String!
  category:              Category!
  frequency:             Frequency!
  daysOfWeek:            [Int!]
  preferredTimeOfDay:    TimeOfDay
  preferredDurationMin:  Int
  notes:                 String
  isActive:              Boolean!
}

type ContextEntry {
  id:        UUID!
  category:  ContextCategory!
  key:       String!
  value:     String!
  isActive:  Boolean!
  createdAt: DateTime!
}

type PlanBlock {
  id:        String!
  time:      String!
  duration:  Int!
  title:     String!
  category:  Category!
  taskId:    UUID
  routineId: UUID
  notes:     String
  skipped:   Boolean!
}

type DayPlan {
  id:        UUID!
  planDate:  Date!
  status:    PlanStatus!
  blocks:    [PlanBlock!]!
  messages:  [PlanMessage!]!
  createdAt: DateTime!
  updatedAt: DateTime!
}

type PlanMessage {
  id:        UUID!
  role:      String!
  content:   String!
  createdAt: DateTime!
}

type TaskConversation {
  id:           UUID!
  parentTaskId: UUID
  status:       String!
  messages:     [TaskMessage!]!
  createdAt:    DateTime!
}

type TaskMessage {
  id:        UUID!
  role:      String!
  content:   String!
  createdAt: DateTime!
}

# ─── Inputs ──────────────────────────────────────────────

input CreateTaskInput {
  title:            String!
  category:         Category!
  priority:         Priority!
  parentId:         UUID
  estimatedMinutes: Int            # defaults to 60 if not provided
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

input CreateRoutineInput {
  title:                String!
  category:             Category!
  frequency:            Frequency!
  daysOfWeek:           [Int!]
  preferredTimeOfDay:   TimeOfDay
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

input UpsertContextInput {
  category: ContextCategory!
  key:      String!
  value:    String!
}

input UpdateBlockInput {
  skipped:  Boolean
  time:     String
  duration: Int
  notes:    String
}

# ─── Query / Mutation ────────────────────────────────────

type Query {
  tasks(category: Category, includeCompleted: Boolean): [Task!]!
  task(id: UUID!): Task
  routines(activeOnly: Boolean): [Routine!]!
  contextEntries(category: ContextCategory): [ContextEntry!]!
  dayPlan(date: Date!): DayPlan          # null if not yet generated
  recentPlans(limit: Int): [DayPlan!]!
  taskConversation(id: UUID!): TaskConversation
}

type Mutation {
  # Tasks
  createTask(input: CreateTaskInput!): Task!
  updateTask(id: UUID!, input: UpdateTaskInput!): Task!
  deleteTask(id: UUID!): Boolean!
  completeTask(id: UUID!): Task!

  # Routines
  createRoutine(input: CreateRoutineInput!): Routine!
  updateRoutine(id: UUID!, input: UpdateRoutineInput!): Routine!
  deleteRoutine(id: UUID!): Boolean!

  # Context
  upsertContext(input: UpsertContextInput!): ContextEntry!
  deleteContext(id: UUID!): Boolean!

  # Plan chat — single interface for generation, refinement, and mid-day replanning
  sendPlanMessage(date: Date!, message: String!): DayPlan!
  acceptPlan(date: Date!): DayPlan!

  # Block actions (only on accepted plans)
  skipBlock(planId: UUID!, blockId: String!): DayPlan!
  updateBlock(planId: UUID!, blockId: String!, input: UpdateBlockInput!): DayPlan!

  # Task scoping chat
  startTaskConversation(message: String!): TaskConversation!
  sendTaskMessage(conversationId: UUID!, message: String!): TaskConversation!
  confirmTaskBreakdown(conversationId: UUID!): [Task!]!  # creates parent + subtasks

  # Carry-over
  resolveSkippedBlock(planId: UUID!, blockId: String!, intentional: Boolean!): Boolean!
}
```

---

## Planner Service (AI Integration)

File: `backend/planner/planner.go`

The planner handles two types of AI conversations:
1. **Plan chat** — generating and refining day plans
2. **Task scoping chat** — breaking down goals into subtasks

### Plan Chat: `SendPlanMessage(date, message)`

This is the unified entry point for all plan interactions. It handles:
- **First message of the day** — generates a fresh plan
- **Subsequent messages (draft)** — refines the plan based on feedback
- **Messages on accepted plans** — mid-day replanning (e.g., "baby's sick, cancel afternoon")

**Prompt construction (system prompt, built fresh each call):**

1. Pull all active `context_entries` → format as bullet list
2. Pull active `routines` → note which apply to today (day-of-week filter)
3. Pull `tasks` where `is_completed = false` and `parent_id IS NOT NULL` (subtasks)
   or `parent_id IS NULL AND estimated_minutes IS NOT NULL` (standalone), ordered by:
   - Priority DESC
   - Deadline urgency (hard dates first, horizon tasks by proximity)
4. For each task, compute `remaining_minutes = estimated_minutes - actual_minutes`
5. Pull carry-over tasks (skipped blocks from previous days)
6. Load existing chat history (`plan_messages` for this plan)

```
You are a daily planning assistant for {user_name}. Your job is to create a realistic,
time-blocked day plan based on their context, routines, and task backlog.

CONTEXT (treat these as ground truth — they define {user_name}'s constraints, life
situation, and preferences. Plan around them.):
{context_entries formatted as: "- {key}: {value}"}

TODAY'S ROUTINES (non-negotiable unless {user_name} says otherwise):
{applicable routines formatted as: "- {title} [{category}] — {preferred_duration_min} min, preferred: {preferred_time_of_day}"}

TASK BACKLOG (ordered by priority/urgency):
{tasks formatted as: "- {title} [{category}] — {remaining_minutes} min remaining (of {estimated_minutes} min) — priority: {priority} — deadline: {deadline info}"}

CARRY-OVER TASKS (skipped from previous days, not intentional):
{list with times_deferred and effective deadline}

Rules for carry-over tasks:
- Schedule at least one carry-over task today, even if backlog is heavy
- Prefer to schedule the most-deferred one first
- If times_deferred >= 3, it goes in the morning block, no exceptions

DEFERRED TASK ESCALATION:
- times_deferred = 1: treat as one priority level higher than assigned
- times_deferred = 2: treat as HIGH priority regardless of assigned priority
- times_deferred >= 3: flag in the plan title with "⚑ OVERDUE" and schedule in the
  first available slot of the day

PLANNING RULES:
- Read the CONTEXT section carefully. It tells you when deep work is possible,
  when family time is, energy limits, and scheduling constraints. Follow it.
- Routines have a preferred time of day, but treat it as a preference, not a
  constraint. If the preferred slot is packed with higher-priority work, move
  the routine to another available slot. Routines marked "any" should fill
  gaps in the schedule.
- Hardest cognitive tasks go in the earliest available deep work slot.
- Always leave 15-min buffers between intense 90+ min sessions.
- Be SPECIFIC in block titles (e.g., "Meta LC: Coin Change II — bottom-up DP"
  not "Interview prep").
- Don't schedule more than is realistic given the energy constraints in CONTEXT.
- A task with remaining_minutes > block duration CAN be scheduled — schedule what
  fits today, the rest goes on subsequent days.
- Never schedule anything in a past time slot.

RESPONSE FORMAT:
Respond ONLY with a JSON array. No explanation. No markdown. Just the array.
Each element: { "id": "uuid-v4", "time": "HH:MM", "duration": 60, "title": "...",
"category": "interview", "task_id": "uuid-or-null", "routine_id": "uuid-or-null",
"notes": "...", "skipped": false }
```

**On subsequent messages (refinement or replanning):**

The system prompt stays the same. The conversation history (from `plan_messages`) is
sent as the message array. For accepted plans being replanned, also include the current
blocks with their skip/duration status, and instruct Claude to preserve the past
(non-skipped) blocks and only reschedule remaining ones.

### Task Scoping Chat: `SendTaskMessage(conversationId, message)`

**System prompt:**

```
You are helping {user_name} break down a goal into concrete, schedulable tasks.

Your job:
1. Ask clarifying questions to understand the scope (what needs to be done, what's
   the deliverable, what are the dependencies between steps)
2. Ask about the deadline
3. Propose a breakdown of subtasks, each with:
   - A specific, actionable title
   - An estimated duration in minutes
   - The category (job|interview|project|meal|baby|exercise|admin)
   - Any notes

When proposing the breakdown, respond with a JSON object:
{
  "status": "proposal",
  "parent": { "title": "...", "category": "...", "priority": "...",
              "deadline_type": "hard|horizon|null", "deadline_date": "YYYY-MM-DD|null",
              "deadline_days": N|null },
  "subtasks": [
    { "title": "...", "estimated_minutes": N, "category": "...", "notes": "..." },
    ...
  ]
}

When asking questions, respond with:
{ "status": "question", "message": "..." }

Keep it conversational. Don't ask more than 2-3 questions before proposing.
Adjust if {user_name} gives feedback on the proposal.
```

### Error handling

- Validate JSON response — if unparseable, retry once with a stricter prompt
- Store raw Claude response alongside parsed blocks for debugging
- Surface error to frontend gracefully ("Couldn't parse plan, please try again")

---

## Frontend Pages

### `/` — Today

**Before plan exists (or plan is draft):**
- Split layout: **chat panel** (left/bottom) + **plan preview** (right/top)
- Chat input for describing the day / giving feedback on the plan
- Plan preview updates live with each Claude response
- "Accept Plan" button to lock in the plan

**After plan is accepted:**
- Full-width time-blocked plan view
- Live "NOW" indicator on current block (based on current time vs block time)
- Category color coding + duration display
- Block actions:
  - **Skip** button — marks block as skipped
  - **Adjust duration** — inline edit of duration (click to change)
- "Something came up" button → reopens the chat panel for replanning
- Chat panel shows conversation history when reopened
- Browser notifications scheduled for each block's start time (Web Notifications API)
- Notifications cleared and rescheduled on replan

### `/backlog` — Task Backlog

- Tasks grouped by parent → subtasks shown nested under parent heading
- Standalone tasks shown in their own section or grouped by category
- Full CRUD: create (manual quick-add), edit (inline), delete, mark complete
- "Scope with AI" button → opens task scoping chat
- Deadline display: hard dates shown as `Mar 15`, horizons as `within 2 weeks`
- Progress display on subtasks: `actual_minutes / estimated_minutes` (e.g., "2hr / 3hr")
- Filter by category, priority
- Completed tasks collapsible at bottom
- Parent completion: auto-completes when all subtasks done, shown with progress indicator

### `/routines` — Routines

- List all routines
- Create/edit/delete
- Toggle active/inactive
- Shows "applies today" indicator based on day-of-week

### `/context` — Persistent Context

- All context entries grouped by category
- Inline edit (click to edit value)
- Add new entries with category + key + value
- Toggle active/inactive (inactive = not sent to planner)
- Shows last-updated timestamp

### `/history` — Past Plans

- Calendar or list of past generated plans
- Click a date to view the plan (read-only)
- Shows which blocks were skipped vs completed
- Useful for seeing what got done, what got skipped

---

## Carry-Over & Missed Task Logic

### Flow: triggered when opening Today page for a new day

When the user opens the app on a new day, before the plan chat appears, the app checks
the most recent past `day_plan` for blocks where `skipped = true` and that have a `task_id`.

If any exist, the frontend shows a **Skipped Tasks Review** screen. This is a blocking
step — the plan chat won't appear until it's dismissed.

```
┌─────────────────────────────────────────────────────┐
│  Yesterday you skipped 3 blocks:                    │
│                                                     │
│  ◈ Meta LC: Coin Change II          [interview]     │
│    → Intentional?  Yes / No                         │
│    → If No: keep as-is / update deadline            │
│                                                     │
│  ◈ Tailor resume for Airbnb         [job]           │
│    → Intentional?  Yes / No                         │
│                                                     │
│  ◈ 30 min walk                      [exercise]      │
│    → Intentional?  Yes / No                         │
│                                                     │
│                         [ Done → Start Planning ]   │
└─────────────────────────────────────────────────────┘
```

Note: Blocks that were NOT skipped are assumed completed. They don't appear in this
review. Their duration was already added to `actual_minutes` on the linked task.

### Resolution options per skipped block

**"Yes, intentional"**
- Do nothing. Task stays in backlog at same priority.
- `times_deferred` is NOT incremented (user chose this).

**"No, I just didn't get to it" (default)**
- Increment `tasks.times_deferred`
- Update `tasks.last_deferred_at`
- Priority escalation rules (see Planner Service section)

**"Update deadline" (optional sub-action on "No")**
- Inline deadline editor appears: user can change to a new hard date or new horizon.
- After editing, still increments `times_deferred`.

### Soft deadline pressure

For `horizon` deadline tasks, compute effective urgency at plan-generation time:

```go
// effectiveDaysRemaining shrinks by 1 for each unintentional deferral
func effectiveDaysRemaining(task Task) int {
    base := task.DeadlineDays
    return base - task.TimesDeferred
}
```

Pass `effectiveDaysRemaining` to the planner prompt, not the raw `deadline_days`.
So a "within 14 days" task that's been skipped 3 times is presented to Claude as
"within 11 days" — it naturally surfaces sooner without any special-case logic.

If `effectiveDaysRemaining <= 3`, treat the task as a hard deadline for scheduling
purposes (same rules as `deadline_type = 'hard'`).

### Updating `actual_minutes`

When a block's time passes and it is not skipped, the system adds the block's `duration`
to the linked task's `actual_minutes`. This happens:
- When the user accepts a plan (for any blocks already in the past)
- At end of day (batch update for all non-skipped blocks)
- When a block is explicitly skipped (confirms it should NOT count)

If a user adjusts a block's duration, the adjusted value is what counts toward
`actual_minutes`.

### Browser notifications

Uses the Web Notifications API (tab must be open). No service worker or push
infrastructure needed.

**Setup:** On first visit, request `Notification.requestPermission()`. Store the
permission state — don't re-prompt if denied.

**Block start notifications:** When a plan is accepted (or on page load if plan is
already accepted), schedule a `setTimeout` for each upcoming block's start time.
When the timeout fires:

```
🔔 "Time for: Meta LC — Coin Change II (90 min)"
```

Clicking the notification focuses the app tab.

**Re-scheduling:** If the plan is replanned (chat reopened → new plan accepted),
clear all existing timeouts and reschedule from the new blocks.

**End-of-day nudge:** At 4pm (or configurable), fire a notification:
"3 blocks haven't been marked — skip any you didn't do, or they'll count as completed."
Only fires if there are non-skipped future blocks remaining.

---

## Deployment (Railway)

### Services

1. **PostgreSQL** — Railway managed Postgres plugin
2. **Backend** — Go service, `PORT` from Railway env
3. **Frontend** — Static Vite build served by the Go backend (embed `frontend/dist`)

### Single-service deployment

Serve the frontend from Go using `//go:embed` on the Vite build output.
This keeps it to one Railway service + the database plugin.

```go
//go:embed frontend/dist
var frontend embed.FS
```

### Environment variables

```
DATABASE_URL=              # Railway provides automatically
ANTHROPIC_API_KEY=         # System default key (used when user has no personal key)
ANTHROPIC_MODEL=           # Default: claude-sonnet-4-6
CLERK_SECRET_KEY=          # Clerk backend secret (sk_live_...)
VITE_CLERK_PUBLISHABLE_KEY= # Clerk frontend key (pk_live_...)
DAILY_AI_CAP=              # Max AI calls per user per day (default: 50)
OWNER_CLERK_ID=            # Clerk user ID of the owner (for data migration)
PORT=8080
```

### Authentication (Clerk)

Multi-user auth via Clerk (SaaS). Invite-only sign-up configured in the Clerk dashboard.
Supports email/password + Google + Apple sign-in.

- Frontend: `@clerk/clerk-react` provides `<ClerkProvider>`, `<SignIn />`, `useAuth()`
- Backend: Clerk JWT verified in HTTP middleware via `github.com/clerk/clerk-sdk-go/v2`
- User identity stored in `context.Context` via `backend/identity/` package
- `ScopedQueries` wrapper (`backend/db/scoped.go`) reads user ID from context and injects
  `user_id` into every sqlc query — store interfaces and resolvers are unchanged
- Zero credentials stored on Railway — all auth data lives on Clerk's servers

### Multi-user data model

- `users` table maps Clerk IDs to internal UUIDs
- Every data table has a `user_id UUID NOT NULL REFERENCES users(id)` column
- All sqlc queries include `WHERE user_id = @user_id` for data isolation
- `plan_messages` and `task_messages` are scoped transitively via their parent FK
- Per-user Google Calendar OAuth (no more singleton `google_auth` constraint)

### Per-user AI and usage caps

- `users.anthropic_api_key` (nullable) — if set, planner uses user's key; otherwise falls back
  to system `ANTHROPIC_API_KEY`
- Daily AI call cap tracked on the `users` table — resets each day
- `WithUserCap` decorator wraps `PlannerService` to enforce cap + key fallback transparently

---

## Database Migrations

Use `golang-migrate`. Migration files in `backend/db/migrations/`.

```
001_create_routines.up.sql
001_create_routines.down.sql
002_create_tasks.up.sql             # includes parent_id, estimated_minutes, actual_minutes, times_deferred
002_create_tasks.down.sql
003_create_context_entries.up.sql
003_create_context_entries.down.sql
004_create_day_plans.up.sql         # includes status, block JSON with skipped field
004_create_day_plans.down.sql
005_create_plan_messages.up.sql
005_create_plan_messages.down.sql
006_create_task_conversations.up.sql  # task_conversations + task_messages
006_create_task_conversations.down.sql
007_seed_context.up.sql
007_seed_context.down.sql
```

Run on startup: `migrate -path db/migrations -database $DATABASE_URL up`

---

## Bounded Context Spec Format

Each bounded context gets its own spec file at `specs/{context-name}.md`.
Claude Code must read the relevant spec file before writing any code for that context.

This is a small single-user app — keep specs lean. Omit sections that don't add value
(e.g. no state machine needed for simple CRUD contexts).

```markdown
# Spec: {Context Name}

## Bounded Context

Owns: {tables, resolvers, and business logic this context is responsible for}
Does not own: {explicit exclusions — what to leave alone}
Depends on: {other contexts this reads from or calls}
Produces: {GraphQL mutations/queries exposed, data written}

## Contracts

### Input
{GraphQL inputs consumed, data read from DB}

### Output
{GraphQL types returned, DB writes, side effects}

### Data Model
{SQL for tables owned by this context — only what's new or changed here}

## Behaviors (EARS syntax)

- When {trigger}, the system shall {response}.
- While {state}, the system shall {behavior}.
- Where {condition}, the system shall {behavior}.
- If {condition} then {behavior} else {alternative}.
- The system shall {behavior}.

## Decision Table (if applicable)

| Input | Condition | Output |
|-------|-----------|--------|

## Test Anchors

{Explicit scenarios that MUST pass. These become TDD seeds.}

1. Given {precondition}, when {action}, then {expected result}.
2. ...
```

### Notes for Claude Code

- If a section would just restate the GraphQL schema with no added information, skip it.
- Decision tables are only useful when there are 3+ branching conditions (e.g. carry-over logic). Skip for simple CRUD.
- State machines are only needed for entities with explicit lifecycle states (e.g. a future alert system). Not needed here.
- Test anchors are mandatory — every context needs at least 3.
- Behaviors must cover both the happy path and the primary error/edge case.

---

## Build Order for Claude Code

Build in this dependency order:

**Wave 1 — Foundation**
- Database migrations (all 7)
- Go module setup + gqlgen config
- sqlc config + query files

**Wave 2 — Backend core**
- GraphQL schema
- gqlgen code generation
- Task CRUD resolvers + DB queries (including parent/subtask hierarchy)
- Routine CRUD resolvers + DB queries

**Wave 3 — Context + Plans + AI**
- Context entry CRUD resolvers
- DayPlan model + DB queries
- Plan messages model + DB queries
- Planner service — plan chat (Claude API integration)
- Planner service — task scoping chat
- sendPlanMessage + acceptPlan mutations
- Task conversation mutations
- Block actions (skip, update duration)

**Wave 4 — Frontend**
- Vite + React + Tailwind + Apollo setup
- GraphQL codegen config
- Today page (/, plan chat + block view)
- Backlog page (/backlog, including "Scope with AI" chat)

**Wave 5 — Polish**
- Routines page (/routines)
- Context page (/context)
- History page (/history)
- Skipped Tasks Review screen (carry-over flow)
- End-of-day nudge (client-side reminder)
- Railway deployment config

**Wave 6 — Hardening**
- Error boundaries + loading states
- actual_minutes computation (end-of-day batch + real-time)
- Simple password auth
- README
