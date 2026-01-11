// Package middleware provides HTTP middleware for the Penshort API.
package middleware

import (
	"net/http"
)

// SecurityConfig holds configuration for security headers.
type SecurityConfig struct {
	// IsDevelopment disables HSTS in dev environments.
	IsDevelopment bool
	// AllowedOrigins for CORS. If empty, CORS headers are not added.
	AllowedOrigins []string
	// MaxRequestBodySize is the max allowed request body in bytes.
	// Default: 1MB (1048576 bytes).
	MaxRequestBodySize int64
}

// DefaultSecurityConfig returns sensible defaults for production.
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		IsDevelopment:      false,
		AllowedOrigins:     []string{},
		MaxRequestBodySize: 1 << 20, // 1MB
	}
}

// Security returns a middleware that applies security headers to all responses.
// This middleware should be applied early in the chain.
//
// Headers applied:
//   - Strict-Transport-Security (HSTS) - only in production
//   - X-Content-Type-Options: nosniff
//   - X-Frame-Options: DENY
//   - X-XSS-Protection: 0 (disabled, CSP is the modern approach)
//   - Referrer-Policy: strict-origin-when-cross-origin
//   - Content-Security-Policy: minimal policy for API responses
//   - Permissions-Policy: restrictive policy
//   - Cache-Control: no-store for API responses
func Security(cfg SecurityConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// === Prevent MIME type sniffing ===
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// === Prevent clickjacking ===
			w.Header().Set("X-Frame-Options", "DENY")

			// === Disable legacy XSS filter (CSP is the modern approach) ===
			// Setting to "0" prevents false positives in older browsers.
			w.Header().Set("X-XSS-Protection", "0")

			// === Control referrer information ===
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// === Content Security Policy (minimal for API) ===
			// Very restrictive since we're an API, not serving HTML.
			w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

			// === Permissions Policy (disable unused browser features) ===
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=(), usb=()")

			// === HSTS (only in production with HTTPS) ===
			// max-age=31536000 = 1 year
			// includeSubDomains ensures all subdomains use HTTPS
			// preload allows submission to browser preload lists
			if !cfg.IsDevelopment {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			}

			// === Prevent caching sensitive data ===
			// API responses should generally not be cached.
			w.Header().Set("Cache-Control", "no-store")

			// === Remove server identification (if not overridden) ===
			w.Header().Del("Server")

			next.ServeHTTP(w, r)
		})
	}
}

// MaxBodySize returns a middleware that limits request body size.
// This prevents denial-of-service via large request bodies.
//
// When the limit is exceeded, the connection is closed and subsequent
// reads return an error.
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil && r.ContentLength > maxBytes {
				http.Error(w, `{"error":{"code":"PAYLOAD_TOO_LARGE","message":"Request body too large"}}`, http.StatusRequestEntityTooLarge)
				return
			}

			// Wrap body with MaxBytesReader for streaming protection
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

			next.ServeHTTP(w, r)
		})
	}
}
