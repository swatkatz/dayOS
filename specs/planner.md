# Spec: Planner

## Bounded Context

Owns: `backend/planner/planner.go` — all Anthropic Claude API integration, prompt construction, JSON response parsing, retry logic. Resolver logic for `sendPlanMessage`, `acceptPlan`, `startTaskConversation`, `sendTaskMessage`, `confirmTaskBreakdown`.

Does not own: `day_plans` table schema or block CRUD (owned by `day-plans`), `tasks` table schema or task CRUD (owned by `tasks`), `plan_messages`/`task_conversations`/`task_messages` table schemas (owned by `day-plans` and `tasks` respectively), carry-over resolution (owned by `carry-over`), frontend rendering

Depends on:
- `foundation` — database schema, sqlc queries
- `validation` — `validate.FormatContextData()` for safe prompt embedding, `validate.ValidateAIOutput()` for response validation, generated system instruction constants
- `tasks` — reads incomplete tasks for backlog, creates parent + subtasks on task breakdown confirmation
- `routines` — reads active routines applicable to today
- `context` — reads all active context entries
- `day-plans` — reads/writes `day_plans` and `plan_messages`, reads block state for replanning

Produces:
- GraphQL mutations: `sendPlanMessage`, `acceptPlan`, `startTaskConversation`, `sendTaskMessage`, `confirmTaskBreakdown`
- Writes to: `day_plans` (blocks JSONB), `plan_messages`, `task_conversations`, `task_messages`, `tasks` (on confirm breakdown)

## Contracts

### Input

**`sendPlanMessage(date: Date!, message: String!): DayPlan!`**

Reads:
- All active `context_entries`
- Active `routines` filtered by today's day-of-week
- Incomplete tasks: subtasks (`parent_id IS NOT NULL, is_completed = false`) and standalone tasks (`parent_id IS NULL, estimated_minutes IS NOT NULL, is_completed = false`), ordered by priority DESC then deadline urgency
- Carry-over tasks: tasks linked to skipped blocks from previous day plans where skip was not intentional (i.e., `times_deferred > 0` and `last_deferred_at` is recent)
- Existing `plan_messages` for this date's plan (conversation history)
- Current `day_plans.blocks` (for replanning on accepted plans)

**`startTaskConversation(message: String!): TaskConversation!`**

No DB reads beyond creating the conversation record.

**`sendTaskMessage(conversationId: UUID!, message: String!): TaskConversation!`**

Reads:
- All `task_messages` for this conversation (conversation history)

**`confirmTaskBreakdown(conversationId: UUID!): [Task!]!`**

Reads:
- Last assistant message in the conversation (must contain a `"status": "proposal"` JSON)

### Output

**`sendPlanMessage`** returns `DayPlan!` — the plan with updated blocks and the full message history. Side effects:
- Upserts `day_plans` row for the date (creates if first message, updates blocks on subsequent)
- Inserts user message and assistant response into `plan_messages`

**`acceptPlan`** returns `DayPlan!` with `status: "accepted"`. Side effect:
- Updates `day_plans.status` to `'accepted'`

**`startTaskConversation`** returns `TaskConversation!` with the first assistant message. Side effects:
- Creates `task_conversations` row with `status: 'active'`
- Inserts user message and assistant response into `task_messages`

**`sendTaskMessage`** returns `TaskConversation!` with updated messages. Side effects:
- Inserts user message and assistant response into `task_messages`

**`confirmTaskBreakdown`** returns `[Task!]!` — the created parent + subtasks. Side effects:
- Creates parent task and subtasks in `tasks` table
- Links `task_conversations.parent_task_id` to the created parent
- Updates `task_conversations.status` to `'completed'`

### Data Model

No new tables — this context uses tables defined by `foundation`. The planner is a service layer that orchestrates reads/writes across existing tables.

## Planner Service Architecture

File: `backend/planner/planner.go`

### Anthropic API Client

- Use the Anthropic Go SDK (`github.com/anthropics/anthropic-sdk-go`)
- Model from `ANTHROPIC_MODEL` env var, default `claude-sonnet-4-6`
- Never hardcode the model string

### Plan Chat: Prompt Construction

