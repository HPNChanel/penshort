package webhook

import (
	"testing"
	"time"
)

func TestGenerateSignature_Format(t *testing.T) {
	t.Parallel()

	sig := GenerateSignature("secret", 1736600000, []byte(`{"event":"test"}`))

	// Signature should be hex-encoded (64 chars for SHA256)
	if len(sig) != 64 {
		t.Errorf("signature length = %d, want 64", len(sig))
	}
}

func TestGenerateSignature_Deterministic(t *testing.T) {
	t.Parallel()

	secret := "whsec_test123"
	timestamp := int64(1736600000)
	payload := []byte(`{"event_type":"click","event_id":"123"}`)

	sig1 := GenerateSignature(secret, timestamp, payload)
	sig2 := GenerateSignature(secret, timestamp, payload)

	if sig1 != sig2 {
		t.Error("signature is not deterministic")
	}
}

func TestGenerateSignature_DifferentSecrets(t *testing.T) {
	t.Parallel()

	timestamp := int64(1736600000)
	payload := []byte(`{"event":"test"}`)

	sig1 := GenerateSignature("secret1", timestamp, payload)
	sig2 := GenerateSignature("secret2", timestamp, payload)

	if sig1 == sig2 {
		t.Error("different secret should produce different signature")
	}
}

func TestGenerateSignature_DifferentTimestamps(t *testing.T) {
	t.Parallel()

	secret := "whsec_test123"
	payload := []byte(`{"event":"test"}`)

	sig1 := GenerateSignature(secret, 1736600000, payload)
	sig2 := GenerateSignature(secret, 1736600001, payload)

	if sig1 == sig2 {
		t.Error("different timestamp should produce different signature")
	}
}

func TestGenerateSignature_DifferentPayloads(t *testing.T) {
	t.Parallel()

	secret := "whsec_test123"
	timestamp := int64(1736600000)

	sig1 := GenerateSignature(secret, timestamp, []byte(`{"event":"click"}`))
	sig2 := GenerateSignature(secret, timestamp, []byte(`{"event":"view"}`))

	if sig1 == sig2 {
		t.Error("different payload should produce different signature")
	}
}

func TestGenerateSignature_EmptyPayload(t *testing.T) {
	t.Parallel()

	// Empty payload should still produce a valid signature (not panic)
	sig := GenerateSignature("secret", 1736600000, []byte{})

	if len(sig) != 64 {
		t.Errorf("signature length for empty payload = %d, want 64", len(sig))
	}

	// Empty json object should also work
	sig2 := GenerateSignature("secret", 1736600000, []byte(`{}`))
	if len(sig2) != 64 {
		t.Errorf("signature length for {} payload = %d, want 64", len(sig2))
	}
}

func TestValidateSignature_Valid(t *testing.T) {
	t.Parallel()

	secret := "test_secret"
	timestamp := time.Now().Unix()
	payload := []byte(`{"test":"data"}`)

	validSig := GenerateSignature(secret, timestamp, payload)

	err := ValidateSignature(secret, validSig, timestamp, payload, 5*time.Minute)
	if err != nil {
		t.Errorf("ValidateSignature() error = %v, want nil", err)
	}
}

func TestValidateSignature_WrongSignature(t *testing.T) {
	t.Parallel()

	secret := "test_secret"
	timestamp := time.Now().Unix()
	payload := []byte(`{"test":"data"}`)

	err := ValidateSignature(secret, "invalid", timestamp, payload, 5*time.Minute)
	if err != ErrInvalidSignature {
		t.Errorf("ValidateSignature() error = %v, want ErrInvalidSignature", err)
	}
}

func TestValidateSignature_ReplayOld(t *testing.T) {
	t.Parallel()

	secret := "test_secret"
	oldTimestamp := time.Now().Add(-10 * time.Minute).Unix()
	payload := []byte(`{"test":"data"}`)
	sig := GenerateSignature(secret, oldTimestamp, payload)

	err := ValidateSignature(secret, sig, oldTimestamp, payload, 5*time.Minute)
	if err != ErrReplayWindowExceeded {
		t.Errorf("ValidateSignature() error = %v, want ErrReplayWindowExceeded", err)
	}
}

