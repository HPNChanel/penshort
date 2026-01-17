package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/penshort/penshort/internal/model"
	"github.com/redis/go-redis/v9"
)

// RequireEnv returns an environment variable or skips the test if missing.
func RequireEnv(t testing.TB, key string) string {
	t.Helper()
	value := os.Getenv(key)
	if value == "" {
		t.Skipf("%s not set", key)
	}
	return value
}

const advisoryLockID int64 = 420420

// AcquireDBLock grabs a global advisory lock to serialize DB tests.
func AcquireDBLock(ctx context.Context, pool *pgxpool.Pool) (func() error, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire connection: %w", err)
	}

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", advisoryLockID); err != nil {
		conn.Release()
		return nil, fmt.Errorf("acquire advisory lock: %w", err)
	}

	unlock := func() error {
		defer conn.Release()
		if _, err := conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", advisoryLockID); err != nil {
			return fmt.Errorf("release advisory lock: %w", err)
		}
		return nil
	}

	return unlock, nil
}

// ResetLinksSchema drops and recreates the links schema for tests.
func ResetLinksSchema(ctx context.Context, pool *pgxpool.Pool) error {
	root, err := ProjectRoot()
	if err != nil {
		return err
	}

	downPath := filepath.Join(root, "migrations", "000002_links.down.sql")
	upPath := filepath.Join(root, "migrations", "000002_links.up.sql")

	downSQL, err := os.ReadFile(downPath)
	if err != nil {
		return fmt.Errorf("read down migration: %w", err)
	}
	if _, err := pool.Exec(ctx, string(downSQL)); err != nil {
		return fmt.Errorf("apply down migration: %w", err)
	}

	upSQL, err := os.ReadFile(upPath)
	if err != nil {
		return fmt.Errorf("read up migration: %w", err)
	}
	if _, err := pool.Exec(ctx, string(upSQL)); err != nil {
		return fmt.Errorf("apply up migration: %w", err)
	}

	return nil
}

// ResetAnalyticsSchema drops and recreates the analytics schema for tests.
func ResetAnalyticsSchema(ctx context.Context, pool *pgxpool.Pool) error {
	root, err := ProjectRoot()
	if err != nil {
		return err
	}

	downPath := filepath.Join(root, "migrations", "000005_analytics.down.sql")
	upPath := filepath.Join(root, "migrations", "000005_analytics.up.sql")

	downSQL, err := os.ReadFile(downPath)
	if err != nil {
		return fmt.Errorf("read analytics down migration: %w", err)
	}
	if _, err := pool.Exec(ctx, string(downSQL)); err != nil {
		return fmt.Errorf("apply analytics down migration: %w", err)
	}

	upSQL, err := os.ReadFile(upPath)
	if err != nil {
		return fmt.Errorf("read analytics up migration: %w", err)
	}
	if _, err := pool.Exec(ctx, string(upSQL)); err != nil {
		return fmt.Errorf("apply analytics up migration: %w", err)
	}

	return nil
}

// ResetAPIKeysSchema drops and recreates the api_keys schema for tests.
func ResetAPIKeysSchema(ctx context.Context, pool *pgxpool.Pool) error {
	root, err := ProjectRoot()
	if err != nil {
		return err
	}

	downPath := filepath.Join(root, "migrations", "000004_api_keys.down.sql")
	upPath := filepath.Join(root, "migrations", "000004_api_keys.up.sql")

	downSQL, err := os.ReadFile(downPath)
	if err != nil {
		return fmt.Errorf("read api_keys down migration: %w", err)
	}
	if _, err := pool.Exec(ctx, string(downSQL)); err != nil {
		return fmt.Errorf("apply api_keys down migration: %w", err)
	}

	upSQL, err := os.ReadFile(upPath)
	if err != nil {
		return fmt.Errorf("read api_keys up migration: %w", err)
	}
	if _, err := pool.Exec(ctx, string(upSQL)); err != nil {
		return fmt.Errorf("apply api_keys up migration: %w", err)
	}

	return nil
}

