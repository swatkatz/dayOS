# Spec: Frontend Backlog

## Bounded Context

Owns: `/backlog` page implementation (`frontend/src/pages/BacklogPage.tsx`), task list components (parent/subtask grouping, standalone section), quick-add form, inline edit, task completion/deletion UI, "Scope with AI" chat panel, category/priority filter controls, completed tasks collapsible section

Does not own: Task CRUD mutations or resolvers (owned by `tasks`), task scoping AI logic (owned by `planner`), GraphQL schema (owned by backend specs), app shell/sidebar/routing (owned by `frontend-shell`), Apollo Client setup (owned by `frontend-shell`)

Depends on: `frontend-shell` (layout, routing, Apollo Client, Tailwind theme, `CATEGORY_COLORS` from `src/constants.ts`), `tasks` (GraphQL queries/mutations for task CRUD), `planner` (GraphQL mutations for task scoping chat: `startTaskConversation`, `sendTaskMessage`, `confirmTaskBreakdown`)

Produces: Fully functional `/backlog` page replacing the stub from `frontend-shell`

## Contracts

### GraphQL Operations Used

**Queries:**

```graphql
query Tasks($category: Category, $includeCompleted: Boolean) {
  tasks(category: $category, includeCompleted: $includeCompleted) {
    id
    title
    category
    priority
    parentId
    estimatedMinutes
    actualMinutes
    deadlineType
    deadlineDate
    deadlineDays
    notes
    isRoutine
    timesDeferred
    isCompleted
    completedAt
    createdAt
    subtasks {
      id
      title
      category
      priority
      estimatedMinutes
      actualMinutes
      notes
      isCompleted
      completedAt
      timesDeferred
    }
  }
}

query TaskConversation($id: UUID!) {
  taskConversation(id: $id) {
    id
    parentTaskId
    status
    messages {
      id
      role
      content
      createdAt
    }
  }
}
```

**Mutations:**

```graphql
mutation CreateTask($input: CreateTaskInput!) {
  createTask(input: $input) { id title category priority estimatedMinutes }
}

mutation UpdateTask($id: UUID!, $input: UpdateTaskInput!) {
  updateTask(id: $id, input: $input) { id title category priority estimatedMinutes notes deadlineType deadlineDate deadlineDays isCompleted }
}

mutation DeleteTask($id: UUID!) {
  deleteTask(id: $id)
}

mutation CompleteTask($id: UUID!) {
  completeTask(id: $id) { id isCompleted completedAt }
}

mutation StartTaskConversation($message: String!) {
  startTaskConversation(message: $message) { id status messages { id role content createdAt } }
}

mutation SendTaskMessage($conversationId: UUID!, $message: String!) {
  sendTaskMessage(conversationId: $conversationId, message: $message) { id status messages { id role content createdAt } }
}

mutation ConfirmTaskBreakdown($conversationId: UUID!) {
  confirmTaskBreakdown(conversationId: $conversationId) { id title category priority parentId estimatedMinutes }
}
```

### Component Structure

```
pages/
  BacklogPage.tsx          # Main page — filter bar, task list, quick-add, AI chat panel
components/
  backlog/
    TaskGroup.tsx           # Parent task heading + nested subtask list
    StandaloneSection.tsx   # Standalone tasks grouped by category
    TaskRow.tsx             # Single task row — inline edit, complete, delete
    QuickAddForm.tsx        # Inline form for creating standalone tasks
    TaskFilters.tsx         # Category + priority filter dropdowns
    ScopeChat.tsx           # "Scope with AI" chat panel (slide-over or modal)
    CompletedSection.tsx    # Collapsible section for completed tasks
```

## Visual Design

### Page Layout

