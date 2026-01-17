//go:build integration

package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/penshort/penshort/internal/model"
	"github.com/penshort/penshort/internal/testutil"
)

// ============================================================================
// API Key Repository Integration Tests
// ============================================================================

func TestIntegrationAPIKeyRepository_CreateAPIKey(t *testing.T) {
	ctx, repo := newAPIKeyTestEnv(t)

	userID := testutil.UniqueID("user")
	key := testutil.NewTestAPIKey(t, userID)

	err := repo.CreateAPIKey(ctx, key)
	if err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	// Verify key exists in DB
	retrieved, err := repo.GetAPIKeyByID(ctx, key.ID)
	if err != nil {
		t.Fatalf("GetAPIKeyByID failed: %v", err)
	}

	if retrieved.UserID != userID {
		t.Errorf("UserID mismatch: got %q, want %q", retrieved.UserID, userID)
	}
	if retrieved.KeyHash != key.KeyHash {
		t.Errorf("KeyHash mismatch: got %q, want %q", retrieved.KeyHash, key.KeyHash)
	}
	if retrieved.KeyPrefix != key.KeyPrefix {
		t.Errorf("KeyPrefix mismatch: got %q, want %q", retrieved.KeyPrefix, key.KeyPrefix)
	}
	if retrieved.RateLimitTier != model.TierFree {
		t.Errorf("RateLimitTier mismatch: got %q, want %q", retrieved.RateLimitTier, model.TierFree)
	}
}

func TestIntegrationAPIKeyRepository_GetByID(t *testing.T) {
	ctx, repo := newAPIKeyTestEnv(t)

	userID := testutil.UniqueID("user")
	key := testutil.NewTestAPIKey(t, userID)

	if err := repo.CreateAPIKey(ctx, key); err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	retrieved, err := repo.GetAPIKeyByID(ctx, key.ID)
	if err != nil {
		t.Fatalf("GetAPIKeyByID failed: %v", err)
	}

	if retrieved.ID != key.ID {
		t.Errorf("ID mismatch: got %q, want %q", retrieved.ID, key.ID)
	}
}

func TestIntegrationAPIKeyRepository_GetByID_NotFound(t *testing.T) {
	ctx, repo := newAPIKeyTestEnv(t)

	_, err := repo.GetAPIKeyByID(ctx, "nonexistent-key-id")
	if !errors.Is(err, ErrAPIKeyNotFound) {
		t.Errorf("Expected ErrAPIKeyNotFound, got: %v", err)
	}
}

