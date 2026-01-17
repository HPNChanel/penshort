package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestLogging_APIKeyRedaction ensures API keys are not logged in plaintext.
// This is a critical security test per the unit testing atlas.
func TestLogging_APIKeyRedaction(t *testing.T) {
	t.Parallel()

	// API key patterns that should NEVER appear in logs
	sensitivePatterns := []string{
		"pk_live_abc123_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b",
		"pk_test_def456_0123456789abcdef0123456789abcdef",
		"pk_live_",
		"pk_test_",
	}

	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	// Create the logging middleware
	loggingMiddleware := Logger(logger)

	// Create a test handler that includes API key in request
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with logging middleware
	wrapped := loggingMiddleware(handler)

	// Create request with Authorization header containing API key
	req := httptest.NewRequest("GET", "/api/v1/links", nil)
	req.Header.Set("Authorization", "Bearer pk_live_abc123_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b")
	req.Header.Set("User-Agent", "TestAgent/1.0")

	// Execute request
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Get log output
	logOutput := buf.String()

	// Verify no sensitive patterns appear in logs
	for _, pattern := range sensitivePatterns {
		if strings.Contains(logOutput, pattern) {
			t.Errorf("Log output contains sensitive pattern %q - API keys should never be logged", pattern)
		}
	}
}

// TestLogging_NoAuthorizationHeaderLogged ensures the Authorization header is not logged.
func TestLogging_NoAuthorizationHeaderLogged(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	loggingMiddleware := Logger(logger)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := loggingMiddleware(handler)

	req := httptest.NewRequest("POST", "/api/v1/keys", nil)
	req.Header.Set("Authorization", "Bearer super_secret_token_12345")

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()

	// Authorization header value should not appear
	if strings.Contains(logOutput, "super_secret_token_12345") {
		t.Error("Log output contains Authorization header value")
	}
	if strings.Contains(logOutput, "Bearer") {
		t.Error("Log output contains 'Bearer' token prefix")
	}
}

// TestLogging_BasicFields verifies that expected non-sensitive fields are logged.
func TestLogging_BasicFields(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	loggingMiddleware := Logger(logger)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	wrapped := loggingMiddleware(handler)

	req := httptest.NewRequest("POST", "/api/v1/links", nil)
	req.Header.Set("User-Agent", "TestBrowser/2.0")

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()

	// These fields should be logged
	expectedFields := []string{
		`"method":"POST"`,
		`"path":"/api/v1/links"`,
		`"status_code":201`,
		`"user_agent":"TestBrowser/2.0"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(logOutput, field) {
			t.Errorf("Expected log field %s not found in output", field)
		}
	}
}

// TestLogging_ErrorStatusLevel verifies error statuses are logged at error level.
func TestLogging_ErrorStatusLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		wantLevel  string
	}{
		{"success", http.StatusOK, "INFO"},
		{"created", http.StatusCreated, "INFO"},
		{"bad request", http.StatusBadRequest, "WARN"},
		{"unauthorized", http.StatusUnauthorized, "WARN"},
		{"not found", http.StatusNotFound, "WARN"},
		{"internal error", http.StatusInternalServerError, "ERROR"},
		{"bad gateway", http.StatusBadGateway, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, nil))

			loggingMiddleware := Logger(logger)
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			})

			wrapped := loggingMiddleware(handler)

			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)

			logOutput := buf.String()

			// Check log level
			if !strings.Contains(logOutput, `"level":"`+tt.wantLevel+`"`) {
				t.Errorf("Expected log level %s for status %d, got output: %s", tt.wantLevel, tt.statusCode, logOutput)
			}
		})
	}
}

// TestResponseWriter_CapturesStatus verifies the response writer correctly captures status codes.
func TestResponseWriter_CapturesStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
	}{
		{"ok", http.StatusOK},
		{"created", http.StatusCreated},
		{"no content", http.StatusNoContent},
		{"bad request", http.StatusBadRequest},
		{"internal error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			wrapped := wrapResponseWriter(rec)

			wrapped.WriteHeader(tt.statusCode)

			if wrapped.status != tt.statusCode {
				t.Errorf("status = %d, want %d", wrapped.status, tt.statusCode)
			}
		})
	}
}

// TestResponseWriter_DefaultStatus verifies default status is 200 OK.
func TestResponseWriter_DefaultStatus(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	wrapped := wrapResponseWriter(rec)

	// Write without explicit WriteHeader
	wrapped.Write([]byte("hello"))

	if wrapped.status != http.StatusOK {
		t.Errorf("default status = %d, want %d", wrapped.status, http.StatusOK)
	}
}

// TestResponseWriter_DoubleWriteHeader ensures only first WriteHeader takes effect.
func TestResponseWriter_DoubleWriteHeader(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	wrapped := wrapResponseWriter(rec)

	wrapped.WriteHeader(http.StatusCreated)
	wrapped.WriteHeader(http.StatusInternalServerError) // Should be ignored

	if wrapped.status != http.StatusCreated {
		t.Errorf("status after double write = %d, want %d", wrapped.status, http.StatusCreated)
	}
}
