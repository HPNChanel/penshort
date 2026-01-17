//go:build integration

package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/penshort/penshort/internal/testutil"
)

// ============================================================================
// Migration Integration Tests
// ============================================================================

func TestIntegrationMigration_ApplyAllTables(t *testing.T) {
	ctx, pool := newMigrationTestEnv(t)

	// Verify all expected tables exist
	tables := []string{
		"links",
		"users",
		"api_keys",
		"click_events",
		"daily_link_stats",
		"webhook_endpoints",
		"webhook_deliveries",
	}

	for _, table := range tables {
		t.Run(table, func(t *testing.T) {
			exists, err := tableExists(ctx, pool, table)
			if err != nil {
				t.Fatalf("tableExists failed: %v", err)
			}
			if !exists {
				t.Errorf("Table %q should exist after migrations", table)
			}
		})
	}
}

func TestIntegrationMigration_LinksTableSchema(t *testing.T) {
	ctx, pool := newMigrationTestEnv(t)

	// Verify links table has expected columns
	expectedColumns := []string{
		"id",
		"short_code",
		"destination",
		"redirect_type",
		"owner_id",
		"enabled",
		"expires_at",
		"deleted_at",
		"click_count",
		"created_at",
		"updated_at",
	}

	for _, col := range expectedColumns {
		t.Run(col, func(t *testing.T) {
			exists, err := columnExists(ctx, pool, "links", col)
			if err != nil {
				t.Fatalf("columnExists failed: %v", err)
			}
			if !exists {
				t.Errorf("Column %q should exist in links table", col)
			}
		})
	}
}

func TestIntegrationMigration_LinksConstraints(t *testing.T) {
	ctx, pool := newMigrationTestEnv(t)

	// Verify redirect_type check constraint
	_, err := pool.Exec(ctx, `
		INSERT INTO links (id, short_code, destination, redirect_type, owner_id)
		VALUES ('test-id', 'test-code', 'https://example.com', 999, 'system')
	`)
	if err == nil {
		t.Error("Expected check constraint violation for invalid redirect_type")
	}

	// Verify destination length constraint
	longDest := "https://example.com/" + string(make([]byte, 2100))
	_, err = pool.Exec(ctx, `
		INSERT INTO links (id, short_code, destination, redirect_type, owner_id)
		VALUES ('test-id', 'test-code', $1, 302, 'system')
	`, longDest)
	if err == nil {
		t.Error("Expected check constraint violation for destination > 2048 chars")
	}

	// Verify short_code length constraint
	_, err = pool.Exec(ctx, `
		INSERT INTO links (id, short_code, destination, redirect_type, owner_id)
		VALUES ('test-id', 'ab', 'https://example.com', 302, 'system')
	`)
	if err == nil {
		t.Error("Expected check constraint violation for short_code < 3 chars")
	}
}

func TestIntegrationMigration_APIKeysTableSchema(t *testing.T) {
	ctx, pool := newMigrationTestEnv(t)

	expectedColumns := []string{
		"id",
		"user_id",
		"key_hash",
		"key_prefix",
		"scopes",
		"rate_limit_tier",
		"name",
		"revoked_at",
		"last_used_at",
		"created_at",
	}

	for _, col := range expectedColumns {
		t.Run(col, func(t *testing.T) {
			exists, err := columnExists(ctx, pool, "api_keys", col)
			if err != nil {
				t.Fatalf("columnExists failed: %v", err)
			}
			if !exists {
				t.Errorf("Column %q should exist in api_keys table", col)
			}
		})
	}
}

func TestIntegrationMigration_AnalyticsTablesSchema(t *testing.T) {
	ctx, pool := newMigrationTestEnv(t)

	// click_events columns
	clickEventCols := []string{
		"id",
		"event_id",
		"short_code",
		"link_id",
		"referrer",
		"user_agent",
		"visitor_hash",
		"country_code",
		"clicked_at",
		"created_at",
	}

	for _, col := range clickEventCols {
		exists, err := columnExists(ctx, pool, "click_events", col)
		if err != nil {
			t.Fatalf("columnExists failed: %v", err)
		}
		if !exists {
			t.Errorf("Column %q should exist in click_events table", col)
		}
	}

	// daily_link_stats columns
	statsColumns := []string{
		"id",
		"link_id",
		"date",
		"total_clicks",
		"unique_visitors",
		"referrer_breakdown",
		"ua_family_breakdown",
		"country_breakdown",
	}

	for _, col := range statsColumns {
		exists, err := columnExists(ctx, pool, "daily_link_stats", col)
		if err != nil {
			t.Fatalf("columnExists failed: %v", err)
		}
		if !exists {
			t.Errorf("Column %q should exist in daily_link_stats table", col)
		}
	}
}

