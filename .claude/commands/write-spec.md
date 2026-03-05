# Write Spec: $ARGUMENTS

You are writing a bounded context spec for DayOS, a personal daily planning app.

## Setup

1. Read `docs/DESIGN.md` thoroughly — it is the source of truth for all requirements.
2. Read the current `CLAUDE.md` for architecture rules and conventions.
3. Read any existing specs in `specs/` to avoid contradictions or duplication.

## Your Task

Write the spec for: **$ARGUMENTS**

The spec file goes at `specs/$ARGUMENTS.md`. Follow the template from the "Bounded Context Spec Format" section in DESIGN.md:

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

## Rules

- **Be precise.** Extract exact details from DESIGN.md — field names, enum values, SQL types, prompt text. Don't paraphrase when quoting is better.
- **Skip empty sections.** If a section would just restate the GraphQL schema with no added information, omit it. Decision tables only when 3+ branching conditions exist. State machines only for entities with explicit lifecycle states.
- **Test anchors are mandatory.** At least 3 per spec. Cover happy path + primary error/edge case.
- **Behaviors must cover both happy path and error cases.**
- **Cross-reference dependencies.** If this context depends on another spec, name it explicitly (e.g., "Depends on: tasks, routines").
- **Include the GraphQL operations** this context owns (queries + mutations) in the Contracts section.
- **Include the exact SQL** for tables this context owns in Data Model.
- **For frontend specs:** describe components, routes, user interactions, Apollo queries/mutations used, and visual requirements (colors, layout). No SQL needed.
- **For the planner spec:** include the full prompt templates from DESIGN.md — both the plan chat system prompt and the task scoping system prompt. This is critical context.
- **For chat-based specs (planner, task scoping):** describe the multi-turn conversation flow, what context is sent on each turn, and how responses are parsed.
- **Parent/subtask hierarchy:** When referencing tasks, be explicit about whether the behavior applies to parent tasks, subtasks, standalone tasks, or all.
- **Block actions:** Blocks can be skipped or have duration adjusted. No "done" toggle — non-skipped blocks are assumed completed.

## Available Specs

These are the bounded contexts for DayOS. Reference this list for cross-dependencies:

1. `foundation` — DB migrations (7 total), Go module + gqlgen + sqlc setup
2. `tasks` — Task CRUD, parent/subtask hierarchy, completion, deferred tracking, priority escalation
3. `routines` — Routine CRUD, day-of-week applicability
4. `context` — Context entry CRUD, active/inactive toggle
5. `day-plans` — Plan storage, block skip/adjust, plan status (draft/accepted)
6. `planner` — AI plan chat, task scoping chat, prompt construction, JSON parsing/retry
7. `carry-over` — Skipped task review, deferred counting, effective deadline, priority escalation, actual_minutes computation
8. `frontend-shell` — App shell, routing, Apollo + Tailwind setup, dark theme, simple auth
9. `frontend-today` — Today page, plan chat interface, block view, NOW indicator, skip/adjust, replan
10. `frontend-backlog` — Task backlog page, parent/subtask grouping, CRUD, "Scope with AI" chat, filtering
11. `frontend-manage` — Routines + Context + History pages (lighter management pages)
12. `deployment` — Railway config, embedded frontend, env vars, simple auth middleware

## After Writing

- Confirm the spec is self-contained: another Claude Code session should be able to implement it by reading only CLAUDE.md + this spec + the specs it depends on.
- Print a summary of what's in the spec and any open questions or ambiguities you noticed.
