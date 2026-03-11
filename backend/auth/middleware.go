package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// RequireAuth returns middleware that validates Bearer tokens against the provided secret.
// Uses constant-time comparison to prevent timing attacks.
func RequireAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				unauthorized(w)
				return
			}

			token := header[len("Bearer "):]
			if len(token) == 0 || subtle.ConstantTimeCompare([]byte(token), []byte(secret)) != 1 {
				unauthorized(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error": "unauthorized"}`))
}
