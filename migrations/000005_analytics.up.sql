-- Phase 4: Analytics tables for click event tracking
-- Migration: 000005_analytics.up.sql

-- ============================================================================
-- CLICK EVENTS TABLE (Raw Events)
-- ============================================================================
CREATE TABLE click_events (
    -- Identity
    id              TEXT PRIMARY KEY,                 -- ULID
    event_id        TEXT NOT NULL UNIQUE,             -- Redis stream ID (idempotency)
    
    -- Foreign keys
    short_code      TEXT NOT NULL,                    -- Link short code
    link_id         TEXT NOT NULL,                    -- FK to links.id
    
    -- Request metadata
    referrer        TEXT,                             -- Truncated to 500 chars
    user_agent      TEXT,                             -- Truncated to 500 chars
    
    -- Visitor identification (privacy-safe)
    visitor_hash    TEXT NOT NULL,                    -- SHA256(IP+UA+salt)[0:16]
    
    -- Optional geo
    country_code    CHAR(2),                          -- ISO 3166-1 alpha-2
    
    -- Timestamps
    clicked_at      TIMESTAMPTZ NOT NULL,             -- Event time
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW() -- Insertion time
);

-- Index for querying by link and time range
CREATE INDEX idx_click_events_link_time 
    ON click_events (link_id, clicked_at DESC);

-- Index for short_code lookup (joins with links)
CREATE INDEX idx_click_events_short_code_time 
    ON click_events (short_code, clicked_at DESC);

-- Index for aggregation queries by day
CREATE INDEX idx_click_events_clicked_at 
    ON click_events (DATE(clicked_at), link_id);

-- ============================================================================
-- DAILY LINK STATS TABLE (Pre-Aggregated)
-- ============================================================================
CREATE TABLE daily_link_stats (
    id              TEXT PRIMARY KEY,                 -- Composite: link_id:date
    link_id         TEXT NOT NULL,
    date            DATE NOT NULL,                    -- UTC date
    
    -- Counters
    total_clicks    BIGINT NOT NULL DEFAULT 0,
    unique_visitors BIGINT NOT NULL DEFAULT 0,
    
    -- Breakdown (JSONB for flexibility)
    referrer_breakdown  JSONB DEFAULT '{}',           -- {"google.com": 42, "direct": 100}
    ua_family_breakdown JSONB DEFAULT '{}',           -- {"Chrome": 50, "Safari": 30}
    country_breakdown   JSONB DEFAULT '{}',           -- {"US": 100, "VN": 50}
    
    -- Timestamps
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT uq_daily_link_stats UNIQUE (link_id, date)
);

-- Index for fast lookups by link and date range
CREATE INDEX idx_daily_link_stats_link_date 
    ON daily_link_stats (link_id, date DESC);

-- ============================================================================
-- COMMENTS
-- ============================================================================
COMMENT ON TABLE click_events IS 'Raw click events for analytics (Phase 4)';
COMMENT ON COLUMN click_events.event_id IS 'Redis stream ID for idempotency';
COMMENT ON COLUMN click_events.visitor_hash IS 'SHA256(IP+UA+daily_salt)[0:16], rotates daily';
COMMENT ON COLUMN click_events.country_code IS 'ISO 3166-1 alpha-2 from CF-IPCountry or IP lookup';

COMMENT ON TABLE daily_link_stats IS 'Pre-aggregated daily statistics for fast analytics queries';
COMMENT ON COLUMN daily_link_stats.referrer_breakdown IS 'JSON object: referrer domain → click count';
COMMENT ON COLUMN daily_link_stats.ua_family_breakdown IS 'JSON object: UA family → click count';
