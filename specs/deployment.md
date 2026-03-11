# Spec: Deployment

## Bounded Context

Owns: Railway configuration (`railway.toml`), embedded frontend serving via `//go:embed`, environment variable handling (`ANTHROPIC_API_KEY`, `ANTHROPIC_MODEL`), production build pipeline (`make build`), CORS configuration

Does not own: Database migrations (owned by `foundation`), frontend build tooling (owned by `frontend-shell`), GraphQL resolvers (owned by backend specs), `main.go` server setup (owned by `foundation` — this spec **extends** it), auth middleware and `APP_SECRET` handling (owned by `auth`)

Depends on: `foundation` (main.go entry point, migration runner, GraphQL handler), `auth` (middleware for protecting GraphQL endpoint), `frontend-shell` (Vite build output at `frontend/dist/`)

Produces: Single deployable Go binary with embedded frontend, Railway service configuration, auth middleware that protects all GraphQL requests

## Contracts

### Embedded Frontend

File: `backend/embed.go`

```go
package main

import "embed"

//go:embed frontend/dist/*
var frontendFS embed.FS
```

The `make build` target builds the frontend first, copies `frontend/dist` into `backend/frontend/dist`, then compiles the Go binary. The Go binary serves the embedded frontend for any request that isn't `/graphql`.

### Route Handling (production)

| Path | Handler |
|------|---------|
| `/graphql` | gqlgen GraphQL handler (POST only) |
| `/*` | Embedded frontend static files (SPA fallback to `index.html`) |

SPA fallback: if the requested file doesn't exist in the embedded FS, serve `index.html` so client-side routing works.

### Auth Middleware

Auth middleware is owned by `specs/auth.md`. The deployment spec wires it into the route setup. See the auth spec for request flow, Playground protection, and introspection protection.

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string (Railway provides) |
| `PORT` | No | `8080` | HTTP server port (Railway sets this) |
| `ANTHROPIC_API_KEY` | Yes | — | Claude API key |
| `ANTHROPIC_MODEL` | No | `claude-sonnet-4-6` | Model ID for planner |
| `APP_SECRET` | Yes | — | Shared secret for auth (see `specs/auth.md`) |

On startup, if any required env var is missing, the system shall log which variable is missing and exit with status 1.

### Railway Config

File: `railway.toml` (project root)

```toml
[build]
builder = "nixpacks"

[build.nixpacks]
providers = ["go", "node"]

[build.nixpacks.phases.setup]
nixPkgs = ["go", "nodejs_20"]

[build.nixpacks.phases.install]
cmds = ["cd frontend && npm ci"]

[build.nixpacks.phases.build]
cmds = [
  "cd frontend && npm run build",
  "mkdir -p backend/frontend/dist",
  "cp -r frontend/dist/* backend/frontend/dist/",
  "cd backend && go build -o dayos ./main.go",
]

[deploy]
startCommand = "cd backend && ./dayos"

[deploy.healthcheck]
path = "/graphql"
timeout = 10
```

### Makefile Changes

Extend the existing Makefile from `foundation`:

```makefile
build:
	cd frontend && npm run build
	mkdir -p backend/frontend/dist
	cp -r frontend/dist/* backend/frontend/dist/
	cd backend && go build -o dayos ./main.go
```

## Behaviors (EARS syntax)

### Auth

Auth behaviors are owned by `specs/auth.md`. See that spec for middleware behavior, Playground/introspection protection, and startup validation.

### Embedded Frontend

- When a request arrives for a path other than `/graphql` and the path matches a file in the embedded frontend FS, the system shall serve that file with appropriate content type.
- When a request arrives for a path other than `/graphql` and no matching file exists in the embedded frontend FS, the system shall serve `index.html` (SPA fallback).
- When running in dev mode (`RAILWAY_ENVIRONMENT` is not set), the system shall NOT serve embedded frontend files and shall instead serve the GraphQL Playground at `/`.

### Startup

- When the application starts, the system shall validate that `DATABASE_URL`, `ANTHROPIC_API_KEY`, and `APP_SECRET` are set before proceeding.
- When any required environment variable is missing, the system shall log the variable name and exit with status 1.
- When `ANTHROPIC_MODEL` is not set, the system shall default to `claude-sonnet-4-6`.

### Build

- When `make build` is run, the system shall build the frontend, copy the output to `backend/frontend/dist/`, and compile the Go binary.
- The system shall include all files from `backend/frontend/dist/` in the Go binary via `//go:embed`.

## Decision Table

| `Authorization` header | Token matches `APP_SECRET` | Request path | Result |
|------------------------|---------------------------|--------------|--------|
| Missing | N/A | `/graphql` | 401 Unauthorized |
| Present, malformed | N/A | `/graphql` | 401 Unauthorized |
| Present, wrong token | No | `/graphql` | 401 Unauthorized |
| Present, correct token | Yes | `/graphql` | Forward to GraphQL handler |
| Any | N/A | `/styles.css` (exists) | Serve static file |
| Any | N/A | `/backlog` (no file) | Serve `index.html` |

## Test Anchors

1. Given the embedded frontend FS contains `index.html` and `assets/main.js`, when a GET to `/assets/main.js` is made, then the file is served with content type `application/javascript`.

2. Given the embedded frontend FS contains `index.html`, when a GET to `/backlog` is made (no matching file), then `index.html` is served (SPA fallback).

3. Given `ANTHROPIC_API_KEY` is not set in the environment, when the application starts, then it exits with status 1 and logs a message containing `"ANTHROPIC_API_KEY"`.

4. Given the app is deployed to Railway with all env vars set, when a browser navigates to the root URL, then `index.html` is served and the frontend app loads.

Auth-related test anchors are in `specs/auth.md`.
