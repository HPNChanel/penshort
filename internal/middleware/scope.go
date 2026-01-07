package middleware

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/penshort/penshort/internal/auth"
	"github.com/penshort/penshort/internal/model"
)

// RequireScope returns middleware that enforces scope requirements.
// Must be applied after Auth middleware.
// If multiple scopes are provided, having ANY of them is sufficient.
func RequireScope(required ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := auth.AuthFromContext(r.Context())
			if authCtx == nil {
				writeScopeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
				return
			}

			// Admin scope grants all permissions
			if slices.Contains(authCtx.Scopes, model.ScopeAdmin) {
				next.ServeHTTP(w, r)
				return
			}

			// Check if any required scope is present
			for _, req := range required {
				if slices.Contains(authCtx.Scopes, req) {
					next.ServeHTTP(w, r)
					return
				}
			}

			writeScopeError(w, http.StatusForbidden, "FORBIDDEN",
				fmt.Sprintf("Insufficient permissions. Required scope: %s", required[0]))
		})
	}
}

// RequireRead is a convenience middleware for read scope.
func RequireRead() func(http.Handler) http.Handler {
	return RequireScope(model.ScopeRead)
}

// RequireWrite is a convenience middleware for write scope.
func RequireWrite() func(http.Handler) http.Handler {
	return RequireScope(model.ScopeWrite)
}

// RequireAdmin is a convenience middleware for admin scope.
func RequireAdmin() func(http.Handler) http.Handler {
	return RequireScope(model.ScopeAdmin)
}

// RequireWebhook is a convenience middleware for webhook scope.
func RequireWebhook() func(http.Handler) http.Handler {
	return RequireScope(model.ScopeWebhook)
}

// writeScopeError writes a scope-related error response.
func writeScopeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"%s","message":"%s"}}`, code, message)))
}
