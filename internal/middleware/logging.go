package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Logger returns a middleware that logs HTTP requests.
// Uses structured logging with slog.
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status
			wrapped := wrapResponseWriter(w)

			// Process request
			next.ServeHTTP(wrapped, r)

			// Calculate duration
			duration := time.Since(start)

			// Get request ID and trace ID from context
			requestID := GetRequestID(r.Context())
			traceID := GetTraceID(r.Context())

			// Build log attributes
			attrs := []slog.Attr{
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status_code", wrapped.status),
				slog.Float64("duration_ms", float64(duration.Microseconds())/1000),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
			}

			// Add trace ID if present
			if traceID != "" {
				attrs = append(attrs, slog.String("trace_id", traceID))
			}

			// Log at appropriate level based on status code
			level := slog.LevelInfo
			if wrapped.status >= 500 {
				level = slog.LevelError
			} else if wrapped.status >= 400 {
				level = slog.LevelWarn
			}

			logger.LogAttrs(r.Context(), level, "http request", attrs...)
		})
	}
}
