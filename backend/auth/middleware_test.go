package auth

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`ok`))
	})
}

func TestRequireAuth(t *testing.T) {
	secret := "test-secret-123"
	handler := RequireAuth(secret)(okHandler())

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "valid bearer token",
			authHeader: "Bearer test-secret-123",
			wantStatus: http.StatusOK,
			wantBody:   "ok",
		},
		{
			name:       "wrong token",
			authHeader: "Bearer wrong-secret",
			wantStatus: http.StatusUnauthorized,
			wantBody:   `{"error": "unauthorized"}`,
		},
		{
			name:       "no authorization header",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
			wantBody:   `{"error": "unauthorized"}`,
		},
		{
			name:       "wrong scheme (Basic)",
			authHeader: "Basic dXNlcjpwYXNz",
			wantStatus: http.StatusUnauthorized,
			wantBody:   `{"error": "unauthorized"}`,
		},
		{
			name:       "empty token after Bearer",
			authHeader: "Bearer ",
			wantStatus: http.StatusUnauthorized,
			wantBody:   `{"error": "unauthorized"}`,
		},
		{
			name:       "bearer lowercase",
			authHeader: "bearer test-secret-123",
			wantStatus: http.StatusUnauthorized,
			wantBody:   `{"error": "unauthorized"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}

			body, _ := io.ReadAll(rr.Body)
			if string(body) != tt.wantBody {
				t.Errorf("body = %q, want %q", string(body), tt.wantBody)
			}

			if tt.wantStatus == http.StatusUnauthorized {
				ct := rr.Header().Get("Content-Type")
				if ct != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", ct)
				}
			}
		})
	}
}
