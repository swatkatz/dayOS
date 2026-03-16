package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"dayos/db"
	"dayos/identity"

	clerk "github.com/clerk/clerk-sdk-go/v2"
	clerkUserAPI "github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/jackc/pgx/v5/pgtype"
)

// RequireClerk returns middleware that validates Clerk JWTs and injects
// the authenticated user into the request context.
// On first login, fetches user profile from Clerk API and seeds default context entries.
func RequireClerk(queries *db.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				unauthorized(w)
				return
			}

			token := header[len("Bearer "):]
			if len(token) == 0 {
				unauthorized(w)
				return
			}

			claims, err := jwt.Verify(r.Context(), &jwt.VerifyParams{Token: token})
			if err != nil {
				unauthorized(w)
				return
			}

			clerkUserID := claims.Subject

			// Upsert with empty email/name initially — updated below if this is a new user
			user, err := queries.UpsertUserByClerkID(r.Context(), db.UpsertUserByClerkIDParams{
				ClerkID: clerkUserID,
				Email:   "",
			})
			if err != nil {
				log.Printf("auth: upserting user %s: %v", clerkUserID, err)
				http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
				return
			}

			// Detect new user: created_at ≈ updated_at (within 1 second)
			isNew := user.UpdatedAt.Time.Sub(user.CreatedAt.Time) < time.Second
			if isNew {
				// Fetch profile from Clerk API to populate email/name
				if clerkUsr, fetchErr := clerkUserAPI.Get(r.Context(), clerkUserID); fetchErr == nil {
					email := primaryEmail(clerkUsr)
					name := displayName(clerkUsr)
					updated, upsertErr := queries.UpsertUserByClerkID(r.Context(), db.UpsertUserByClerkIDParams{
						ClerkID:     clerkUserID,
						Email:       email,
						DisplayName: &name,
					})
					if upsertErr == nil {
						user = updated
					}
				} else {
					log.Printf("auth: fetching clerk profile for %s: %v", clerkUserID, fetchErr)
				}

				if err := SeedDefaultContextEntries(r.Context(), queries, user.ID); err != nil {
					log.Printf("auth: seeding context entries for %s: %v", clerkUserID, err)
				}
			}

			var apiKey *string
			if user.AnthropicApiKey != nil {
				apiKey = user.AnthropicApiKey
			}

			ctx := identity.WithUser(r.Context(), identity.User{
				ID:              user.ID,
				ClerkID:         user.ClerkID,
				Email:           user.Email,
				DisplayName:     derefStr(user.DisplayName),
				AnthropicAPIKey: apiKey,
				DailyAICap:      user.DailyAiCap,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func primaryEmail(u *clerk.User) string {
	if u.PrimaryEmailAddressID == nil || u.EmailAddresses == nil {
		return ""
	}
	for _, ea := range u.EmailAddresses {
		if ea.ID == *u.PrimaryEmailAddressID {
			return ea.EmailAddress
		}
	}
	if len(u.EmailAddresses) > 0 {
		return u.EmailAddresses[0].EmailAddress
	}
	return ""
}

func displayName(u *clerk.User) string {
	parts := []string{}
	if u.FirstName != nil && *u.FirstName != "" {
		parts = append(parts, *u.FirstName)
	}
	if u.LastName != nil && *u.LastName != "" {
		parts = append(parts, *u.LastName)
	}
	return strings.Join(parts, " ")
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error": "unauthorized"}`))
}

// SeedDefaultContextEntries inserts example context entries for a new user.
func SeedDefaultContextEntries(ctx context.Context, q *db.Queries, userID pgtype.UUID) error {
	seeds := []struct {
		Category string
		Key      string
		Value    string
	}{
		{"constraints", "energy", "Set your energy and scheduling constraints here. Example: Cap deep cognitive work at 5h/day."},
		{"constraints", "work_window", "Set your available hours here. Example: Deep focus work 9am-5pm weekdays."},
		{"preferences", "planning_style", "Describe your preferred planning style. Example: Time-blocked days with buffers between sessions."},
	}

	for _, s := range seeds {
		_, err := q.UpsertContextEntry(ctx, db.UpsertContextEntryParams{
			UserID:   userID,
			Category: s.Category,
			Key:      s.Key,
			Value:    s.Value,
		})
		if err != nil {
			return fmt.Errorf("seeding %s/%s: %w", s.Category, s.Key, err)
		}
	}
	return nil
}