// ResetWebhooksSchema drops and recreates the webhooks schema for tests.
func ResetWebhooksSchema(ctx context.Context, pool *pgxpool.Pool) error {
	root, err := ProjectRoot()
	if err != nil {
		return err
	}

	downPath := filepath.Join(root, "migrations", "000006_webhooks.down.sql")
	upPath := filepath.Join(root, "migrations", "000006_webhooks.up.sql")

	downSQL, err := os.ReadFile(downPath)
	if err != nil {
		return fmt.Errorf("read webhooks down migration: %w", err)
	}
	if _, err := pool.Exec(ctx, string(downSQL)); err != nil {
		return fmt.Errorf("apply webhooks down migration: %w", err)
	}

	upSQL, err := os.ReadFile(upPath)
	if err != nil {
		return fmt.Errorf("read webhooks up migration: %w", err)
	}
	if _, err := pool.Exec(ctx, string(upSQL)); err != nil {
		return fmt.Errorf("apply webhooks up migration: %w", err)
	}

	return nil
}

// ResetUsersSchema drops and recreates the users schema for tests.
func ResetUsersSchema(ctx context.Context, pool *pgxpool.Pool) error {
	root, err := ProjectRoot()
	if err != nil {
		return err
	}

	downPath := filepath.Join(root, "migrations", "000003_users.down.sql")
	upPath := filepath.Join(root, "migrations", "000003_users.up.sql")

	downSQL, err := os.ReadFile(downPath)
	if err != nil {
		return fmt.Errorf("read users down migration: %w", err)
	}
	if _, err := pool.Exec(ctx, string(downSQL)); err != nil {
		return fmt.Errorf("apply users down migration: %w", err)
	}

	upSQL, err := os.ReadFile(upPath)
	if err != nil {
		return fmt.Errorf("read users up migration: %w", err)
	}
	if _, err := pool.Exec(ctx, string(upSQL)); err != nil {
		return fmt.Errorf("apply users up migration: %w", err)
	}

	return nil
}

// FlushRedis clears the current Redis database.
func FlushRedis(ctx context.Context, client *redis.Client) error {
	return client.FlushDB(ctx).Err()
}

// ProjectRoot returns the project root directory.
func ProjectRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to resolve testutil path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	return root, nil
}

// ============================================================================
// Test Data Factories
// ============================================================================

// NewTestLink creates a test link with sensible defaults.
func NewTestLink(t testing.TB, shortCode string) *model.Link {
	t.Helper()
	now := time.Now().UTC()
	return &model.Link{
		ID:           fmt.Sprintf("link-%d", now.UnixNano()),
		ShortCode:    shortCode,
		Destination:  "https://example.com/" + shortCode,
		RedirectType: model.RedirectTemporary,
		OwnerID:      "test-user",
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// NewTestLinkWithExpiry creates a test link with an expiry time.
func NewTestLinkWithExpiry(t testing.TB, shortCode string, expiresAt time.Time) *model.Link {
	t.Helper()
	link := NewTestLink(t, shortCode)
	link.ExpiresAt = &expiresAt
	return link
}

// NewTestAPIKey creates a test API key with sensible defaults.
func NewTestAPIKey(t testing.TB, userID string) *model.APIKey {
	t.Helper()
	now := time.Now().UTC()
	return &model.APIKey{
		ID:            fmt.Sprintf("key-%d", now.UnixNano()),
		UserID:        userID,
		KeyHash:       fmt.Sprintf("hash-%d", now.UnixNano()),
		KeyPrefix:     "pk_test_",
		Scopes:        []string{model.ScopeRead, model.ScopeWrite},
		RateLimitTier: model.TierFree,
		Name:          "Test Key",
		CreatedAt:     now,
	}
}

// NewTestAPIKeyWithTier creates a test API key with a specific tier.
func NewTestAPIKeyWithTier(t testing.TB, userID string, tier string) *model.APIKey {
	t.Helper()
	key := NewTestAPIKey(t, userID)
	key.RateLimitTier = tier
	return key
}

// UniqueShortCode generates a unique short code for tests.
func UniqueShortCode(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// UniqueID generates a unique ID for tests.
func UniqueID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

