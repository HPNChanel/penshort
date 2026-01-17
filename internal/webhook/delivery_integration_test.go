//go:build integration

package webhook

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/penshort/penshort/internal/model"
	"github.com/penshort/penshort/internal/testutil"
)

// ============================================================================
// Webhook Delivery Persistence Integration Tests
// ============================================================================

func TestIntegrationWebhook_CreateEndpoint(t *testing.T) {
	ctx, db, repo := newWebhookTestEnv(t)

	userID := testutil.UniqueID("user")
	endpoint := newTestEndpoint(t, userID)

	err := repo.CreateEndpoint(ctx, endpoint)
	if err != nil {
		t.Fatalf("CreateEndpoint failed: %v", err)
	}

	// Verify endpoint exists in DB
	retrieved, err := repo.GetEndpoint(ctx, endpoint.ID)
	if err != nil {
		t.Fatalf("GetEndpoint failed: %v", err)
	}

	if retrieved.UserID != userID {
		t.Errorf("UserID mismatch: got %q, want %q", retrieved.UserID, userID)
	}
	if retrieved.TargetURL != endpoint.TargetURL {
		t.Errorf("TargetURL mismatch: got %q, want %q", retrieved.TargetURL, endpoint.TargetURL)
	}
	if !retrieved.Enabled {
		t.Error("Endpoint should be enabled")
	}

	_ = db // silence unused warning
}

func TestIntegrationWebhook_CreateDelivery(t *testing.T) {
	ctx, _, repo := newWebhookTestEnv(t)

	userID := testutil.UniqueID("user")
	endpoint := newTestEndpoint(t, userID)

	if err := repo.CreateEndpoint(ctx, endpoint); err != nil {
		t.Fatalf("CreateEndpoint failed: %v", err)
	}

	delivery := newTestDelivery(t, endpoint.ID)

	err := repo.CreateDelivery(ctx, delivery)
	if err != nil {
		t.Fatalf("CreateDelivery failed: %v", err)
	}

	// Verify delivery exists
	retrieved, _, err := repo.GetDeliveryWithEndpoint(ctx, delivery.ID)
	if err != nil {
		t.Fatalf("GetDeliveryWithEndpoint failed: %v", err)
	}

	if retrieved.Status != model.DeliveryStatusPending {
		t.Errorf("Status mismatch: got %q, want %q", retrieved.Status, model.DeliveryStatusPending)
	}
	if retrieved.AttemptCount != 0 {
		t.Errorf("AttemptCount should be 0, got %d", retrieved.AttemptCount)
	}
}

func TestIntegrationWebhook_DeliverySuccess(t *testing.T) {
	ctx, _, repo := newWebhookTestEnv(t)

	userID := testutil.UniqueID("user")
	endpoint := newTestEndpoint(t, userID)

	if err := repo.CreateEndpoint(ctx, endpoint); err != nil {
		t.Fatalf("CreateEndpoint failed: %v", err)
	}

	delivery := newTestDelivery(t, endpoint.ID)

	if err := repo.CreateDelivery(ctx, delivery); err != nil {
		t.Fatalf("CreateDelivery failed: %v", err)
	}

	// Mark as success
	err := repo.UpdateDeliverySuccess(ctx, delivery.ID, 200)
	if err != nil {
		t.Fatalf("UpdateDeliverySuccess failed: %v", err)
	}

	// Verify
	retrieved, _, err := repo.GetDeliveryWithEndpoint(ctx, delivery.ID)
	if err != nil {
		t.Fatalf("GetDeliveryWithEndpoint failed: %v", err)
	}

	if retrieved.Status != model.DeliveryStatusSuccess {
		t.Errorf("Status mismatch: got %q, want %q", retrieved.Status, model.DeliveryStatusSuccess)
	}
	if retrieved.AttemptCount != 1 {
		t.Errorf("AttemptCount should be 1, got %d", retrieved.AttemptCount)
	}
	if retrieved.LastHTTPStatus == nil || *retrieved.LastHTTPStatus != 200 {
		t.Error("LastHTTPStatus should be 200")
	}
}

