package middleware

import (
	"strings"
	"testing"
)

func TestValidateShortCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		code    string
		wantErr error
	}{
		// Valid cases
		{"empty is valid (auto-generate)", "", nil},
		{"valid short code", "abc123", nil},
		{"valid with hyphen", "my-link", nil},
		{"valid with underscore", "test_code", nil},
		{"valid mixed case", "MyLink123", nil},
		{"exactly 3 chars", "abc", nil},
		{"exactly 32 chars", "abcdefghijklmnopqrstuvwxyz012345", nil},

		// Too short/long
		{"too short 2 chars", "ab", ErrShortCodeTooShort},
		{"too short 1 char", "a", ErrShortCodeTooShort},
		{"too long 33 chars", "abcdefghijklmnopqrstuvwxyz0123456", ErrShortCodeTooLong},
		{"too long 36 chars", "abcdefghijklmnopqrstuvwxyz0123456789", ErrShortCodeTooLong},

		// Invalid characters
		{"space in middle", "hello world", ErrShortCodeInvalid},
		{"at sign", "test@code", ErrShortCodeInvalid},
		{"special chars", "abc!@#", ErrShortCodeInvalid},
		{"unicode chinese", "中文", ErrShortCodeInvalid},
		{"unicode japanese", "アプリ", ErrShortCodeInvalid},
		{"unicode emoji", "test🎉", ErrShortCodeInvalid},
		{"period", "my.link", ErrShortCodeInvalid},
		{"slash", "my/link", ErrShortCodeInvalid},

		// Reserved aliases (case-insensitive)
		{"reserved api lowercase", "api", ErrShortCodeReserved},
		{"reserved API uppercase", "API", ErrShortCodeReserved},
		{"reserved Admin mixed", "Admin", ErrShortCodeReserved},
		{"reserved healthz", "healthz", ErrShortCodeReserved},
		{"reserved HEALTHZ", "HEALTHZ", ErrShortCodeReserved},
		{"reserved login", "login", ErrShortCodeReserved},
		{"reserved webhook", "webhook", ErrShortCodeReserved},
		{"reserved metrics", "metrics", ErrShortCodeReserved},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateShortCode(tt.code)
			if err != tt.wantErr {
				t.Errorf("ValidateShortCode(%q) = %v, want %v", tt.code, err, tt.wantErr)
			}
		})
	}
}

func TestValidateDestinationURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		wantErr error
	}{
		// Valid URLs
		{"valid https", "https://example.com", nil},
		{"valid http", "http://example.com", nil},
		{"valid with path", "https://example.com/path/to/page", nil},
		{"valid with query", "https://example.com?foo=bar", nil},
		{"valid localhost http", "http://localhost:8080", nil},
		{"valid with port", "https://example.com:8443/api", nil},

		// Invalid schemes
		{"javascript scheme", "javascript:alert('xss')", ErrDestinationInvalid},
		{"data scheme", "data:text/html,<h1>test</h1>", ErrDestinationInvalid},
		{"file scheme", "file:///etc/passwd", ErrDestinationInvalid},
		{"ftp scheme", "ftp://ftp.example.com", ErrDestinationInvalid},
		{"mailto scheme", "mailto:test@example.com", ErrDestinationInvalid},
		{"no scheme", "example.com", ErrDestinationInvalid},

		// Dangerous schemes in URL parts (XSS payloads)
		{"javascript in query", "https://example.com?redirect=javascript:alert(1)", ErrDestinationUnsafe},
		{"data in path", "https://example.com/data:text/html", ErrDestinationUnsafe},
		{"vbscript in url", "https://example.com?x=vbscript:msgbox", ErrDestinationUnsafe},
		{"file in query", "https://example.com?path=file:///etc/passwd", ErrDestinationUnsafe},

		// Too long
		{"too long URL", "https://example.com/" + strings.Repeat("x", 2100), ErrDestinationTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateDestinationURL(tt.url)
			if err != tt.wantErr {
				t.Errorf("ValidateDestinationURL(%q) = %v, want %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestValidateWebhookURL(t *testing.T) {
	t.Parallel()

	// "https://example.com/" is 20 chars, so we need 1004 more for 1024 total
	tests := []struct {
		name    string
		url     string
		wantErr error
	}{
		{"valid webhook url", "https://webhook.example.com/callback", nil},
		{"valid with path", "https://api.example.com/v1/webhooks", nil},
		{"exactly 1024 chars", "https://example.com/" + strings.Repeat("x", 1004), nil},
		{"too long 1025 chars", "https://example.com/" + strings.Repeat("x", 1005), ErrWebhookURLTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateWebhookURL(tt.url)
			if err != tt.wantErr {
				t.Errorf("ValidateWebhookURL (len=%d) = %v, want %v", len(tt.url), err, tt.wantErr)
			}
		})
	}
}

func TestValidateAlias(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		alias   string
		wantErr error
	}{
		// Valid cases
		{"empty is valid", "", nil},
		{"ascii only lowercase", "myalias", nil},
		{"ascii with numbers", "myalias123", nil},
		{"ascii with hyphen", "my-alias", nil},
		{"ascii all lower", "abcdefghij", nil},

		// Unicode blocked
		{"cyrillic a instead of latin", "аdmin", ErrAliasInvalidUnicode}, // Cyrillic 'а'
		{"cyrillic е in test", "tеst", ErrAliasInvalidUnicode},           // Cyrillic 'е'
		{"japanese katakana", "アプリ", ErrAliasInvalidUnicode},
		{"chinese characters", "链接", ErrAliasInvalidUnicode},
		{"mixed script", "hellΦworld", ErrAliasInvalidUnicode},
		{"accented latin", "hëllo", ErrAliasInvalidUnicode},

		// Confusable overload (>50% confusable chars for len > 3)
		{"confusable overload", "0O0l1I1l", ErrAliasInvalidUnicode},
		{"mostly zeros and ohs", "00OO00", ErrAliasInvalidUnicode},
		{"mostly ones and els", "1l1l1l", ErrAliasInvalidUnicode},

		// Some confusables are OK if < 50%
		{"some confusables OK", "hello0", nil},
		{"mixed normal", "test123", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateAlias(tt.alias)
			if err != tt.wantErr {
				t.Errorf("ValidateAlias(%q) = %v, want %v", tt.alias, err, tt.wantErr)
			}
		})
	}
}

func TestValidateShortCode_BoundaryConditions(t *testing.T) {
	t.Parallel()

	// Test exact boundary lengths
	exactMin := strings.Repeat("a", MinShortCodeLength)
	exactMax := strings.Repeat("a", MaxShortCodeLength)
	belowMin := strings.Repeat("a", MinShortCodeLength-1)
	aboveMax := strings.Repeat("a", MaxShortCodeLength+1)

	tests := []struct {
		name    string
		code    string
		wantErr error
	}{
		{"exact min length", exactMin, nil},
		{"exact max length", exactMax, nil},
		{"below min length", belowMin, ErrShortCodeTooShort},
		{"above max length", aboveMax, ErrShortCodeTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateShortCode(tt.code)
			if err != tt.wantErr {
				t.Errorf("ValidateShortCode (len=%d) = %v, want %v", len(tt.code), err, tt.wantErr)
			}
		})
	}
}

