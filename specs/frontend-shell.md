# Spec: Frontend Shell

## Bounded Context

Owns: Vite + React + TypeScript project setup, Tailwind CSS configuration (dark theme), Apollo Client setup with auth header, GraphQL codegen config, app shell layout (sidebar navigation, page container), React Router route definitions, simple auth gate (password entry → localStorage), notification permission request

Does not own: Page implementations (owned by `frontend-today`, `frontend-backlog`, `frontend-manage`), backend auth middleware (owned by `deployment`), GraphQL schema (owned by backend specs)

Depends on: `foundation` (GraphQL endpoint must exist at `/graphql`), `deployment` (defines `APP_SECRET` env var and backend auth middleware)

Produces: Runnable frontend dev server, Apollo Client instance available to all pages, route structure, shared layout components, auth token in localStorage, Tailwind theme with DayOS design tokens

## Contracts

### Project Structure

```
frontend/
├── src/
│   ├── main.tsx              # React entry point
│   ├── App.tsx               # Router + Apollo Provider + Auth Gate
│   ├── apollo.ts             # Apollo Client setup
│   ├── components/
│   │   ├── Layout.tsx        # Sidebar + page container
│   │   ├── Sidebar.tsx       # Nav links, active indicator
│   │   └── AuthGate.tsx      # Password prompt if not authed
│   ├── pages/
│   │   ├── TodayPage.tsx     # stub — owned by frontend-today
│   │   ├── BacklogPage.tsx   # stub — owned by frontend-backlog
│   │   ├── RoutinesPage.tsx  # stub — owned by frontend-manage
│   │   ├── ContextPage.tsx   # stub — owned by frontend-manage
│   │   └── HistoryPage.tsx   # stub — owned by frontend-manage
│   └── graphql/
│       └── (codegen output lives here)
├── index.html
├── package.json
├── tsconfig.json
├── vite.config.ts
├── tailwind.config.ts
├── postcss.config.js
└── codegen.ts                # GraphQL codegen config
```

### Dependencies

```json
{
  "dependencies": {
    "react": "^19",
    "react-dom": "^19",
    "react-router-dom": "^7",
    "@apollo/client": "^3",
    "graphql": "^16"
  },
  "devDependencies": {
    "typescript": "^5",
    "vite": "^6",
    "@vitejs/plugin-react": "^4",
    "tailwindcss": "^4",
    "@tailwindcss/vite": "^4",
    "@graphql-codegen/cli": "^5",
    "@graphql-codegen/client-preset": "^4",
    "@graphql-codegen/typescript": "^4",
    "@graphql-codegen/typescript-operations": "^4"
  }
}
```

### Routes

| Path | Page Component | Sidebar Label |
|------|---------------|---------------|
| `/` | `TodayPage` | Today |
| `/backlog` | `BacklogPage` | Backlog |
| `/routines` | `RoutinesPage` | Routines |
| `/context` | `ContextPage` | Context |
| `/history` | `HistoryPage` | History |

### GraphQL Codegen Config

File: `frontend/codegen.ts`

```typescript
import type { CodegenConfig } from '@graphql-codegen/cli';

const config: CodegenConfig = {
  schema: 'http://localhost:8080/graphql',
  documents: 'src/**/*.{ts,tsx}',
  generates: {
    './src/graphql/generated/': {
      preset: 'client',
      presetConfig: {
        gqlTagName: 'gql',
      },
    },
  },
};

export default config;
```

## Visual Design

### Dark Theme

All colors defined in Tailwind config:

| Token | Value | Usage |
|-------|-------|-------|
| `bg-primary` | `#0f0f11` | Page background |
| `bg-surface` | `#1a1a1f` | Cards, sidebar, input backgrounds |
| `bg-surface-hover` | `#25252b` | Hover states on surfaces |
| `text-primary` | `#e8e6e1` | Main text |
| `text-secondary` | `#9a9a9a` | Secondary/muted text |
| `accent` | `#c5a55a` | Gold accent — active nav, buttons, highlights |
| `accent-hover` | `#d4b86a` | Accent hover state |
| `border-default` | `#2a2a30` | Borders, dividers |

### Category Colors

Applied as left border or badge background on blocks/tasks:

