package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// rateLimitAPIPrefix is the Redis key prefix for API key rate limits.
	rateLimitAPIPrefix = "ratelimit:apikey:"
	// rateLimitIPPrefix is the Redis key prefix for IP rate limits.
	rateLimitIPPrefix = "ratelimit:ip:"
	// rateLimitAPITTL is the TTL for API rate limit keys.
	rateLimitAPITTL = 120 * time.Second
	// rateLimitIPTTL is the TTL for IP rate limit keys.
	rateLimitIPTTL = 10 * time.Second
)

// RateLimitResult contains the result of a rate limit check.
type RateLimitResult struct {
	Allowed    bool
	Remaining  int64
	ResetAt    time.Time
	RetryAfter time.Duration
}

// tokenBucketScript is a Lua script implementing the token bucket algorithm.
// It's atomic and handles token refill and consumption in a single operation.
var tokenBucketScript = redis.NewScript(`
	local key = KEYS[1]
	local rate = tonumber(ARGV[1])      -- tokens per second
	local burst = tonumber(ARGV[2])     -- max tokens (bucket capacity)
	local now = tonumber(ARGV[3])       -- current time in seconds
	local ttl = tonumber(ARGV[4])       -- TTL in seconds

	-- Get current state
	local data = redis.call('HMGET', key, 'tokens', 'last_update')
	local tokens = tonumber(data[1]) or burst
	local last_update = tonumber(data[2]) or now

	-- Refill tokens based on elapsed time
	local elapsed = now - last_update
	tokens = math.min(burst, tokens + (elapsed * rate))

	-- Check if request is allowed
	local allowed = 0
	local retry_after = 0

	if tokens >= 1 then
		tokens = tokens - 1
		allowed = 1
	else
		-- Calculate when 1 token will be available
		retry_after = math.ceil((1 - tokens) / rate)
	end

	-- Update state
	redis.call('HMSET', key, 'tokens', tokens, 'last_update', now)
	redis.call('EXPIRE', key, ttl)

	return {allowed, retry_after, math.floor(tokens)}
`)

// CheckAPIRateLimit checks and updates the rate limit for an API key.
// Returns whether the request is allowed and rate limit metadata.
func (c *Cache) CheckAPIRateLimit(ctx context.Context, keyID string, ratePerMinute, burst int) (*RateLimitResult, error) {
	// Unlimited tier
	if ratePerMinute == 0 {
		return &RateLimitResult{
			Allowed:   true,
			Remaining: int64(burst),
			ResetAt:   time.Now().Add(time.Minute),
		}, nil
	}

	key := rateLimitAPIPrefix + keyID
	ratePerSecond := float64(ratePerMinute) / 60.0

	return c.checkRateLimit(ctx, key, ratePerSecond, burst, int(rateLimitAPITTL.Seconds()))
}

// CheckIPRateLimit checks and updates the rate limit for an IP address.
// IP is hashed to avoid storing raw IP addresses.
func (c *Cache) CheckIPRateLimit(ctx context.Context, ip string, ratePerSecond, burst int) (*RateLimitResult, error) {
	// Hash IP for privacy
	hashedIP := hashIP(ip)
	key := rateLimitIPPrefix + hashedIP

	return c.checkRateLimit(ctx, key, float64(ratePerSecond), burst, int(rateLimitIPTTL.Seconds()))
}

// checkRateLimit is the common rate limit implementation.
func (c *Cache) checkRateLimit(ctx context.Context, key string, rate float64, burst, ttl int) (*RateLimitResult, error) {
	now := time.Now().Unix()

	result, err := tokenBucketScript.Run(ctx, c.client,
		[]string{key},
		rate, burst, now, ttl,
	).Int64Slice()

	if err != nil {
		// Fail open on Redis errors - allow the request
		return &RateLimitResult{
			Allowed:   true,
			Remaining: int64(burst),
			ResetAt:   time.Now().Add(time.Minute),
		}, nil
	}

	allowed := result[0] == 1
	retryAfterSec := result[1]
	remaining := result[2]

	return &RateLimitResult{
		Allowed:    allowed,
		Remaining:  remaining,
		ResetAt:    time.Now().Add(time.Duration(float64(time.Second) / rate)),
		RetryAfter: time.Duration(retryAfterSec) * time.Second,
	}, nil
}

// hashIP creates a truncated SHA256 hash of an IP address.
// This provides privacy while maintaining uniqueness.
func hashIP(ip string) string {
	hash := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(hash[:8]) // 16 hex chars
}
