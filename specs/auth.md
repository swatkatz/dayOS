# Spec: Auth

## Bounded Context

Owns: Auth middleware (`backend/auth/middleware.go`), environment variable validation for `APP_SECRET`, protection of GraphQL endpoint, GraphQL Playground, and introspection queries

Does not own: Route registration in `main.go` (owned by `foundation` — this spec **extends** it), frontend token storage (owned by `frontend-shell`), Railway environment config (owned by `deployment`)

Depends on: foundation (`main.go` entry point, HTTP handler setup)

Produces: HTTP middleware that wraps all sensitive endpoints, ensuring unauthenticated requests cannot access data or development tools

## Contracts

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `APP_SECRET` | Yes | Shared secret for Bearer token auth. Must be set in all environments. |
| `RAILWAY_ENVIRONMENT` | No | Set by Railway in production. Used to determine environment. |

### Middleware Interface

```go
// backend/auth/middleware.go

// RequireAuth returns middleware that validates Bearer tokens against APP_SECRET.
// Uses constant-time comparison to prevent timing attacks.
func RequireAuth(secret string) func(http.Handler) http.Handler
```

### Route Protection Matrix

| Path | Method | Auth required | Condition |
|------|--------|---------------|-----------|
| `/graphql` | POST | Yes | Always |
| `/graphql` | GET | Yes | Always (blocks Playground GET requests in prod) |
| `/` (Playground) | GET | Yes | Only when Playground is enabled |
| `/*` (static files) | GET | No | Frontend assets are public |

### Introspection Protection

GraphQL introspection queries (`__schema`, `__type`) are a development tool that exposes the full API surface — every query, mutation, field, and type. In production, unauthenticated introspection lets an attacker map the entire API without credentials.

**Approach:** Since auth middleware wraps the entire `/graphql` endpoint, introspection is protected by default — no separate logic needed. An unauthenticated request (whether introspection or not) gets a 401 before reaching gqlgen.

### GraphQL Playground Protection

The GraphQL Playground is an interactive IDE served at `/` (or `/graphql` via GET) that lets anyone craft and execute queries. In production:

- Playground should be **disabled entirely** (not served)
- In development (`RAILWAY_ENVIRONMENT` not set), Playground is enabled but still behind auth

**Implementation in `main.go`:**

```go
if os.Getenv("RAILWAY_ENVIRONMENT") == "" {
    // Dev mode: serve playground, but behind auth
    http.Handle("/", auth.RequireAuth(secret)(playground.Handler("DayOS", "/graphql")))
} else {
    // Production: no playground, static files served instead
    http.Handle("/", staticFileHandler(frontendFS))
}

// GraphQL endpoint always requires auth
http.Handle("/graphql", auth.RequireAuth(secret)(graphqlHandler))
```

## Auth Flow

### Request validation

```
1. Extract Authorization header
2. If missing → 401 {"error": "unauthorized"}
3. If not "Bearer {token}" format → 401 {"error": "unauthorized"}
4. Compare token to APP_SECRET using crypto/subtle.ConstantTimeCompare
5. If mismatch → 401 {"error": "unauthorized"}
6. If match → call next handler
```

### Frontend token flow (owned by frontend-shell, documented here for context)

```
1. App loads → check localStorage for "dayos_token"
2. If missing → show password input
3. User enters secret → store in localStorage
4. Every Apollo request includes Authorization: Bearer {token}
5. On 401 response → clear localStorage, show password input again
```

## Behaviors (EARS syntax)

### Startup

- When the application starts and `APP_SECRET` is not set, the system shall log `"APP_SECRET environment variable is required"` and exit with status 1.
- When the application starts and `APP_SECRET` is set, the system shall initialize the auth middleware with the secret.

### Request authentication

- When a POST to `/graphql` arrives without an `Authorization` header, the system shall respond with HTTP 401 and body `{"error": "unauthorized"}`.
- When a POST to `/graphql` arrives with `Authorization: Bearer {token}` where `{token}` does not match `APP_SECRET`, the system shall respond with HTTP 401 and body `{"error": "unauthorized"}`.
- When a POST to `/graphql` arrives with `Authorization: Bearer {token}` where `{token}` matches `APP_SECRET`, the system shall pass the request to the GraphQL handler.
- The system shall use `crypto/subtle.ConstantTimeCompare` for token comparison to prevent timing attacks.
- When a malformed `Authorization` header is received (e.g., no "Bearer " prefix, empty token), the system shall respond with HTTP 401.

