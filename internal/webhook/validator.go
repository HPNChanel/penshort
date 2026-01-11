package webhook

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

var (
	// ErrInvalidScheme is returned when URL scheme is not HTTPS.
	ErrInvalidScheme = errors.New("only HTTPS allowed")
	// ErrPrivateIP is returned when URL resolves to private IP.
	ErrPrivateIP = errors.New("private IP addresses not allowed")
	// ErrLocalhostBlocked is returned when localhost is used.
	ErrLocalhostBlocked = errors.New("localhost not allowed")
	// ErrInvalidPort is returned when non-standard port is used.
	ErrInvalidPort = errors.New("only port 443 allowed")
	// ErrInvalidURL is returned when URL parsing fails.
	ErrInvalidURL = errors.New("invalid URL format")
	// ErrEmptyHost is returned when URL has no host.
	ErrEmptyHost = errors.New("URL must have a host")
)

// BlockedCIDRs contains private/internal IP ranges.
var BlockedCIDRs = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"127.0.0.0/8",
	"169.254.0.0/16", // Link-local
	"0.0.0.0/8",      // This network
	"::1/128",        // IPv6 loopback
	"fc00::/7",       // IPv6 private
	"fe80::/10",      // IPv6 link-local
}

var blockedNetworks []*net.IPNet

func init() {
	for _, cidr := range BlockedCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil {
			blockedNetworks = append(blockedNetworks, network)
		}
	}
}

// ValidateTargetURL checks webhook URL for security issues.
// It enforces HTTPS, blocks private IPs, and prevents SSRF attacks.
func ValidateTargetURL(targetURL string) error {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return ErrInvalidURL
	}

	// 1. HTTPS only
	if parsed.Scheme != "https" {
		return ErrInvalidScheme
	}

	// 2. Must have a host
	host := parsed.Hostname()
	if host == "" {
		return ErrEmptyHost
	}

	// 3. No localhost/loopback hostnames
	if isLocalhostHostname(host) {
		return ErrLocalhostBlocked
	}

	// 4. Resolve and check against blocked CIDRs
	ips, err := net.LookupIP(host)
	if err != nil {
		// DNS resolution failed - could be invalid domain
		// Allow this to fail at delivery time instead
		return nil
	}

	for _, ip := range ips {
		if isBlockedIP(ip) {
			return ErrPrivateIP
		}
	}

	// 5. Default port only (443 for HTTPS)
	if port := parsed.Port(); port != "" && port != "443" {
		return ErrInvalidPort
	}

	return nil
}

// isLocalhostHostname checks if hostname is localhost variant.
func isLocalhostHostname(host string) bool {
	host = strings.ToLower(host)
	return host == "localhost" ||
		strings.HasSuffix(host, ".localhost") ||
		strings.HasSuffix(host, ".local") ||
		host == "127.0.0.1" ||
		host == "::1"
}

// isBlockedIP checks if IP is in any blocked CIDR range.
func isBlockedIP(ip net.IP) bool {
	for _, network := range blockedNetworks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// ExtractHost extracts host from URL for safe logging.
// Never log full URLs as they may contain secrets in path/query.
func ExtractHost(targetURL string) string {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return "(invalid)"
	}
	return parsed.Host
}
