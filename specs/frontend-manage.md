# Spec: Frontend Manage

## Bounded Context

Owns: `RoutinesPage.tsx`, `ContextPage.tsx`, `HistoryPage.tsx` — the three lighter management pages at `/routines`, `/context`, `/history`. Includes all components, Apollo operations, and interaction logic for these pages.

Does not own: Sidebar/layout/routing (owned by `frontend-shell`), backend resolvers (owned by `routines`, `context`, `day-plans`), GraphQL schema, Today/Backlog pages

Depends on: `frontend-shell` (routes, layout, theme tokens, `CATEGORY_COLORS`, Apollo Client), `routines` (GraphQL queries/mutations), `context` (GraphQL queries/mutations + `toggleContext`), `day-plans` (GraphQL queries: `dayPlan`, `recentPlans`)

Produces: Three fully functional page components replacing the stubs created by `frontend-shell`

## Contracts

### GraphQL Operations Used

**Routines Page:**

```graphql
query Routines($activeOnly: Boolean) {
  routines(activeOnly: $activeOnly) {
    id title category frequency daysOfWeek
    preferredTimeOfDay preferredDurationMin notes isActive
  }
}

mutation CreateRoutine($input: CreateRoutineInput!) {
  createRoutine(input: $input) { id title category frequency daysOfWeek preferredTimeOfDay preferredDurationMin notes isActive }
}

mutation UpdateRoutine($id: UUID!, $input: UpdateRoutineInput!) {
  updateRoutine(id: $id, input: $input) { id title category frequency daysOfWeek preferredTimeOfDay preferredDurationMin notes isActive }
}

mutation DeleteRoutine($id: UUID!) {
  deleteRoutine(id: $id)
}
```

**Context Page:**

```graphql
query ContextEntries($category: ContextCategory) {
  contextEntries(category: $category) {
    id category key value isActive createdAt
  }
}

mutation UpsertContext($input: UpsertContextInput!) {
  upsertContext(input: $input) { id category key value isActive createdAt }
}

mutation ToggleContext($id: UUID!, $isActive: Boolean!) {
  toggleContext(id: $id, isActive: $isActive) { id isActive }
}

mutation DeleteContext($id: UUID!) {
  deleteContext(id: $id)
}
```

**History Page:**

```graphql
query RecentPlans($limit: Int) {
  recentPlans(limit: $limit) {
    id planDate status blocks { id time duration title category taskId routineId notes skipped } createdAt
  }
}

query DayPlan($date: Date!) {
  dayPlan(date: $date) {
    id planDate status
    blocks { id time duration title category taskId routineId notes skipped }
    messages { id role content createdAt }
    createdAt updatedAt
  }
}
```

## Components

### File Structure

```
frontend/src/pages/
├── RoutinesPage.tsx
├── ContextPage.tsx
└── HistoryPage.tsx
```

No shared sub-components directory needed — these pages are simple enough to be self-contained. If a reusable component emerges (e.g., category badge), extract to `frontend/src/components/`.

---

## Routines Page (`/routines`)

### Layout

- Page heading: "Routines"
- "Add Routine" button (accent colored) top-right
- Routines listed as cards in a single column, each card showing:
  - Title (left-aligned, `text-primary`)
  - Category badge (colored dot + label, using `CATEGORY_COLORS`)
  - Frequency label: "Daily", "Weekdays", "Weekly", or "Custom (Mon, Wed, Fri)" (expand `daysOfWeek` to day names)
  - Preferred time of day (if set): "Morning", "Midday", etc.
  - Duration: e.g., "45 min"
  - Notes (if present): truncated to 1 line, expand on click
  - Active/inactive toggle switch (right side)
  - "Applies today" indicator: green dot if the routine applies to the current day of week
  - Edit (pencil icon) and Delete (trash icon) action buttons

### Interactions

- **Add Routine:** Opens a form (inline or modal) with fields: title, category (dropdown), frequency (dropdown), days of week (multi-select checkboxes, shown only when frequency is WEEKLY or CUSTOM), preferred time of day (dropdown), preferred duration (number input, minutes), notes (textarea). Submit calls `createRoutine`.
- **Edit Routine:** Clicking edit opens the same form pre-populated with current values. Submit calls `updateRoutine`.
- **Delete Routine:** Clicking delete shows a confirmation prompt ("Delete this routine?"). On confirm, calls `deleteRoutine`.
- **Toggle Active:** Toggling the switch calls `updateRoutine(id, { isActive: !current })`. Inactive routines are visually dimmed (`opacity-50`).

### "Applies Today" Logic