```
┌─────────────────────────────────────────────────────────────┐
│  Task Backlog                        [+ Quick Add] [Scope with AI] │
│                                                             │
│  Filters: [Category ▾] [Priority ▾]                        │
│                                                             │
│  ── Parent Tasks ──────────────────────────────────────────│
│                                                             │
│  ◈ Prepare for Meta interview        [interview] due Mar 15 │
│    Progress: ████████░░ 4/6 subtasks                        │
│    ├─ ✓ Review DP fundamentals           2hr / 2hr          │
│    ├─ □ Coin Change II variations        0hr / 1.5hr        │
│    ├─ □ Graph BFS/DFS practice           0hr / 2hr          │
│    └─ ...                                                   │
│                                                             │
│  ◈ Set up BabyBaton MVP              [project] within 10d   │
│    Progress: ██░░░░░░░░ 1/5 subtasks                        │
│    ├─ ...                                                   │
│                                                             │
│  ── Standalone Tasks ──────────────────────────────────────│
│                                                             │
│  □ Tailor resume for Airbnb          [job] HIGH   60 min    │
│  □ Buy groceries for dal makhani     [meal] LOW   30 min    │
│                                                             │
│  ── Completed (3) ▾ ───────────────────────────────────────│
│    ✓ Submit Google application        [job]  completed 2d ago│
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Styling Rules

- Page title: `text-2xl font-semibold text-primary`
- Section headers ("Parent Tasks", "Standalone Tasks", "Completed"): `text-sm font-medium text-secondary uppercase tracking-wide`, with `border-b border-default` divider
- Task rows: `bg-surface rounded-lg p-3 mb-2`, category color as `border-l-4` using `CATEGORY_COLORS`
- Subtask rows: indented `ml-6`, slightly smaller text `text-sm`
- Completed tasks: `text-secondary line-through` for title
- Quick-add and filter controls: `bg-surface` inputs with `border-default` borders, `text-primary` text
- Buttons: accent gold for primary actions (`bg-accent text-bg-primary`), `bg-surface-hover` for secondary
- Priority badges: HIGH = `text-red-400`, MEDIUM = `text-amber-400`, LOW = `text-secondary`
- Deferred indicator: if `timesDeferred >= 2`, show a small `text-amber-400` indicator (e.g., "deferred 3x")

### Deadline Display

| Deadline Type | Display Format |
|---|---|
| `HARD` | `due Mar 15` (formatted date) |
| `HORIZON` | `within 14 days` |
| None | No deadline shown |
| Overdue (hard, past date) | `OVERDUE — due Mar 1` in `text-red-400` |

### Progress Display (Parent Tasks)

- Text: `{completed_subtasks}/{total_subtasks} subtasks`
- Progress bar: thin horizontal bar, filled portion uses category color, unfilled is `bg-surface-hover`

### Progress Display (Subtasks / Standalone)

- Text: `{actual_minutes formatted} / {estimated_minutes formatted}` (e.g., "2hr / 3hr", "30min / 60min")
- Format: minutes < 60 → `{m}min`, minutes >= 60 → `{h}hr` (or `{h}hr {m}min` if not even)

## Behaviors (EARS syntax)

### Task List & Grouping

- When the page loads, the system shall query `tasks(includeCompleted: true)` and group them into: (1) parent tasks with their subtasks nested, (2) standalone tasks (no `parentId`, has `estimatedMinutes`), (3) completed tasks.
- The system shall display parent tasks first, then standalone tasks, then completed tasks in a collapsible section.
- When a parent task has all subtasks completed, the system shall display the parent as completed in the completed section.
- Where a task has `timesDeferred >= 2`, the system shall show a deferred count indicator next to the task title.

### Quick Add (Standalone Task)

- When the user clicks "+ Quick Add", the system shall show an inline form with fields: title (required), category (dropdown, default `ADMIN`), priority (dropdown, default `MEDIUM`), estimated minutes (number, default 60).
- When the user submits the quick-add form, the system shall call `createTask` with the provided values and no `parentId`.
- When the task is created, the system shall add it to the standalone section and clear the form.
- When the user presses Escape or clicks cancel, the system shall close the form without creating a task.

### Inline Edit

- When the user clicks on a task title, the system shall make it editable inline (contentEditable or input replacement).
- When the user clicks on estimated minutes, priority, or deadline fields, the system shall show an inline editor for that field.
- When the user confirms an inline edit (Enter or blur), the system shall call `updateTask` with only the changed field.
- If `updateTask` fails, the system shall revert the field to its previous value and show an error toast.

### Completion

- When the user clicks the checkbox on a standalone task or subtask, the system shall call `completeTask` with that task's ID.
- When `completeTask` returns with the parent also completed (checked via refetch), the system shall move the parent to the completed section.
- When the user clicks the checkbox on a completed task, the system shall call `updateTask(id, { isCompleted: false })` to uncomplete it.
- The system shall not render a checkbox on parent tasks (parent completion is automatic).

### Deletion

- When the user clicks the delete button on a task, the system shall show a confirmation prompt: "Delete this task?" (for parent tasks: "Delete this task and all subtasks?").
- When confirmed, the system shall call `deleteTask` and remove the task from the UI.
- When the user cancels deletion, no action is taken.

### Filtering

- When the user selects a category filter, the system shall re-query `tasks(category: $selected)` or filter client-side.
- When the user selects a priority filter, the system shall filter the displayed tasks client-side to only show tasks matching that priority.
- When "All" is selected for either filter, the system shall show all tasks (no filter applied).
- Filters shall apply to parent tasks, standalone tasks, and completed tasks uniformly.

### Completed Section

- The system shall render completed tasks in a collapsible section at the bottom, collapsed by default.
- When the user clicks the section header, the system shall toggle visibility of completed tasks.
- The section header shall show the count of completed tasks: `"Completed (N)"`.

### "Scope with AI" Chat

- When the user clicks "Scope with AI", the system shall open a slide-over panel on the right side of the page.
- The panel shall contain a chat interface: message history (scrollable) + text input at the bottom.
- When the user types a message and presses Enter (or clicks Send), the system shall call `startTaskConversation` (first message) or `sendTaskMessage` (subsequent messages).
- While waiting for the AI response, the system shall show a loading indicator in the chat.
- When the AI responds with `{"status": "question", "message": "..."}`, the system shall display the `message` text as an assistant chat bubble.
- When the AI responds with `{"status": "proposal", ...}`, the system shall parse and display the proposal in a structured format: parent task title, deadline, and a list of subtasks with their estimated minutes.
- When a proposal is displayed, the system shall show a "Create Tasks" button below the proposal.
- When the user clicks "Create Tasks", the system shall call `confirmTaskBreakdown` and, on success, close the chat panel and refetch the task list.
- When `confirmTaskBreakdown` returns an error, the system shall display the error in the chat panel.
- When the user clicks outside the panel or clicks a close button, the system shall close the panel. The conversation is not lost — reopening continues from where it left off if the conversation is still active.

### Proposal Display in Chat

When an assistant message contains a proposal JSON, render it as:

```
┌─────────────────────────────────────────┐
│  📋 Proposed Breakdown                  │
│                                         │
│  Parent: Set up BabyBaton MVP           │
│  Category: project | Priority: high     │
│  Deadline: within 14 days               │
│                                         │
│  Subtasks:                              │
│  1. Set up Next.js project — 60 min     │
│  2. Design data models — 90 min         │
│  3. Build auth flow — 120 min           │
│                                         │
│  Total: 4hr 30min                       │
│                                         │
│           [Create Tasks]  [Adjust...]   │
└─────────────────────────────────────────┘
```

- "Adjust..." sends a message like "I'd like to adjust..." pre-filling the chat input so the user can provide feedback.

### Error Handling

- When a GraphQL query or mutation fails, the system shall display an inline error message (not a modal) near the affected component.
- When the task list fails to load, the system shall show a centered error state with a "Retry" button.
- While any mutation is in flight, the system shall disable the triggering button to prevent double-submission.

## Decision Table: Assistant Message Rendering

| Raw content parses as JSON? | `status` field | Rendering |
|---|---|---|
| Yes | `"question"` | Display `.message` as plain text chat bubble |
| Yes | `"proposal"` | Display structured proposal card with "Create Tasks" button |
| No | N/A | Display raw content as plain text chat bubble (fallback) |

## Test Anchors

1. Given 2 parent tasks (with 3 and 2 subtasks respectively) and 3 standalone tasks exist, when the backlog page loads, then parent tasks are shown with nested subtasks, standalone tasks are shown in a separate section, and the grouping is visually distinct.

2. Given the quick-add form is open, when the user fills in title "Buy diapers", selects category BABY, and submits, then `createTask` is called with `{title: "Buy diapers", category: BABY, priority: MEDIUM, estimatedMinutes: 60}` and the new task appears in the standalone section.

3. Given a parent task with 2 subtasks (1 completed, 1 incomplete), when the user completes the remaining subtask, then both the subtask and parent move to the completed section.

4. Given tasks exist across categories, when the user selects "Interview" in the category filter, then only interview tasks (and parent tasks containing interview subtasks) are visible.

5. Given the user clicks "Scope with AI" and types "I want to prepare for my Meta onsite interview", when the AI responds with a question, then the question is displayed as a chat bubble and the input is ready for the next message.

6. Given a task scoping conversation has a proposal response, when the user clicks "Create Tasks", then `confirmTaskBreakdown` is called, the chat panel closes, and the task list is refetched showing the new parent + subtasks.

7. Given a task with `deadlineType: HARD` and `deadlineDate: "2026-03-15"`, when rendered in the task list, then the deadline displays as "due Mar 15".

8. Given a task with `actualMinutes: 120` and `estimatedMinutes: 180`, when rendered as a subtask row, then the progress displays as "2hr / 3hr".

9. Given 5 completed tasks exist, when the page loads, then the completed section header shows "Completed (5)" and the section is collapsed by default.

10. Given a parent task exists, when the user clicks delete on it, then a confirmation prompt mentions subtasks will also be deleted, and on confirm, `deleteTask` is called and the parent + subtasks are removed from the UI.
