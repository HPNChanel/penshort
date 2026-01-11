package middleware

import (
	"testing"
)

func TestValidateShortCode(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr error
	}{
		{
			name:    "empty is valid (auto-generate)",
			code:    "",
			wantErr: nil,
		},
		{
			name:    "valid short code",
			code:    "abc123",
			wantErr: nil,
		},
		{
			name:    "valid with hyphen",
			code:    "my-link",
			wantErr: nil,
		},
		{
			name:    "valid with underscore",
			code:    "my_link",
			wantErr: nil,
		},
		{
			name:    "too short",
			code:    "ab",
			wantErr: ErrShortCodeTooShort,
		},
		{
			name:    "too long",
			code:    "abcdefghijklmnopqrstuvwxyz0123456789",
			wantErr: ErrShortCodeTooLong,
		},
		{
			name:    "invalid characters",
			code:    "abc!@#",
			wantErr: ErrShortCodeInvalid,
		},
		{
			name:    "reserved alias - api",
			code:    "api",
			wantErr: ErrShortCodeReserved,
		},
		{
			name:    "reserved alias - admin (case insensitive)",
			code:    "Admin",
			wantErr: ErrShortCodeReserved,
		},
		{
			name:    "reserved alias - healthz",
			code:    "healthz",
			wantErr: ErrShortCodeReserved,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateShortCode(tt.code)
			if err != tt.wantErr {
				t.Errorf("ValidateShortCode(%q) = %v, want %v", tt.code, err, tt.wantErr)
			}
		})
	}
}

func TestValidateDestinationURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr error
	}{
		{
			name:    "valid https",
			url:     "https://example.com",
			wantErr: nil,
		},
		{
			name:    "valid http",
			url:     "http://example.com",
			wantErr: nil,
		},
		{
			name:    "valid with path",
			url:     "https://example.com/path/to/page",
			wantErr: nil,
		},
		{
			name:    "javascript scheme blocked",
			url:     "javascript:alert('xss')",
			wantErr: ErrDestinationInvalid,
		},
		{
			name:    "data scheme blocked",
			url:     "data:text/html,<h1>test</h1>",
			wantErr: ErrDestinationInvalid,
		},
		{
			name:    "file scheme blocked",
			url:     "file:///etc/passwd",
			wantErr: ErrDestinationInvalid,
		},
		{
			name:    "too long URL",
			url:     "https://example.com/" + string(make([]byte, 2100)),
			wantErr: ErrDestinationTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDestinationURL(tt.url)
			if err != tt.wantErr {
				t.Errorf("ValidateDestinationURL(%q) = %v, want %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestValidateAlias(t *testing.T) {
	tests := []struct {
		name    string
		alias   string
		wantErr error
	}{
		{
			name:    "empty is valid",
			alias:   "",
			wantErr: nil,
		},
		{
			name:    "ascii only is valid",
			alias:   "myalias123",
			wantErr: nil,
		},
		{
			name:    "unicode blocked",
			alias:   "аdmin", // Cyrillic 'а' instead of Latin 'a'
			wantErr: ErrAliasInvalidUnicode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAlias(tt.alias)
			if err != tt.wantErr {
				t.Errorf("ValidateAlias(%q) = %v, want %v", tt.alias, err, tt.wantErr)
			}
		})
	}
}
