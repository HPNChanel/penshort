-- Phase 5: Webhook system tables
-- Migration: 000006_webhooks.up.sql

-- ============================================================================
-- WEBHOOK ENDPOINTS TABLE (Configuration)
-- ============================================================================
CREATE TABLE webhook_endpoints (
    id              TEXT PRIMARY KEY,                 -- ULID
    user_id         TEXT NOT NULL,                    -- Owner (FK to users.id)
    
    -- Configuration
    target_url      TEXT NOT NULL,                    -- HTTPS endpoint URL
    secret_hash     TEXT NOT NULL,                    -- SHA256(secret), never store plaintext
    enabled         BOOLEAN NOT NULL DEFAULT true,
    
    -- Event filtering
    event_types     TEXT[] NOT NULL DEFAULT '{click}', -- Subscribed events
    
    -- Metadata
    name            TEXT,                             -- Human-readable label
    description     TEXT,
    
    -- Lifecycle
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ                       -- Soft delete
);

-- Index for user lookups
CREATE INDEX idx_webhook_endpoints_user ON webhook_endpoints (user_id) 
    WHERE deleted_at IS NULL;

-- Index for enabled endpoints (delivery queries)
CREATE INDEX idx_webhook_endpoints_active ON webhook_endpoints (enabled) 
    WHERE deleted_at IS NULL AND enabled = true;


-- ============================================================================
-- WEBHOOK DELIVERIES TABLE (Outbox/Job Queue)
-- ============================================================================
CREATE TABLE webhook_deliveries (
    id              TEXT PRIMARY KEY,                 -- ULID
    
    -- References
    endpoint_id     TEXT NOT NULL REFERENCES webhook_endpoints(id),
    event_id        TEXT NOT NULL,                    -- Source event ID (click_events.id)
    event_type      TEXT NOT NULL DEFAULT 'click',
    
    -- Payload (stored for retry)
    payload_json    TEXT NOT NULL,                    -- Signed payload JSON
    
    -- Delivery state
    status          TEXT NOT NULL DEFAULT 'pending',  -- pending, success, failed, exhausted
    attempt_count   INT NOT NULL DEFAULT 0,
    max_attempts    INT NOT NULL DEFAULT 5,
    
    -- Timing
    next_retry_at   TIMESTAMPTZ NOT NULL,             -- Next delivery attempt
    last_attempt_at TIMESTAMPTZ,
    
    -- Error tracking (for debugging, sanitized)
    last_http_status INT,                             -- HTTP status code
    last_error      TEXT,                             -- Error message (truncated 500)
    
    -- Timestamps
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for pending deliveries (worker polling)
CREATE INDEX idx_webhook_deliveries_pending ON webhook_deliveries (next_retry_at)
    WHERE status IN ('pending', 'failed');

-- Index for endpoint delivery history
CREATE INDEX idx_webhook_deliveries_endpoint ON webhook_deliveries (endpoint_id, created_at DESC);

-- Index for event lookup (prevent duplicates)
CREATE UNIQUE INDEX idx_webhook_deliveries_event_endpoint 
    ON webhook_deliveries (event_id, endpoint_id);


-- ============================================================================
-- COMMENTS
-- ============================================================================
COMMENT ON TABLE webhook_endpoints IS 'Webhook configuration per user/team (Phase 5)';
COMMENT ON COLUMN webhook_endpoints.secret_hash IS 'SHA256 of secret, used for payload signing';
COMMENT ON COLUMN webhook_endpoints.event_types IS 'Array of event types to subscribe to';

COMMENT ON TABLE webhook_deliveries IS 'Outbox pattern for reliable webhook delivery';
COMMENT ON COLUMN webhook_deliveries.status IS 'pending → success | failed → exhausted';
COMMENT ON COLUMN webhook_deliveries.payload_json IS 'Full signed payload for retry consistency';
