# Spec: Context

## Bounded Context

Owns: Context entry CRUD resolvers, upsert logic (by category+key), sqlc queries for context_entries

Does not own: `context_entries` table DDL (foundation), seed data insertion (foundation migration 007), formatting context for AI prompts (planner)

Depends on: foundation (table + schema + seed data exist)

Produces: GraphQL query (`contextEntries`) and mutations (`upsertContext`, `deleteContext`)

## Contracts

### Input

```graphql
input UpsertContextInput {
  category: ContextCategory!   # LIFE | CONSTRAINTS | EQUIPMENT | PREFERENCES | CUSTOM
  key:      String!            # short label, e.g. "nanny_hours"
  value:    String!            # the actual content
}
```

### Output

```graphql
type ContextEntry {
  id, category, key, value, isActive, createdAt
}

# Queries
contextEntries(category: ContextCategory): [ContextEntry!]!

# Mutations
upsertContext(input: UpsertContextInput!): ContextEntry!
deleteContext(id: UUID!): Boolean!
```

### sqlc Queries

File: `backend/db/queries/context_entries.sql`

```sql
-- name: ListContextEntries :many
SELECT * FROM context_entries
WHERE (sqlc.narg('category')::TEXT IS NULL OR category = sqlc.narg('category'))
ORDER BY category, key;

-- name: GetContextEntry :one
SELECT * FROM context_entries WHERE id = $1;

-- name: UpsertContextEntry :one
INSERT INTO context_entries (category, key, value)
VALUES ($1, $2, $3)
ON CONFLICT (category, key) DO UPDATE SET
  value = EXCLUDED.value,
  updated_at = now()
RETURNING *;

-- name: ToggleContextEntry :one
UPDATE context_entries SET is_active = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteContextEntry :exec
DELETE FROM context_entries WHERE id = $1;

-- name: ListActiveContextEntries :many
-- Used by planner: all active entries for prompt construction
SELECT * FROM context_entries
WHERE is_active = true
ORDER BY category, key;
```

**Important:** The `UpsertContextEntry` query requires a unique index on `(category, key)`. This must be added to the foundation migrations:

```sql
-- Add to 003_create_context_entries.up.sql
CREATE UNIQUE INDEX idx_context_entries_category_key ON context_entries(category, key);
```

## Behaviors (EARS syntax)

### Input validation

Input validation uses gqlgen directives (see `specs/validation.md`). Fields annotated with `@validate` in `schema.graphqls` are automatically sanitized and length-checked by the directive handler before the resolver runs. Converters do NOT perform text validation.

- When `upsertContext` is called with an annotated field, the `@validate` directive handler shall sanitize and length-check the value before the resolver executes.
- When validation fails, the directive handler shall return the validation error without the resolver or database being reached.

### GraphQL directive annotations

```graphql
input UpsertContextInput {
  category: ContextCategory!
  key:      String!  @validate(rule: SINGLE_LINE_SHORT)
  value:    String!  @validate(rule: PLAIN_TEXT)
}
```

### Upsert behavior

- When `upsertContext` is called with a `category`+`key` that already exists, the system shall update the `value` and `updated_at` (not create a duplicate).
- When `upsertContext` is called with a new `category`+`key`, the system shall insert a new entry with `isActive = true`.
- When `contextEntries(category: CONSTRAINTS)` is called, the system shall return only entries in the `constraints` category.
- When `contextEntries()` is called with no category filter, the system shall return all entries grouped by category then key.
- When `deleteContext` is called, the system shall hard-delete the entry.
- When `deleteContext` is called with a non-existent ID, the system shall return an error.

### Active/inactive toggle

- The `upsertContext` mutation does not change `isActive` — it only sets/updates the value. Toggling active/inactive is done via a separate resolver or `updateBlock`-style approach. For simplicity, add an `updateContext` mutation or handle via a dedicated `toggleContextEntry(id, isActive)` pattern.

**Decision:** Add a `toggleContext(id: UUID!, isActive: Boolean!): ContextEntry!` mutation to the GraphQL schema. This is not in the original DESIGN.md schema but is needed for the frontend toggle functionality described on the `/context` page. Alternatively, reuse `upsertContext` — but upsert is keyed on category+key, not id, making it awkward for toggling. A dedicated mutation is cleaner.

```graphql
# Additional mutation (add to schema.graphqls)
toggleContext(id: UUID!, isActive: Boolean!): ContextEntry!
```

## Test Anchors

1. Given seed context exists (9 entries from migration 007), when `contextEntries()` is called, then all 9 entries are returned ordered by category then key.

2. Given a context entry exists with `category=constraints, key=work_window`, when `upsertContext({category: CONSTRAINTS, key: "work_window", value: "New schedule"})` is called, then the existing entry is updated (not duplicated) and the new value is returned.

3. Given no entry with `key=commute` exists, when `upsertContext({category: CONSTRAINTS, key: "commute", value: "15 min walk"})` is called, then a new entry is created with `isActive = true`.

4. Given an active context entry exists, when `toggleContext(id, false)` is called, then the entry's `isActive` is false. When `ListActiveContextEntries` is queried, it is excluded.

5. Given a context entry exists, when `deleteContext(id)` is called, then the entry is permanently removed and subsequent queries do not return it.

6. Given a key of 101 characters, when `upsertContext({category: CONSTRAINTS, key: longKey, value: "test"})` is called, then a validation error `"key must be 100 characters or fewer"` is returned.

7. Given a key `"work\nwindow"`, when `upsertContext({category: CONSTRAINTS, key: "work\nwindow", value: "9-5"})` is called, then the entry is created with key `"workwindow"` (newline stripped).

8. Given a value of 1001 characters, when `upsertContext({category: CONSTRAINTS, key: "test", value: longValue})` is called, then a validation error is returned.
