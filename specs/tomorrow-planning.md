# Spec: Tomorrow Planning

## Bounded Context

Owns: Date-aware planner prompt construction (plan date label, date-conditional rules), frontend Today/Tomorrow toggle, read-only accepted plan mode for future dates, `PlanDateLabel` field in `PlanChatInput`

Does not own: `day_plans` table or block CRUD (owned by `day-plans`), AI plan generation core (owned by `planner`), carry-over resolution logic (owned by `carry-over`), GraphQL schema (no changes — `sendPlanMessage(date: Date!)` already accepts any date), task/routine/context CRUD

Depends on:
- `planner` — `PlanChatInput` struct, `buildPlanSystemPrompt`, plan chat flow
- `carry-over` — carry-over task retrieval (already date-parameterized via `planDate`)
- `frontend-today` — `TodayPage`, `AcceptedPlanView`, `ChatPanel`, `useNotifications`
- `frontend-shell` — routing, layout

Produces:
- Backend: date-aware prompt labels and conditional scheduling rules in the planner
- Frontend: Today/Tomorrow toggle on the Today page, read-only accepted plan view for future dates

## Contracts

### Input

No new GraphQL operations. All existing operations already accept a `Date!` parameter:
- `dayPlan(date: Date!)` — fetch plan for any date
- `sendPlanMessage(date: Date!, message: String!)` — create/refine plan for any date
- `acceptPlan(date: Date!)` — accept plan for any date

The frontend passes tomorrow's date (`YYYY-MM-DD`) instead of today's when the user selects the "Tomorrow" tab.

### Output

No new GraphQL types or DB writes. The only backend output change is in the planner system prompt content — it becomes date-aware.

### Data Model

No schema changes. The `day_plans` table already supports one plan per `(user_id, plan_date)` pair. Tomorrow's plan is just another row with a different `plan_date`.

## Backend Changes

### `PlanChatInput` — new field

Add to `backend/planner/planner.go`:

```go
type PlanChatInput struct {
    // ... existing fields ...
    PlanDateLabel string // e.g. "TODAY (Tuesday, March 17)", "TOMORROW (Wednesday, March 18)"
}
```

When `PlanDateLabel` is empty (backward compatibility with existing tests), `buildPlanSystemPrompt` defaults it to `"TODAY'S"`.

### `buildPlanChatInput` — compute date label and gate CurrentTime

In `backend/graph/planner_helpers.go`, after extracting `userName` and before building `input`:

```go
userNow := time.Now().In(tz.FromContext(ctx))
userToday := time.Date(userNow.Year(), userNow.Month(), userNow.Day(), 0, 0, 0, 0, userNow.Location())
planDay := time.Date(planDate.Year(), planDate.Month(), planDate.Day(), 0, 0, 0, 0, userNow.Location())
isFuturePlan := planDay.After(userToday)

var planDateLabel string
switch {
case planDay.Equal(userToday):
    planDateLabel = fmt.Sprintf("TODAY (%s, %s)", planDate.Weekday(), planDate.Format("January 2"))
case planDay.Equal(userToday.AddDate(0, 0, 1)):
    planDateLabel = fmt.Sprintf("TOMORROW (%s, %s)", planDate.Weekday(), planDate.Format("January 2"))
default:
    planDateLabel = strings.ToUpper(planDate.Format("Monday, January 2"))
}
```

Set `PlanDateLabel: planDateLabel` on the `PlanChatInput`.

In the `if isReplan` block, only set `CurrentTime` for today's plan:

```go
if isReplan {
    if !isFuturePlan {
        input.CurrentTime = userNow.Format("15:04")
    }
    // ... rest of replan block splitting unchanged ...
}
```

### `buildPlanSystemPrompt` — date-aware labels and rules

In `backend/planner/planner.go`, four changes to `buildPlanSystemPrompt`:

**1. Default empty label for backward compat:**

```go
dateLabel := input.PlanDateLabel
if dateLabel == "" {
    dateLabel = "TODAY'S"
}
```

**2. Add plan date near top of prompt** (after safety instructions):

```
PLAN DATE: {dateLabel}
```

**3. Replace hardcoded labels:**

- `"TODAY'S CALENDAR EVENTS (fixed — ...)"` → `"{dateLabel} CALENDAR EVENTS (fixed — ...)"`
- `"TODAY'S ROUTINES — EVERY routine below ..."` → `"{dateLabel} ROUTINES — EVERY routine below ..."`

