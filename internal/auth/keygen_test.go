package auth

import (
	"strings"
	"testing"
)

func TestGenerateAPIKey_Live(t *testing.T) {
	t.Parallel()

	key, err := GenerateAPIKey(EnvLive)
	if err != nil {
		t.Fatalf("GenerateAPIKey failed: %v", err)
	}

	// Check plaintext format
	if !strings.HasPrefix(key.Plaintext, "pk_live_") {
		t.Errorf("Key should start with pk_live_, got: %s", key.Plaintext)
	}

	// Check prefix length
	if len(key.Prefix) != KeyPrefixLen {
		t.Errorf("Prefix should be %d chars, got: %d", KeyPrefixLen, len(key.Prefix))
	}

	// Check hash is not empty and in PHC format
	if key.Hash == "" {
		t.Error("Hash should not be empty")
	}
	if !strings.HasPrefix(key.Hash, "$argon2id$v=") {
		t.Errorf("Hash should be in PHC format, got: %s", key.Hash)
	}

	// Verify plaintext contains prefix
	if !strings.Contains(key.Plaintext, key.Prefix) {
		t.Error("Plaintext should contain prefix")
	}
}

func TestGenerateAPIKey_Test(t *testing.T) {
	t.Parallel()

	key, err := GenerateAPIKey(EnvTest)
	if err != nil {
		t.Fatalf("GenerateAPIKey failed: %v", err)
	}

	if !strings.HasPrefix(key.Plaintext, "pk_test_") {
		t.Errorf("Key should start with pk_test_, got: %s", key.Plaintext)
	}
}

func TestGenerateAPIKey_DefaultsToLive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  string
	}{
		{"invalid env", "invalid"},
		{"empty env", ""},
		{"prod env", "prod"},
		{"staging env", "staging"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			key, err := GenerateAPIKey(tt.env)
			if err != nil {
				t.Fatalf("GenerateAPIKey failed: %v", err)
			}
			if !strings.HasPrefix(key.Plaintext, "pk_live_") {
				t.Errorf("Expected pk_live_ prefix for env %q, got: %s", tt.env, key.Plaintext)
			}
		})
	}
}

func TestGenerateAPIKey_UniquePrefixes(t *testing.T) {
	t.Parallel()

	const numKeys = 100
	prefixes := make(map[string]bool, numKeys)

	for i := 0; i < numKeys; i++ {
		key, err := GenerateAPIKey(EnvLive)
		if err != nil {
			t.Fatalf("GenerateAPIKey failed: %v", err)
		}

		if prefixes[key.Prefix] {
			t.Errorf("Duplicate prefix found: %s (iteration %d)", key.Prefix, i)
		}
		prefixes[key.Prefix] = true
	}

	// Verify all prefixes are unique (high probability)
	if len(prefixes) != numKeys {
		t.Errorf("Expected %d unique prefixes, got %d", numKeys, len(prefixes))
	}
}

func TestGenerateAPIKey_UniqueSecrets(t *testing.T) {
	t.Parallel()

	const numKeys = 100
	secrets := make(map[string]bool, numKeys)

	for i := 0; i < numKeys; i++ {
		key, err := GenerateAPIKey(EnvLive)
		if err != nil {
			t.Fatalf("GenerateAPIKey failed: %v", err)
		}

		// Extract secret from plaintext (last 32 chars after final underscore)
		parts := strings.Split(key.Plaintext, "_")
		if len(parts) != 4 {
			t.Fatalf("Expected 4 parts in key, got %d", len(parts))
		}
		secret := parts[3]

		if secrets[secret] {
			t.Errorf("Duplicate secret found at iteration %d", i)
		}
		secrets[secret] = true
	}

	if len(secrets) != numKeys {
		t.Errorf("Expected %d unique secrets, got %d", numKeys, len(secrets))
	}
}

func TestParseAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		key        string
		wantEnv    string
		wantPrefix string
		wantErr    error
	}{
		{
			name:       "valid live key",
			key:        "pk_live_abc123_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b",
			wantEnv:    "live",
			wantPrefix: "abc123",
			wantErr:    nil,
		},
		{
			name:       "valid test key",
			key:        "pk_test_def456_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b",
			wantEnv:    "test",
			wantPrefix: "def456",
			wantErr:    nil,
		},
		{
			name:    "wrong prefix sk_",
			key:     "sk_live_abc123_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b",
			wantErr: ErrInvalidKeyFormat,
		},
		{
			name:    "wrong env prod",
			key:     "pk_prod_abc123_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b",
			wantErr: ErrInvalidKeyFormat,
		},
		{
			name:    "short prefix 3 chars",
			key:     "pk_live_abc_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b",
			wantErr: ErrInvalidKeyFormat,
		},
		{
			name:    "short secret 8 chars",
			key:     "pk_live_abc123_4f8d2e1b",
			wantErr: ErrInvalidKeyFormat,
		},
		{
			name:    "long secret 33 chars",
			key:     "pk_live_abc123_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1bx",
			wantErr: ErrInvalidKeyFormat,
		},
		{
			name:    "empty string",
			key:     "",
			wantErr: ErrInvalidKeyFormat,
		},
		{
			name:    "just invalid",
			key:     "invalid",
			wantErr: ErrInvalidKeyFormat,
		},
		{
			name:    "missing parts pk_abc",
			key:     "pk_abc",
			wantErr: ErrInvalidKeyFormat,
		},
		{
			name:    "pk_live_ only",
			key:     "pk_live_",
			wantErr: ErrInvalidKeyFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parsed, err := ParseAPIKey(tt.key)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("ParseAPIKey(%q) error = %v, want %v", tt.key, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseAPIKey(%q) unexpected error: %v", tt.key, err)
			}

			if parsed.Env != tt.wantEnv {
				t.Errorf("Env = %s, want %s", parsed.Env, tt.wantEnv)
			}

			if parsed.Prefix != tt.wantPrefix {
				t.Errorf("Prefix = %s, want %s", parsed.Prefix, tt.wantPrefix)
			}
		})
	}
}

func TestValidateKeyFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"valid live key", "pk_live_abc123_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b", true},
		{"valid test key", "pk_test_def456_0123456789abcdef0123456789abcdef", true},
		{"not a key", "not-a-key", false},
		{"wrong prefix", "sk_live_abc123_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b", false},
		{"empty", "", false},
		{"uppercase hex", "pk_live_ABC123_4F8D2E1B9C7A5F3D2E1B9C7A5F3D2E1B", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ValidateKeyFormat(tt.key)
			if got != tt.want {
				t.Errorf("ValidateKeyFormat(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}
