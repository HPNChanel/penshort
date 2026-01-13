package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurity(t *testing.T) {
	tests := []struct {
		name          string
		isDev         bool
		checkHeader   string
		wantPresent   bool
		wantValue     string
	}{
		{
			name:        "X-Content-Type-Options is set",
			isDev:       false,
			checkHeader: "X-Content-Type-Options",
			wantPresent: true,
			wantValue:   "nosniff",
		},
		{
			name:        "X-Frame-Options is set",
			isDev:       false,
			checkHeader: "X-Frame-Options",
			wantPresent: true,
			wantValue:   "DENY",
		},
		{
			name:        "Referrer-Policy is set",
			isDev:       false,
			checkHeader: "Referrer-Policy",
			wantPresent: true,
			wantValue:   "strict-origin-when-cross-origin",
		},
		{
			name:        "CSP is set",
			isDev:       false,
			checkHeader: "Content-Security-Policy",
			wantPresent: true,
			wantValue:   "default-src 'none'; frame-ancestors 'none'",
		},
		{
			name:        "HSTS is set in production",
			isDev:       false,
			checkHeader: "Strict-Transport-Security",
			wantPresent: true,
			wantValue:   "max-age=31536000; includeSubDomains; preload",
		},
		{
			name:        "HSTS is NOT set in development",
			isDev:       true,
			checkHeader: "Strict-Transport-Security",
			wantPresent: false,
		},
		{
			name:        "Cache-Control is set",
			isDev:       false,
			checkHeader: "Cache-Control",
			wantPresent: true,
			wantValue:   "no-store",
		},
		{
			name:        "Cross-Origin-Opener-Policy is set",
			isDev:       false,
			checkHeader: "Cross-Origin-Opener-Policy",
			wantPresent: true,
			wantValue:   "same-origin",
		},
		{
			name:        "Cross-Origin-Resource-Policy is set",
			isDev:       false,
			checkHeader: "Cross-Origin-Resource-Policy",
			wantPresent: true,
			wantValue:   "same-origin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := SecurityConfig{
				IsDevelopment: tt.isDev,
			}

			handler := Security(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			got := rec.Header().Get(tt.checkHeader)
			if tt.wantPresent {
				if got == "" {
					t.Errorf("header %s not present, want %s", tt.checkHeader, tt.wantValue)
				} else if got != tt.wantValue {
					t.Errorf("header %s = %q, want %q", tt.checkHeader, got, tt.wantValue)
				}
			} else {
				if got != "" {
					t.Errorf("header %s = %q, want empty", tt.checkHeader, got)
				}
			}
		})
	}
}

func TestMaxBodySize(t *testing.T) {
	tests := []struct {
		name           string
		maxBytes       int64
		contentLength  int64
		body           string
		wantStatus     int
	}{
		{
			name:          "small body allowed",
			maxBytes:      1024,
			contentLength: 10,
			body:          "small body",
			wantStatus:    http.StatusOK,
		},
		{
			name:          "content-length exceeds limit",
			maxBytes:      10,
			contentLength: 100,
			body:          "this is a much longer body that exceeds the limit",
			wantStatus:    http.StatusRequestEntityTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := MaxBodySize(tt.maxBytes)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			req.ContentLength = tt.contentLength
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
