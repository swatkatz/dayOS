package tz

import (
	"context"
	"net/http"
	"time"
)

type ctxKey struct{}

// Middleware reads the X-Timezone header and stores the parsed *time.Location in
// the request context. Falls back to UTC if the header is missing or invalid.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loc := time.UTC
		if name := r.Header.Get("X-Timezone"); name != "" {
			if parsed, err := time.LoadLocation(name); err == nil {
				loc = parsed
			}
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, loc)))
	})
}

// FromContext returns the timezone from the context, defaulting to UTC.
func FromContext(ctx context.Context) *time.Location {
	if loc, ok := ctx.Value(ctxKey{}).(*time.Location); ok {
		return loc
	}
	return time.UTC
}
