package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins []string
		requestOrigin  string
		method         string
		wantStatus     int
		wantHeader     string
	}{
		{
			name:           "no origins configured blocks all",
			allowedOrigins: []string{},
			requestOrigin:  "https://example.com",
			method:         http.MethodGet,
			wantStatus:     http.StatusOK,
			wantHeader:     "", // No CORS header
		},
		{
			name:           "allowed origin gets header",
			allowedOrigins: []string{"https://example.com"},
			requestOrigin:  "https://example.com",
			method:         http.MethodGet,
			wantStatus:     http.StatusOK,
			wantHeader:     "https://example.com",
		},
		{
			name:           "disallowed origin blocked on preflight",
			allowedOrigins: []string{"https://example.com"},
			requestOrigin:  "https://evil.com",
			method:         http.MethodOptions,
			wantStatus:     http.StatusForbidden,
			wantHeader:     "",
		},
		{
			name:           "preflight returns no content",
			allowedOrigins: []string{"https://example.com"},
			requestOrigin:  "https://example.com",
			method:         http.MethodOptions,
			wantStatus:     http.StatusNoContent,
			wantHeader:     "https://example.com",
		},
		{
			name:           "case insensitive origin match",
			allowedOrigins: []string{"HTTPS://EXAMPLE.COM"},
			requestOrigin:  "https://example.com",
			method:         http.MethodGet,
			wantStatus:     http.StatusOK,
			wantHeader:     "https://example.com",
		},
		{
			name:           "no origin header skips CORS",
			allowedOrigins: []string{"https://example.com"},
			requestOrigin:  "",
			method:         http.MethodGet,
			wantStatus:     http.StatusOK,
			wantHeader:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultCORSConfig()
			cfg.AllowedOrigins = tt.allowedOrigins

			handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(tt.method, "/", nil)
			if tt.requestOrigin != "" {
				req.Header.Set("Origin", tt.requestOrigin)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			got := rec.Header().Get("Access-Control-Allow-Origin")
			if got != tt.wantHeader {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tt.wantHeader)
			}
		})
	}
}

func TestCORSPreflightHeaders(t *testing.T) {
	cfg := DefaultCORSConfig()
	cfg.AllowedOrigins = []string{"https://example.com"}

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Check preflight headers are set
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("Access-Control-Allow-Methods not set on preflight")
	}

	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Error("Access-Control-Allow-Headers not set on preflight")
	}

	if got := rec.Header().Get("Access-Control-Max-Age"); got == "" {
		t.Error("Access-Control-Max-Age not set on preflight")
	}
}
