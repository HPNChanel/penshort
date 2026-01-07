// Package model defines domain entities for the application.
package model

import (
	"slices"
	"time"
)

// Scope constants for API key authorization.
const (
	ScopeRead    = "read"
	ScopeWrite   = "write"
	ScopeWebhook = "webhook"
	ScopeAdmin   = "admin"
)

// ValidScopes contains all valid scope values.
var ValidScopes = []string{ScopeRead, ScopeWrite, ScopeWebhook, ScopeAdmin}

// RateLimitTier constants.
const (
	TierFree      = "free"
	TierPro       = "pro"
	TierUnlimited = "unlimited"
)

// RateLimitConfig defines rate limit parameters per tier.
type RateLimitConfig struct {
	RequestsPerMinute int
	Burst             int
}

// TierConfigs maps tier names to their rate limit configurations.
var TierConfigs = map[string]RateLimitConfig{
	TierFree:      {RequestsPerMinute: 60, Burst: 10},
	TierPro:       {RequestsPerMinute: 600, Burst: 50},
	TierUnlimited: {RequestsPerMinute: 0, Burst: 0}, // 0 means unlimited
}

// APIKey represents an API key entity.
type APIKey struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	KeyHash       string     `json:"-"` // Never serialize
	KeyPrefix     string     `json:"key_prefix"`
	Scopes        []string   `json:"scopes"`
	RateLimitTier string     `json:"rate_limit_tier"`
	Name          string     `json:"name,omitempty"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
	LastUsedAt    *time.Time `json:"last_used_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// IsRevoked returns true if the key has been revoked.
func (k *APIKey) IsRevoked() bool {
	return k.RevokedAt != nil
}

// HasScope checks if the key has a specific scope.
// Admin scope implies all other scopes.
func (k *APIKey) HasScope(scope string) bool {
	if slices.Contains(k.Scopes, ScopeAdmin) {
		return true
	}
	return slices.Contains(k.Scopes, scope)
}

// GetRateLimitConfig returns the rate limit configuration for this key.
func (k *APIKey) GetRateLimitConfig() RateLimitConfig {
	if config, ok := TierConfigs[k.RateLimitTier]; ok {
		return config
	}
	return TierConfigs[TierFree] // Default to free tier
}

// AuthContext holds authenticated request context.
// This is injected into the request context by auth middleware.
type AuthContext struct {
	KeyID         string
	KeyPrefix     string
	UserID        string
	Scopes        []string
	RateLimitTier string
}

// HasScope checks if the auth context has a specific scope.
func (a *AuthContext) HasScope(scope string) bool {
	if slices.Contains(a.Scopes, ScopeAdmin) {
		return true
	}
	return slices.Contains(a.Scopes, scope)
}

// APIKeyCreateRequest represents a request to create a new API key.
type APIKeyCreateRequest struct {
	Name   string   `json:"name,omitempty"`
	Scopes []string `json:"scopes"`
}

// APIKeyResponse represents the response for an API key (without secrets).
type APIKeyResponse struct {
	ID            string     `json:"id"`
	Name          string     `json:"name,omitempty"`
	KeyPrefix     string     `json:"key_prefix"`
	Scopes        []string   `json:"scopes"`
	RateLimitTier string     `json:"rate_limit_tier"`
	CreatedAt     time.Time  `json:"created_at"`
	LastUsedAt    *time.Time `json:"last_used_at,omitempty"`
	Revoked       bool       `json:"revoked"`
}

// ToResponse converts an APIKey to APIKeyResponse.
func (k *APIKey) ToResponse() APIKeyResponse {
	return APIKeyResponse{
		ID:            k.ID,
		Name:          k.Name,
		KeyPrefix:     k.KeyPrefix,
		Scopes:        k.Scopes,
		RateLimitTier: k.RateLimitTier,
		CreatedAt:     k.CreatedAt,
		LastUsedAt:    k.LastUsedAt,
		Revoked:       k.IsRevoked(),
	}
}

// APIKeyCreateResponse includes the plaintext key (shown only once).
type APIKeyCreateResponse struct {
	ID            string    `json:"id"`
	Key           string    `json:"key"` // Plaintext - display once only!
	Name          string    `json:"name,omitempty"`
	KeyPrefix     string    `json:"key_prefix"`
	Scopes        []string  `json:"scopes"`
	RateLimitTier string    `json:"rate_limit_tier"`
	CreatedAt     time.Time `json:"created_at"`
}

// APIKeyRotateResponse includes both old and new key information.
type APIKeyRotateResponse struct {
	OldKeyID        string               `json:"old_key_id"`
	OldKeyRevokedAt time.Time            `json:"old_key_revoked_at"`
	NewKey          APIKeyCreateResponse `json:"new_key"`
}