**4. Gate "past time slot" rule on `CurrentTime`:**

The planning rule `"Never schedule anything in a past time slot."` is only meaningful for today. Make it conditional:

```go
if input.CurrentTime != "" {
    b.WriteString("- Never schedule anything in a past time slot.\n")
}
```

**5. Conditional replanning section:**

When replanning a future-date plan (`CurrentTime` is empty), omit the current-time constraint:

```
REPLANNING RULES:
- Return ONLY new/rescheduled blocks for the full day.
- Do NOT include completed or skipped blocks in your response.
- The user is asking to adjust the schedule.
```

When replanning today (`CurrentTime` is set), use the existing rules with `Current time: {HH:MM}` and time-gating.

### Carry-over behavior for tomorrow

No backend changes needed. The existing carry-over logic in `getCarryOverTasks(ctx, planDate)` already uses `planDate` (not hardcoded "today") to find the most recent accepted plan before the target date. When `planDate` is tomorrow, it will find today's accepted plan and pull its skipped tasks — which is the desired behavior.

### Routine day-of-week filtering

No backend changes needed. `buildPlanChatInput` already computes `dow := int32(planDate.Weekday())` — this returns the correct day-of-week for any date.

### Calendar events

No backend changes needed. `r.Calendar.GetEvents(ctx, planDate)` already accepts any date.

## Frontend Changes

### `TodayPage.tsx` — Today/Tomorrow toggle and date switching

**New helper function:**

```ts
function tomorrowDate(): string {
  const d = new Date()
  d.setDate(d.getDate() + 1)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}
```

**State changes:**

Replace `const date = useMemo(todayDate, [])` with:

```ts
const [selectedDay, setSelectedDay] = useState<'today' | 'tomorrow'>('today')
const today = useMemo(todayDate, [])
const tomorrow = useMemo(tomorrowDate, [])
const date = selectedDay === 'today' ? today : tomorrow
const isFuture = selectedDay === 'tomorrow'
```

**Tab switch handler** — resets transient state:

```ts
const handleDaySwitch = (day: 'today' | 'tomorrow') => {
  setSelectedDay(day)
  setReplanning(false)
  setShowReplanBanner(false)
  setPlanCalendarVersion(null)
  setMobileTab('chat')
}
```

**Carry-over gate** — only for today:

```ts
const needsReview = !isFuture && showReview && skippedBlocks.length > 0
```

**Notifications** — disabled for future plans:

```ts
useNotifications(blocks, isAccepted && !replanning, !isFuture)
```

**Toggle UI** — rendered at the top of both the accepted-plan and draft/chat views. Visual: pill-shaped toggle with two buttons ("Today" / "Tomorrow"), accent background on active, positioned center-top with a thin bottom border.

```tsx
<div className="flex gap-1 p-1 bg-bg-surface rounded-xl w-fit">
  {(['today', 'tomorrow'] as const).map((day) => (
    <button
      key={day}
      onClick={() => handleDaySwitch(day)}
      className={`px-4 py-1.5 rounded-lg text-sm font-medium transition-colors ${
        selectedDay === day
          ? 'bg-accent text-[#0f0f11]'
          : 'text-text-secondary hover:text-text-primary'
      }`}
    >
      {day === 'today' ? 'Today' : 'Tomorrow'}
    </button>
  ))}
</div>
```

**Props passed to children:**

- `AcceptedPlanView`: add `readOnly={isFuture}`
- `ChatPanel`: add `isFuture={isFuture}`

### `AcceptedPlanView.tsx` — read-only mode for future plans

Add `readOnly?: boolean` to the props interface.

When `readOnly` is `true`:
- Heading: `"Tomorrow's Plan"` instead of `"Today's Plan"`
- Progress bar and `{doneCount}/{activeCount} done`: hidden
- `BlockList` receives `readOnly={true}` and `showNow={false}` (no NowIndicator)
- "Something came up" button: hidden
- Confetti on complete: suppressed
- Skip/complete/unskip/duration-edit actions: suppressed via existing `readOnly` prop on `BlockCard`/`BlockList`

### `ChatPanel.tsx` — future-aware copy

Add `isFuture?: boolean` to the props interface.

When `isFuture` is `true`:
- Empty-state heading: `"Plan your tomorrow"` instead of `"Plan your day"`
- Empty-state body: `"Tell me what's on your plate tomorrow and I'll draft a schedule."`
- Input placeholder (first message): `"Plan your tomorrow..."` instead of `"Describe your day..."`