The system prompt is rebuilt fresh on every `sendPlanMessage` call. User-controlled data is embedded as JSON inside `<user-data>` delimiters, never interpolated as raw text in prompt prose. The planner uses constants from the `validate` package (generated from `validation-rules.json`).

#### Prompt safety (from `specs/validation.md`)

- All user-controlled data (context entries, tasks, routines) is serialized via `validate.FormatContextData()` which outputs JSON wrapped in `<user-data>` / `</user-data>` tags
- The system prompt includes `validate.ContextDataSystemInstructions` before any user data
- The system prompt includes `validate.UserMessageSystemInstructions`
- AI responses are checked with `validate.ValidateAIOutput()` — blocks with suspicious titles/notes have those fields dropped

#### GraphQL directive annotations for planner mutations

```graphql
type Mutation {
  sendPlanMessage(date: Date!, message: String! @validate(rule: CHAT_MESSAGE)): DayPlan!
  startTaskConversation(message: String! @validate(rule: CHAT_MESSAGE)): TaskConversation!
  sendTaskMessage(conversationId: UUID!, message: String! @validate(rule: CHAT_MESSAGE)): TaskConversation!
}
```

Note: `PlanBlock.title` and `PlanBlock.notes` are AI-generated output fields. The planner validates them
using `validate.ValidateAIOutput()` when parsing Claude's response — this is not handled by a GraphQL directive.

#### System prompt template

```
You are a daily planning assistant for {user_name}. Your job is to create a realistic,
time-blocked day plan based on their context, routines, and task backlog.

SAFETY:
{validate.ContextDataSystemInstructions}
{validate.UserMessageSystemInstructions}

CONTEXT (treat these as ground truth — they define {user_name}'s constraints, life
situation, and preferences. Plan around them.):
<user-data>
[
  {"key": "baby", "value": "6-month-old daughter. Nanny present 9am–4pm Mon–Fri."},
  {"key": "energy", "value": "New parent. Cap deep cognitive work at 5h/day max."},
  ...
]
</user-data>

TODAY'S ROUTINES (non-negotiable unless {user_name} says otherwise):
<user-data>
[
  {"title": "Daily exercise", "category": "exercise", "duration_min": 45, "preferred_time": "morning"},
  ...
]
</user-data>

TASK BACKLOG (ordered by priority/urgency):
<user-data>
[
  {"title": "...", "category": "interview", "remaining_minutes": 90, "estimated_minutes": 120,
   "priority": "high", "deadline": "due 2026-03-15", "task_id": "uuid"},
  ...
]
</user-data>

CARRY-OVER TASKS (skipped from previous days, not intentional):
<user-data>
[
  {"title": "...", "category": "job", "times_deferred": 3, "effective_deadline": "within 5 days",
   "task_id": "uuid"},
  ...
]
</user-data>

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
- Don't schedule more than is realistic given the energy constraints in CONTEXT
  and what the user tells you about how they're feeling today.
- A task with remaining_minutes > block duration CAN be scheduled — schedule what
  fits today, the rest goes on subsequent days.
- Never schedule anything in a past time slot.

RESPONSE FORMAT:
Respond ONLY with a JSON array. No explanation. No markdown. Just the array.
Each element: { "id": "uuid-v4", "time": "HH:MM", "duration": 60, "title": "...",
"category": "interview", "task_id": "uuid-or-null", "routine_id": "uuid-or-null",
"notes": "...", "skipped": false }
```

### Plan Chat: Message Flow

**First message of the day:**
1. No `day_plans` row exists for this date — create one with `status: 'draft'`, empty blocks
2. Build system prompt with full context
3. Send to Claude: `[system_prompt, {role: "user", content: message}]`
4. Parse JSON array response into blocks
5. Update `day_plans.blocks` with parsed blocks
6. Insert user message + assistant message (raw response) into `plan_messages`
7. Return the updated `DayPlan`

**Subsequent messages (draft plan — refinement):**
1. `day_plans` row exists with `status: 'draft'`
2. Build system prompt (same as above — always fresh)
3. Load all `plan_messages` for this plan as conversation history
4. Send to Claude: `[system_prompt, ...history, {role: "user", content: message}]`
5. Parse JSON array response, update blocks
6. Insert new user + assistant messages
7. Return updated `DayPlan`

