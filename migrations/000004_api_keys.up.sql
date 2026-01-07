-- Phase 3: API Keys table for authentication
-- Migration: 000004_api_keys.up.sql

CREATE TABLE api_keys (
    -- Identity
    id              TEXT PRIMARY KEY,           -- ULID
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Key material
    key_hash        TEXT NOT NULL,              -- Argon2id hash
    key_prefix      TEXT NOT NULL,              -- 6-char visible prefix for identification
    
    -- Authorization
    scopes          TEXT[] NOT NULL DEFAULT '{}',
    
    -- Rate limiting
    rate_limit_tier TEXT NOT NULL DEFAULT 'free',
    
    -- State
    revoked_at      TIMESTAMPTZ,                -- NULL = active
    
    -- Audit
    last_used_at    TIMESTAMPTZ,                -- Updated on successful auth
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Metadata
    name            TEXT,                       -- Optional friendly name
    
    -- Constraints
    CONSTRAINT chk_key_prefix_length CHECK (LENGTH(key_prefix) = 6),
    CONSTRAINT chk_scopes_valid CHECK (
        scopes <@ ARRAY['read', 'write', 'webhook', 'admin']::TEXT[]
    ),
    CONSTRAINT chk_rate_limit_tier CHECK (
        rate_limit_tier IN ('free', 'pro', 'unlimited')
    )
);

-- Index for user's active keys
CREATE INDEX idx_api_keys_user_id 
    ON api_keys (user_id) 
    WHERE revoked_at IS NULL;

-- Index for prefix lookup during authentication
CREATE INDEX idx_api_keys_prefix 
    ON api_keys (key_prefix) 
    WHERE revoked_at IS NULL;

-- Comments
COMMENT ON TABLE api_keys IS 'API keys for authentication (Phase 3)';
COMMENT ON COLUMN api_keys.id IS 'ULID primary key';
COMMENT ON COLUMN api_keys.user_id IS 'Owner user reference';
COMMENT ON COLUMN api_keys.key_hash IS 'Argon2id hash of full plaintext key';
COMMENT ON COLUMN api_keys.key_prefix IS 'Visible 6-char prefix for key identification';
COMMENT ON COLUMN api_keys.scopes IS 'Authorization scopes: read, write, webhook, admin';
COMMENT ON COLUMN api_keys.rate_limit_tier IS 'Rate limit tier: free (60/min), pro (600/min), unlimited';
COMMENT ON COLUMN api_keys.revoked_at IS 'NULL means active; set to revoke key';
COMMENT ON COLUMN api_keys.last_used_at IS 'Last successful authentication timestamp';
COMMENT ON COLUMN api_keys.name IS 'Optional friendly name for key identification';