### `useNotifications.ts` — enabled parameter

Add `enabled = true` third parameter:

```ts
export function useNotifications(blocks: Block[], isAccepted: boolean, enabled = true)
```

When `!enabled`: clear all scheduled timeouts, skip permission request, bail from scheduling effect. Add `enabled` to effect dependency arrays.

## Behaviors (EARS syntax)

### Backend — Prompt date awareness

- When `sendPlanMessage` is called for today's date, the system shall label the plan as `"TODAY ({weekday}, {month day})"` in the system prompt.
- When `sendPlanMessage` is called for tomorrow's date, the system shall label the plan as `"TOMORROW ({weekday}, {month day})"` in the system prompt.
- When `sendPlanMessage` is called for a date beyond tomorrow, the system shall label the plan as `"{WEEKDAY}, {MONTH DAY}"` (uppercase) in the system prompt.
- When the plan date is in the future relative to the user's local time, the system shall NOT include the `"Never schedule anything in a past time slot"` rule in the system prompt.
- When replanning a future-date plan, the system shall NOT inject `CurrentTime` into the replanning context and shall instruct the AI to reschedule the full day (not time-gated).
- When replanning today's plan, the system shall continue to inject `CurrentTime` and gate blocks to `>= current time` (existing behavior, unchanged).
- Where `PlanDateLabel` is empty (backward compatibility), the system shall default it to `"TODAY'S"`.

### Backend — Carry-over for tomorrow

- When building carry-over tasks for tomorrow's plan, the system shall use the existing `getCarryOverTasks(ctx, planDate)` which finds the most recent accepted plan before `planDate` — this will be today's accepted plan.
- Where today's plan has skipped blocks with `task_id` values, those tasks shall appear as carry-over in tomorrow's planner prompt.

### Frontend — Day toggle

- When the Today page loads, the system shall default to the "Today" tab.
- When the user clicks "Tomorrow", the system shall switch `date` to tomorrow's date and re-fetch the plan for that date.
- When the user switches tabs, the system shall reset `replanning`, `showReplanBanner`, `planCalendarVersion`, and `mobileTab` to their initial values.
- When the "Tomorrow" tab is selected and no plan exists for tomorrow, the system shall show the chat panel with the empty state "Plan your tomorrow".

### Frontend — Carry-over gate

- When the "Tomorrow" tab is selected, the system shall NOT show the `SkippedTasksReview` carry-over gate, regardless of whether past plans have skipped blocks.
- When the "Today" tab is selected, carry-over behavior is unchanged from `frontend-today` spec.

### Frontend — Read-only accepted plan

- When a future-date plan has `status: ACCEPTED`, the system shall render `AcceptedPlanView` with `readOnly={true}`.
- While `readOnly` is `true`, the accepted plan view shall NOT show skip, complete, unskip, or duration-edit controls.
- While `readOnly` is `true`, the accepted plan view shall NOT show the "Something came up" button.
- While `readOnly` is `true`, the accepted plan view shall NOT show the NowIndicator.
- While `readOnly` is `true`, the accepted plan view shall NOT show the progress bar.
- While `readOnly` is `true`, the heading shall read `"Tomorrow's Plan"`.

### Frontend — Notifications

- When the "Tomorrow" tab is selected, the system shall disable all browser notification scheduling (clear existing timeouts, do not schedule new ones).
- When the user switches back to "Today", the system shall resume notification scheduling for today's plan (if accepted).

## Decision Table: Plan Date Label

| Plan date vs user's local today | `PlanDateLabel` value | "Past time slot" rule | `CurrentTime` on replan |
|---|---|---|---|
| Same day | `"TODAY (Wednesday, March 17)"` | Included | Set to `HH:MM` |
| Next day | `"TOMORROW (Thursday, March 18)"` | Omitted | Not set |
| 2+ days ahead | `"FRIDAY, MARCH 20"` (uppercase) | Omitted | Not set |

## Decision Table: Frontend View State (extends `frontend-today` spec)

| Tab | Plan exists? | Plan status | Has carry-over? | View shown |
|---|---|---|---|---|
| Today | - | - | Yes | `SkippedTasksReview` (blocking) |
| Today | No | N/A | No | Chat + empty preview |
| Today | Yes | DRAFT | No | Chat + plan preview |
| Today | Yes | ACCEPTED | No | Full-width accepted view (interactive) |
| Tomorrow | No | N/A | (ignored) | Chat + empty preview ("Plan your tomorrow") |
| Tomorrow | Yes | DRAFT | (ignored) | Chat + plan preview |
| Tomorrow | Yes | ACCEPTED | (ignored) | Full-width accepted view (read-only) |

