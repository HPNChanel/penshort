//go:build integration

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/penshort/penshort/internal/cache"
	"github.com/penshort/penshort/internal/model"
)

// TestRateLimitConcurrency verifies rate limiting under concurrent load.
// This test requires Redis to be running.
func TestRateLimitConcurrency(t *testing.T) {
	ctx := context.Background()

	// Skip if Redis not available
	redisURL := "redis://localhost:6379"
	cacheClient, err := cache.New(ctx, redisURL)
	if err != nil {
		t.Skipf("Skipping integration test: Redis not available: %v", err)
	}
	defer cacheClient.Close()

	// Clear any existing rate limit state
	_ = cacheClient.Client().FlushDB(ctx).Err()

	// Test parameters
	keyID := "test-key-concurrent"
	rpm := 10  // Low limit to trigger easily
	burst := 5

	// Track allowed vs rejected
	var allowed, rejected int64

	// Spawn 20 concurrent goroutines, each making 3 requests
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 3; j++ {
				result, err := cacheClient.CheckAPIRateLimit(ctx, keyID, rpm, burst)
				if err != nil {
					t.Errorf("CheckAPIRateLimit error: %v", err)
					return
				}
				if result.Allowed {
					atomic.AddInt64(&allowed, 1)
				} else {
					atomic.AddInt64(&rejected, 1)
				}
			}
		}()
	}

	wg.Wait()

	total := allowed + rejected
	t.Logf("Concurrency test: %d allowed, %d rejected (total: %d)", allowed, rejected, total)

	// We expect roughly burst amount to be allowed initially
	// With 60 requests total and 10 RPM (burst 5), most should be rejected
	if allowed > int64(burst+rpm) {
		t.Errorf("Too many requests allowed: %d (expected <= %d)", allowed, burst+rpm)
	}

	if rejected == 0 {
		t.Error("Expected some requests to be rejected")
	}
}

// TestIPRateLimitConcurrency verifies IP-based rate limiting concurrency.
func TestIPRateLimitConcurrency(t *testing.T) {
	ctx := context.Background()

	redisURL := "redis://localhost:6379"
	cacheClient, err := cache.New(ctx, redisURL)
	if err != nil {
		t.Skipf("Skipping integration test: Redis not available: %v", err)
	}
	defer cacheClient.Close()

	_ = cacheClient.Client().FlushDB(ctx).Err()

	testIP := "192.168.1.100"
	rps := 5
	burst := 3

	var allowed, rejected int64
	var wg sync.WaitGroup

	// 30 concurrent requests
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, _ := cacheClient.CheckIPRateLimit(ctx, testIP, rps, burst)
			if result.Allowed {
				atomic.AddInt64(&allowed, 1)
			} else {
				atomic.AddInt64(&rejected, 1)
			}
		}()
	}

	wg.Wait()

	t.Logf("IP rate limit: %d allowed, %d rejected", allowed, rejected)

	if rejected == 0 {
		t.Error("Expected some requests to be rejected")
	}
}

// TestRateLimitHeaders verifies rate limit headers are set correctly.
func TestRateLimitHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate setting rate limit headers
		setRateLimitHeaders(w, 60, 45, r.Context().Deadline())
		w.WriteHeader(http.StatusOK)
	})

	// The function should set headers
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-RateLimit-Limit") != "60" {
		t.Errorf("Expected X-RateLimit-Limit=60, got %s", rec.Header().Get("X-RateLimit-Limit"))
	}

	if rec.Header().Get("X-RateLimit-Remaining") != "45" {
		t.Errorf("Expected X-RateLimit-Remaining=45, got %s", rec.Header().Get("X-RateLimit-Remaining"))
	}
}

// Test429Response verifies the rate limit error response format.
func Test429Response(t *testing.T) {
	rec := httptest.NewRecorder()
	writeRateLimitError(rec, 5*1e9) // 5 seconds in nanoseconds

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected JSON content type")
	}

	body := rec.Body.String()
	if len(body) == 0 {
		t.Error("Expected error body")
	}
}

// TestTierConfigs verifies tier configuration is correct.
func TestTierConfigs(t *testing.T) {
	tests := []struct {
		tier    string
		wantRPM int
	}{
		{model.TierFree, 60},
		{model.TierPro, 600},
		{model.TierUnlimited, 0},
	}

	for _, tc := range tests {
		t.Run(tc.tier, func(t *testing.T) {
			config := model.TierConfigs[tc.tier]
			if config.RequestsPerMinute != tc.wantRPM {
				t.Errorf("Tier %s: expected RPM %d, got %d", tc.tier, tc.wantRPM, config.RequestsPerMinute)
			}
		})
	}
}
