package cache

import (
	"testing"
)

func TestHashIP_Deterministic(t *testing.T) {
	t.Parallel()

	ip := "192.168.1.100"

	hash1 := hashIP(ip)
	hash2 := hashIP(ip)

	if hash1 != hash2 {
		t.Error("Same IP should produce same hash")
	}
}

func TestHashIP_Length(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ip   string
	}{
		{"IPv4", "192.168.1.1"},
		{"IPv4 localhost", "127.0.0.1"},
		{"IPv6 localhost", "::1"},
		{"IPv6 full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hash := hashIP(tt.ip)
			// hashIP uses first 8 bytes of SHA256, encoded as 16 hex chars
			if len(hash) != 16 {
				t.Errorf("hashIP(%q) length = %d, want 16", tt.ip, len(hash))
			}
		})
	}
}

func TestHashIP_Different(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ip1  string
		ip2  string
	}{
		{"different IPv4", "192.168.1.1", "192.168.1.2"},
		{"different last octet", "10.0.0.1", "10.0.0.2"},
		{"IPv4 vs IPv6", "127.0.0.1", "::1"},
		{"public vs private", "8.8.8.8", "192.168.1.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hash1 := hashIP(tt.ip1)
			hash2 := hashIP(tt.ip2)

			if hash1 == hash2 {
				t.Errorf("Different IPs should produce different hashes: %q and %q both produced %s", tt.ip1, tt.ip2, hash1)
			}
		})
	}
}

func TestExtractShortCodeFromClickKey_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{"simple", "clicks:abc123", "abc123"},
		{"with hyphen", "clicks:my-link", "my-link"},
		{"with underscore", "clicks:test_code", "test_code"},
		{"long code", "clicks:verylongshortcodehere123", "verylongshortcodehere123"},
		{"numbers only", "clicks:12345", "12345"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ExtractShortCodeFromClickKey(tt.key)
			if result != tt.expected {
				t.Errorf("ExtractShortCodeFromClickKey(%q) = %q, want %q", tt.key, result, tt.expected)
			}
		})
	}
}

func TestExtractShortCodeFromClickKey_Empty(t *testing.T) {
	t.Parallel()

	// Key with prefix but no content after colon
	result := ExtractShortCodeFromClickKey("clicks:")

	if result != "" {
		t.Errorf("ExtractShortCodeFromClickKey(\"clicks:\") = %q, want empty string", result)
	}
}

func TestExtractShortCodeFromClickKey_NoPrefix(t *testing.T) {
	t.Parallel()

	// The function doesn't validate the prefix, just returns substring after
	// the prefix length (7 chars for "clicks:"). Keys shorter than 7 chars
	// return empty string.
	tests := []struct {
		name string
		key  string
	}{
		{"empty string", ""},
		{"just colon", ":"},
		{"too short", "abc"},
		{"exactly 7 chars no prefix", "abcdefg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Keys with length <= 7 should return empty string
			result := ExtractShortCodeFromClickKey(tt.key)
			if len(tt.key) <= 7 && result != "" {
				t.Errorf("ExtractShortCodeFromClickKey(%q) = %q, want empty string for short key", tt.key, result)
			}
		})
	}
}
