// Package auth provides authentication utilities for API keys.
package auth

import (
	"context"

	"github.com/penshort/penshort/internal/model"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// authContextKey is the context key for storing AuthContext.
	authContextKey contextKey = "auth_context"
)

// ContextWithAuth adds AuthContext to the context.
func ContextWithAuth(ctx context.Context, auth *model.AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey, auth)
}

// AuthFromContext retrieves AuthContext from the context.
// Returns nil if not present.
func AuthFromContext(ctx context.Context) *model.AuthContext {
	auth, ok := ctx.Value(authContextKey).(*model.AuthContext)
	if !ok {
		return nil
	}
	return auth
}

// MustAuthFromContext retrieves AuthContext from the context.
// Panics if not present (use only when auth middleware has run).
func MustAuthFromContext(ctx context.Context) *model.AuthContext {
	auth := AuthFromContext(ctx)
	if auth == nil {
		panic("auth context not found - ensure auth middleware is applied")
	}
	return auth
}

// UserIDFromContext is a convenience function to get user ID from context.
// Returns empty string if not authenticated.
func UserIDFromContext(ctx context.Context) string {
	auth := AuthFromContext(ctx)
	if auth == nil {
		return ""
	}
	return auth.UserID
}

// KeyIDFromContext is a convenience function to get key ID from context.
// Returns empty string if not authenticated.
func KeyIDFromContext(ctx context.Context) string {
	auth := AuthFromContext(ctx)
	if auth == nil {
		return ""
	}
	return auth.KeyID
}
