package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/penshort/penshort/internal/auth"
	"github.com/penshort/penshort/internal/cache"
	"github.com/penshort/penshort/internal/model"
)

// RateLimitConfig holds configuration for rate limiting middleware.
type RateLimitConfig struct {
	Logger *slog.Logger
	Cache  *cache.Cache
	// API rate limiting (per API key)
	APIEnabled bool
	// Redirect rate limiting (per IP)
	RedirectEnabled  bool
	RedirectRPS      int // Requests per second
	RedirectBurst    int
}

// RateLimitAPI returns middleware that rate limits API requests per API key.
// Must be applied after Auth middleware.
func RateLimitAPI(cfg RateLimitConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.APIEnabled {
				next.ServeHTTP(w, r)
				return
			}

			authCtx := auth.AuthFromContext(r.Context())
			if authCtx == nil {
				// No auth context - should not happen if Auth middleware ran first
				next.ServeHTTP(w, r)
				return
			}

			// Get rate limit config for this key's tier
			tierConfig := model.TierConfigs[authCtx.RateLimitTier]
			if tierConfig.RequestsPerMinute == 0 {
				// Unlimited tier
				setRateLimitHeaders(w, 0, 0, time.Now())
				next.ServeHTTP(w, r)
				return
			}

			result, err := cfg.Cache.CheckAPIRateLimit(
				r.Context(),
				authCtx.KeyID,
				tierConfig.RequestsPerMinute,
				tierConfig.Burst,
			)
			if err != nil {
				cfg.Logger.Error("rate limit check failed",
					slog.String("error", err.Error()),
					slog.String("key_id", authCtx.KeyID),
				)
				// Fail open - allow request
				next.ServeHTTP(w, r)
				return
			}

			setRateLimitHeaders(w, tierConfig.RequestsPerMinute, result.Remaining, result.ResetAt)

			if !result.Allowed {
				cfg.Logger.Warn("rate limit exceeded",
					slog.String("key_id", authCtx.KeyID),
					slog.String("type", "api"),
					slog.String("ip", r.RemoteAddr),
					slog.String("endpoint", r.Method+" "+r.URL.Path),
					slog.Int64("retry_after_seconds", int64(result.RetryAfter.Seconds())),
					slog.String("request_id", GetRequestID(r.Context())),
				)

				w.Header().Set("Retry-After", strconv.Itoa(int(result.RetryAfter.Seconds())))
				writeRateLimitError(w, result.RetryAfter)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitIP returns middleware that rate limits requests per IP.
// Used for the redirect endpoint to prevent abuse.
func RateLimitIP(cfg RateLimitConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.RedirectEnabled {
				next.ServeHTTP(w, r)
				return
			}

			ip := getClientIP(r)

			result, err := cfg.Cache.CheckIPRateLimit(
				r.Context(),
				ip,
				cfg.RedirectRPS,
				cfg.RedirectBurst,
			)
			if err != nil {
				cfg.Logger.Error("IP rate limit check failed",
					slog.String("error", err.Error()),
					slog.String("ip", ip),
				)
				// Fail open - allow request
				next.ServeHTTP(w, r)
				return
			}

			if !result.Allowed {
				cfg.Logger.Warn("rate limit exceeded",
					slog.String("type", "redirect"),
					slog.String("ip", ip),
					slog.String("endpoint", r.Method+" "+r.URL.Path),
					slog.Int64("retry_after_seconds", int64(result.RetryAfter.Seconds())),
					slog.String("request_id", GetRequestID(r.Context())),
				)

				w.Header().Set("Retry-After", strconv.Itoa(int(result.RetryAfter.Seconds())))
				writeRateLimitError(w, result.RetryAfter)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// setRateLimitHeaders sets standard rate limit response headers.
func setRateLimitHeaders(w http.ResponseWriter, limit int, remaining int64, resetAt time.Time) {
	if limit > 0 {
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))
	}
}

// writeRateLimitError writes a 429 Too Many Requests response.
func writeRateLimitError(w http.ResponseWriter, retryAfter time.Duration) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	msg := fmt.Sprintf(`{"error":{"code":"RATE_LIMITED","message":"Rate limit exceeded. Retry after %d seconds."}}`,
		int(retryAfter.Seconds()))
	_, _ = w.Write([]byte(msg))
}

// getClientIP extracts the client IP from the request.
// Checks X-Forwarded-For and X-Real-IP headers for proxied requests.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For first (may contain multiple IPs)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP (client IP)
		for i := range xff {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	// Check X-Real-IP
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
