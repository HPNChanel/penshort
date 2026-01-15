//go:build integration

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test401Unauthorized verifies the auth error response format.
func TestIntegration401Unauthorized(t *testing.T) {
	rec := httptest.NewRecorder()
	writeAuthError(rec)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected JSON content type")
	}

	body := rec.Body.String()
	if body == "" {
		t.Error("Expected error body")
	}

	// Verify it contains expected fields
	expectedCode := `"code":"UNAUTHORIZED"`
	if !contains(body, expectedCode) {
		t.Errorf("Response should contain %s, got: %s", expectedCode, body)
	}
}

// Test403Forbidden verifies the forbidden error format.
func TestIntegration403Forbidden(t *testing.T) {
	rec := httptest.NewRecorder()
	writeScopeError(rec, http.StatusForbidden, "FORBIDDEN", "Insufficient permissions")

	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected JSON content type")
	}

	body := rec.Body.String()
	if !contains(body, `"code":"FORBIDDEN"`) {
		t.Errorf("Response should contain FORBIDDEN code, got: %s", body)
	}
}

// TestExtractAPIKey tests API key extraction from headers.
func TestIntegrationExtractAPIKey(t *testing.T) {
	testCases := []struct {
		name       string
		authHeader string
		apiKeyHeader string
		want       string
	}{
		{
			name:       "Bearer token",
			authHeader: "Bearer pk_live_abc123_secret",
			want:       "pk_live_abc123_secret",
		},
		{
			name:         "X-API-Key header",
			apiKeyHeader: "pk_live_abc123_secret",
			want:         "pk_live_abc123_secret",
		},
		{
			name:       "Bearer takes precedence",
			authHeader: "Bearer bearer_key",
			apiKeyHeader: "apikey_header",
			want:       "bearer_key",
		},
		{
			name: "No key",
			want: "",
		},
		{
			name:       "Invalid Bearer format",
			authHeader: "Basic abc123",
			want:       "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			if tc.apiKeyHeader != "" {
				req.Header.Set("X-API-Key", tc.apiKeyHeader)
			}

			got := extractAPIKey(req)
			if got != tc.want {
				t.Errorf("extractAPIKey() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestGetClientIP verifies IP extraction from various headers.
func TestIntegrationGetClientIP(t *testing.T) {
	testCases := []struct {
		name        string
		xff         string
		xri         string
		remoteAddr  string
		want        string
	}{
		{
			name:       "X-Forwarded-For single",
			xff:        "1.2.3.4",
			remoteAddr: "127.0.0.1:8080",
			want:       "1.2.3.4",
		},
		{
			name:       "X-Forwarded-For multiple",
			xff:        "1.2.3.4, 5.6.7.8, 9.10.11.12",
			remoteAddr: "127.0.0.1:8080",
			want:       "1.2.3.4",
		},
		{
			name:       "X-Real-IP",
			xri:        "1.2.3.4",
			remoteAddr: "127.0.0.1:8080",
			want:       "1.2.3.4",
		},
		{
			name:       "Fallback to RemoteAddr",
			remoteAddr: "192.168.1.1:12345",
			want:       "192.168.1.1:12345",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			if tc.xri != "" {
				req.Header.Set("X-Real-IP", tc.xri)
			}
			req.RemoteAddr = tc.remoteAddr

			got := getClientIP(req)
			if got != tc.want {
				t.Errorf("getClientIP() = %q, want %q", got, tc.want)
			}
		})
	}
}

// contains is a helper to check if a string contains a substring.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
