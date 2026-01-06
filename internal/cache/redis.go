// Package cache provides Redis cache access layer.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache provides Redis cache access methods.
type Cache struct {
	client *redis.Client
}

// New creates a new Cache with a Redis client.
func New(ctx context.Context, redisURL string) (*Cache, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Connection pool settings
	opt.PoolSize = 10
	opt.MinIdleConns = 2
	opt.PoolTimeout = 4 * time.Second
	opt.ConnMaxIdleTime = 5 * time.Minute

	client := redis.NewClient(opt)

	// Verify connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return &Cache{client: client}, nil
}

// Ping checks Redis connectivity.
func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Close closes the Redis client.
func (c *Cache) Close() error {
	return c.client.Close()
}

// Client returns the underlying Redis client.
// Use sparingly - prefer adding methods to Cache.
func (c *Cache) Client() *redis.Client {
	return c.client
}