**Messages on accepted plan (mid-day replanning):**
1. `day_plans` row exists with `status: 'accepted'`
2. Build system prompt, plus append this additional instruction:

```
REPLANNING CONTEXT:
The current plan has been accepted and is in progress. Here are the current blocks:
{blocks JSON with skip/duration status}

Current time: {HH:MM}

REPLANNING RULES:
- Blocks in the past (before current time) that are NOT skipped must be preserved exactly as-is.
- Blocks that are skipped can be replaced or removed.
- Only reschedule blocks from current time onward.
- The user is asking to adjust the remaining schedule.
```

3. Load conversation history, send to Claude
4. Parse response, update blocks
5. Set `status` back to `'draft'` (user must re-accept the revised plan)
6. Insert messages, return updated `DayPlan`

### Plan Chat: Task Backlog Formatting

For each task in the backlog:
- Compute `remaining_minutes = estimated_minutes - actual_minutes`
- Skip tasks where `remaining_minutes <= 0`
- For parent tasks: do NOT include the parent itself (it's not schedulable). Include its incomplete subtasks individually.
- For `horizon` deadline tasks: compute `effectiveDaysRemaining = deadline_days - times_deferred`. Display as `"within {effectiveDaysRemaining} days"`. If `effectiveDaysRemaining <= 3`, mark as `"URGENT — within {N} days"`
- For `hard` deadline tasks: display as `"due {deadline_date}"`. If deadline is within 3 days, mark as `"URGENT — due {date}"`
- Sort order: priority DESC (high > medium > low), then deadline urgency (hard dates first sorted by proximity, then horizon by effective days remaining), then `times_deferred` DESC

### Task Scoping Chat: Prompt & Flow

**System prompt (used for all task scoping conversations):**

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

**`startTaskConversation(message)`:**
1. Create `task_conversations` row with `status: 'active'`, `parent_task_id: NULL`
2. Send to Claude: `[system_prompt, {role: "user", content: message}]`
3. Parse response — expect either `{"status": "question", ...}` or `{"status": "proposal", ...}`
4. Insert user message + assistant response (raw) into `task_messages`
5. Return the conversation with messages

**`sendTaskMessage(conversationId, message)`:**
1. Load all `task_messages` for this conversation
2. Send to Claude: `[system_prompt, ...history, {role: "user", content: message}]`
3. Parse response, insert messages
4. Return updated conversation

**`confirmTaskBreakdown(conversationId)`:**
1. Find the last assistant message in the conversation
2. Parse it as JSON — must have `"status": "proposal"`
3. Create parent task from `proposal.parent`:
   - `title`, `category`, `priority` from proposal
   - `deadline_type`, `deadline_date`, `deadline_days` from proposal
   - `estimated_minutes` = NULL (parent tasks don't have this)
   - `parent_id` = NULL
4. Create each subtask from `proposal.subtasks`:
   - `title`, `estimated_minutes`, `category`, `notes` from proposal
   - `parent_id` = created parent's ID
   - `priority` = parent's priority (inherited)
5. Set `task_conversations.parent_task_id` = parent ID
6. Set `task_conversations.status` = `'completed'`
7. Return all created tasks (parent + subtasks)

### JSON Parsing & Retry Logic

**Plan chat responses:**
- Attempt to parse the assistant response as a JSON array of block objects
- Validate each block has required fields: `id`, `time`, `duration`, `title`, `category`
- Default `skipped` to `false` if missing, `task_id` and `routine_id` to `null` if missing

**Task scoping responses:**
- Attempt to parse as JSON object with `"status"` field
- If `status` is `"question"`, extract `"message"` field
- If `status` is `"proposal"`, validate `parent` and `subtasks` structure

**On parse failure (either chat type):**
1. Retry ONCE with a stricter prompt appended to the user message:

```
Your previous response was not valid JSON. You MUST respond with ONLY a valid JSON
array/object as specified in the system prompt. No markdown, no explanation, no code
fences. Just the raw JSON.
```

2. If the retry also fails, return a user-facing error: `"Couldn't parse AI response, please try again"`
3. Still store both the failed raw responses in `plan_messages`/`task_messages` for debugging (but with role `"assistant"`)

**Additional JSON robustness:**
- Strip leading/trailing whitespace before parsing
- Strip markdown code fences (` ```json ... ``` `) if present — Claude sometimes wraps JSON in fences despite instructions
- Handle both `null` and missing keys for optional fields

## Decision Table: sendPlanMessage Behavior

| Plan exists? | Plan status | Action |
|---|---|---|
| No | N/A | Create new `day_plans` row with `status: 'draft'`, generate fresh plan |
| Yes | `draft` | Refinement — rebuild prompt, include conversation history, update blocks |
| Yes | `accepted` | Replanning — add replanning context, preserve past blocks, set status back to `'draft'` |

## Decision Table: Task Scoping Response Handling

| `status` field | Action |
|---|---|
| `"question"` | Extract `message`, store as assistant response, return conversation |
| `"proposal"` | Store raw JSON as assistant response, return conversation (user can review then call `confirmTaskBreakdown`) |
| Missing/invalid | Trigger retry logic |

## Behaviors (EARS syntax)

- When `sendPlanMessage` is called for a date with no existing plan, the system shall create a new `day_plans` row with `status: 'draft'` and generate a plan by calling the Claude API with full context.
- When `sendPlanMessage` is called for a date with an existing draft plan, the system shall include all prior `plan_messages` as conversation history and update the plan blocks with Claude's response.
- When `sendPlanMessage` is called for a date with an accepted plan, the system shall include replanning instructions that preserve past non-skipped blocks and only reschedule from the current time onward, and shall set the plan status back to `'draft'`.
- When `sendPlanMessage` constructs the system prompt, the system shall pull ALL active `context_entries`, ALL applicable routines for today's day-of-week, and ALL incomplete schedulable tasks (subtasks + standalone), plus carry-over tasks.
- When constructing the system prompt, the system shall embed all user-controlled data (context entries, tasks, routines) as JSON arrays inside `<user-data>` / `</user-data>` delimiters using `validate.FormatContextData()`.
- When constructing the system prompt, the system shall include `validate.ContextDataSystemInstructions` and `validate.UserMessageSystemInstructions` in the SAFETY section before any user data.
- When constructing the CONTEXT section, the system shall present context entries as ground truth that defines constraints, life situation, and preferences, instructing the AI to plan around them.
- When constructing the PLANNING RULES section, the system shall instruct the AI to read the CONTEXT section for time windows and constraints, and to also factor in what the user says about their energy/mood in their chat message.
- When a routine's preferred time slot conflicts with higher-priority work, the system shall instruct the AI to move the routine to another available slot rather than treating preferred time as a hard constraint.
- When computing `remaining_minutes` for a task, the system shall subtract `actual_minutes` from `estimated_minutes` and exclude tasks where `remaining_minutes <= 0`.
- When formatting horizon deadline tasks for the prompt, the system shall compute `effectiveDaysRemaining = deadline_days - times_deferred` and use that value instead of the raw `deadline_days`.
- When `effectiveDaysRemaining <= 3` for a horizon task, the system shall present it as urgent in the prompt.
- When Claude returns a response that is not valid JSON, the system shall retry once with a stricter prompt instructing JSON-only output.
- If the retry also fails to produce valid JSON, the system shall return the error `"Couldn't parse AI response, please try again"` and still store both raw responses in the message history.
- When `acceptPlan` is called, the system shall update the plan's status to `'accepted'`.
- When `acceptPlan` is called on a plan that is already accepted, the system shall return the plan unchanged (idempotent).
- When `startTaskConversation` is called, the system shall create a new conversation record, call Claude with the task scoping system prompt, and return the conversation with both the user message and Claude's response.
- When `sendTaskMessage` is called, the system shall load the full conversation history and send it to Claude along with the new message.
- When `confirmTaskBreakdown` is called, the system shall parse the last assistant message as a proposal, create the parent task and all subtasks, link the conversation, and mark it as completed.
- When `confirmTaskBreakdown` is called but the last assistant message is not a valid proposal, the system shall return an error `"No valid task proposal found. Continue the conversation first."`.
- The system shall never hardcode the Anthropic model — it shall read from `ANTHROPIC_MODEL` env var with default `claude-sonnet-4-6`.
- The system shall strip markdown code fences from Claude responses before JSON parsing.
- When a parsed plan block is returned from Claude, the system shall call `validate.ValidateAIOutput()` on the `title` and `notes` fields.
- When `validate.ValidateAIOutput()` returns an error for a block field, the system shall set that field to an empty string rather than rejecting the entire plan.
- When including parent tasks in the backlog, the system shall NOT include the parent itself (not schedulable) and shall instead include its incomplete subtasks individually.

## Test Anchors

1. **Given** 3 active context entries, 2 applicable routines, and 5 incomplete tasks exist, **when** `sendPlanMessage` is called for today with `"Light day, just interview prep and exercise"`, **then** a `day_plans` row is created with `status: 'draft'`, blocks are a valid JSON array, and 2 `plan_messages` rows are inserted (user + assistant).

2. **Given** a draft plan exists for today with 2 messages in history, **when** `sendPlanMessage` is called with `"Move interview prep to after lunch"`, **then** the Claude API receives 5 messages (system + 2 history + new user + expects response), blocks are updated, and 2 new `plan_messages` are added.

3. **Given** an accepted plan exists for today with 3 blocks (one at 09:00 completed, one at 11:00 upcoming, one at 14:00 upcoming), **when** `sendPlanMessage` is called at 10:30 with `"Cancel the afternoon, baby is sick"`, **then** the replanning prompt includes current blocks with instructions to preserve the 09:00 block, and `day_plans.status` is set back to `'draft'`.

4. **Given** Claude returns `"Here's your plan: ```json [...] ```"` (JSON wrapped in markdown fences), **when** the system parses the response, **then** it strips the fences and successfully parses the JSON array.

5. **Given** Claude returns completely invalid text (not JSON even after stripping), **when** the system retries with the stricter prompt, **and** the retry also returns invalid text, **then** the mutation returns an error message and both raw responses are stored in `plan_messages`.

6. **Given** a task scoping conversation where the last assistant message is a valid proposal JSON with 1 parent and 3 subtasks, **when** `confirmTaskBreakdown` is called, **then** 4 tasks are created (1 parent + 3 subtasks), subtasks have `parent_id` set to the parent's ID, the conversation's `parent_task_id` is set, and the conversation status is `'completed'`.

7. **Given** a task scoping conversation where the last assistant message has `"status": "question"`, **when** `confirmTaskBreakdown` is called, **then** the system returns an error `"No valid task proposal found. Continue the conversation first."`.

8. **Given** a task with `deadline_type: 'horizon'`, `deadline_days: 14`, and `times_deferred: 3`, **when** the system formats the backlog for the prompt, **then** the task is displayed as `"within 11 days"` (not 14).

9. **Given** a parent task with 2 incomplete subtasks and 1 completed subtask, **when** the system builds the task backlog for the prompt, **then** only the 2 incomplete subtasks appear in the backlog (not the parent, not the completed subtask).

10. **Given** no `ANTHROPIC_MODEL` env var is set, **when** the planner initializes, **then** it defaults to `claude-sonnet-4-6`.

11. **Given** the system prompt is constructed with context entries containing scheduling constraints, **when** a user message says `"Exhausted today, keep it very light"`, **then** the planning rules instruct the AI to factor in both the CONTEXT constraints and the user's stated energy level when determining how much to schedule.

12. **Given** the system prompt is being constructed, **when** context entries are embedded, **then** they appear as a JSON array inside `<user-data>` / `</user-data>` tags, not as inline bullet text.

13. **Given** Claude returns a block with `title` containing `"Send to user@example.com for review"`, **when** the response is parsed, **then** `validate.ValidateAIOutput("title", ...)` flags the email pattern and the title is set to `""`.

14. **Given** Claude returns a block with a 250-character `notes` field, **when** the response is parsed, **then** `validate.ValidateAIOutput("notes", ...)` passes and the notes are stored as-is.
