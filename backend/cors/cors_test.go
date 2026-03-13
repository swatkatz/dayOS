package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAllowFrontend(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name           string
		frontendURL    string
		method         string
		origin         string
		wantStatus     int
		wantCORS       bool
		wantNextCalled bool
	}{
		{
			name:           "preflight with matching origin returns 204",
			frontendURL:    "http://localhost:5173",
			method:         http.MethodOptions,
			origin:         "http://localhost:5173",
			wantStatus:     http.StatusNoContent,
			wantCORS:       true,
			wantNextCalled: false,
		},
		{
			name:           "POST with matching origin includes CORS headers",
			frontendURL:    "http://localhost:5173",
			method:         http.MethodPost,
			origin:         "http://localhost:5173",
			wantStatus:     http.StatusOK,
			wantCORS:       true,
			wantNextCalled: true,
		},
		{
			name:           "POST with non-matching origin has no CORS headers",
			frontendURL:    "http://localhost:5173",
			method:         http.MethodPost,
			origin:         "http://evil.com",
			wantStatus:     http.StatusOK,
			wantCORS:       false,
			wantNextCalled: true,
		},
		{
			name:           "POST with no origin header has no CORS headers",
			frontendURL:    "http://localhost:5173",
			method:         http.MethodPost,
			origin:         "",
			wantStatus:     http.StatusOK,
			wantCORS:       false,
			wantNextCalled: true,
		},
		{
			name:           "preflight with non-matching origin has no CORS headers",
			frontendURL:    "http://localhost:5173",
			method:         http.MethodOptions,
			origin:         "http://evil.com",
			wantStatus:     http.StatusNoContent,
			wantCORS:       false,
			wantNextCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled = false

			middleware := AllowFrontend(tt.frontendURL)
			handler := middleware(next)

			req := httptest.NewRequest(tt.method, "/graphql", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if tt.wantCORS {
				if allowOrigin != tt.frontendURL {
					t.Errorf("Access-Control-Allow-Origin = %q, want %q", allowOrigin, tt.frontendURL)
				}
				if w.Header().Get("Access-Control-Allow-Methods") == "" {
					t.Error("missing Access-Control-Allow-Methods header")
				}
				if w.Header().Get("Access-Control-Allow-Headers") == "" {
					t.Error("missing Access-Control-Allow-Headers header")
				}
			} else {
				if allowOrigin != "" {
					t.Errorf("Access-Control-Allow-Origin = %q, want empty", allowOrigin)
				}
			}

			if nextCalled != tt.wantNextCalled {
				t.Errorf("next handler called = %v, want %v", nextCalled, tt.wantNextCalled)
			}
		})
	}
}
