# Spec: Deployment

## Bounded Context

Owns: Railway configuration (`railway.toml`), embedded frontend serving via `//go:embed`, auth middleware (validates `APP_SECRET` on every GraphQL request), environment variable handling (`ANTHROPIC_API_KEY`, `ANTHROPIC_MODEL`, `APP_SECRET`), production build pipeline (`make build`), CORS configuration

Does not own: Database migrations (owned by `foundation`), frontend build tooling (owned by `frontend-shell`), GraphQL resolvers (owned by backend specs), `main.go` server setup (owned by `foundation` — this spec **extends** it)

Depends on: `foundation` (main.go entry point, migration runner, GraphQL handler), `frontend-shell` (Vite build output at `frontend/dist/`)

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

Middleware wraps the `/graphql` handler. Does NOT apply to static file serving (the frontend itself is public; data access is protected).

```
Request flow:
  1. Extract `Authorization` header
  2. Expect format: `Bearer {token}`
  3. Compare token to APP_SECRET env var (constant-time comparison)
  4. If match → proceed to GraphQL handler
  5. If no match or missing → return 401 Unauthorized
```

Use `crypto/subtle.ConstantTimeCompare` for the token comparison.

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string (Railway provides) |
| `PORT` | No | `8080` | HTTP server port (Railway sets this) |
| `ANTHROPIC_API_KEY` | Yes | — | Claude API key |
| `ANTHROPIC_MODEL` | No | `claude-sonnet-4-6` | Model ID for planner |
| `APP_SECRET` | Yes | — | Shared secret for auth |

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

- When a GraphQL request arrives without an `Authorization` header, the system shall respond with HTTP 401 and body `{"error": "unauthorized"}`.
- When a GraphQL request arrives with an `Authorization` header whose Bearer token does not match `APP_SECRET`, the system shall respond with HTTP 401 and body `{"error": "unauthorized"}`.
- When a GraphQL request arrives with `Authorization: Bearer {token}` where `{token}` matches `APP_SECRET`, the system shall pass the request to the GraphQL handler.
- The system shall use constant-time comparison for token validation to prevent timing attacks.
- When `APP_SECRET` is not set, the system shall exit on startup with a message: `"APP_SECRET environment variable is required"`.

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

1. Given `APP_SECRET` is set to `"test-secret"`, when a POST to `/graphql` is made with `Authorization: Bearer test-secret`, then the request is forwarded to the GraphQL handler and returns a 200 response.

2. Given `APP_SECRET` is set to `"test-secret"`, when a POST to `/graphql` is made with `Authorization: Bearer wrong-secret`, then the response is HTTP 401 with body `{"error": "unauthorized"}`.

3. Given `APP_SECRET` is set to `"test-secret"`, when a POST to `/graphql` is made with no `Authorization` header, then the response is HTTP 401.

4. Given the embedded frontend FS contains `index.html` and `assets/main.js`, when a GET to `/assets/main.js` is made, then the file is served with content type `application/javascript`.

5. Given the embedded frontend FS contains `index.html`, when a GET to `/backlog` is made (no matching file), then `index.html` is served (SPA fallback).

6. Given `ANTHROPIC_API_KEY` is not set in the environment, when the application starts, then it exits with status 1 and logs a message containing `"ANTHROPIC_API_KEY"`.

7. Given `APP_SECRET` is not set in the environment, when the application starts, then it exits with status 1 and logs a message containing `"APP_SECRET"`.

8. Given the app is deployed to Railway with all env vars set, when a browser navigates to the root URL, then `index.html` is served and the frontend app loads.
