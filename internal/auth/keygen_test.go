package auth

import (
	"strings"
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
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

	// Check hash is not empty
	if key.Hash == "" {
		t.Error("Hash should not be empty")
	}

	// Verify plaintext contains prefix
	if !strings.Contains(key.Plaintext, key.Prefix) {
		t.Error("Plaintext should contain prefix")
	}
}

func TestGenerateAPIKey_TestEnv(t *testing.T) {
	key, err := GenerateAPIKey(EnvTest)
	if err != nil {
		t.Fatalf("GenerateAPIKey failed: %v", err)
	}

	if !strings.HasPrefix(key.Plaintext, "pk_test_") {
		t.Errorf("Key should start with pk_test_, got: %s", key.Plaintext)
	}
}

func TestGenerateAPIKey_Uniqueness(t *testing.T) {
	key1, _ := GenerateAPIKey(EnvLive)
	key2, _ := GenerateAPIKey(EnvLive)

	if key1.Plaintext == key2.Plaintext {
		t.Error("Generated keys should be unique")
	}

	if key1.Prefix == key2.Prefix {
		// This could happen by chance, but very unlikely
		t.Log("Warning: prefixes are the same (very unlikely)")
	}
}

func TestParseAPIKey(t *testing.T) {
	testCases := []struct {
		name    string
		key     string
		wantEnv string
		wantErr bool
	}{
		{
			name:    "valid live key",
			key:     "pk_live_abc123_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b",
			wantEnv: "live",
			wantErr: false,
		},
		{
			name:    "valid test key",
			key:     "pk_test_def456_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b",
			wantEnv: "test",
			wantErr: false,
		},
		{
			name:    "wrong prefix",
			key:     "sk_live_abc123_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b",
			wantErr: true,
		},
		{
			name:    "wrong env",
			key:     "pk_prod_abc123_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b",
			wantErr: true,
		},
		{
			name:    "short prefix",
			key:     "pk_live_abc_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b",
			wantErr: true,
		},
		{
			name:    "short secret",
			key:     "pk_live_abc123_4f8d2e1b",
			wantErr: true,
		},
		{
			name:    "empty",
			key:     "",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := ParseAPIKey(tc.key)
			if tc.wantErr {
				if err == nil {
					t.Error("Expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if parsed.Env != tc.wantEnv {
				t.Errorf("Expected env %s, got %s", tc.wantEnv, parsed.Env)
			}
		})
	}
}

func TestValidateKeyFormat(t *testing.T) {
	valid := "pk_live_abc123_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b"
	invalid := "not-a-key"

	if !ValidateKeyFormat(valid) {
		t.Error("Should validate correct format")
	}

	if ValidateKeyFormat(invalid) {
		t.Error("Should reject invalid format")
	}
}
