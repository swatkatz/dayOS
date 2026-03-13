# Spec: Deployment

## Bounded Context

Owns: Dockerfiles (two services: backend + frontend), CORS middleware for the backend, environment variable handling (`ANTHROPIC_API_KEY`, `ANTHROPIC_MODEL`, `FRONTEND_URL`), `make dev` target that starts both servers, production Apollo Client URL wiring, production build targets

Does not own: Database migrations (owned by `foundation`), frontend build tooling (owned by `frontend-shell`), GraphQL resolvers (owned by backend specs), `main.go` server setup (owned by `foundation` — this spec **extends** it), auth middleware and `APP_SECRET` handling (owned by `auth`)

Depends on: `foundation` (main.go entry point, migration runner, GraphQL handler), `auth` (middleware for protecting GraphQL endpoint), `frontend-shell` (Vite dev server + build, Apollo Client setup)

Produces: Two Docker containers (Go backend + static frontend), CORS-enabled backend, dev workflow that runs both servers locally

## Contracts

### Two-Service Architecture

Railway runs two services from this monorepo:

1. **Backend** — Go server serving `/graphql` on `PORT` (Railway-assigned)
2. **Frontend** — Static Vite build served via `npx serve` with SPA fallback

The frontend talks to the backend via its public Railway URL. CORS on the backend allows requests from the frontend's origin.

### CORS Middleware

File: `backend/cors/cors.go`

```go
// AllowFrontend returns middleware that sets CORS headers for the given origin.
// In dev mode (FRONTEND_URL not set), allows http://localhost:5173.
func AllowFrontend(frontendURL string) func(http.Handler) http.Handler
```

CORS headers to set on every response when `Origin` matches `frontendURL`:
- `Access-Control-Allow-Origin: {frontendURL}`
- `Access-Control-Allow-Methods: POST, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type, Authorization`
- `Access-Control-Max-Age: 86400`

Preflight (`OPTIONS`) requests get a 204 with the headers and no body.

### Environment Variables

**Backend:**

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string (Railway provides) |
| `PORT` | No | `8080` | HTTP server port (Railway sets this) |
| `ANTHROPIC_API_KEY` | Yes | — | Claude API key |
| `ANTHROPIC_MODEL` | No | `claude-sonnet-4-6` | Model ID for planner |
| `APP_SECRET` | Yes | — | Shared secret for auth (see `specs/auth.md`) |
| `FRONTEND_URL` | No | `http://localhost:5173` | Frontend origin for CORS (set in Railway to the frontend service URL) |

On startup, if any required env var is missing, the system shall log which variable is missing and exit with status 1.

**Frontend (build-time only):**

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `VITE_GRAPHQL_URL` | No | — | Backend GraphQL endpoint URL. When not set, Apollo uses `/graphql` (works with Vite dev proxy). Set in Railway to e.g. `https://dayos-backend.up.railway.app/graphql`. |

### Apollo Client URL Change

**File: `frontend/src/apollo.ts`**

Current:
```typescript
const httpLink = new HttpLink({ uri: '/graphql' })
```

Change to:
```typescript
const httpLink = new HttpLink({
  uri: import.meta.env.VITE_GRAPHQL_URL || '/graphql',
})
```

In dev mode, `VITE_GRAPHQL_URL` is not set → uses `/graphql` → Vite proxy forwards to `localhost:8080`. Same behavior as today.

In production, `VITE_GRAPHQL_URL` is set at build time in Railway → Apollo calls the backend service directly. CORS middleware allows it.

### main.go Changes

Extend `main.go` to:
1. Validate `ANTHROPIC_API_KEY` on startup (currently only `APP_SECRET` and `DATABASE_URL` are checked)
2. Read `FRONTEND_URL` (default `http://localhost:5173`)
3. Wrap the `/graphql` handler with CORS middleware (CORS must be outermost, then auth)

```go
// Middleware order on /graphql: CORS → Auth → GraphQL handler
cors := cors.AllowFrontend(frontendURL)
http.Handle("/graphql", cors(authMiddleware(srv)))
```

CORS must wrap auth because the browser sends a preflight `OPTIONS` request *without* the `Authorization` header. If auth runs first, it would 401 the preflight and the browser would never send the real request.

### Dockerfiles

Two Dockerfiles, one per service. Each Railway service points to its Dockerfile.

**File: `backend/Dockerfile`**

```dockerfile
FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o dayos .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=build /app/dayos /usr/local/bin/dayos
CMD ["dayos"]
```

**File: `frontend/Dockerfile`**

```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG VITE_GRAPHQL_URL
ENV VITE_GRAPHQL_URL=$VITE_GRAPHQL_URL
RUN npm run build

FROM node:20-alpine
WORKDIR /app
RUN npm install -g serve@14
COPY --from=builder /app/dist ./dist
ENV PORT=8080
EXPOSE $PORT
CMD ["sh", "-c", "serve -s dist -l $PORT"]
```