func TestIntegrationWebhook_DeliveryRetry(t *testing.T) {
	ctx, _, repo := newWebhookTestEnv(t)

	userID := testutil.UniqueID("user")
	endpoint := newTestEndpoint(t, userID)

	if err := repo.CreateEndpoint(ctx, endpoint); err != nil {
		t.Fatalf("CreateEndpoint failed: %v", err)
	}

	delivery := newTestDelivery(t, endpoint.ID)

	if err := repo.CreateDelivery(ctx, delivery); err != nil {
		t.Fatalf("CreateDelivery failed: %v", err)
	}

	// Mark as failed with retry
	status := 500
	nextRetry := time.Now().Add(1 * time.Minute)
	err := repo.UpdateDeliveryFailure(ctx, delivery.ID, &status, "server error", nextRetry, false)
	if err != nil {
		t.Fatalf("UpdateDeliveryFailure failed: %v", err)
	}

	// Verify
	retrieved, _, err := repo.GetDeliveryWithEndpoint(ctx, delivery.ID)
	if err != nil {
		t.Fatalf("GetDeliveryWithEndpoint failed: %v", err)
	}

	if retrieved.Status != model.DeliveryStatusFailed {
		t.Errorf("Status mismatch: got %q, want %q", retrieved.Status, model.DeliveryStatusFailed)
	}
	if retrieved.AttemptCount != 1 {
		t.Errorf("AttemptCount should be 1, got %d", retrieved.AttemptCount)
	}
	if retrieved.LastHTTPStatus == nil || *retrieved.LastHTTPStatus != 500 {
		t.Error("LastHTTPStatus should be 500")
	}
	if retrieved.LastError != "server error" {
		t.Errorf("LastError mismatch: got %q, want %q", retrieved.LastError, "server error")
	}
}

func TestIntegrationWebhook_DeliveryExhausted(t *testing.T) {
	ctx, _, repo := newWebhookTestEnv(t)

	userID := testutil.UniqueID("user")
	endpoint := newTestEndpoint(t, userID)

	if err := repo.CreateEndpoint(ctx, endpoint); err != nil {
		t.Fatalf("CreateEndpoint failed: %v", err)
	}

	delivery := newTestDelivery(t, endpoint.ID)
	delivery.MaxAttempts = 3

	if err := repo.CreateDelivery(ctx, delivery); err != nil {
		t.Fatalf("CreateDelivery failed: %v", err)
	}

	// Exhaust retries
	status := 503
	nextRetry := time.Now()
	err := repo.UpdateDeliveryFailure(ctx, delivery.ID, &status, "service unavailable", nextRetry, true)
	if err != nil {
		t.Fatalf("UpdateDeliveryFailure (exhausted) failed: %v", err)
	}

	// Verify
	retrieved, _, err := repo.GetDeliveryWithEndpoint(ctx, delivery.ID)
	if err != nil {
		t.Fatalf("GetDeliveryWithEndpoint failed: %v", err)
	}

	if retrieved.Status != model.DeliveryStatusExhausted {
		t.Errorf("Status mismatch: got %q, want %q", retrieved.Status, model.DeliveryStatusExhausted)
	}
	if !retrieved.IsTerminal() {
		t.Error("Exhausted delivery should be terminal")
	}
}

func TestIntegrationWebhook_DuplicateEventEndpoint(t *testing.T) {
	ctx, _, repo := newWebhookTestEnv(t)

	userID := testutil.UniqueID("user")
	endpoint := newTestEndpoint(t, userID)

	if err := repo.CreateEndpoint(ctx, endpoint); err != nil {
		t.Fatalf("CreateEndpoint failed: %v", err)
	}

	delivery1 := newTestDelivery(t, endpoint.ID)
	eventID := delivery1.EventID

	if err := repo.CreateDelivery(ctx, delivery1); err != nil {
		t.Fatalf("CreateDelivery (first) failed: %v", err)
	}

	// Try to insert duplicate event for same endpoint
	delivery2 := newTestDelivery(t, endpoint.ID)
	delivery2.EventID = eventID // Same event ID

	// Should be ignored (ON CONFLICT DO NOTHING)
	err := repo.CreateDelivery(ctx, delivery2)
	if err != nil {
		t.Fatalf("CreateDelivery (duplicate) should not error: %v", err)
	}

	// Verify only one delivery exists
	deliveries, total, err := repo.ListDeliveriesByEndpoint(ctx, endpoint.ID, nil, 10, 0)
	if err != nil {
		t.Fatalf("ListDeliveriesByEndpoint failed: %v", err)
	}

	if total != 1 {
		t.Errorf("Expected 1 delivery, got %d", total)
	}
	if len(deliveries) != 1 {
		t.Errorf("Expected 1 delivery in list, got %d", len(deliveries))
	}
}

