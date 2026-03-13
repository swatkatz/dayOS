# Spec: Calendar

## Bounded Context

Owns: `backend/calendar/` package (Google Calendar OAuth, event fetching, memcached caching), `google_auth` table, GraphQL `calendarEvents` query, `CalendarEvent` and `CalendarEventsPayload` types, OAuth callback HTTP handler, frontend calendar polling + replan trigger

Does not own: plan generation (planner), plan storage (day-plans), block CRUD (day-plans), memcached infrastructure (ops/deployment)

Depends on:
- `foundation` — shared Postgres for `google_auth` table
- `planner` — planner reads calendar events when building prompts (planner calls this package)
- `day-plans` — replan triggered via existing `sendPlanMessage` mutation
- `auth` — `APP_SECRET` protects GraphQL endpoint (existing middleware)
- `validation` — `validate.FormatContextData()` for safe prompt embedding of calendar events

Produces:
- GraphQL query: `calendarEvents(date: Date!): CalendarEventsPayload!`
- GraphQL mutation: `connectGoogleCalendar(code: String!): Boolean!`
- GraphQL mutation: `disconnectGoogleCalendar: Boolean!`
- GraphQL query: `googleCalendarStatus: GoogleCalendarStatus!`
- HTTP endpoint: `GET /auth/google/callback` (OAuth redirect handler)
- Data written: `google_auth` table (OAuth tokens)
- Data cached: calendar events in memcached (keyed by date, 20-min TTL)

## Contracts

### Input

**`calendarEvents(date: Date!): CalendarEventsPayload!`**

Reads:
- Memcached: cached events for the given date
- Google Calendar API: if cache miss, fetches events for the date using stored OAuth tokens
- `google_auth` table: OAuth access/refresh tokens

**`connectGoogleCalendar(code: String!): Boolean!`**

Receives:
- OAuth authorization code from Google's OAuth consent flow

**`disconnectGoogleCalendar: Boolean!`**

No external reads — deletes the `google_auth` row.

**`googleCalendarStatus: GoogleCalendarStatus!`**

Reads:
- `google_auth` table: checks if a valid token exists

### Output

**`CalendarEventsPayload`**
```graphql
type CalendarEvent {
  title:     String!
  startTime: String!    # "HH:MM" format (local time)
  duration:  Int!       # minutes
  allDay:    Boolean!
}

type CalendarEventsPayload {
  events:    [CalendarEvent!]!
  version:   String!    # hash of event list — used for change detection
  connected: Boolean!   # whether Google Calendar is connected
}

type GoogleCalendarStatus {
  connected:    Boolean!
  calendarName: String   # primary calendar name, null if not connected
}
```

**GraphQL additions:**
```graphql
type Query {
  calendarEvents(date: Date!): CalendarEventsPayload!
  googleCalendarStatus: GoogleCalendarStatus!
}

type Mutation {
  connectGoogleCalendar(code: String!): Boolean!
  disconnectGoogleCalendar: Boolean!
}
```

### Data Model

```sql
CREATE TABLE google_auth (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  access_token   TEXT NOT NULL,
  refresh_token  TEXT NOT NULL,
  token_expiry   TIMESTAMPTZ NOT NULL,
  calendar_id    TEXT NOT NULL DEFAULT 'primary',  -- which calendar to read
  created_at     TIMESTAMPTZ DEFAULT now(),
  updated_at     TIMESTAMPTZ DEFAULT now()
);
```

Single row — this is a single-user app. Only one Google account connected at a time.

### sqlc Queries

File: `backend/db/queries/google_auth.sql`

```sql
-- name: GetGoogleAuth :one
SELECT * FROM google_auth LIMIT 1;

-- name: UpsertGoogleAuth :one
INSERT INTO google_auth (access_token, refresh_token, token_expiry, calendar_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT ((true)) DO UPDATE SET   -- single-row upsert
  access_token = EXCLUDED.access_token,
  refresh_token = EXCLUDED.refresh_token,
  token_expiry = EXCLUDED.token_expiry,
  calendar_id = EXCLUDED.calendar_id,
  updated_at = now()
RETURNING *;

-- name: DeleteGoogleAuth :exec
DELETE FROM google_auth;
```

Note: The single-row upsert requires a unique constraint. Add a partial unique index:
```sql
CREATE UNIQUE INDEX idx_google_auth_singleton ON google_auth ((true));
```