Computed client-side from current `new Date().getDay()` (JS: 0=Sun, 6=Sat — matches the `daysOfWeek` convention):

| frequency | Applies today when |
|---|---|
| `DAILY` | Always |
| `WEEKDAYS` | `dayOfWeek >= 1 && dayOfWeek <= 5` |
| `WEEKLY` | `daysOfWeek.includes(dayOfWeek)` |
| `CUSTOM` | `daysOfWeek.includes(dayOfWeek)` |

---

## Context Page (`/context`)

### Layout

- Page heading: "Context"
- Entries grouped by category. Each group has a section header: "Life", "Constraints", "Equipment", "Preferences", "Custom"
- Only show category groups that have entries (plus always show "Custom" with an add button)
- Each entry displayed as a row:
  - Key (bold, `text-primary`, left)
  - Value (regular weight, `text-secondary`, wrapping allowed)
  - Active/inactive toggle (right side)
  - Delete button (trash icon, right side)
- "Add Entry" button at the bottom of each category group
- Last updated timestamp shown as relative time (e.g., "2 days ago") in `text-secondary`, small font

### Interactions

- **Inline Edit:** Clicking on a value text makes it editable (contenteditable or textarea swap). Pressing Enter or clicking away saves via `upsertContext({ category, key, value })`. Pressing Escape cancels.
- **Add Entry:** Opens an inline form at the bottom of the group with: category (pre-filled from the group, or dropdown if using the global add), key (text input), value (textarea). Submit calls `upsertContext`.
- **Toggle Active/Inactive:** Toggle switch calls `toggleContext(id, isActive)`. Inactive entries are dimmed (`opacity-50`) and show a label "inactive — not sent to planner".
- **Delete Entry:** Confirmation prompt, then calls `deleteContext(id)`.

---

## History Page (`/history`)

### Layout

- Page heading: "History"
- List of past plans, most recent first
- Initial load: fetch `recentPlans(limit: 30)`
- Each plan shown as a row/card:
  - Date (formatted: "Thursday, Mar 5, 2026")
  - Status badge: "Draft" (muted) or "Accepted" (accent)
  - Summary: "{N} blocks, {M} skipped" computed from blocks array
  - Click to expand/navigate to detail view

### Plan Detail View

When a plan row is clicked, expand inline (or navigate to a sub-view) showing the full plan in read-only mode:

- Time-blocked list identical to the accepted plan view in Today page, but **read-only** (no skip/adjust buttons)
- Each block shows:
  - Time: "09:00" (left column)
  - Duration: "60 min"
  - Title
  - Category color bar (left border, using `CATEGORY_COLORS`)
  - Skipped blocks: shown with strikethrough text and `opacity-50`, plus a "Skipped" label
  - Non-skipped blocks: normal display (assumed completed)
- Chat history (collapsible section below the blocks): shows all `messages` in chronological order, styled as a chat log (user messages right-aligned, assistant messages left-aligned)

### Navigation

- "Load more" button or infinite scroll to load older plans (increment limit or use cursor-based pagination via `recentPlans`)
- No date picker needed — the list view is sufficient for a single user

---

## Visual Rules (all pages)

- Cards/rows use `bg-surface` with `border border-border-default rounded-lg`
- Hover states on interactive rows: `bg-surface-hover`
- Form inputs: `bg-surface border border-border-default text-primary` with `focus:border-accent focus:ring-1 focus:ring-accent`
- Buttons:
  - Primary (Add/Save): `bg-accent text-black font-medium rounded px-4 py-2 hover:bg-accent-hover`
  - Destructive (Delete): `text-red-400 hover:text-red-300` (text button, no background)
  - Cancel: `text-secondary hover:text-primary` (text button)
- Toggle switch: custom Tailwind toggle — small rounded pill, accent when active, `bg-gray-600` when inactive
- Empty states: centered `text-secondary` message (e.g., "No routines yet. Add one to get started.")
- Loading states: skeleton placeholders or a simple spinner in `text-secondary`

---

## Behaviors (EARS syntax)

### Routines Page

- When the Routines page loads, the system shall fetch all routines (including inactive) via `routines(activeOnly: false)` and display them.
- When the user clicks "Add Routine" and submits the form, the system shall call `createRoutine` and prepend the new routine to the list without a full refetch (Apollo cache update).
- When the user toggles a routine's active switch, the system shall call `updateRoutine` with the new `isActive` value and update the routine's visual state immediately (optimistic update).
- When the user clicks delete and confirms, the system shall call `deleteRoutine` and remove the routine from the list.
- When `createRoutine` is called with `frequency = WEEKLY` or `CUSTOM` and no `daysOfWeek` selected, the system shall show a client-side validation error ("Select at least one day") before submitting.
- While a routine is inactive, the system shall render it with `opacity-50` and the toggle in the off position.
- The system shall display a green "Applies today" dot next to routines that match the current day of week.

