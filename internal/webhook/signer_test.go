package webhook

import (
	"testing"
	"time"
)

func TestGenerateSignature(t *testing.T) {
	tests := []struct {
		name        string
		secret      string
		timestamp   int64
		payloadJSON []byte
	}{
		{
			name:        "basic signature",
			secret:      "whsec_test123",
			timestamp:   1736600000,
			payloadJSON: []byte(`{"event_type":"click","event_id":"123"}`),
		},
		{
			name:        "empty payload",
			secret:      "secret",
			timestamp:   1000000000,
			payloadJSON: []byte(`{}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig := GenerateSignature(tt.secret, tt.timestamp, tt.payloadJSON)

			// Signature should be hex-encoded (64 chars for SHA256)
			if len(sig) != 64 {
				t.Errorf("signature length = %d, want 64", len(sig))
			}

			// Same inputs should produce same signature
			sig2 := GenerateSignature(tt.secret, tt.timestamp, tt.payloadJSON)
			if sig != sig2 {
				t.Error("signature is not deterministic")
			}

			// Different timestamp should produce different signature
			sig3 := GenerateSignature(tt.secret, tt.timestamp+1, tt.payloadJSON)
			if sig == sig3 {
				t.Error("different timestamp should produce different signature")
			}

			// Different secret should produce different signature
			sig4 := GenerateSignature(tt.secret+"x", tt.timestamp, tt.payloadJSON)
			if sig == sig4 {
				t.Error("different secret should produce different signature")
			}
		})
	}
}

func TestValidateSignature(t *testing.T) {
	secret := "test_secret"
	timestamp := time.Now().Unix()
	payload := []byte(`{"test":"data"}`)

	validSig := GenerateSignature(secret, timestamp, payload)

	tests := []struct {
		name        string
		secret      string
		signature   string
		timestamp   int64
		payload     []byte
		window      time.Duration
		wantErr     error
	}{
		{
			name:      "valid signature",
			secret:    secret,
			signature: validSig,
			timestamp: timestamp,
			payload:   payload,
			window:    5 * time.Minute,
			wantErr:   nil,
		},
		{
			name:      "invalid signature",
			secret:    secret,
			signature: "invalid",
			timestamp: timestamp,
			payload:   payload,
			window:    5 * time.Minute,
			wantErr:   ErrInvalidSignature,
		},
		{
			name:      "expired timestamp",
			secret:    secret,
			signature: GenerateSignature(secret, time.Now().Add(-10*time.Minute).Unix(), payload),
			timestamp: time.Now().Add(-10 * time.Minute).Unix(),
			payload:   payload,
			window:    5 * time.Minute,
			wantErr:   ErrReplayWindowExceeded,
		},
		{
			name:      "future timestamp beyond window",
			secret:    secret,
			signature: GenerateSignature(secret, time.Now().Add(10*time.Minute).Unix(), payload),
			timestamp: time.Now().Add(10 * time.Minute).Unix(),
			payload:   payload,
			window:    5 * time.Minute,
			wantErr:   ErrReplayWindowExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSignature(tt.secret, tt.signature, tt.timestamp, tt.payload, tt.window)
			if err != tt.wantErr {
				t.Errorf("ValidateSignature() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestHashSecret(t *testing.T) {
	secret := "my_secret_key"
	hash := HashSecret(secret)

	// Should be hex-encoded SHA256 (64 chars)
	if len(hash) != 64 {
		t.Errorf("hash length = %d, want 64", len(hash))
	}

	// Same input should produce same hash
	hash2 := HashSecret(secret)
	if hash != hash2 {
		t.Error("hash is not deterministic")
	}

	// Different input should produce different hash
	hash3 := HashSecret(secret + "x")
	if hash == hash3 {
		t.Error("different input should produce different hash")
	}
}
