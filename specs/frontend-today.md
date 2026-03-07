# Spec: Frontend — Today Page

## Bounded Context

Owns: Today page (`/`), plan chat interface, plan preview panel, accepted plan block view, NOW indicator, block skip/adjust controls, "Something came up" replan trigger, skipped tasks review screen (carry-over), browser notification scheduling

Does not own: App shell, routing, Apollo setup, Tailwind config (owned by `frontend-shell`), GraphQL resolvers or backend logic (owned by `day-plans`, `planner`, `carry-over`), task/routine/context management UI (owned by other frontend specs)

Depends on: `frontend-shell` (app shell, routing, Apollo client, dark theme, auth header), `day-plans` (GraphQL queries/mutations: `dayPlan`, `acceptPlan`, `skipBlock`, `updateBlock`), `planner` (`sendPlanMessage` mutation), `carry-over` (`resolveSkippedBlock` mutation, carry-over task data)

Produces: React components and page for the `/` route. No backend writes — all mutations go through existing GraphQL API.

## Route

`/` — the default and primary page of the app.

## Components

### Page-Level

**`TodayPage`** — root component for `/`. Orchestrates state across sub-components. On mount:
1. Query `dayPlan(date: today)` and the most recent past plan (for carry-over check)
2. If past plan has unresolved skipped blocks with `task_id`, show `SkippedTasksReview` (blocking)
3. Otherwise, show the plan interface based on plan state

### Skipped Tasks Review (carry-over gate)

**`SkippedTasksReview`** — blocking modal/screen shown before the plan chat when there are unresolved skipped blocks from the previous day.

- Queries the most recent past `dayPlan` via `recentPlans(limit: 1)` where `plan_date < today`
- Filters blocks where `skipped = true` and `task_id` is not null
- Displays each skipped block with title, category color dot, and resolution controls
- Per block:
  - "Intentional?" — two buttons: **Yes** / **No** (default)
  - Clicking **Yes** calls `resolveSkippedBlock(planId, blockId, intentional: true)`
  - Clicking **No** calls `resolveSkippedBlock(planId, blockId, intentional: false)`
- Bottom: **"Done — Start Planning"** button, enabled only when all skipped blocks are resolved
- After dismissal, transitions to the plan chat / accepted plan view

**Tracking resolution state:** Use local component state to track which blocks have been resolved. The `resolveSkippedBlock` mutation returns `Boolean!` — on success, mark the block as resolved in local state.

### Plan Chat (draft state)

**`PlanChat`** — split-panel chat interface for creating and refining a plan.

Layout: Two-panel split (responsive)
- **Left panel (or bottom on mobile):** Chat messages + input
- **Right panel (or top on mobile):** Plan preview (block list)

Sub-components:

**`ChatPanel`**
- Displays `DayPlan.messages` as a conversation thread
- User messages right-aligned, assistant messages left-aligned
- User messages in accent color background (`#c5a55a` with dark text), assistant messages in slightly lighter background (`#1a1a1e`)
- Text input at bottom with send button
- On send: calls `sendPlanMessage(date: today, message: input)` mutation
- Shows loading spinner while mutation is in flight
- After response, the returned `DayPlan` updates both the chat and preview panel
- Input placeholder: `"Describe your day..."` (first message) or `"Adjust the plan..."` (subsequent)