The `-s` flag enables SPA fallback — any route that doesn't match a static file serves `index.html`, so React Router handles client-side routes like `/backlog`. `VITE_GRAPHQL_URL` is passed as a build arg in Railway and baked into the frontend bundle at build time.

### Makefile Changes

Replace the existing `dev` and `build` targets:

```makefile
dev:
	$(MAKE) dev-backend & $(MAKE) dev-frontend & wait

dev-backend:
	cd backend && go run .

dev-frontend:
	cd frontend && npm run dev

build:
	cd frontend && npm run build
	cd backend && go build -o dayos .
```

`make dev` starts both the Go server (port 8080) and the Vite dev server (port 5173) in parallel. The Vite proxy config (already in `vite.config.ts`) forwards `/graphql` to the Go server.

## Behaviors (EARS syntax)

### Auth

Auth behaviors are owned by `specs/auth.md`. See that spec for middleware behavior, Playground/introspection protection, and startup validation.

### CORS

- When a request arrives at `/graphql` with an `Origin` header matching `FRONTEND_URL`, the system shall include CORS headers in the response.
- When a preflight `OPTIONS` request arrives at `/graphql` with an `Origin` header matching `FRONTEND_URL`, the system shall respond with 204 and the CORS headers without forwarding to the auth or GraphQL handler.
- When a request arrives with an `Origin` header that does NOT match `FRONTEND_URL`, the system shall NOT include CORS headers (the browser will block the response).
- When `FRONTEND_URL` is not set, the system shall default to `http://localhost:5173` for CORS (dev mode).

### Startup

- When the application starts, the system shall validate that `DATABASE_URL`, `ANTHROPIC_API_KEY`, and `APP_SECRET` are set before proceeding.
- When any required environment variable is missing, the system shall log the variable name and exit with status 1.
- When `ANTHROPIC_MODEL` is not set, the system shall default to `claude-sonnet-4-6`.
- When `FRONTEND_URL` is not set, the system shall default to `http://localhost:5173`.

### Dev Workflow

- When `make dev` is run, the system shall start both the Go backend server and the Vite frontend dev server concurrently.
- The Vite dev server shall proxy `/graphql` requests to `http://localhost:8080` (already configured in `vite.config.ts`).

## Decision Table

| `Origin` header | Matches `FRONTEND_URL` | Request method | Path | Result |
|-----------------|------------------------|----------------|------|--------|
| Missing | N/A | POST | `/graphql` | No CORS headers, forward to auth → handler |
| Present, matches | Yes | OPTIONS | `/graphql` | 204 with CORS headers, no auth check |
| Present, matches | Yes | POST | `/graphql` | Forward to auth → handler, CORS headers on response |
| Present, doesn't match | No | OPTIONS | `/graphql` | 204 but no CORS headers (browser blocks) |
| Present, doesn't match | No | POST | `/graphql` | Forward to auth → handler, no CORS headers |

## Files

| File | Action | Description |
|------|--------|-------------|
| `backend/cors/cors.go` | Create | CORS middleware |
| `backend/cors/cors_test.go` | Create | CORS middleware tests |
| `backend/main.go` | Modify | Add ANTHROPIC_API_KEY validation, FRONTEND_URL, wire CORS |
| `frontend/src/apollo.ts` | Modify | Use `VITE_GRAPHQL_URL` env var for production |
| `backend/Dockerfile` | Create | Backend Docker container |
| `frontend/Dockerfile` | Create | Frontend Docker container |
| `Makefile` | Modify | Add dev-backend, dev-frontend targets, update dev target |

## Test Anchors

1. Given `FRONTEND_URL` is `http://localhost:5173`, when an OPTIONS request to `/graphql` arrives with `Origin: http://localhost:5173`, then the response is 204 with `Access-Control-Allow-Origin: http://localhost:5173` and the request is NOT forwarded to the next handler.

2. Given `FRONTEND_URL` is `http://localhost:5173`, when a POST to `/graphql` arrives with `Origin: http://localhost:5173`, then the response includes `Access-Control-Allow-Origin: http://localhost:5173` and the request IS forwarded to the next handler.

3. Given `FRONTEND_URL` is `http://localhost:5173`, when a POST to `/graphql` arrives with `Origin: http://evil.com`, then the response does NOT include `Access-Control-Allow-Origin`.

4. Given `ANTHROPIC_API_KEY` is not set in the environment, when the application starts, then it exits with status 1 and logs a message containing `"ANTHROPIC_API_KEY"`.

5. Given `FRONTEND_URL` is not set, when the CORS middleware initializes, then it defaults to allowing `http://localhost:5173`.