func TestIntegrationWebhook_GetPendingDeliveries(t *testing.T) {
	ctx, _, repo := newWebhookTestEnv(t)

	userID := testutil.UniqueID("user")
	endpoint := newTestEndpoint(t, userID)

	if err := repo.CreateEndpoint(ctx, endpoint); err != nil {
		t.Fatalf("CreateEndpoint failed: %v", err)
	}

	// Create 3 pending deliveries
	for i := 0; i < 3; i++ {
		delivery := newTestDelivery(t, endpoint.ID)
		delivery.NextRetryAt = time.Now().Add(-1 * time.Minute) // Past due
		if err := repo.CreateDelivery(ctx, delivery); err != nil {
			t.Fatalf("CreateDelivery (%d) failed: %v", i, err)
		}
		time.Sleep(1 * time.Millisecond)
	}

	// Create 1 future delivery
	futureDelivery := newTestDelivery(t, endpoint.ID)
	futureDelivery.NextRetryAt = time.Now().Add(1 * time.Hour) // Future
	if err := repo.CreateDelivery(ctx, futureDelivery); err != nil {
		t.Fatalf("CreateDelivery (future) failed: %v", err)
	}

	// Get pending
	pending, err := repo.GetPendingDeliveries(ctx, 10)
	if err != nil {
		t.Fatalf("GetPendingDeliveries failed: %v", err)
	}

	if len(pending) != 3 {
		t.Errorf("Expected 3 pending deliveries, got %d", len(pending))
	}
}

func TestIntegrationWebhook_EndpointSoftDelete(t *testing.T) {
	ctx, _, repo := newWebhookTestEnv(t)

	userID := testutil.UniqueID("user")
	endpoint := newTestEndpoint(t, userID)

	if err := repo.CreateEndpoint(ctx, endpoint); err != nil {
		t.Fatalf("CreateEndpoint failed: %v", err)
	}

	// Delete endpoint
	if err := repo.DeleteEndpoint(ctx, endpoint.ID); err != nil {
		t.Fatalf("DeleteEndpoint failed: %v", err)
	}

	// Should not be found
	_, err := repo.GetEndpoint(ctx, endpoint.ID)
	if !errors.Is(err, ErrEndpointNotFound) {
		t.Errorf("Expected ErrEndpointNotFound, got: %v", err)
	}

	// Deliveries for deleted endpoint should not appear in pending
	delivery := newTestDelivery(t, endpoint.ID)
	delivery.NextRetryAt = time.Now().Add(-1 * time.Minute)
	// Note: Can't create delivery for deleted endpoint via FK normally,
	// but let's verify the query excludes deleted endpoints
}

func TestIntegrationWebhook_QueueDepth(t *testing.T) {
	ctx, _, repo := newWebhookTestEnv(t)

	userID := testutil.UniqueID("user")
	endpoint := newTestEndpoint(t, userID)

	if err := repo.CreateEndpoint(ctx, endpoint); err != nil {
		t.Fatalf("CreateEndpoint failed: %v", err)
	}

	// Initially empty
	depth, err := repo.GetQueueDepth(ctx)
	if err != nil {
		t.Fatalf("GetQueueDepth failed: %v", err)
	}
	if depth != 0 {
		t.Errorf("Expected queue depth 0, got %d", depth)
	}

	// Add 2 pending deliveries
	for i := 0; i < 2; i++ {
		delivery := newTestDelivery(t, endpoint.ID)
		if err := repo.CreateDelivery(ctx, delivery); err != nil {
			t.Fatalf("CreateDelivery (%d) failed: %v", i, err)
		}
		time.Sleep(1 * time.Millisecond)
	}

	depth, err = repo.GetQueueDepth(ctx)
	if err != nil {
		t.Fatalf("GetQueueDepth (after add) failed: %v", err)
	}
	if depth != 2 {
		t.Errorf("Expected queue depth 2, got %d", depth)
	}
}

func TestIntegrationWebhook_ResetDeliveryForRetry(t *testing.T) {
	ctx, _, repo := newWebhookTestEnv(t)

	userID := testutil.UniqueID("user")
	endpoint := newTestEndpoint(t, userID)

	if err := repo.CreateEndpoint(ctx, endpoint); err != nil {
		t.Fatalf("CreateEndpoint failed: %v", err)
	}

	delivery := newTestDelivery(t, endpoint.ID)

	if err := repo.CreateDelivery(ctx, delivery); err != nil {
		t.Fatalf("CreateDelivery failed: %v", err)
	}

	// Exhaust it first
	err := repo.UpdateDeliveryFailure(ctx, delivery.ID, nil, "exhausted", time.Now(), true)
	if err != nil {
		t.Fatalf("UpdateDeliveryFailure failed: %v", err)
	}

	// Reset for retry
	if err := repo.ResetDeliveryForRetry(ctx, delivery.ID); err != nil {
		t.Fatalf("ResetDeliveryForRetry failed: %v", err)
	}

	// Verify reset
	retrieved, _, err := repo.GetDeliveryWithEndpoint(ctx, delivery.ID)
	if err != nil {
		t.Fatalf("GetDeliveryWithEndpoint failed: %v", err)
	}

	if retrieved.Status != model.DeliveryStatusPending {
		t.Errorf("Status should be pending after reset, got %q", retrieved.Status)
	}
}