func TestIntegrationAPIKeyRepository_GetByPrefix(t *testing.T) {
	ctx, repo := newAPIKeyTestEnv(t)

	userID := testutil.UniqueID("user")
	prefix := "pk_prefix_"

	// Create multiple keys with same prefix
	key1 := testutil.NewTestAPIKey(t, userID)
	key1.KeyPrefix = prefix
	key2 := testutil.NewTestAPIKey(t, userID)
	key2.KeyPrefix = prefix

	if err := repo.CreateAPIKey(ctx, key1); err != nil {
		t.Fatalf("CreateAPIKey (1) failed: %v", err)
	}
	time.Sleep(1 * time.Millisecond)
	if err := repo.CreateAPIKey(ctx, key2); err != nil {
		t.Fatalf("CreateAPIKey (2) failed: %v", err)
	}

	keys, err := repo.GetAPIKeysByPrefix(ctx, prefix)
	if err != nil {
		t.Fatalf("GetAPIKeysByPrefix failed: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}

	for _, k := range keys {
		if k.KeyPrefix != prefix {
			t.Errorf("KeyPrefix mismatch: got %q, want %q", k.KeyPrefix, prefix)
		}
	}
}

func TestIntegrationAPIKeyRepository_GetByPrefix_ExcludesRevoked(t *testing.T) {
	ctx, repo := newAPIKeyTestEnv(t)

	userID := testutil.UniqueID("user")
	prefix := "pk_revoke_test_"

	key1 := testutil.NewTestAPIKey(t, userID)
	key1.KeyPrefix = prefix
	key2 := testutil.NewTestAPIKey(t, userID)
	key2.KeyPrefix = prefix

	if err := repo.CreateAPIKey(ctx, key1); err != nil {
		t.Fatalf("CreateAPIKey (1) failed: %v", err)
	}
	time.Sleep(1 * time.Millisecond)
	if err := repo.CreateAPIKey(ctx, key2); err != nil {
		t.Fatalf("CreateAPIKey (2) failed: %v", err)
	}

	// Revoke key1
	if err := repo.RevokeAPIKey(ctx, key1.ID); err != nil {
		t.Fatalf("RevokeAPIKey failed: %v", err)
	}

	// GetByPrefix should only return active keys
	keys, err := repo.GetAPIKeysByPrefix(ctx, prefix)
	if err != nil {
		t.Fatalf("GetAPIKeysByPrefix failed: %v", err)
	}

	if len(keys) != 1 {
		t.Errorf("Expected 1 active key, got %d", len(keys))
	}

	if len(keys) > 0 && keys[0].ID != key2.ID {
		t.Errorf("Expected key2, got key %s", keys[0].ID)
	}
}

func TestIntegrationAPIKeyRepository_ListByUserID(t *testing.T) {
	ctx, repo := newAPIKeyTestEnv(t)

	userID := testutil.UniqueID("user")

	// Create 3 keys
	for i := 0; i < 3; i++ {
		key := testutil.NewTestAPIKey(t, userID)
		if err := repo.CreateAPIKey(ctx, key); err != nil {
			t.Fatalf("CreateAPIKey (%d) failed: %v", i, err)
		}
		time.Sleep(1 * time.Millisecond)
	}

	keys, err := repo.ListAPIKeysByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("ListAPIKeysByUserID failed: %v", err)
	}

	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	for _, k := range keys {
		if k.UserID != userID {
			t.Errorf("UserID mismatch: got %q, want %q", k.UserID, userID)
		}
	}
}

func TestIntegrationAPIKeyRepository_RevokeAPIKey(t *testing.T) {
	ctx, repo := newAPIKeyTestEnv(t)

	userID := testutil.UniqueID("user")
	key := testutil.NewTestAPIKey(t, userID)

	if err := repo.CreateAPIKey(ctx, key); err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	// Revoke
	if err := repo.RevokeAPIKey(ctx, key.ID); err != nil {
		t.Fatalf("RevokeAPIKey failed: %v", err)
	}

	// Verify revoked
	retrieved, err := repo.GetAPIKeyByID(ctx, key.ID)
	if err != nil {
		t.Fatalf("GetAPIKeyByID failed: %v", err)
	}

	if retrieved.RevokedAt == nil {
		t.Error("RevokedAt should be set after revocation")
	}
	if !retrieved.IsRevoked() {
		t.Error("IsRevoked() should return true")
	}
}

func TestIntegrationAPIKeyRepository_RevokeAPIKey_DoubleRevoke(t *testing.T) {
	ctx, repo := newAPIKeyTestEnv(t)

	userID := testutil.UniqueID("user")
	key := testutil.NewTestAPIKey(t, userID)

	if err := repo.CreateAPIKey(ctx, key); err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	// First revoke
	if err := repo.RevokeAPIKey(ctx, key.ID); err != nil {
		t.Fatalf("RevokeAPIKey (first) failed: %v", err)
	}

	// Second revoke should fail (already revoked)
	err := repo.RevokeAPIKey(ctx, key.ID)
	if !errors.Is(err, ErrAPIKeyNotFound) {
		t.Errorf("Expected ErrAPIKeyNotFound on double revoke, got: %v", err)
	}
}

func TestIntegrationAPIKeyRepository_UpdateLastUsed(t *testing.T) {
	ctx, repo := newAPIKeyTestEnv(t)

	userID := testutil.UniqueID("user")
	key := testutil.NewTestAPIKey(t, userID)

	if err := repo.CreateAPIKey(ctx, key); err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	// Initial state: last_used_at is nil
	retrieved, _ := repo.GetAPIKeyByID(ctx, key.ID)
	if retrieved.LastUsedAt != nil {
		t.Error("LastUsedAt should be nil initially")
	}

	// Update last used
	if err := repo.UpdateAPIKeyLastUsed(ctx, key.ID); err != nil {
		t.Fatalf("UpdateAPIKeyLastUsed failed: %v", err)
	}

	// Verify updated
	retrieved, _ = repo.GetAPIKeyByID(ctx, key.ID)
	if retrieved.LastUsedAt == nil {
		t.Error("LastUsedAt should be set after update")
	}
}

func TestIntegrationAPIKeyRepository_ScopesPersistence(t *testing.T) {
	ctx, repo := newAPIKeyTestEnv(t)

	userID := testutil.UniqueID("user")
	key := testutil.NewTestAPIKey(t, userID)
	key.Scopes = []string{model.ScopeRead, model.ScopeWrite, model.ScopeWebhook}

	if err := repo.CreateAPIKey(ctx, key); err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	retrieved, err := repo.GetAPIKeyByID(ctx, key.ID)
	if err != nil {
		t.Fatalf("GetAPIKeyByID failed: %v", err)
	}

	if len(retrieved.Scopes) != 3 {
		t.Errorf("Expected 3 scopes, got %d", len(retrieved.Scopes))
	}

	// Verify HasScope works
	if !retrieved.HasScope(model.ScopeRead) {
		t.Error("Key should have read scope")
	}
	if !retrieved.HasScope(model.ScopeWebhook) {
		t.Error("Key should have webhook scope")
	}
	if retrieved.HasScope(model.ScopeAdmin) {
		t.Error("Key should not have admin scope")
	}
}

func TestIntegrationAPIKeyRepository_TierPersistence(t *testing.T) {
	ctx, repo := newAPIKeyTestEnv(t)

	tests := []struct {
		tier string
	}{
		{model.TierFree},
		{model.TierPro},
		{model.TierUnlimited},
	}

	for _, tc := range tests {
		t.Run(tc.tier, func(t *testing.T) {
			userID := testutil.UniqueID("user")
			key := testutil.NewTestAPIKeyWithTier(t, userID, tc.tier)

			if err := repo.CreateAPIKey(ctx, key); err != nil {
				t.Fatalf("CreateAPIKey failed: %v", err)
			}

			retrieved, err := repo.GetAPIKeyByID(ctx, key.ID)
			if err != nil {
				t.Fatalf("GetAPIKeyByID failed: %v", err)
			}

			if retrieved.RateLimitTier != tc.tier {
				t.Errorf("RateLimitTier mismatch: got %q, want %q", retrieved.RateLimitTier, tc.tier)
			}

			// Verify config is correct
			config := retrieved.GetRateLimitConfig()
			expectedConfig := model.TierConfigs[tc.tier]
			if config.RequestsPerMinute != expectedConfig.RequestsPerMinute {
				t.Errorf("RPM mismatch: got %d, want %d", config.RequestsPerMinute, expectedConfig.RequestsPerMinute)
			}
		})
	}
}

// ============================================================================
// Test Environment Setup
// ============================================================================

func newAPIKeyTestEnv(t *testing.T) (context.Context, *Repository) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	ctx := context.Background()
	dbURL := testutil.RequireEnv(t, "DATABASE_URL")

	repo, err := New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	t.Cleanup(repo.Close)

	unlock, err := testutil.AcquireDBLock(ctx, repo.Pool())
	if err != nil {
		t.Fatalf("acquire db lock: %v", err)
	}
	t.Cleanup(func() {
		_ = unlock()
	})

	// Reset users first (api_keys depends on users)
	if err := testutil.ResetUsersSchema(ctx, repo.Pool()); err != nil {
		t.Fatalf("reset users schema: %v", err)
	}

	if err := testutil.ResetAPIKeysSchema(ctx, repo.Pool()); err != nil {
		t.Fatalf("reset api_keys schema: %v", err)
	}

	return ctx, repo
}
