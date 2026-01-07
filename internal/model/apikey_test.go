package model

import (
	"slices"
	"testing"
)

func TestAPIKey_HasScope(t *testing.T) {
	testCases := []struct {
		name      string
		keyScopes []string
		checkFor  string
		want      bool
	}{
		{
			name:      "has exact scope",
			keyScopes: []string{ScopeRead, ScopeWrite},
			checkFor:  ScopeRead,
			want:      true,
		},
		{
			name:      "does not have scope",
			keyScopes: []string{ScopeRead},
			checkFor:  ScopeWrite,
			want:      false,
		},
		{
			name:      "admin implies all",
			keyScopes: []string{ScopeAdmin},
			checkFor:  ScopeRead,
			want:      true,
		},
		{
			name:      "admin implies write",
			keyScopes: []string{ScopeAdmin},
			checkFor:  ScopeWrite,
			want:      true,
		},
		{
			name:      "admin implies webhook",
			keyScopes: []string{ScopeAdmin},
			checkFor:  ScopeWebhook,
			want:      true,
		},
		{
			name:      "empty scopes",
			keyScopes: []string{},
			checkFor:  ScopeRead,
			want:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key := &APIKey{Scopes: tc.keyScopes}
			got := key.HasScope(tc.checkFor)
			if got != tc.want {
				t.Errorf("HasScope(%s) = %v, want %v", tc.checkFor, got, tc.want)
			}
		})
	}
}

func TestAuthContext_HasScope(t *testing.T) {
	testCases := []struct {
		name      string
		scopes    []string
		checkFor  string
		want      bool
	}{
		{
			name:     "has scope",
			scopes:   []string{ScopeRead},
			checkFor: ScopeRead,
			want:     true,
		},
		{
			name:     "admin grants all",
			scopes:   []string{ScopeAdmin},
			checkFor: ScopeWrite,
			want:     true,
		},
		{
			name:     "missing scope",
			scopes:   []string{ScopeRead},
			checkFor: ScopeAdmin,
			want:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &AuthContext{Scopes: tc.scopes}
			got := ctx.HasScope(tc.checkFor)
			if got != tc.want {
				t.Errorf("HasScope(%s) = %v, want %v", tc.checkFor, got, tc.want)
			}
		})
	}
}

func TestAPIKey_IsRevoked(t *testing.T) {
	key := &APIKey{}
	if key.IsRevoked() {
		t.Error("new key should not be revoked")
	}
}

func TestAPIKey_GetRateLimitConfig(t *testing.T) {
	testCases := []struct {
		tier        string
		wantRPM     int
		wantBurst   int
	}{
		{TierFree, 60, 10},
		{TierPro, 600, 50},
		{TierUnlimited, 0, 0},
		{"unknown", 60, 10}, // Falls back to free
	}

	for _, tc := range testCases {
		t.Run(tc.tier, func(t *testing.T) {
			key := &APIKey{RateLimitTier: tc.tier}
			config := key.GetRateLimitConfig()
			if config.RequestsPerMinute != tc.wantRPM {
				t.Errorf("RPM = %d, want %d", config.RequestsPerMinute, tc.wantRPM)
			}
			if config.Burst != tc.wantBurst {
				t.Errorf("Burst = %d, want %d", config.Burst, tc.wantBurst)
			}
		})
	}
}

func TestValidScopes(t *testing.T) {
	expected := []string{ScopeRead, ScopeWrite, ScopeWebhook, ScopeAdmin}
	for _, scope := range expected {
		if !slices.Contains(ValidScopes, scope) {
			t.Errorf("ValidScopes should contain %s", scope)
		}
	}
}

func TestAPIKey_ToResponse(t *testing.T) {
	key := &APIKey{
		ID:            "key123",
		Name:          "Test Key",
		KeyPrefix:     "abc123",
		Scopes:        []string{ScopeRead},
		RateLimitTier: TierFree,
	}

	resp := key.ToResponse()
	if resp.ID != key.ID {
		t.Errorf("ID mismatch")
	}
	if resp.KeyPrefix != key.KeyPrefix {
		t.Errorf("KeyPrefix mismatch")
	}
	if resp.Revoked != false {
		t.Errorf("Revoked should be false for active key")
	}
}