func TestIntegrationMigration_WebhookTablesSchema(t *testing.T) {
	ctx, pool := newMigrationTestEnv(t)

	// webhook_endpoints columns
	endpointCols := []string{
		"id",
		"user_id",
		"target_url",
		"secret_hash",
		"enabled",
		"event_types",
		"name",
		"description",
		"created_at",
		"updated_at",
		"deleted_at",
	}

	for _, col := range endpointCols {
		exists, err := columnExists(ctx, pool, "webhook_endpoints", col)
		if err != nil {
			t.Fatalf("columnExists failed: %v", err)
		}
		if !exists {
			t.Errorf("Column %q should exist in webhook_endpoints table", col)
		}
	}

	// webhook_deliveries columns
	deliveryCols := []string{
		"id",
		"endpoint_id",
		"event_id",
		"event_type",
		"payload_json",
		"status",
		"attempt_count",
		"max_attempts",
		"next_retry_at",
		"last_attempt_at",
		"last_http_status",
		"last_error",
		"created_at",
		"updated_at",
	}

	for _, col := range deliveryCols {
		exists, err := columnExists(ctx, pool, "webhook_deliveries", col)
		if err != nil {
			t.Fatalf("columnExists failed: %v", err)
		}
		if !exists {
			t.Errorf("Column %q should exist in webhook_deliveries table", col)
		}
	}
}

func TestIntegrationMigration_RollbackLinks(t *testing.T) {
	ctx, pool := newMigrationTestEnv(t)

	root, err := testutil.ProjectRoot()
	if err != nil {
		t.Fatalf("ProjectRoot failed: %v", err)
	}

	// Apply down migration
	downPath := filepath.Join(root, "migrations", "000002_links.down.sql")
	downSQL, err := os.ReadFile(downPath)
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(downSQL)); err != nil {
		t.Fatalf("apply down migration: %v", err)
	}

	// Verify table doesn't exist
	exists, err := tableExists(ctx, pool, "links")
	if err != nil {
		t.Fatalf("tableExists failed: %v", err)
	}
	if exists {
		t.Error("links table should not exist after rollback")
	}

	// Re-apply up migration for cleanup
	upPath := filepath.Join(root, "migrations", "000002_links.up.sql")
	upSQL, err := os.ReadFile(upPath)
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(upSQL)); err != nil {
		t.Fatalf("reapply up migration: %v", err)
	}
}

func TestIntegrationMigration_Idempotency(t *testing.T) {
	ctx, pool := newMigrationTestEnv(t)

	root, err := testutil.ProjectRoot()
	if err != nil {
		t.Fatalf("ProjectRoot failed: %v", err)
	}

	// Apply up migration again (should be idempotent via IF NOT EXISTS)
	// Note: This tests the CREATE EXTENSION IF NOT EXISTS clause
	upPath := filepath.Join(root, "migrations", "000001_init.up.sql")
	upSQL, err := os.ReadFile(upPath)
	if err != nil {
		t.Fatalf("read init up migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(upSQL)); err != nil {
		t.Fatalf("second apply should not fail: %v", err)
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

func tableExists(ctx context.Context, pool *pgxpool.Pool, tableName string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = $1
		)
	`, tableName).Scan(&exists)
	return exists, err
}

func columnExists(ctx context.Context, pool *pgxpool.Pool, tableName, columnName string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.columns 
			WHERE table_schema = 'public' 
			AND table_name = $1 
			AND column_name = $2
		)
	`, tableName, columnName).Scan(&exists)
	return exists, err
}

// ============================================================================
// Test Environment Setup
// ============================================================================

func newMigrationTestEnv(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	ctx := context.Background()
	dbURL := testutil.RequireEnv(t, "DATABASE_URL")

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	t.Cleanup(pool.Close)

	unlock, err := testutil.AcquireDBLock(ctx, pool)
	if err != nil {
		t.Fatalf("acquire db lock: %v", err)
	}
	t.Cleanup(func() {
		_ = unlock()
	})

	return ctx, pool
}