## File Changes

### Backend (2 files modified)

```
backend/planner/planner.go          # Add PlanDateLabel field, update buildPlanSystemPrompt
backend/graph/planner_helpers.go     # Compute planDateLabel, gate CurrentTime on isFuturePlan
```

### Frontend (4 files modified)

```
frontend/src/pages/TodayPage.tsx                      # Add toggle, tomorrowDate(), selectedDay state, isFuture
frontend/src/components/today/AcceptedPlanView.tsx     # Add readOnly prop, gate actions/progress/heading
frontend/src/components/today/ChatPanel.tsx            # Add isFuture prop, update copy
frontend/src/hooks/useNotifications.ts                 # Add enabled parameter
```

### No new files

The toggle is inline JSX in `TodayPage` — not worth extracting to a separate component for a two-button element.

## Test Anchors

### Backend

1. **Given** `PlanDateLabel` is `"TOMORROW (Thursday, March 18)"`, **when** `buildPlanSystemPrompt` is called, **then** the prompt contains `"PLAN DATE: TOMORROW (Thursday, March 18)"`, `"TOMORROW (Thursday, March 18) CALENDAR EVENTS"`, and `"TOMORROW (Thursday, March 18) ROUTINES"`.

2. **Given** `PlanDateLabel` is empty (not set), **when** `buildPlanSystemPrompt` is called, **then** the prompt uses `"TODAY'S"` as the default label for calendar events and routines sections.

3. **Given** `CurrentTime` is empty (future-date plan), **when** `buildPlanSystemPrompt` is called, **then** the prompt does NOT contain `"Never schedule anything in a past time slot"`.

4. **Given** `CurrentTime` is `"14:30"` (today's plan), **when** `buildPlanSystemPrompt` is called, **then** the prompt contains `"Never schedule anything in a past time slot"`.

5. **Given** `IsReplan` is `true` and `CurrentTime` is empty (future-date replan), **when** `buildPlanSystemPrompt` is called, **then** the replanning section says `"Return ONLY new/rescheduled blocks for the full day"` and does NOT contain `"Current time:"`.

6. **Given** `IsReplan` is `true` and `CurrentTime` is `"10:00"` (today's replan), **when** `buildPlanSystemPrompt` is called, **then** the replanning section contains `"Current time: 10:00"` and `"All blocks must have \"time\" >= current time"`.

7. **Given** `planDate` is tomorrow and the user's timezone is `America/Toronto`, **when** `buildPlanChatInput` is called with `isReplan = true`, **then** `input.CurrentTime` is empty and `input.PlanDateLabel` starts with `"TOMORROW"`.

8. **Given** `planDate` is today, **when** `buildPlanChatInput` is called with `isReplan = true`, **then** `input.CurrentTime` is set to the current local time in HH:MM format.

9. **Given** today's accepted plan has 2 skipped blocks with `task_id`, **when** `buildPlanChatInput` is called for tomorrow's plan, **then** `input.CarryOverTasks` contains those 2 tasks (carry-over works across dates).

### Frontend

10. **Given** the Today page loads, **when** the user clicks the "Tomorrow" toggle, **then** the `dayPlan` query fires with tomorrow's date and the chat panel shows `"Plan your tomorrow"` as the empty-state heading.

11. **Given** an accepted plan exists for tomorrow, **when** the "Tomorrow" tab is selected, **then** the accepted view renders with heading `"Tomorrow's Plan"`, no progress bar, no skip/complete buttons, no "Something came up" button, and no NowIndicator.

12. **Given** yesterday's plan has skipped blocks with `task_id`, **when** the user selects the "Tomorrow" tab, **then** the `SkippedTasksReview` gate does NOT appear (carry-over gate is skipped for future dates).

13. **Given** the user is on the "Tomorrow" tab with an accepted plan, **when** the view renders, **then** `useNotifications` does not schedule any browser notification timeouts.

14. **Given** the user is on the "Tomorrow" tab mid-replan, **when** they switch to the "Today" tab, **then** `replanning` is reset to `false`, `mobileTab` resets to `"chat"`, and the Today plan loads normally.

15. **Given** no plan exists for tomorrow, **when** the user sends a message in the chat panel on the "Tomorrow" tab, **then** `sendPlanMessage` is called with tomorrow's date and a new draft plan is created for tomorrow.