### Playground protection

- When `RAILWAY_ENVIRONMENT` is not set (dev mode), the system shall serve the GraphQL Playground at `/`, protected by the auth middleware.
- When `RAILWAY_ENVIRONMENT` is set (production), the system shall NOT serve the GraphQL Playground at any path.
- When an unauthenticated GET request arrives at `/graphql` in any environment, the system shall respond with HTTP 401 (this blocks browser-based Playground access without a token).

### Introspection protection

- When an unauthenticated request containing an introspection query (`__schema` or `__type`) arrives at `/graphql`, the system shall respond with HTTP 401 — the auth middleware rejects it before gqlgen processes the query.
- When an authenticated request containing an introspection query arrives in dev mode, the system shall allow it (useful for codegen and development tools).
- When an authenticated request containing an introspection query arrives in production, the system shall allow it (the user has already proven identity via APP_SECRET).

### Error responses

- The system shall return `Content-Type: application/json` for all 401 responses.
- The system shall NOT include details about why authentication failed (e.g., "wrong password" vs "missing header") — all failures return the same `{"error": "unauthorized"}` body.

## Decision Table

| `Authorization` header | Token matches | Environment | Path | Result |
|------------------------|---------------|-------------|------|--------|
| Missing | N/A | Any | `/graphql` | 401 |
| `Bearer wrong` | No | Any | `/graphql` | 401 |
| `Bearer correct` | Yes | Any | `/graphql` | Forward to GraphQL handler |
| Missing | N/A | Dev | `/` | 401 (Playground behind auth) |
| `Bearer correct` | Yes | Dev | `/` | Serve Playground |
| Any | N/A | Production | `/` | Serve `index.html` (no auth needed) |
| Any | N/A | Any | `/style.css` | Serve static file (no auth needed) |

## Files

| File | Generated? | Description |
|------|-----------|-------------|
| `backend/auth/middleware.go` | No | `RequireAuth()` middleware function |
| `backend/auth/middleware_test.go` | No | Tests for auth middleware |
| `backend/main.go` | Modified | Wire up auth middleware on routes |

## Test Anchors

1. Given `APP_SECRET` is `"test-secret-123"`, when a POST to `/graphql` is made with `Authorization: Bearer test-secret-123`, then the request reaches the handler and returns 200.

2. Given `APP_SECRET` is `"test-secret-123"`, when a POST to `/graphql` is made with `Authorization: Bearer wrong-secret`, then the response is HTTP 401 with body `{"error": "unauthorized"}`.

3. Given `APP_SECRET` is `"test-secret-123"`, when a POST to `/graphql` is made with no `Authorization` header, then the response is HTTP 401.

4. Given `APP_SECRET` is `"test-secret-123"`, when a POST to `/graphql` is made with `Authorization: Basic dXNlcjpwYXNz` (wrong scheme), then the response is HTTP 401.

5. Given `APP_SECRET` is `"test-secret-123"`, when a POST to `/graphql` is made with `Authorization: Bearer ` (empty token), then the response is HTTP 401.

6. Given `RAILWAY_ENVIRONMENT` is not set (dev mode) and `APP_SECRET` is set, when an unauthenticated GET to `/` is made, then the response is HTTP 401.

7. Given `RAILWAY_ENVIRONMENT` is not set (dev mode) and `APP_SECRET` is set, when an authenticated GET to `/` is made, then the Playground HTML is served.

8. Given `RAILWAY_ENVIRONMENT` is set (production), when a GET to `/` is made (any auth), then `index.html` is served from the embedded frontend (no Playground).

9. Given `APP_SECRET` is not set in the environment, when the application starts, then it exits with status 1 and logs a message containing `"APP_SECRET"`.

10. Given auth middleware is active, when a timing attack is attempted by measuring response times for different tokens, then response times are constant regardless of how many characters match (verified by `crypto/subtle.ConstantTimeCompare` usage, not by timing test).
