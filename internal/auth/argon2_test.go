package auth

import (
	"strings"
	"testing"
)

func TestHashPassword_Format(t *testing.T) {
	t.Parallel()

	password := "pk_live_abc123_secretsecretsecretsecret1234"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// Verify PHC format: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	if !strings.HasPrefix(hash, "$argon2id$v=") {
		t.Errorf("Hash should be in PHC format, got: %s", hash)
	}

	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		t.Errorf("Hash should have 6 parts, got: %d", len(parts))
	}

	// Verify parameters are correct
	if parts[1] != "argon2id" {
		t.Errorf("Expected argon2id algorithm, got: %s", parts[1])
	}
	if parts[2] != "v=19" {
		t.Errorf("Expected v=19, got: %s", parts[2])
	}
	if parts[3] != "m=65536,t=3,p=4" {
		t.Errorf("Expected m=65536,t=3,p=4, got: %s", parts[3])
	}
}

func TestHashPassword_Uniqueness(t *testing.T) {
	t.Parallel()

	password := "the_same_password_12345"

	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// Same password should produce different hashes (different salts)
	if hash1 == hash2 {
		t.Error("Same password should produce different hashes due to random salt")
	}

	// But both should be valid and verify correctly
	match1, _ := VerifyPassword(password, hash1)
	match2, _ := VerifyPassword(password, hash2)

	if !match1 || !match2 {
		t.Error("Both hashes should verify correctly")
	}
}

func TestVerifyPassword_Correct(t *testing.T) {
	t.Parallel()

	password := "pk_live_abc123_secretsecretsecretsecret1234"

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
}

func TestVerifyPassword_Incorrect(t *testing.T) {
	t.Parallel()

	password := "pk_live_abc123_secretsecretsecretsecret1234"
	wrongPassword := "pk_live_abc123_wrongwrongwrongwrongwrong1234"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// Wrong password should not verify (but no error)
	match, err := VerifyPassword(wrongPassword, hash)
	if err != nil {
		t.Fatalf("VerifyPassword should not return error for wrong password: %v", err)
	}
	if match {
		t.Error("Wrong password should not match")
	}
}

func TestVerifyPassword_InvalidHashFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		hash    string
		wantErr error
	}{
		{"empty", "", ErrInvalidHash},
		{"wrong format", "not-a-hash", ErrInvalidHash},
		{"wrong algorithm", "$bcrypt$v=19$m=65536,t=3,p=4$salt$hash", ErrInvalidHash},
		{"missing parts", "$argon2id$v=19$m=65536", ErrInvalidHash},
		{"wrong part count", "$argon2id$v=19", ErrInvalidHash},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := VerifyPassword("password", tt.hash)
			if err != tt.wantErr {
				t.Errorf("VerifyPassword with %q error = %v, want %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func TestVerifyPassword_WrongVersion(t *testing.T) {
	t.Parallel()

	// Construct a hash with v=18 instead of v=19
	// This simulates an incompatible argon2 version
	invalidVersionHash := "$argon2id$v=18$m=65536,t=3,p=4$c29tZXNhbHRoZXJl$c29tZWhhc2hoZXJl"

	match, err := VerifyPassword("password", invalidVersionHash)
	if err != ErrIncompatibleVersion {
		t.Errorf("Expected ErrIncompatibleVersion, got: %v", err)
	}
	if match {
		t.Error("Should not match with incompatible version")
	}
}

func TestQuickHash_Deterministic(t *testing.T) {
	t.Parallel()

	input := "pk_live_abc123_secretsecretsecretsecret1234"

	hash1 := QuickHash(input)
	hash2 := QuickHash(input)

	// Same input should produce same hash
	if hash1 != hash2 {
		t.Error("Same input should produce same hash")
	}
}

func TestQuickHash_Length(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"api key", "pk_live_abc123_secretsecretsecretsecret1234"},
		{"short string", "abc"},
		{"empty string", ""},
		{"long string", strings.Repeat("x", 1000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hash := QuickHash(tt.input)
			if len(hash) != 32 {
				t.Errorf("Hash should be 32 chars, got: %d", len(hash))
			}
		})
	}
}

func TestQuickHash_Different(t *testing.T) {
	t.Parallel()

	hash1 := QuickHash("input-one")
	hash2 := QuickHash("input-two")

	if hash1 == hash2 {
		t.Error("Different input should produce different hash")
	}
}