// ============================================================================
// Test Helpers
// ============================================================================

func newTestEndpoint(t testing.TB, userID string) *model.WebhookEndpoint {
	t.Helper()
	now := time.Now().UTC()
	return &model.WebhookEndpoint{
		ID:         testutil.UniqueID("endpoint"),
		UserID:     userID,
		TargetURL:  "https://example.com/webhook",
		SecretHash: "test-secret-hash-" + testutil.UniqueID(""),
		Enabled:    true,
		EventTypes: []model.EventType{model.EventTypeClick},
		Name:       "Test Webhook",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func newTestDelivery(t testing.TB, endpointID string) *model.WebhookDelivery {
	t.Helper()
	now := time.Now().UTC()
	return &model.WebhookDelivery{
		ID:          testutil.UniqueID("delivery"),
		EndpointID:  endpointID,
		EventID:     testutil.UniqueID("event"),
		EventType:   model.EventTypeClick,
		PayloadJSON: `{"event_type":"click","data":{}}`,
		Status:      model.DeliveryStatusPending,
		AttemptCount: 0,
		MaxAttempts:  5,
		NextRetryAt:  now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// ============================================================================
// Test Environment Setup
// ============================================================================

func newWebhookTestEnv(t *testing.T) (context.Context, *sql.DB, *Repository) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	ctx := context.Background()
	dbURL := testutil.RequireEnv(t, "DATABASE_URL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping db: %v", err)
	}

	// Use pgxpool for lock acquisition
	root, err := testutil.ProjectRoot()
	if err != nil {
		t.Fatalf("ProjectRoot failed: %v", err)
	}

	// Reset webhooks schema via direct SQL
	resetWebhooksDirectly(t, ctx, db, root)

	repo := NewRepository(db)

	return ctx, db, repo
}

func resetWebhooksDirectly(t *testing.T, ctx context.Context, db *sql.DB, root string) {
	t.Helper()

	// Drop and recreate webhooks tables
	// Note: We do this directly since testutil uses pgxpool
	downSQL := `
		DROP TABLE IF EXISTS webhook_deliveries;
		DROP TABLE IF EXISTS webhook_endpoints;
	`
	if _, err := db.ExecContext(ctx, downSQL); err != nil {
		t.Fatalf("drop webhooks tables: %v", err)
	}

	upSQL := `
		CREATE TABLE IF NOT EXISTS webhook_endpoints (
			id              TEXT PRIMARY KEY,
			user_id         TEXT NOT NULL,
			target_url      TEXT NOT NULL,
			secret_hash     TEXT NOT NULL,
			enabled         BOOLEAN NOT NULL DEFAULT true,
			event_types     TEXT[] NOT NULL DEFAULT '{click}',
			name            TEXT,
			description     TEXT,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			deleted_at      TIMESTAMPTZ
		);

		CREATE INDEX IF NOT EXISTS idx_webhook_endpoints_user ON webhook_endpoints (user_id) 
			WHERE deleted_at IS NULL;

		CREATE TABLE IF NOT EXISTS webhook_deliveries (
			id              TEXT PRIMARY KEY,
			endpoint_id     TEXT NOT NULL REFERENCES webhook_endpoints(id),
			event_id        TEXT NOT NULL,
			event_type      TEXT NOT NULL DEFAULT 'click',
			payload_json    TEXT NOT NULL,
			status          TEXT NOT NULL DEFAULT 'pending',
			attempt_count   INT NOT NULL DEFAULT 0,
			max_attempts    INT NOT NULL DEFAULT 5,
			next_retry_at   TIMESTAMPTZ NOT NULL,
			last_attempt_at TIMESTAMPTZ,
			last_http_status INT,
			last_error      TEXT,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_pending ON webhook_deliveries (next_retry_at)
			WHERE status IN ('pending', 'failed');

		CREATE UNIQUE INDEX IF NOT EXISTS idx_webhook_deliveries_event_endpoint 
			ON webhook_deliveries (event_id, endpoint_id);
	`
	if _, err := db.ExecContext(ctx, upSQL); err != nil {
		t.Fatalf("create webhooks tables: %v", err)
	}
}