## Calendar Package Architecture

File: `backend/calendar/calendar.go`

```go
type Service struct {
    DB        *db.Queries
    Memcached *memcache.Client
    OAuthCfg  *oauth2.Config
}

type Event struct {
    Title     string `json:"title"`
    StartTime string `json:"start_time"` // "HH:MM"
    Duration  int    `json:"duration"`    // minutes
    AllDay    bool   `json:"all_day"`
}

type EventsResult struct {
    Events  []Event
    Version string // SHA-256 hash of serialized events
}

func (s *Service) GetEvents(ctx context.Context, date time.Time) (*EventsResult, error)
func (s *Service) ExchangeCode(ctx context.Context, code string) error
func (s *Service) Disconnect(ctx context.Context) error
func (s *Service) IsConnected(ctx context.Context) (bool, error)
```

### Memcached key scheme

Key: `cal:events:YYYY-MM-DD`
Value: JSON-encoded `{events: [...], version: "hash"}`
TTL: 20 minutes

### Version computation

`version` = hex-encoded SHA-256 of the JSON-serialized events array (sorted by start time). This is deterministic — same events always produce the same version, so the frontend can reliably detect changes.

### Token refresh

When the stored `access_token` is expired (checked via `token_expiry`):
1. Use `refresh_token` to get a new access token from Google
2. Update the `google_auth` row with the new access token + expiry
3. Proceed with the API call

If the refresh token is also invalid (revoked, expired), return `connected: false` in the payload so the frontend can prompt re-authorization.

### Data minimization

When fetching from Google Calendar API, extract only:
- `summary` → `title`
- `start.dateTime` → `startTime` (converted to "HH:MM" in local timezone)
- `end.dateTime` - `start.dateTime` → `duration` (in minutes)
- `start.date` (no time) → `allDay: true`

Discard: attendees, description, location, meeting links, organizer, reminders, attachments.

## OAuth Flow

### Setup
- Google Cloud project with Calendar API enabled
- OAuth 2.0 credentials (client ID + client secret)
- Env vars: `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URI`
- Scopes: `https://www.googleapis.com/auth/calendar.events.readonly`

### Connect flow
1. Frontend: user clicks "Connect Google Calendar" on the manage page
2. Frontend: opens Google OAuth consent URL in a popup/redirect
   - URL constructed from `GOOGLE_CLIENT_ID`, `GOOGLE_REDIRECT_URI`, scope
   - The OAuth URL can be served via a `googleOAuthURL` query or constructed client-side
3. Google: user consents, redirected to `GOOGLE_REDIRECT_URI` with `?code=...`
4. Backend: `GET /auth/google/callback` receives the code, calls `connectGoogleCalendar(code)` internally (or the frontend extracts the code and calls the mutation)
5. Backend: exchanges code for access + refresh tokens via Google's token endpoint
6. Backend: stores tokens in `google_auth` table
7. Frontend: detects success, shows "Connected" status

### Disconnect flow
1. Frontend: user clicks "Disconnect" on manage page
2. Calls `disconnectGoogleCalendar` mutation
3. Backend: deletes `google_auth` row, optionally revokes the token with Google
4. Frontend: shows "Not connected" status

## Planner Integration

### Prompt section

When `sendPlanMessage` builds the system prompt, if Google Calendar is connected:

1. Resolver calls `calendar.GetEvents(ctx, date)` to get today's events
2. Events are passed to the planner as a new field on `PlanChatInput`:

```go
type PlanChatInput struct {
    // ... existing fields ...
    CalendarEvents []CalendarEventInfo
}

type CalendarEventInfo struct {
    Title     string `json:"title"`
    StartTime string `json:"start_time"`
    Duration  int    `json:"duration_min"`
    AllDay    bool   `json:"all_day"`
}
```

3. Planner adds a new section to the system prompt (between CONTEXT and ROUTINES):

```
TODAY'S CALENDAR EVENTS (fixed — do NOT move, overlap, or reschedule these):
<user-data>
[
  {"title": "Team standup", "start_time": "09:30", "duration_min": 30, "all_day": false},
  {"title": "Design review", "start_time": "13:00", "duration_min": 60, "all_day": false}
]
</user-data>

ALL-DAY EVENTS (for awareness, not time-blocked):
<user-data>
[
  {"title": "Elijah's birthday", "all_day": true}
]
</user-data>

CALENDAR RULES:
- Calendar events are IMMOVABLE. Schedule all tasks and routines around them.
- Leave 10 min buffer before and after meetings for context switching.
- Do not schedule deep focus work in gaps shorter than 45 min between meetings.
- All-day events are informational — mention them in relevant block notes if useful.
```

