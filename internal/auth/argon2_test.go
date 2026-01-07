package auth

import (
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "pk_live_abc123_secretsecretsecretsecret1234"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// Verify PHC format
	if !strings.HasPrefix(hash, "$argon2id$v=") {
		t.Errorf("Hash should be in PHC format, got: %s", hash)
	}

	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		t.Errorf("Hash should have 6 parts, got: %d", len(parts))
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "pk_live_abc123_secretsecretsecretsecret1234"
	wrongPassword := "pk_live_abc123_wrongwrongwrongwrongwrong1234"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// Correct password should verify
	match, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if !match {
		t.Error("Correct password should match")
	}

	// Wrong password should not verify
	match, err = VerifyPassword(wrongPassword, hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if match {
		t.Error("Wrong password should not match")
	}
}

func TestVerifyPassword_InvalidHash(t *testing.T) {
	testCases := []struct {
		name string
		hash string
	}{
		{"empty", ""},
		{"wrong format", "not-a-hash"},
		{"wrong algorithm", "$bcrypt$v=19$m=65536,t=3,p=4$salt$hash"},
		{"missing parts", "$argon2id$v=19$m=65536"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := VerifyPassword("password", tc.hash)
			if err == nil {
				t.Error("Expected error for invalid hash")
			}
		})
	}
}

func TestQuickHash(t *testing.T) {
	input := "pk_live_abc123_secretsecretsecretsecret1234"

	hash1 := QuickHash(input)
	hash2 := QuickHash(input)

	// Same input should produce same hash
	if hash1 != hash2 {
		t.Error("Same input should produce same hash")
	}

	// Hash should be 32 hex characters
	if len(hash1) != 32 {
		t.Errorf("Hash should be 32 chars, got: %d", len(hash1))
	}

	// Different input should produce different hash
	hash3 := QuickHash("different-input")
	if hash1 == hash3 {
		t.Error("Different input should produce different hash")
	}
}
