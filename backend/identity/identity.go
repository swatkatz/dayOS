package identity

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// User represents the authenticated user extracted from a Clerk JWT.
type User struct {
	ID              pgtype.UUID
	ClerkID         string
	Email           string
	DisplayName     string
	AnthropicAPIKey *string
	DailyAICap      int32
}

type ctxKey struct{}

// WithUser stores the authenticated user in the context.
func WithUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, ctxKey{}, u)
}

// FromContext returns the authenticated user from the context.
// Returns false if no user is present.
func FromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(ctxKey{}).(User)
	return u, ok
}

// MustFromContext returns the authenticated user or panics.
// Use only after auth middleware has run.
func MustFromContext(ctx context.Context) User {
	u, ok := FromContext(ctx)
	if !ok {
		panic("identity: no user in context")
	}
	return u
}
