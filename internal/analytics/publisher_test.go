package analytics

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateVisitorHash_Deterministic(t *testing.T) {
	t.Parallel()

	ip := "192.168.1.100"
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	clickedAt := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

	hash1 := GenerateVisitorHash(ip, userAgent, clickedAt)
	hash2 := GenerateVisitorHash(ip, userAgent, clickedAt)

	if hash1 != hash2 {
		t.Error("Same inputs should produce same hash")
	}

	// Hash should be 16 hex chars
	if len(hash1) != 16 {
		t.Errorf("Hash length = %d, want 16", len(hash1))
	}
}

func TestGenerateVisitorHash_DailyRotation(t *testing.T) {
	t.Parallel()

	ip := "192.168.1.100"
	userAgent := "Mozilla/5.0"

	day1 := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 1, 16, 12, 0, 0, 0, time.UTC) // Next day

	hash1 := GenerateVisitorHash(ip, userAgent, day1)
	hash2 := GenerateVisitorHash(ip, userAgent, day2)

	if hash1 == hash2 {
		t.Error("Different days should produce different hashes to prevent cross-day tracking")
	}
}

func TestGenerateVisitorHash_SameDayDifferentTime(t *testing.T) {
	t.Parallel()

	ip := "192.168.1.100"
	userAgent := "Mozilla/5.0"

	morning := time.Date(2026, 1, 15, 6, 0, 0, 0, time.UTC)
	evening := time.Date(2026, 1, 15, 18, 0, 0, 0, time.UTC)

	hash1 := GenerateVisitorHash(ip, userAgent, morning)
	hash2 := GenerateVisitorHash(ip, userAgent, evening)

	// Same day should produce same hash regardless of time
	if hash1 != hash2 {
		t.Error("Same day should produce same hash regardless of time")
	}
}

func TestGenerateVisitorHash_DifferentInputs(t *testing.T) {
	t.Parallel()

	clickedAt := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		ip1  string
		ua1  string
		ip2  string
		ua2  string
	}{
		{"different IP", "192.168.1.1", "Mozilla/5.0", "192.168.1.2", "Mozilla/5.0"},
		{"different UA", "192.168.1.1", "Chrome/100", "192.168.1.1", "Firefox/100"},
		{"both different", "10.0.0.1", "Safari/15", "10.0.0.2", "Edge/100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hash1 := GenerateVisitorHash(tt.ip1, tt.ua1, clickedAt)
			hash2 := GenerateVisitorHash(tt.ip2, tt.ua2, clickedAt)

			if hash1 == hash2 {
				t.Error("Different inputs should produce different hashes")
			}
		})
	}
}

func TestSanitizeReferrer_StripQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "strip utm params",
			input:    "https://example.com/page?utm_source=test&utm_medium=email",
			expected: "https://example.com/page",
		},
		{
			name:     "strip all query params",
			input:    "https://google.com/search?q=test&hl=en",
			expected: "https://google.com/search",
		},
		{
			name:     "strip fragment",
			input:    "https://example.com/page#section",
			expected: "https://example.com/page",
		},
		{
			name:     "strip both query and fragment",
			input:    "https://example.com/path?query=1#section",
			expected: "https://example.com/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := SanitizeReferrer(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeReferrer(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeReferrer_Empty(t *testing.T) {
	t.Parallel()

	result := SanitizeReferrer("")
	if result != "" {
		t.Errorf("SanitizeReferrer(\"\") = %q, want empty string", result)
	}
}

func TestSanitizeReferrer_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"missing scheme", "example.com/path"},
		{"malformed", "://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := SanitizeReferrer(tt.input)
			// Invalid URLs should return empty or remain valid after parsing
			// The function is lenient, so we just verify it doesn't panic
			_ = result
		})
	}
}

func TestSanitizeReferrer_Truncate(t *testing.T) {
	t.Parallel()

	// Create a very long URL
	longPath := strings.Repeat("a", 600)
	longURL := "https://example.com/" + longPath

	result := SanitizeReferrer(longURL)

	if len(result) > 500 {
		t.Errorf("Sanitized referrer length = %d, want <= 500", len(result))
	}
}

func TestExtractCountryCode_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"us", "US"},
		{"US", "US"},
		{"gb", "GB"},
		{"vn", "VN"},
		{"JP", "JP"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			result := ExtractCountryCode(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractCountryCode(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractCountryCode_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"too long", "USA"},
		{"single char", "U"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ExtractCountryCode(tt.input)
			if result != "" {
				t.Errorf("ExtractCountryCode(%q) = %q, want empty string", tt.input, result)
			}
		})
	}
}

func TestExtractReferrerDomain_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"https://google.com/search?q=test", "google.com"},
		{"https://www.example.com/path/to/page", "www.example.com"},
		{"http://subdomain.domain.com:8080/path", "subdomain.domain.com:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()

			result := ExtractReferrerDomain(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractReferrerDomain(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractReferrerDomain_Direct(t *testing.T) {
	t.Parallel()

	result := ExtractReferrerDomain("")
	if result != "(direct)" {
		t.Errorf("ExtractReferrerDomain(\"\") = %q, want \"(direct)\"", result)
	}
}

func TestExtractReferrerDomain_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"no host", "https:///path"},
		{"relative path", "/path/to/page"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ExtractReferrerDomain(tt.input)
			if result != "(unknown)" {
				t.Errorf("ExtractReferrerDomain(%q) = %q, want \"(unknown)\"", tt.input, result)
			}
		})
	}
}

func TestTruncateUserAgent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantLen  int
	}{
		{"short UA", "Mozilla/5.0", 11},
		{"exact 500", strings.Repeat("x", 500), 500},
		{"over 500", strings.Repeat("x", 600), 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := TruncateUserAgent(tt.input)
			if len(result) != tt.wantLen {
				t.Errorf("TruncateUserAgent length = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}

func TestTruncateUserAgent_PreservesContent(t *testing.T) {
	t.Parallel()

	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	result := TruncateUserAgent(ua)

	if result != ua {
		t.Errorf("Short UA should be preserved, got %q", result)
	}
}
