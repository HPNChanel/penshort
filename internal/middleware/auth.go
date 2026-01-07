package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/penshort/penshort/internal/auth"
	"github.com/penshort/penshort/internal/cache"
	"github.com/penshort/penshort/internal/model"
	"github.com/penshort/penshort/internal/repository"
)

const (
	// minAuthDuration is the minimum time to spend on auth to prevent timing attacks.
	minAuthDuration = 200 * time.Millisecond
)

// AuthConfig holds configuration for the auth middleware.
type AuthConfig struct {
	Logger     *slog.Logger
	Repository *repository.Repository
	Cache      *cache.Cache
}

// Auth returns a middleware that authenticates API requests.
// It extracts the API key from the Authorization header,
// verifies it, and injects the auth context into the request.
func Auth(cfg AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()

			// Ensure consistent timing regardless of outcome
			defer func() {
				elapsed := time.Since(startTime)
				if elapsed < minAuthDuration {
					time.Sleep(minAuthDuration - elapsed)
				}
			}()

			// Extract key from header
			key := extractAPIKey(r)
			if key == "" {
				cfg.Logger.Warn("authentication failed",
					slog.String("reason", "missing_key"),
					slog.String("ip", r.RemoteAddr),
					slog.String("endpoint", r.Method+" "+r.URL.Path),
					slog.String("request_id", GetRequestID(r.Context())),
				)
				writeAuthError(w)
				return
			}

			// Validate key format
			parsed, err := auth.ParseAPIKey(key)
			if err != nil {
				cfg.Logger.Warn("authentication failed",
					slog.String("reason", "invalid_format"),
					slog.String("ip", r.RemoteAddr),
					slog.String("endpoint", r.Method+" "+r.URL.Path),
					slog.String("request_id", GetRequestID(r.Context())),
				)
				writeAuthError(w)
				return
			}

			// Check cache first
			cacheKey := auth.QuickHash(key)
			authCtx, _ := cfg.Cache.GetAuthContext(r.Context(), cacheKey)

			if authCtx != nil {
				// Cache hit - use cached auth context
				cfg.Logger.Info("authentication successful",
					slog.String("key_id", authCtx.KeyID),
					slog.String("key_prefix", authCtx.KeyPrefix),
					slog.String("user_id", authCtx.UserID),
					slog.String("ip", r.RemoteAddr),
					slog.String("endpoint", r.Method+" "+r.URL.Path),
					slog.Bool("cache_hit", true),
					slog.String("request_id", GetRequestID(r.Context())),
				)

				ctx := auth.ContextWithAuth(r.Context(), authCtx)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Cache miss - lookup by prefix
			keys, err := cfg.Repository.GetAPIKeysByPrefix(r.Context(), parsed.Prefix)
			if err != nil {
				cfg.Logger.Error("database error during auth",
					slog.String("error", err.Error()),
					slog.String("request_id", GetRequestID(r.Context())),
				)
				writeAuthError(w)
				return
			}

			if len(keys) == 0 {
				cfg.Logger.Warn("authentication failed",
					slog.String("reason", "invalid_key"),
					slog.String("ip", r.RemoteAddr),
					slog.String("endpoint", r.Method+" "+r.URL.Path),
					slog.String("request_id", GetRequestID(r.Context())),
				)
				writeAuthError(w)
				return
			}

			// Verify against each candidate key (handles prefix collisions)
			var matchedKey *model.APIKey
			for _, k := range keys {
				match, err := auth.VerifyPassword(key, k.KeyHash)
				if err != nil {
					continue
				}
				if match {
					matchedKey = k
					break
				}
			}

			if matchedKey == nil {
				cfg.Logger.Warn("authentication failed",
					slog.String("reason", "invalid_key"),
					slog.String("ip", r.RemoteAddr),
					slog.String("endpoint", r.Method+" "+r.URL.Path),
					slog.String("request_id", GetRequestID(r.Context())),
				)
				writeAuthError(w)
				return
			}

			// Build auth context
			authCtx = &model.AuthContext{
				KeyID:         matchedKey.ID,
				KeyPrefix:     matchedKey.KeyPrefix,
				UserID:        matchedKey.UserID,
				Scopes:        matchedKey.Scopes,
				RateLimitTier: matchedKey.RateLimitTier,
			}

			// Cache the result
			_ = cfg.Cache.SetAuthContext(r.Context(), cacheKey, authCtx)

			// Update last_used_at asynchronously
			go func() {
				_ = cfg.Repository.UpdateAPIKeyLastUsed(r.Context(), matchedKey.ID)
			}()

			cfg.Logger.Info("authentication successful",
				slog.String("key_id", authCtx.KeyID),
				slog.String("key_prefix", authCtx.KeyPrefix),
				slog.String("user_id", authCtx.UserID),
				slog.String("ip", r.RemoteAddr),
				slog.String("endpoint", r.Method+" "+r.URL.Path),
				slog.Bool("cache_hit", false),
				slog.String("request_id", GetRequestID(r.Context())),
			)

			ctx := auth.ContextWithAuth(r.Context(), authCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractAPIKey extracts the API key from the request.
// Supports both "Authorization: Bearer <key>" and "X-API-Key: <key>" headers.
func extractAPIKey(r *http.Request) string {
	// Try Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	// Fall back to X-API-Key header
	return r.Header.Get("X-API-Key")
}

// writeAuthError writes a 401 Unauthorized response.
// Uses the same message for all auth failures to prevent enumeration.
func writeAuthError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":{"code":"UNAUTHORIZED","message":"Invalid or missing API key"}}`))
}