| Category | Color | Tailwind class |
|----------|-------|---------------|
| `job` | `#6366f1` | `bg-indigo-500` |
| `interview` | `#0ea5e9` | `bg-sky-500` |
| `project` | `#8b5cf6` | `bg-violet-500` |
| `meal` | `#10b981` | `bg-emerald-500` |
| `baby` | `#f59e0b` | `bg-amber-500` |
| `exercise` | `#ef4444` | `bg-red-500` |
| `admin` | `#6b7280` | `bg-gray-500` |

Export a `CATEGORY_COLORS` map from a shared `src/constants.ts` file so page components can use them.

### Layout

```
┌──────────┬──────────────────────────────────┐
│          │                                  │
│ Sidebar  │       Page Content               │
│          │                                  │
│ ● Today  │                                  │
│   Backlog│                                  │
│   Routine│                                  │
│   Context│                                  │
│   History│                                  │
│          │                                  │
│          │                                  │
│          │                                  │
│          │                                  │
│          │                                  │
└──────────┴──────────────────────────────────┘
```

- Sidebar: fixed width `w-48`, full height, `bg-surface`
- Sidebar header: "DayOS" in accent color, `text-lg font-semibold`
- Nav items: vertical stack, `py-2 px-4`, accent left border + accent text on active route, `text-secondary` on inactive
- Page container: fills remaining width, `bg-primary`, `p-6` padding
- Responsive: on screens < `md` (768px), sidebar collapses to a top horizontal nav bar

### Fonts

Use the system font stack (no custom fonts to load):

```css
font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
```

## Behaviors (EARS syntax)

### Auth Gate

- When the app loads and no auth token exists in `localStorage` (key: `dayos_token`), the system shall display a full-screen password prompt instead of the app.
- When the user submits a password, the system shall store it in `localStorage` under `dayos_token`.
- When an Apollo Client request returns a 401 status, the system shall clear the stored token and redirect to the password prompt.
- The system shall include the stored token as an `Authorization: Bearer {token}` header on every GraphQL request via Apollo link.

### Apollo Client

- The system shall configure Apollo Client with an `HttpLink` pointing to `/graphql` (relative URL, no hardcoded host).
- The system shall configure an `InMemoryCache` with default settings.
- The system shall configure an error link that intercepts network errors with status 401 and triggers the auth gate.
- The system shall wrap the entire app in `ApolloProvider`.

### Routing

- The system shall use `react-router-dom` `BrowserRouter` with the 5 routes defined above.
- When navigating to an undefined route, the system shall redirect to `/`.
- The system shall render the `Layout` component (sidebar + content area) around all routes.

### Sidebar

- The system shall highlight the current route's nav item with the accent color and a left border.
- When a nav item is clicked, the system shall navigate to the corresponding route using React Router's `Link` component (no full page reload).

### Notification Permission

- When the app first loads (after auth), the system shall check `Notification.permission`.
- If permission is `'default'` (not yet asked), the system shall call `Notification.requestPermission()`.
- The system shall not re-prompt if permission was previously denied.

### Vite Dev Server

- The system shall configure a Vite proxy so that `/graphql` requests are forwarded to `http://localhost:8080` during development.

```typescript
// vite.config.ts
server: {
  proxy: {
    '/graphql': 'http://localhost:8080',
  },
}
```

### Page Stubs

- Each page component stub shall render a centered heading with the page name (e.g., `<h1>Today</h1>`) so routing is visually verifiable.
- Stubs are replaced by their owning specs (`frontend-today`, `frontend-backlog`, `frontend-manage`).

## Test Anchors

1. Given the app is loaded with no `dayos_token` in localStorage, when the page renders, then the password prompt is displayed and no sidebar/routes are visible.

2. Given the user enters a password and submits, when the auth gate processes it, then the token is stored in `localStorage` under `dayos_token` and the app shell (sidebar + page content) is rendered.

3. Given the user is authenticated, when navigating to `/backlog`, then the Backlog page stub is rendered and the "Backlog" sidebar item is highlighted with the accent color.

4. Given the user is authenticated, when navigating to an unknown route like `/foo`, then the app redirects to `/` (Today page).

5. Given a valid auth token exists, when an Apollo query is made, then the `Authorization: Bearer {token}` header is included in the request.

6. Given an Apollo request returns a 401 response, when the error link processes it, then `dayos_token` is removed from localStorage and the auth gate is shown.

7. Given the app is running in dev mode, when a GraphQL request is made to `/graphql`, then Vite proxies it to `http://localhost:8080/graphql`.

8. Given all 5 routes exist, when clicking each sidebar nav item in sequence, then each page stub renders with the correct heading and the active indicator moves to the clicked item.
