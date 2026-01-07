// Package auth provides authentication utilities for API keys.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
)

// Key format: pk_{env}_{prefix}_{secret}
// Example: pk_live_7a9x3k_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b
const (
	KeyPrefixLen = 6  // Visible prefix length (hex encoded 3 bytes)
	KeySecretLen = 32 // Secret length (hex encoded 16 bytes)
)

// Environment indicators for key prefix.
const (
	EnvLive = "live"
	EnvTest = "test"
)

var (
	// ErrInvalidKeyFormat indicates the key format is invalid.
	ErrInvalidKeyFormat = errors.New("invalid API key format")
	// keyFormatRegex validates the key format.
	keyFormatRegex = regexp.MustCompile(`^pk_(live|test)_([a-f0-9]{6})_([a-f0-9]{32})$`)
)

// GeneratedKey contains the parts of a newly generated API key.
type GeneratedKey struct {
	Plaintext string // Full key (show once only)
	Hash      string // Argon2id hash for storage
	Prefix    string // 6-char visible prefix
}

// GenerateAPIKey creates a new API key with the specified environment.
// Returns the plaintext key (to show once), hash (to store), and prefix (for lookup).
func GenerateAPIKey(env string) (*GeneratedKey, error) {
	if env != EnvLive && env != EnvTest {
		env = EnvLive // Default to live
	}

	// Generate 3-byte prefix (6 hex chars)
	prefixBytes := make([]byte, 3)
	if _, err := rand.Read(prefixBytes); err != nil {
		return nil, fmt.Errorf("generate prefix: %w", err)
	}
	prefix := hex.EncodeToString(prefixBytes)

	// Generate 16-byte secret (32 hex chars)
	secretBytes := make([]byte, 16)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, fmt.Errorf("generate secret: %w", err)
	}
	secret := hex.EncodeToString(secretBytes)

	// Assemble plaintext key
	plaintext := fmt.Sprintf("pk_%s_%s_%s", env, prefix, secret)

	// Hash for storage
	hash, err := HashPassword(plaintext)
	if err != nil {
		return nil, fmt.Errorf("hash key: %w", err)
	}

	return &GeneratedKey{
		Plaintext: plaintext,
		Hash:      hash,
		Prefix:    prefix,
	}, nil
}

// ParsedKey contains the parsed parts of an API key.
type ParsedKey struct {
	Env    string
	Prefix string
	Secret string
}

// ParseAPIKey extracts the components from a plaintext API key.
// Returns an error if the format is invalid.
func ParseAPIKey(key string) (*ParsedKey, error) {
	matches := keyFormatRegex.FindStringSubmatch(key)
	if matches == nil {
		return nil, ErrInvalidKeyFormat
	}

	return &ParsedKey{
		Env:    matches[1],
		Prefix: matches[2],
		Secret: matches[3],
	}, nil
}

// ValidateKeyFormat checks if the key matches the expected format.
func ValidateKeyFormat(key string) bool {
	return keyFormatRegex.MatchString(key)
}