### Context Page

- When the Context page loads, the system shall fetch all context entries via `contextEntries()` (no category filter) and group them by category.
- When the user clicks on a value to edit it, the system shall replace the text with an editable input. On save (Enter/blur), the system shall call `upsertContext` with the entry's category, key, and new value.
- When the user toggles an entry's active switch, the system shall call `toggleContext(id, isActive)` and dim/undim the entry immediately.
- When the user deletes an entry and confirms, the system shall call `deleteContext` and remove it from the list.
- While an entry is inactive, the system shall render it with `opacity-50` and show "inactive — not sent to planner" text.

### History Page

- When the History page loads, the system shall fetch the 30 most recent plans via `recentPlans(limit: 30)`.
- When the user clicks a plan row, the system shall expand it inline to show the block detail view (read-only).
- While a plan detail is expanded, the system shall show skipped blocks with strikethrough styling and `opacity-50`.
- When the user clicks "Load more", the system shall fetch additional plans by increasing the limit.
- If no plans exist, the system shall show an empty state: "No plans yet. Start planning on the Today page."

---

## Decision Table: Routine "Applies Today"

| frequency | daysOfWeek | Current day (JS) | Applies today |
|-----------|-----------|-------------------|---------------|
| DAILY     | any       | any               | Yes           |
| WEEKDAYS  | any       | 1–5 (Mon–Fri)    | Yes           |
| WEEKDAYS  | any       | 0 or 6 (Sat/Sun) | No            |
| WEEKLY    | [1,3,5]   | 3 (Wed)           | Yes           |
| WEEKLY    | [1,3,5]   | 2 (Tue)           | No            |
| CUSTOM    | [0,6]     | 0 (Sun)           | Yes           |
| CUSTOM    | [0,6]     | 3 (Wed)           | No            |

---

## Test Anchors

### Routines Page

1. Given 3 routines exist (2 active, 1 inactive), when the Routines page loads, then all 3 routines are displayed, with the inactive routine rendered at `opacity-50`.

2. Given today is Wednesday (day=3) and routines exist with frequencies DAILY, WEEKDAYS, and CUSTOM([0,6]), when the page renders, then the DAILY and WEEKDAYS routines show the "Applies today" indicator, but the CUSTOM weekend routine does not.

3. Given the user fills out the "Add Routine" form with title "Morning run", category EXERCISE, frequency DAILY, duration 30, when they submit, then `createRoutine` is called and the new routine appears in the list without a page refresh.

4. Given a routine with frequency WEEKLY exists, when the user clicks edit and changes `daysOfWeek` from [1,3] to [1,3,5], then `updateRoutine` is called with the updated days and the routine's display updates.

5. Given the user clicks delete on a routine and confirms, when `deleteRoutine` succeeds, then the routine is removed from the list.

6. Given the user opens the "Add Routine" form and selects frequency CUSTOM but does not select any days, when they click submit, then a validation error "Select at least one day" is shown and no mutation is called.

### Context Page

7. Given 9 seed context entries exist across 4 categories, when the Context page loads, then entries are grouped under "Life", "Constraints", "Equipment", "Preferences" headings in that order.

8. Given a context entry with key "work_window" and value "Deep focus: 9–4", when the user clicks the value and changes it to "Deep focus: 8–3" and presses Enter, then `upsertContext` is called with the updated value and the displayed text updates.

9. Given an active context entry, when the user toggles it inactive, then `toggleContext(id, false)` is called and the entry renders dimmed with "inactive — not sent to planner" text.

10. Given the user clicks "Add Entry" under the Constraints group and fills in key "commute" and value "15 min walk", when they submit, then `upsertContext` is called with `category: CONSTRAINTS` and the new entry appears in the Constraints group.

### History Page

11. Given 5 accepted plans exist with various dates, when the History page loads, then plans are listed most-recent-first with date, status badge, and block summary.

12. Given an accepted plan for Mar 4 with 6 blocks (2 skipped), when the user clicks on that plan row, then the detail view expands showing all 6 blocks, with the 2 skipped blocks rendered with strikethrough and "Skipped" label.

13. Given no plans exist, when the History page loads, then the empty state "No plans yet. Start planning on the Today page." is displayed.

14. Given a plan has 4 chat messages (2 user, 2 assistant), when the user expands the plan detail and opens the chat history section, then all 4 messages are shown in chronological order with correct left/right alignment by role.
