package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/penshort/penshort/internal/model"
)

// Cache key prefixes and TTLs.
const (
	linkKeyPrefix     = "link:"
	negCacheKeySuffix = ":neg"
	clicksKeyPrefix   = "clicks:"

	// DefaultLinkTTL is the TTL for cached link data.
	DefaultLinkTTL = 24 * time.Hour

	// NegativeCacheTTL is the TTL for negative cache entries.
	NegativeCacheTTL = 5 * time.Minute
)

// Common cache errors.
var (
	ErrCacheMiss = errors.New("cache miss")
)

// GetLink retrieves a link from cache by short code.
// Returns ErrCacheMiss if not found.
func (c *Cache) GetLink(ctx context.Context, shortCode string) (*model.CachedLink, error) {
	key := linkKeyPrefix + shortCode

	result, err := c.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis hgetall failed: %w", err)
	}

	if len(result) == 0 {
		return nil, ErrCacheMiss
	}

	cached := &model.CachedLink{
		Destination:  result["destination"],
		RedirectType: result["redirect_type"],
		ExpiresAt:    result["expires_at"],
		Enabled:      result["enabled"],
		DeletedAt:    result["deleted_at"],
		UpdatedAt:    result["updated_at"],
	}

	return cached, nil
}

// SetLink stores a link in cache.
func (c *Cache) SetLink(ctx context.Context, shortCode string, link *model.Link) error {
	key := linkKeyPrefix + shortCode
	cached := link.ToCachedLink()

	ttl := DefaultLinkTTL
	if link.ExpiresAt != nil {
		expiresIn := time.Until(*link.ExpiresAt)
		if expiresIn <= 0 {
			c.client.Del(ctx, key, key+negCacheKeySuffix)
			return nil
		}
		if expiresIn < ttl {
			ttl = expiresIn
		}
	}

	fields := map[string]any{
		"destination":   cached.Destination,
		"redirect_type": cached.RedirectType,
		"enabled":       cached.Enabled,
		"updated_at":    cached.UpdatedAt,
	}

	// Only set optional fields if they have values
	if cached.ExpiresAt != "" {
		fields["expires_at"] = cached.ExpiresAt
	}
	if cached.DeletedAt != "" {
		fields["deleted_at"] = cached.DeletedAt
	}

	pipe := c.client.Pipeline()
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, ttl)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to cache link: %w", err)
	}

	// Remove negative cache if exists
	c.client.Del(ctx, key+negCacheKeySuffix)

	return nil
}

// DeleteLink removes a link from cache.
func (c *Cache) DeleteLink(ctx context.Context, shortCode string) error {
	key := linkKeyPrefix + shortCode

	pipe := c.client.Pipeline()
	pipe.Del(ctx, key)
	pipe.Del(ctx, key+negCacheKeySuffix)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete link from cache: %w", err)
	}

	return nil
}

// IsNegativelyCached checks if a short code is in negative cache.
func (c *Cache) IsNegativelyCached(ctx context.Context, shortCode string) (bool, error) {
	key := linkKeyPrefix + shortCode + negCacheKeySuffix

	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check negative cache: %w", err)
	}

	return exists > 0, nil
}

// SetNegativeCache marks a short code as not found.
func (c *Cache) SetNegativeCache(ctx context.Context, shortCode string) error {
	key := linkKeyPrefix + shortCode + negCacheKeySuffix

	err := c.client.SetEx(ctx, key, "", NegativeCacheTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to set negative cache: %w", err)
	}

	return nil
}

// IncrementClicks increments the click counter in Redis.
// This is fire-and-forget for the redirect path.
func (c *Cache) IncrementClicks(ctx context.Context, shortCode string) error {
	key := clicksKeyPrefix + shortCode

	err := c.client.Incr(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to increment clicks: %w", err)
	}

	return nil
}

// GetAndResetClicks gets the current click count and resets it.
// Used by the background job to flush to PostgreSQL.
func (c *Cache) GetAndResetClicks(ctx context.Context, shortCode string) (int64, error) {
	key := clicksKeyPrefix + shortCode

	// GETSET is deprecated, use GETDEL or custom logic
	result, err := c.client.GetDel(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get and reset clicks: %w", err)
	}

	count, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse click count: %w", err)
	}

	return count, nil
}

// ScanClickKeys scans for all click counter keys.
// Used by the background job to find which links have pending click updates.
func (c *Cache) ScanClickKeys(ctx context.Context) ([]string, error) {
	var keys []string
	var cursor uint64

	for {
		var scanKeys []string
		var err error

		scanKeys, cursor, err = c.client.Scan(ctx, cursor, clicksKeyPrefix+"*", 100).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to scan click keys: %w", err)
		}

		keys = append(keys, scanKeys...)

		if cursor == 0 {
			break
		}
	}

	return keys, nil
}

// ExtractShortCodeFromClickKey extracts the short code from a click key.
func ExtractShortCodeFromClickKey(key string) string {
	if len(key) > len(clicksKeyPrefix) {
		return key[len(clicksKeyPrefix):]
	}
	return ""
}
