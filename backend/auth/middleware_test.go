package auth

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"dayos/identity"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Verify user is in context if auth succeeded
		if u, ok := identity.FromContext(r.Context()); ok {
			w.Write([]byte("user:" + u.ClerkID))
		} else {
			w.Write([]byte("ok"))
		}
	})
}

func TestRequireClerk_NoHeader(t *testing.T) {
	// We can't easily test with real Clerk JWTs, but we can test the
	// header extraction logic by sending requests without valid tokens.
	handler := RequireClerk(nil)(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	body, _ := io.ReadAll(rr.Body)
	if string(body) != `{"error": "unauthorized"}` {
		t.Errorf("body = %q, want unauthorized JSON", string(body))
	}
	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestRequireClerk_EmptyBearer(t *testing.T) {
	handler := RequireClerk(nil)(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	req.Header.Set("Authorization", "Bearer ")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestRequireClerk_WrongScheme(t *testing.T) {
	handler := RequireClerk(nil)(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestRequireClerk_InvalidJWT(t *testing.T) {
	handler := RequireClerk(nil)(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	req.Header.Set("Authorization", "Bearer invalid-jwt-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}
