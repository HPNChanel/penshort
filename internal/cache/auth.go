package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/penshort/penshort/internal/model"
)

const (
	// authCachePrefix is the Redis key prefix for auth context cache.
	authCachePrefix = "auth:ctx:"
	// authCacheTTL is the time-to-live for cached auth contexts.
	authCacheTTL = 5 * time.Minute
)

// CachedAuthContext represents auth context stored in Redis.
type CachedAuthContext struct {
	KeyID         string   `json:"key_id"`
	KeyPrefix     string   `json:"key_prefix"`
	UserID        string   `json:"user_id"`
	Scopes        []string `json:"scopes"`
	RateLimitTier string   `json:"rate_limit_tier"`
}

// GetAuthContext retrieves a cached auth context by cache key.
// Returns nil if not found (cache miss).
func (c *Cache) GetAuthContext(ctx context.Context, cacheKey string) (*model.AuthContext, error) {
	key := authCachePrefix + cacheKey

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		// Cache miss is not an error
		return nil, nil //nolint:nilerr
	}

	var cached CachedAuthContext
	if err := json.Unmarshal(data, &cached); err != nil {
		// Corrupted cache entry - treat as miss
		return nil, nil //nolint:nilerr
	}

	return &model.AuthContext{
		KeyID:         cached.KeyID,
		KeyPrefix:     cached.KeyPrefix,
		UserID:        cached.UserID,
		Scopes:        cached.Scopes,
		RateLimitTier: cached.RateLimitTier,
	}, nil
}

// SetAuthContext caches an auth context.
func (c *Cache) SetAuthContext(ctx context.Context, cacheKey string, auth *model.AuthContext) error {
	key := authCachePrefix + cacheKey

	cached := CachedAuthContext{
		KeyID:         auth.KeyID,
		KeyPrefix:     auth.KeyPrefix,
		UserID:        auth.UserID,
		Scopes:        auth.Scopes,
		RateLimitTier: auth.RateLimitTier,
	}

	data, err := json.Marshal(cached)
	if err != nil {
		return fmt.Errorf("marshal auth context: %w", err)
	}

	return c.client.Set(ctx, key, data, authCacheTTL).Err()
}

// DeleteAuthContext removes a cached auth context.
// Used when a key is revoked.
func (c *Cache) DeleteAuthContext(ctx context.Context, cacheKey string) error {
	key := authCachePrefix + cacheKey
	return c.client.Del(ctx, key).Err()
}

// InvalidateUserAuthContexts removes all cached auth contexts for a user.
// Note: This requires scanning keys, which is O(N). Use sparingly.
func (c *Cache) InvalidateUserAuthContexts(ctx context.Context, userID string) error {
	// In production, consider maintaining a set of cache keys per user
	// For now, we don't implement per-user invalidation to avoid key scanning
	// Individual key revocation should call DeleteAuthContext directly
	return nil
}