func TestValidateSignature_ReplayFuture(t *testing.T) {
	t.Parallel()

	secret := "test_secret"
	futureTimestamp := time.Now().Add(10 * time.Minute).Unix()
	payload := []byte(`{"test":"data"}`)
	sig := GenerateSignature(secret, futureTimestamp, payload)

	err := ValidateSignature(secret, sig, futureTimestamp, payload, 5*time.Minute)
	if err != ErrReplayWindowExceeded {
		t.Errorf("ValidateSignature() error = %v, want ErrReplayWindowExceeded", err)
	}
}

func TestValidateSignature_EdgeOfWindow(t *testing.T) {
	t.Parallel()

	secret := "test_secret"
	payload := []byte(`{"test":"data"}`)
	window := 5 * time.Minute

	// Timestamp exactly at 5 minute boundary (just inside)
	// Use 4 minutes 59 seconds ago to be safely inside window
	edgeTimestamp := time.Now().Add(-4*time.Minute - 59*time.Second).Unix()
	sig := GenerateSignature(secret, edgeTimestamp, payload)

	err := ValidateSignature(secret, sig, edgeTimestamp, payload, window)
	if err != nil {
		t.Errorf("ValidateSignature at edge of window error = %v, want nil", err)
	}
}

func TestValidateSignature_CustomWindow(t *testing.T) {
	t.Parallel()

	secret := "test_secret"
	payload := []byte(`{"test":"data"}`)

	// Use 1 minute window
	customWindow := 1 * time.Minute

	// 30 seconds ago should be valid
	recentTimestamp := time.Now().Add(-30 * time.Second).Unix()
	sig := GenerateSignature(secret, recentTimestamp, payload)

	err := ValidateSignature(secret, sig, recentTimestamp, payload, customWindow)
	if err != nil {
		t.Errorf("ValidateSignature with custom window error = %v, want nil", err)
	}

	// 2 minutes ago should be invalid with 1 min window
	oldTimestamp := time.Now().Add(-2 * time.Minute).Unix()
	oldSig := GenerateSignature(secret, oldTimestamp, payload)

	err = ValidateSignature(secret, oldSig, oldTimestamp, payload, customWindow)
	if err != ErrReplayWindowExceeded {
		t.Errorf("ValidateSignature with expired custom window error = %v, want ErrReplayWindowExceeded", err)
	}
}

func TestHashSecret_Format(t *testing.T) {
	t.Parallel()

	hash := HashSecret("my_secret_key")

	// Should be hex-encoded SHA256 (64 chars)
	if len(hash) != 64 {
		t.Errorf("hash length = %d, want 64", len(hash))
	}
}

func TestHashSecret_Deterministic(t *testing.T) {
	t.Parallel()

	secret := "my_secret_key"
	hash1 := HashSecret(secret)
	hash2 := HashSecret(secret)

	if hash1 != hash2 {
		t.Error("hash is not deterministic")
	}
}

func TestHashSecret_Different(t *testing.T) {
	t.Parallel()

	hash1 := HashSecret("secret1")
	hash2 := HashSecret("secret2")

	if hash1 == hash2 {
		t.Error("different input should produce different hash")
	}
}

func TestGenerateSecret_Length(t *testing.T) {
	t.Parallel()

	secret, err := GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret() error = %v", err)
	}

	// Should be 64 hex chars (32 bytes = 256 bits)
	if len(secret) != 64 {
		t.Errorf("secret length = %d, want 64", len(secret))
	}
}

func TestGenerateSecret_Uniqueness(t *testing.T) {
	t.Parallel()

	const numSecrets = 100
	secrets := make(map[string]bool, numSecrets)

	for i := 0; i < numSecrets; i++ {
		secret, err := GenerateSecret()
		if err != nil {
			t.Fatalf("GenerateSecret() error = %v", err)
		}

		if secrets[secret] {
			t.Errorf("Duplicate secret found at iteration %d", i)
		}
		secrets[secret] = true
	}

	if len(secrets) != numSecrets {
		t.Errorf("Expected %d unique secrets, got %d", numSecrets, len(secrets))
	}
}

func TestGenerateSecret_Entropy(t *testing.T) {
	t.Parallel()

	secret, err := GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret() error = %v", err)
	}

	// 64 hex chars = 32 bytes = 256 bits of entropy
	// Verify it's valid hex
	for _, c := range secret {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("secret contains invalid hex character: %c", c)
		}
	}
}