**`PlanPreview`**
- Renders blocks from the current draft plan as a vertical timeline
- Each block shows: time, duration badge, title, category color bar
- Visual-only — no actions on draft blocks (can't skip/adjust until accepted)
- Empty state: "Send a message to generate your plan"
- Bottom: **"Accept Plan"** button (calls `acceptPlan(date: today)`)
- Button is disabled if no blocks exist

### Accepted Plan View

**`AcceptedPlanView`** — full-width time-blocked schedule for the day.

Shown when `dayPlan.status === 'ACCEPTED'` and the chat panel is closed.

**`BlockList`** — vertical list of plan blocks, ordered by `time` ASC.

**`BlockCard`** — individual block display:
- Left border colored by category (see colors below)
- Time display: `"9:00 AM"` (12-hour format, derived from `"09:00"`)
- Title (bold)
- Duration badge: `"60 min"` — clickable for inline adjustment
- Category label (small, colored)
- Notes (if present, muted text below title)
- Skip button (icon: X or slash) — calls `skipBlock(planId, blockId)`
- Skipped blocks: greyed out with strikethrough title, skip button replaced with "Skipped" label
- Duration adjustment: clicking the duration badge opens an inline input (number). On blur or Enter, calls `updateBlock(planId, blockId, { duration: newValue })`. Validates > 0 client-side.

**`NowIndicator`**
- A horizontal line with label "NOW" positioned between blocks based on current time
- Updates every 60 seconds via `setInterval`
- Positioned by comparing current time to block times
- If current time is before all blocks: show at the top
- If current time is after all blocks: show at the bottom
- If between two blocks: show between them
- Color: accent `#c5a55a`
- Auto-scrolls into view on page load and on each interval tick

**`ReplanButton`** — "Something came up" button at the bottom of the accepted view
- Clicking it opens the `ChatPanel` alongside the block view (returns to split layout)
- Chat panel shows full conversation history
- New messages sent here trigger replanning (backend handles `accepted` → `draft` transition)
- After the plan updates, user must re-accept

### Notification Manager

**`useNotifications`** — custom hook managing browser notifications.

- On mount: calls `Notification.requestPermission()` if permission is `'default'`
- Stores permission state in React state (and localStorage to avoid re-prompting if denied)
- When plan is accepted (or on page load with accepted plan):
  - Clears any existing scheduled timeouts
  - For each upcoming block (time > now): schedules a `setTimeout` for `blockTime - now` milliseconds
  - Notification text: `"Time for: {block.title} ({block.duration} min)"`
  - Clicking notification calls `window.focus()`
- When plan transitions back to draft (replan): clears all scheduled timeouts
- End-of-day nudge: schedules a notification at 16:00 if there are non-skipped future blocks remaining. Text: `"Review your plan — mark skipped blocks so they don't count as completed."`
- Cleanup: clears all timeouts on unmount

## GraphQL Operations Used

### Queries

```graphql
query GetTodayPlan($date: Date!) {
  dayPlan(date: $date) {
    id
    planDate
    status
    blocks {
      id
      time
      duration
      title
      category
      taskId
      routineId
      notes
      skipped
    }
    messages {
      id
      role
      content
      createdAt
    }
    createdAt
    updatedAt
  }
}

query GetRecentPlans($limit: Int) {
  recentPlans(limit: $limit) {
    id
    planDate
    status
    blocks {
      id
      time
      duration
      title
      category
      taskId
      routineId
      notes
      skipped
    }
  }
}
```

### Mutations

```graphql
mutation SendPlanMessage($date: Date!, $message: String!) {
  sendPlanMessage(date: $date, message: $message) {
    id
    planDate
    status
    blocks { id time duration title category taskId routineId notes skipped }
    messages { id role content createdAt }
  }
}

mutation AcceptPlan($date: Date!) {
  acceptPlan(date: $date) {
    id
    status
    blocks { id time duration title category taskId routineId notes skipped }
  }
}

mutation SkipBlock($planId: UUID!, $blockId: String!) {
  skipBlock(planId: $planId, blockId: $blockId) {
    id
    blocks { id time duration title category taskId routineId notes skipped }
  }
}

mutation UpdateBlock($planId: UUID!, $blockId: String!, $input: UpdateBlockInput!) {
  updateBlock(planId: $planId, blockId: $blockId, input: $input) {
    id
    blocks { id time duration title category taskId routineId notes skipped }
  }
}

mutation ResolveSkippedBlock($planId: UUID!, $blockId: String!, $intentional: Boolean!) {
  resolveSkippedBlock(planId: $planId, blockId: $blockId, intentional: $intentional)
}
```

## Visual Design

### Colors (from CLAUDE.md)

- Background: `#0f0f11`
- Text: `#e8e6e1`
- Accent: `#c5a55a`
- Muted text: `#6b7280` (gray-500)
- Card/panel background: `#1a1a1e`
- Category left-border colors:
  - `job`: `#6366f1` (indigo)
  - `interview`: `#0ea5e9` (sky)
  - `project`: `#8b5cf6` (violet)
  - `meal`: `#10b981` (emerald)
  - `baby`: `#f59e0b` (amber)
  - `exercise`: `#ef4444` (red)
  - `admin`: `#6b7280` (gray)

### Layout Details

- Skipped tasks review: centered card, max-width `32rem`, vertically centered
- Chat panel: messages area with overflow-y scroll, auto-scroll to bottom on new messages
- Plan preview / block list: overflow-y scroll, max-height based on viewport
- Block cards: rounded corners, `4px` left border for category color, padding `1rem`
- NOW indicator: full-width dashed line, `2px`, accent color, with "NOW" label left-aligned
- Duration badge: small pill, background slightly lighter than card
- Skip button: small, positioned top-right of block card

### Responsive Behavior

- Desktop (≥768px): side-by-side split for chat + preview
- Mobile (<768px): stacked — preview on top, chat below (or tab toggle)

## Behaviors (EARS syntax)

### Page Load

- When the Today page loads, the system shall query `dayPlan` for today's date and `recentPlans(limit: 1)`.
- When the most recent past plan has skipped blocks with `task_id` values, the system shall show the `SkippedTasksReview` screen before any other content.
- When no past plan exists or the past plan has no unresolved skipped blocks, the system shall show the plan interface directly.
- If today's plan exists with `status: DRAFT`, the system shall show the split chat+preview layout with existing messages and blocks.
- If today's plan exists with `status: ACCEPTED`, the system shall show the full-width accepted plan view.
- If no plan exists for today, the system shall show the chat panel with an empty preview and prompt the user to describe their day.

### Carry-Over Review

- When a skipped block's "Yes" (intentional) button is clicked, the system shall call `resolveSkippedBlock` with `intentional: true` and visually mark the block as resolved.
- When a skipped block's "No" (not intentional) button is clicked, the system shall call `resolveSkippedBlock` with `intentional: false` and visually mark the block as resolved.
- While any skipped block remains unresolved, the "Done — Start Planning" button shall be disabled.
- When all skipped blocks are resolved and "Done — Start Planning" is clicked, the system shall dismiss the review screen and show the plan interface.

### Plan Chat

- When the user sends a message, the system shall call `sendPlanMessage` and display a loading state in the chat.
- When `sendPlanMessage` returns successfully, the system shall update the chat messages and plan preview from the returned `DayPlan`.
- When `sendPlanMessage` returns an error, the system shall display the error message in the chat area (styled distinctly from normal messages) and re-enable the input.
- While a `sendPlanMessage` mutation is in flight, the send button and input shall be disabled.
- When blocks are present in the draft plan, the "Accept Plan" button shall be enabled.
- When "Accept Plan" is clicked, the system shall call `acceptPlan` and transition to the accepted plan view.

### Accepted Plan — Block Actions

- When the skip button is clicked on a non-skipped block, the system shall call `skipBlock` and update the block's appearance to show it as skipped (greyed out, strikethrough).
- When a block is already skipped, the skip button shall not be shown (replaced by "Skipped" label). Skipping is not reversible from the UI.
- When the duration badge is clicked on a non-skipped block, the system shall show an inline number input pre-filled with the current duration.
- When the user confirms a duration change (Enter or blur), the system shall call `updateBlock` with the new duration. If the value is ≤ 0 or non-numeric, show a validation error and do not submit.
- When `skipBlock` or `updateBlock` returns an error, the system shall show a toast/inline error and revert the optimistic UI update.

### NOW Indicator

- While the accepted plan view is visible, the system shall update the NOW indicator position every 60 seconds.
- When the page loads with an accepted plan, the system shall scroll the NOW indicator into view.
- Where the current time falls before the first block, the NOW indicator shall appear above the first block.
- Where the current time falls after the last block, the NOW indicator shall appear below the last block.
- Where the current time falls between two blocks, the NOW indicator shall appear between those blocks.

### Replanning

- When "Something came up" is clicked, the system shall open the chat panel alongside the block view.
- When a message is sent during replanning, `sendPlanMessage` shall trigger the backend's replanning flow (preserves past blocks, reschedules future ones).
- When the plan transitions from `ACCEPTED` to `DRAFT` (after replanning response), the system shall show the split chat+preview layout and require re-acceptance.
- When a replanned plan is accepted, the system shall reschedule all browser notifications.

### Browser Notifications

- When the page loads, the system shall request notification permission if not already granted or denied.
- When permission is denied, the system shall not re-prompt and shall store the denial in localStorage.
- When a plan is accepted, the system shall schedule `setTimeout`-based notifications for each upcoming block.
- When the plan transitions back to draft (replan), the system shall clear all scheduled notification timeouts.
- When a new plan is accepted after replanning, the system shall schedule fresh notifications for the new blocks.
- When 16:00 arrives and there are non-skipped blocks after 16:00, the system shall fire the end-of-day nudge notification.

## Decision Table: Page State

| Plan exists? | Plan status | Has unresolved carry-over? | View shown |
|---|---|---|---|
| - | - | Yes | `SkippedTasksReview` (blocking) |
| No | N/A | No | Chat panel + empty preview |
| Yes | DRAFT | No | Chat panel + plan preview (with existing messages/blocks) |
| Yes | ACCEPTED | No | Full-width accepted plan view |
| Yes | ACCEPTED (replanning) | No | Split: chat panel + block view |

## Decision Table: Block Rendering

| Block skipped? | Time vs now | Rendering |
|---|---|---|
| false | future | Normal card, skip button + duration editable |
| false | past | Normal card, slightly dimmed, skip button still available, duration editable |
| true | any | Greyed out, strikethrough title, "Skipped" label, no actions |

## File Structure

```
frontend/src/
├── pages/
│   └── TodayPage.tsx
├── components/
│   └── today/
│       ├── SkippedTasksReview.tsx
│       ├── ChatPanel.tsx
│       ├── PlanPreview.tsx
│       ├── AcceptedPlanView.tsx
│       ├── BlockCard.tsx
│       ├── BlockList.tsx
│       ├── NowIndicator.tsx
│       └── ReplanButton.tsx
├── hooks/
│   └── useNotifications.ts
└── graphql/
    └── today.graphql        # queries + mutations for codegen
```

## Test Anchors

1. **Given** no plan exists for today and no past plan with skipped blocks, **when** the Today page loads, **then** the chat panel is shown with an empty preview and the input placeholder reads "Describe your day...".

2. **Given** yesterday's accepted plan has 2 skipped blocks with `task_id` values, **when** the Today page loads, **then** the `SkippedTasksReview` screen is displayed with 2 blocks listed and the "Done — Start Planning" button is disabled.

3. **Given** the `SkippedTasksReview` shows 2 blocks, **when** the user clicks "No" on both and then clicks "Done — Start Planning", **then** `resolveSkippedBlock` is called twice with `intentional: false`, the review screen dismisses, and the plan interface appears.

4. **Given** a draft plan exists with 4 blocks and 3 messages, **when** the Today page loads, **then** the chat panel shows all 3 messages and the plan preview shows all 4 blocks with the "Accept Plan" button enabled.

5. **Given** the user is in the chat panel, **when** they type a message and click send, **then** `sendPlanMessage` is called, the input is disabled while loading, and on success the new messages and updated blocks appear.

6. **Given** a draft plan with blocks, **when** the user clicks "Accept Plan", **then** `acceptPlan` is called and the view transitions to the full-width `AcceptedPlanView`.

7. **Given** an accepted plan with 5 blocks (one at 10:00, current time is 10:30), **when** the accepted plan view renders, **then** the NOW indicator appears between the 10:00 block and the next block, and it is scrolled into view.

8. **Given** an accepted plan with a non-skipped block, **when** the user clicks the skip button on that block, **then** `skipBlock` is called and the block renders as greyed out with strikethrough title.

9. **Given** an accepted plan with a block showing `60 min`, **when** the user clicks the duration badge, enters `45`, and presses Enter, **then** `updateBlock` is called with `{ duration: 45 }` and the badge updates to `"45 min"`.

10. **Given** an accepted plan, **when** the user clicks "Something came up", **then** the chat panel opens alongside the block view showing the full conversation history, and the input is ready for a replanning message.

11. **Given** notification permission is granted and a plan is accepted with 3 future blocks, **when** the plan is accepted, **then** 3 `setTimeout` calls are scheduled for the block start times. **When** the plan is later replanned and re-accepted with 2 future blocks, **then** the old timeouts are cleared and 2 new ones are scheduled.

12. **Given** `sendPlanMessage` returns an error `"Couldn't parse AI response, please try again"`, **when** the chat panel receives the error, **then** the error is displayed as a distinct message in the chat and the input is re-enabled.

13. **Given** the user clicks the duration badge and enters `0`, **when** they press Enter, **then** a validation error is shown and `updateBlock` is NOT called.
