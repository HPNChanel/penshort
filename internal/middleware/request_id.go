// Package middleware provides HTTP middleware components.
package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// contextKey is a type for context keys to avoid collisions.
type contextKey string

const (
	// RequestIDKey is the context key for request ID.
	RequestIDKey contextKey = "request_id"
	// TraceIDKey is the context key for trace ID.
	TraceIDKey contextKey = "trace_id"
)

// RequestIDHeader is the HTTP header for request ID.
const RequestIDHeader = "X-Request-ID"

// TraceIDHeader is the HTTP header for trace ID.
const TraceIDHeader = "X-Trace-ID"

// RequestID injects a unique request ID into each request.
// If the X-Request-ID header is present, it uses that value.
// Otherwise, it generates a new UUID.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Get trace ID from header if present
		traceID := r.Header.Get(TraceIDHeader)

		// Add to context
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		if traceID != "" {
			ctx = context.WithValue(ctx, TraceIDKey, traceID)
		}

		// Add to response headers
		w.Header().Set(RequestIDHeader, requestID)
		if traceID != "" {
			w.Header().Set(TraceIDHeader, traceID)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// GetTraceID retrieves the trace ID from context.
func GetTraceID(ctx context.Context) string {
	if id, ok := ctx.Value(TraceIDKey).(string); ok {
		return id
	}
	return ""
}
