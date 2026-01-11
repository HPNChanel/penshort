package webhook

import (
	"net"
	"testing"
)

func TestValidateTargetURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr error
	}{
		{
			name:    "valid https url",
			url:     "https://example.com/webhook",
			wantErr: nil,
		},
		{
			name:    "valid https with path",
			url:     "https://api.example.com/v1/webhooks",
			wantErr: nil,
		},
		{
			name:    "http not allowed",
			url:     "http://example.com/webhook",
			wantErr: ErrInvalidScheme,
		},
		{
			name:    "localhost blocked",
			url:     "https://localhost/webhook",
			wantErr: ErrLocalhostBlocked,
		},
		{
			name:    "127.0.0.1 blocked",
			url:     "https://127.0.0.1/webhook",
			wantErr: ErrLocalhostBlocked,
		},
		{
			name:    ".local domain blocked",
			url:     "https://myserver.local/webhook",
			wantErr: ErrLocalhostBlocked,
		},
		{
			name:    "non-standard port blocked",
			url:     "https://example.com:8443/webhook",
			wantErr: ErrInvalidPort,
		},
		{
			name:    "port 443 allowed",
			url:     "https://example.com:443/webhook",
			wantErr: nil,
		},
		{
			name:    "empty host",
			url:     "https:///webhook",
			wantErr: ErrEmptyHost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTargetURL(tt.url)
			if err != tt.wantErr {
				t.Errorf("ValidateTargetURL(%q) error = %v, want %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestIsBlockedIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		blocked bool
	}{
		{"private 10.x", "10.0.0.1", true},
		{"private 172.16.x", "172.16.0.1", true},
		{"private 192.168.x", "192.168.1.1", true},
		{"loopback", "127.0.0.1", true},
		{"link-local", "169.254.1.1", true},
		{"public IP", "8.8.8.8", false},
		{"public IP 2", "93.184.216.34", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP: %s", tt.ip)
			}
			got := isBlockedIP(ip)
			if got != tt.blocked {
				t.Errorf("isBlockedIP(%q) = %v, want %v", tt.ip, got, tt.blocked)
			}
		})
	}
}

func TestExtractHost(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/webhook", "example.com"},
		{"https://api.example.com:443/v1", "api.example.com:443"},
		{"invalid-url", ""}, // url.Parse is lenient, returns empty host for relative paths
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := ExtractHost(tt.url)
			if got != tt.want {
				t.Errorf("ExtractHost(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

