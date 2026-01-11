// Package middleware provides HTTP middleware for the Penshort API.
package middleware

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

// Validation limits.
const (
	// MaxShortCodeLength is the maximum length for a short code.
	MaxShortCodeLength = 32

	// MinShortCodeLength is the minimum length for a custom alias.
	MinShortCodeLength = 3

	// MaxDestinationURLLength is the maximum length for destination URLs.
	MaxDestinationURLLength = 2048

	// MaxWebhookURLLength is the maximum length for webhook URLs.
	MaxWebhookURLLength = 1024

	// MaxAliasLength is the maximum length for custom aliases.
	MaxAliasLength = 32
)

// Validation errors.
var (
	ErrShortCodeTooLong    = errors.New("short code exceeds maximum length")
	ErrShortCodeTooShort   = errors.New("short code is too short")
	ErrShortCodeInvalid    = errors.New("short code contains invalid characters")
	ErrShortCodeReserved   = errors.New("short code is reserved")
	ErrDestinationTooLong  = errors.New("destination URL exceeds maximum length")
	ErrDestinationInvalid  = errors.New("destination URL is invalid")
	ErrDestinationUnsafe   = errors.New("destination URL uses unsafe scheme")
	ErrWebhookURLTooLong   = errors.New("webhook URL exceeds maximum length")
	ErrAliasInvalidUnicode = errors.New("alias contains confusable unicode characters")
)

// ReservedAliases contains short codes that cannot be used as custom aliases.
// These are reserved for system routes and common paths.
var ReservedAliases = map[string]bool{
	// System routes
	"api":        true,
	"admin":      true,
	"healthz":    true,
	"readyz":     true,
	"metrics":    true,
	"static":     true,
	"assets":     true,
	"public":     true,
	"private":    true,

	// Common paths attackers might try
	"login":      true,
	"logout":     true,
	"auth":       true,
	"oauth":      true,
	"callback":   true,
	"webhook":    true,
	"webhooks":   true,

	// Brand protection (add your brand here)
	"penshort":   true,
	"pen":        true,

	// Common abuse targets
	"password":   true,
	"reset":      true,
	"verify":     true,
	"confirm":    true,
	"activate":   true,
	"unsubscribe": true,

	// Common file extensions
	"robots":     true,
	"sitemap":    true,
	"favicon":    true,
	"well-known": true,
}

// validShortCodePattern matches valid short code characters.
// Allowed: a-z, A-Z, 0-9, hyphen, underscore
var validShortCodePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ValidateShortCode validates a short code for use as a custom alias.
func ValidateShortCode(code string) error {
	if code == "" {
		return nil // Empty is valid (will be auto-generated)
	}

	if len(code) > MaxShortCodeLength {
		return ErrShortCodeTooLong
	}

	if len(code) < MinShortCodeLength {
		return ErrShortCodeTooShort
	}

	if !validShortCodePattern.MatchString(code) {
		return ErrShortCodeInvalid
	}

	// Check reserved aliases (case-insensitive)
	if ReservedAliases[strings.ToLower(code)] {
		return ErrShortCodeReserved
	}

	return nil
}

// ValidateDestinationURL validates a destination URL for link creation.
func ValidateDestinationURL(url string) error {
	if len(url) > MaxDestinationURLLength {
		return ErrDestinationTooLong
	}

	// Basic scheme validation
	lowerURL := strings.ToLower(url)
	if !strings.HasPrefix(lowerURL, "http://") && !strings.HasPrefix(lowerURL, "https://") {
		return ErrDestinationInvalid
	}

	// Block dangerous schemes (in case of URL encoding tricks)
	forbiddenSchemes := []string{"javascript:", "data:", "vbscript:", "file:"}
	for _, scheme := range forbiddenSchemes {
		if strings.Contains(lowerURL, scheme) {
			return ErrDestinationUnsafe
		}
	}

	return nil
}

// ValidateWebhookURL validates a webhook target URL.
func ValidateWebhookURL(url string) error {
	if len(url) > MaxWebhookURLLength {
		return ErrWebhookURLTooLong
	}

	// Additional validation is done in webhook.ValidateTargetURL
	return nil
}

// ValidateAlias checks for unicode normalization issues.
// Prevents homograph attacks using lookalike characters.
func ValidateAlias(alias string) error {
	if alias == "" {
		return nil
	}

	// Check for any non-ASCII characters
	for _, r := range alias {
		if r > unicode.MaxASCII {
			// For now, reject all non-ASCII to prevent homograph attacks
			// This is strict but safe; can be relaxed with proper normalization
			return ErrAliasInvalidUnicode
		}
	}

	// Check for confusable ASCII patterns
	// These are common substitutions in phishing
	confusables := map[string]bool{
		"0":  true, // Can look like 'O' or 'o'
		"1":  true, // Can look like 'l' or 'I'
		"l":  true, // Can look like '1' or 'I'
		"I":  true, // Can look like '1' or 'l'
		"O":  true, // Can look like '0'
	}

	// Count confusable characters - too many is suspicious
	confusableCount := 0
	for _, r := range alias {
		if confusables[string(r)] {
			confusableCount++
		}
	}

	// If more than 50% of characters are confusable, reject
	if len(alias) > 3 && float64(confusableCount)/float64(len(alias)) > 0.5 {
		return ErrAliasInvalidUnicode
	}

	return nil
}
