// Package middleware provides HTTP middleware for the Penshort API.
package middleware

import (
	"net/http"
	"strings"
)

// CORSConfig holds CORS configuration options.
type CORSConfig struct {
	// AllowedOrigins is a list of origins allowed to make cross-origin requests.
	// Use specific origins in production; never use "*" with credentials.
	AllowedOrigins []string

	// AllowedMethods specifies the allowed HTTP methods.
	// Default: GET, POST, PUT, PATCH, DELETE, OPTIONS
	AllowedMethods []string

	// AllowedHeaders specifies the allowed request headers.
	// Default: Content-Type, Authorization, X-API-Key, X-Request-ID
	AllowedHeaders []string

	// ExposedHeaders specifies which headers the browser can access.
	// Default: X-Request-ID, X-RateLimit-Remaining, X-RateLimit-Reset
	ExposedHeaders []string

	// AllowCredentials indicates whether credentials (cookies, auth) are allowed.
	// Be careful: if true, AllowedOrigins cannot contain "*".
	AllowCredentials bool

	// MaxAge is the value for Access-Control-Max-Age header (in seconds).
	// Default: 86400 (24 hours)
	MaxAge int
}

// DefaultCORSConfig returns production-safe CORS defaults.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{
			"Content-Type",
			"Authorization",
			"X-API-Key",
			"X-Request-ID",
			"Accept",
			"Accept-Language",
		},
		ExposedHeaders: []string{
			"X-Request-ID",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
		},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// CORS returns a middleware that handles Cross-Origin Resource Sharing.
//
// Important security notes:
//   - Never use "*" for AllowedOrigins in production with credentials
//   - Validate and whitelist origins explicitly
//   - This middleware handles preflight OPTIONS requests
func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	// Pre-compute joined strings for performance
	methodsStr := strings.Join(cfg.AllowedMethods, ", ")
	headersStr := strings.Join(cfg.AllowedHeaders, ", ")
	exposedStr := strings.Join(cfg.ExposedHeaders, ", ")
	maxAgeStr := ""
	if cfg.MaxAge > 0 {
		maxAgeStr = itoa(cfg.MaxAge)
	}

	// Build origin lookup map for O(1) checks
	originMap := make(map[string]bool, len(cfg.AllowedOrigins))
	for _, origin := range cfg.AllowedOrigins {
		originMap[strings.ToLower(origin)] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// No Origin header = same-origin request, skip CORS
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check if origin is allowed
			allowed := isOriginAllowed(origin, originMap, cfg.AllowedOrigins)
			if !allowed {
				// Origin not allowed - don't add CORS headers
				// For preflight, respond with 403
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				// For actual requests, proceed without CORS headers
				// Browser will block the response
				next.ServeHTTP(w, r)
				return
			}

			// Add CORS headers for allowed origin
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")

			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if exposedStr != "" {
				w.Header().Set("Access-Control-Expose-Headers", exposedStr)
			}

			// Handle preflight request
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", methodsStr)
				w.Header().Set("Access-Control-Allow-Headers", headersStr)

				if maxAgeStr != "" {
					w.Header().Set("Access-Control-Max-Age", maxAgeStr)
				}

				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isOriginAllowed checks if the given origin is in the allowed list.
func isOriginAllowed(origin string, originMap map[string]bool, allowedOrigins []string) bool {
	// If no origins configured, deny all cross-origin requests
	if len(allowedOrigins) == 0 {
		return false
	}

	// Normalize origin for comparison
	normalizedOrigin := strings.ToLower(origin)

	// Direct match
	if originMap[normalizedOrigin] {
		return true
	}

	// Check for wildcard subdomain patterns like "*.example.com"
	for _, allowed := range allowedOrigins {
		if strings.HasPrefix(allowed, "*.") {
			// Extract domain suffix
			suffix := strings.TrimPrefix(allowed, "*")
			if strings.HasSuffix(normalizedOrigin, strings.ToLower(suffix)) {
				// Ensure we're matching a subdomain, not a partial domain
				// e.g., "*.example.com" should match "sub.example.com" but not "notexample.com"
				prefix := strings.TrimSuffix(normalizedOrigin, strings.ToLower(suffix))
				if strings.HasSuffix(prefix, "://") || strings.Contains(prefix, ".") {
					return true
				}
			}
		}
	}

	return false
}

// itoa converts int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	var buf [20]byte
	i := len(buf)
	negative := n < 0
	if negative {
		n = -n
	}

	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}

	if negative {
		i--
		buf[i] = '-'
	}

	return string(buf[i:])
}