4. If Google Calendar is not connected, this section is omitted entirely (existing behavior preserved).

### Calendar events as blocks

The AI will generate blocks for calendar events like any other block. These blocks should have:
- `task_id: null`
- `routine_id: null`
- `category`: the AI's best guess based on the event title (e.g., "interview" for "Meta phone screen")
- The block times should match the calendar event times exactly

## Frontend Integration

### Today page: polling + replan detection

```typescript
// Poll every 15 mins + on mount + on tab focus
const POLL_INTERVAL = 15 * 60 * 1000; // 15 minutes

// Store the version from when the plan was last built/replanned
const [planCalendarVersion, setPlanCalendarVersion] = useState<string | null>(null);
const [showReplanBanner, setShowReplanBanner] = useState(false);

// On each poll response:
if (data.calendarEvents.version !== planCalendarVersion && plan?.status === 'ACCEPTED') {
  setShowReplanBanner(true);
}
```

### Replan banner

When calendar events change on an accepted plan:

```
┌─────────────────────────────────────────────────────────┐
│  📅 Your calendar has changed. Replan remaining blocks? │
│                              [ Dismiss ]  [ Replan ]    │
└─────────────────────────────────────────────────────────┘
```

- **Replan**: calls `sendPlanMessage` with an auto-generated message: `"Calendar events have changed. Replan from now."` — this triggers the existing replanning flow (preserves past blocks, reschedules from current time).
- **Dismiss**: hides the banner, updates `planCalendarVersion` to the new version (so it won't re-trigger until the next change).

### Manage page: connection UI

On the `/manage` page (or `/context` page — wherever settings live):

```
┌─────────────────────────────────────────────────┐
│  Google Calendar                                │
│                                                 │
│  Status: Connected (swati@gmail.com)            │
│  Calendar: Primary                              │
│                                                 │
│  [ Disconnect ]                                 │
└─────────────────────────────────────────────────┘
```

Or if not connected:

```
┌─────────────────────────────────────────────────┐
│  Google Calendar                                │
│                                                 │
│  Connect your Google Calendar to automatically  │
│  include meetings in your daily plan.           │
│                                                 │
│  [ Connect Google Calendar ]                    │
└─────────────────────────────────────────────────┘
```

## Environment Variables

```
GOOGLE_CLIENT_ID=          # Google OAuth client ID
GOOGLE_CLIENT_SECRET=      # Google OAuth client secret
GOOGLE_REDIRECT_URI=       # OAuth callback URL (e.g., https://dayos.up.railway.app/auth/google/callback)
MEMCACHED_URL=             # Memcached connection string (e.g., localhost:11211)
```

## Behaviors (EARS syntax)

### Event fetching

- When `calendarEvents(date)` is called and Google Calendar is connected, the system shall check memcached for cached events for that date.
- When the memcached cache contains events for the requested date (cache hit), the system shall return the cached events and version without calling the Google Calendar API.
- When the memcached cache does not contain events for the requested date (cache miss), the system shall fetch events from the Google Calendar API, write them to memcached with a 20-minute TTL, and return the events and version.
- When `calendarEvents(date)` is called and Google Calendar is not connected, the system shall return an empty events list with `connected: false`.
- When fetching events from Google Calendar API, the system shall extract only title, start time, duration, and all-day status — discarding attendees, description, location, meeting links, and all other fields.
- When computing the version hash, the system shall sort events by start time, serialize to JSON, and compute SHA-256 — producing a deterministic hash for the same set of events.

### OAuth

- When `connectGoogleCalendar(code)` is called, the system shall exchange the authorization code for access and refresh tokens and store them in the `google_auth` table.
- When a `google_auth` row already exists and `connectGoogleCalendar` is called, the system shall overwrite the existing tokens (single-user, single-row upsert).
- When `disconnectGoogleCalendar` is called, the system shall delete the `google_auth` row.
- When the stored access token is expired, the system shall use the refresh token to obtain a new access token and update the `google_auth` row before proceeding.
- When the refresh token is invalid (revoked or expired), the system shall return `connected: false` so the frontend can prompt re-authorization.

### Planner integration

- When `sendPlanMessage` builds the system prompt and Google Calendar is connected, the system shall include a `TODAY'S CALENDAR EVENTS` section with time-blocked events and an `ALL-DAY EVENTS` section with all-day events, both formatted as JSON inside `<user-data>` tags.
- When Google Calendar is not connected, the system shall omit the calendar sections entirely (backward-compatible).
- When calendar events are included in the prompt, the system shall instruct the AI to treat them as immovable and schedule everything else around them.

### Frontend change detection

- When the Today page mounts, the system shall fetch `calendarEvents` for today.
- When the Today page is in the foreground, the system shall poll `calendarEvents` every 15 minutes.
- When the browser tab regains focus, the system shall refetch `calendarEvents`.
- When the returned version differs from the version used to build the current accepted plan, the system shall show a replan banner.
- When the user confirms the replan, the system shall call `sendPlanMessage` with a message indicating calendar changes, triggering the existing replanning flow from the current time.
- When the user dismisses the replan banner, the system shall update the stored version to the new version and hide the banner.

### Version tracking

- When a plan is created or replanned via `sendPlanMessage`, the resolver shall store the current calendar events version alongside the plan. This can be stored as a field on `day_plans` or tracked in the frontend state.
- When comparing versions for replan detection, the system shall compare the latest polled version against the version that was current when the plan was last built.

## Decision Table: calendarEvents behavior

| Connected? | Cache hit? | Token valid? | Action |
|-----------|-----------|-------------|--------|
| No | - | - | Return `{events: [], version: "", connected: false}` |
| Yes | Yes | - | Return cached events + version |
| Yes | No | Yes | Fetch from Google, cache, return events + version |
| Yes | No | Expired (refresh works) | Refresh token, fetch from Google, cache, return |
| Yes | No | Expired (refresh fails) | Delete `google_auth`, return `{events: [], connected: false}` |

## Test Anchors

1. **Given** Google Calendar is connected and memcached has cached events for today, **when** `calendarEvents(date: today)` is called, **then** cached events and version are returned without calling the Google Calendar API.

2. **Given** Google Calendar is connected and memcached has no cached events for today, **when** `calendarEvents(date: today)` is called, **then** events are fetched from Google Calendar API, written to memcached with 20-min TTL, and returned with a version hash.

3. **Given** Google Calendar is not connected, **when** `calendarEvents(date: today)` is called, **then** `{events: [], version: "", connected: false}` is returned.

4. **Given** a valid OAuth authorization code, **when** `connectGoogleCalendar(code)` is called, **then** tokens are stored in `google_auth` and subsequent `calendarEvents` calls return real events.

5. **Given** Google Calendar is connected, **when** `disconnectGoogleCalendar` is called, **then** the `google_auth` row is deleted and subsequent `calendarEvents` calls return `connected: false`.

6. **Given** the stored access token is expired but the refresh token is valid, **when** `calendarEvents` is called, **then** a new access token is obtained, stored, and events are fetched successfully.

7. **Given** both access and refresh tokens are invalid, **when** `calendarEvents` is called, **then** the `google_auth` row is deleted and `connected: false` is returned.

8. **Given** Google Calendar returns 3 timed events and 1 all-day event, **when** events are fetched, **then** only title, start time, duration, and all-day flag are extracted — no attendees, descriptions, or links.

9. **Given** the same set of calendar events is fetched twice, **when** the version is computed each time, **then** both versions are identical (deterministic hash).

10. **Given** a calendar event is added between two fetches, **when** the version is computed, **then** the new version differs from the old version.

11. **Given** Google Calendar is connected and events exist, **when** `sendPlanMessage` builds the system prompt, **then** the prompt includes a `TODAY'S CALENDAR EVENTS` section with events as JSON inside `<user-data>` tags, and a `CALENDAR RULES` section instructing the AI to treat them as immovable.

12. **Given** Google Calendar is not connected, **when** `sendPlanMessage` builds the system prompt, **then** no calendar section appears in the prompt.

13. **Given** an accepted plan was built with calendar version "abc", **when** the frontend polls and receives version "def", **then** the replan banner is shown.

14. **Given** a draft plan exists, **when** the frontend polls and receives a changed calendar version, **then** no replan banner is shown (draft plans pick up fresh events on next message).
