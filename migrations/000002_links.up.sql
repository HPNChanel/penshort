-- Phase 2: Links table for URL shortening
-- Migration: 000002_links.up.sql

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Links table
CREATE TABLE links (
    id              TEXT PRIMARY KEY,  -- ULID
    short_code      TEXT NOT NULL,
    destination     TEXT NOT NULL,
    redirect_type   SMALLINT NOT NULL DEFAULT 302,
    
    -- Ownership (placeholder for Phase 3)
    owner_id        TEXT NOT NULL DEFAULT 'system',
    
    -- State
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at      TIMESTAMPTZ,
    deleted_at      TIMESTAMPTZ,
    
    -- Counters (denormalized for read performance)
    click_count     BIGINT NOT NULL DEFAULT 0,
    
    -- Timestamps
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT chk_redirect_type CHECK (redirect_type IN (301, 302)),
    CONSTRAINT chk_destination_length CHECK (LENGTH(destination) <= 2048),
    CONSTRAINT chk_short_code_length CHECK (LENGTH(short_code) BETWEEN 3 AND 50)
);

-- Unique constraint on short_code (global uniqueness, only non-deleted)
CREATE UNIQUE INDEX idx_links_short_code 
    ON links (short_code) 
    WHERE deleted_at IS NULL;

-- Fast lookup for redirect (hot path)
CREATE INDEX idx_links_redirect_lookup 
    ON links (short_code, enabled, deleted_at, expires_at)
    WHERE deleted_at IS NULL;

-- List by owner with pagination
CREATE INDEX idx_links_owner_created 
    ON links (owner_id, created_at DESC)
    WHERE deleted_at IS NULL;

-- Cleanup job index (find expired links)
CREATE INDEX idx_links_expires_at 
    ON links (expires_at) 
    WHERE expires_at IS NOT NULL AND deleted_at IS NULL;

-- Updated timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_links_updated_at
    BEFORE UPDATE ON links
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments for documentation
COMMENT ON TABLE links IS 'Short URL links with redirect configuration';
COMMENT ON COLUMN links.id IS 'ULID primary key for time-sortable IDs';
COMMENT ON COLUMN links.short_code IS 'Unique short code for URL (the slug)';
COMMENT ON COLUMN links.destination IS 'Target URL to redirect to';
COMMENT ON COLUMN links.redirect_type IS '301 (permanent) or 302 (temporary)';
COMMENT ON COLUMN links.owner_id IS 'Owner identifier (system for Phase 2)';
COMMENT ON COLUMN links.enabled IS 'Whether link is active for redirects';
COMMENT ON COLUMN links.expires_at IS 'NULL means never expires';
COMMENT ON COLUMN links.deleted_at IS 'Soft delete timestamp';
COMMENT ON COLUMN links.click_count IS 'Denormalized counter; async increment';
