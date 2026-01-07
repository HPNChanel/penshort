-- Phase 3: Users table for API key ownership
-- Migration: 000003_users.up.sql

CREATE TABLE users (
    -- Identity
    id              TEXT PRIMARY KEY,           -- ULID
    email           TEXT UNIQUE NOT NULL,
    
    -- Timestamps
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for email lookup
CREATE INDEX idx_users_email ON users (email);

-- Comments
COMMENT ON TABLE users IS 'Minimal user entity for API key ownership (Phase 3)';
COMMENT ON COLUMN users.id IS 'ULID primary key';
COMMENT ON COLUMN users.email IS 'Unique email identifier';
