package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
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
	root, err := projectRoot()
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

// FlushRedis clears the current Redis database.
func FlushRedis(ctx context.Context, client *redis.Client) error {
	return client.FlushDB(ctx).Err()
}

func projectRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to resolve testutil path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	return root, nil
}
