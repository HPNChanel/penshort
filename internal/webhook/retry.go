package webhook

import (
	"math/rand"
	"time"
)

// Retry delays for exponential backoff.
// Attempt 1: 1 min, Attempt 2: 5 min, Attempt 3: 30 min,
// Attempt 4: 2 hours, Attempt 5: 12 hours
var retryDelays = []time.Duration{
	1 * time.Minute,
	5 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
	12 * time.Hour,
}

const (
	// DefaultMaxAttempts is the default maximum delivery attempts.
	DefaultMaxAttempts = 5

	// JitterFactor is the ±percentage of jitter applied to delays.
	JitterFactor = 0.2 // ±20%
)

// NextRetryDelay calculates next retry delay with exponential backoff + jitter.
// attemptCount is 0-indexed (after first failed attempt, attemptCount = 0).
func NextRetryDelay(attemptCount int) time.Duration {
	if attemptCount < 0 {
		attemptCount = 0
	}
	if attemptCount >= len(retryDelays) {
		attemptCount = len(retryDelays) - 1
	}

	base := retryDelays[attemptCount]

	// Add ±20% jitter to prevent thundering herd
	jitterRange := float64(base) * JitterFactor
	jitter := (rand.Float64()*2 - 1) * jitterRange // -20% to +20%

	return time.Duration(float64(base) + jitter)
}

// NextRetryAt calculates the time for next retry attempt.
func NextRetryAt(attemptCount int) time.Time {
	return time.Now().Add(NextRetryDelay(attemptCount))
}

// IsExhausted returns true if max attempts have been reached.
func IsExhausted(attemptCount, maxAttempts int) bool {
	return attemptCount >= maxAttempts
}

// GetRetryDelays returns the configured retry delays (for testing/docs).
func GetRetryDelays() []time.Duration {
	return append([]time.Duration{}, retryDelays...)
}

// EstimatedMaxDeliveryWindow returns the maximum time span for all retries.
// This is useful for documentation and SLA communication.
func EstimatedMaxDeliveryWindow() time.Duration {
	var total time.Duration
	for _, d := range retryDelays {
		total += d
	}
	// Add ~20% for jitter overhead
	return time.Duration(float64(total) * 1.2)
}
