package webhook

import (
	"testing"
	"time"
)

func TestNextRetryDelay(t *testing.T) {
	// Test that delays are in expected ranges
	tests := []struct {
		attempt   int
		minDelay  time.Duration
		maxDelay  time.Duration
	}{
		{0, 48 * time.Second, 72 * time.Second},         // 1min ± 20%
		{1, 4 * time.Minute, 6 * time.Minute},           // 5min ± 20%
		{2, 24 * time.Minute, 36 * time.Minute},         // 30min ± 20%
		{3, 96 * time.Minute, 144 * time.Minute},        // 2h ± 20%
		{4, 576 * time.Minute, 864 * time.Minute},       // 12h ± 20%
		{10, 576 * time.Minute, 864 * time.Minute},      // beyond max stays at last
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			// Run multiple times to account for jitter
			for i := 0; i < 10; i++ {
				delay := NextRetryDelay(tt.attempt)
				if delay < tt.minDelay || delay > tt.maxDelay {
					t.Errorf("NextRetryDelay(%d) = %v, want between %v and %v",
						tt.attempt, delay, tt.minDelay, tt.maxDelay)
				}
			}
		})
	}
}

func TestNextRetryDelay_Negative(t *testing.T) {
	// Negative attempt should be treated as 0
	delay := NextRetryDelay(-1)
	if delay < 48*time.Second || delay > 72*time.Second {
		t.Errorf("NextRetryDelay(-1) should use attempt 0, got %v", delay)
	}
}

func TestIsExhausted(t *testing.T) {
	tests := []struct {
		attempt     int
		maxAttempts int
		want        bool
	}{
		{0, 5, false},
		{4, 5, false},
		{5, 5, true},
		{6, 5, true},
	}

	for _, tt := range tests {
		got := IsExhausted(tt.attempt, tt.maxAttempts)
		if got != tt.want {
			t.Errorf("IsExhausted(%d, %d) = %v, want %v",
				tt.attempt, tt.maxAttempts, got, tt.want)
		}
	}
}

func TestGetRetryDelays(t *testing.T) {
	delays := GetRetryDelays()
	if len(delays) != 5 {
		t.Errorf("expected 5 retry delays, got %d", len(delays))
	}
	
	// Verify delays are in increasing order
	for i := 1; i < len(delays); i++ {
		if delays[i] <= delays[i-1] {
			t.Errorf("delays should be increasing: %v <= %v", delays[i], delays[i-1])
		}
	}
}

func TestEstimatedMaxDeliveryWindow(t *testing.T) {
	window := EstimatedMaxDeliveryWindow()
	
	// Should be approximately 14h 36m + 20% ≈ 17.5h
	minExpected := 14 * time.Hour
	maxExpected := 20 * time.Hour
	
	if window < minExpected || window > maxExpected {
		t.Errorf("EstimatedMaxDeliveryWindow() = %v, want between %v and %v",
			window, minExpected, maxExpected)
	}
}
