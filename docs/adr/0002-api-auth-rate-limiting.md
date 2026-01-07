# ADR-0002: API Key Authentication and Rate Limiting (Phase 3)

**Status**: Accepted  
**Date**: 2026-01-07  
**Deciders**: Security Engineer, API Architect  
**Technical Story**: Phase 3 of Penshort roadmap — API keys and rate limiting

---

## 1. Context and Problem Statement

Penshort is a developer-first URL shortener that requires:
- **Authentication**: Secure API access via API keys per user/team
- **Authorization**: Scope-based access control for endpoints
- **Rate Limiting**: Protection against abuse on both API and redirect endpoints

The system must prevent timing attacks, key enumeration, and information leakage while maintaining high performance for the redirect hot path.

---

## 2. Decision Drivers

- **Security**: No plaintext keys ever; timing-safe comparisons; no key existence leakage
- **Developer Experience**: Clear key format, intuitive scopes, helpful error messages
- **Operational Simplicity**: Small team, single Redis instance, minimal infrastructure
- **Performance**: Rate limiting must not degrade redirect latency significantly

---

## 3. API Key Lifecycle

### 3.1 Key Generation Format

```
Format: pk_{prefix}_{random}
Example: pk_live_7a9x3k_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b
```

| Component | Description | Length |
|-----------|-------------|--------|
| `pk_` | Static prefix identifying Penshort keys | 3 chars |
| `{prefix}` | Environment indicator: `live`, `test` | 4-5 chars |
| `7a9x3k` | Visible prefix for key identification (hex) | 6 chars |
| `_` | Separator | 1 char |
| Random part | Cryptographically random hex string | 32 chars |

**Generation Algorithm**:
```go
func GenerateAPIKey(env string) (plaintext, hash, prefix string, err error) {
    // 1. Generate 6-byte visible prefix (12 hex chars → use first 6)
    prefixBytes := make([]byte, 3)
    if _, err := crypto_rand.Read(prefixBytes); err != nil {
        return "", "", "", err
    }
    prefix = hex.EncodeToString(prefixBytes) // 6 chars
    
    // 2. Generate 16-byte secret (32 hex chars)
    secretBytes := make([]byte, 16)
    if _, err := crypto_rand.Read(secretBytes); err != nil {
        return "", "", "", err
    }
    secret := hex.EncodeToString(secretBytes) // 32 chars
    
    // 3. Assemble plaintext
    plaintext = fmt.Sprintf("pk_%s_%s_%s", env, prefix, secret)
    
    // 4. Hash full plaintext for storage
    hash, err = argon2id.CreateHash(plaintext, argon2id.DefaultParams)
    if err != nil {
        return "", "", "", err
    }
    
    return plaintext, hash, prefix, nil
}
```

> [!IMPORTANT]
> **Display-Once Policy**: The plaintext key is shown ONCE at creation and NEVER again. The API returns 201 with the key; subsequent GETs return only metadata (prefix, scopes, created_at).

### 3.2 Hashing Algorithm: Argon2id

**Rationale**:
| Algorithm | Memory-Hard | GPU Resistant | OWASP Recommended | Notes |
|-----------|------------|---------------|-------------------|-------|
| bcrypt | Partial | Medium | Yes | 72-byte limit problematic for long keys |
| scrypt | Yes | High | Yes | Complex parameter tuning |
| **Argon2id** | Yes | High | Yes (preferred) | Hybrid approach, modern standard |

**Argon2id Parameters** (OWASP 2024 minimum):
```go
var DefaultParams = &argon2id.Params{
    Memory:      64 * 1024,  // 64 MB
    Iterations:  3,
    Parallelism: 4,
    SaltLength:  16,
    KeyLength:   32,
}
```

> [!CAUTION]
> Verification is intentionally slow (~100-300ms). Use Redis caching to avoid repeated hash verification per request.

### 3.3 Database Schema

```sql
-- Migration: 000003_api_keys.up.sql

CREATE TABLE api_keys (
    -- Identity
    id              TEXT PRIMARY KEY,           -- ULID
    user_id         TEXT NOT NULL,              -- Owner reference
    
    -- Key material
    key_hash        TEXT NOT NULL,              -- Argon2id hash
    key_prefix      TEXT NOT NULL,              -- 6-char visible prefix
    
    -- Authorization
    scopes          TEXT[] NOT NULL DEFAULT '{}', -- Array of scope names
    
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
    )
);

-- Indexes
CREATE INDEX idx_api_keys_user_id 
    ON api_keys (user_id) 
    WHERE revoked_at IS NULL;

CREATE INDEX idx_api_keys_prefix 
    ON api_keys (key_prefix) 
    WHERE revoked_at IS NULL;

-- Comments
COMMENT ON TABLE api_keys IS 'API keys for authentication (Phase 3)';
COMMENT ON COLUMN api_keys.key_hash IS 'Argon2id hash of full plaintext key';
COMMENT ON COLUMN api_keys.key_prefix IS 'Visible prefix for key identification';
COMMENT ON COLUMN api_keys.scopes IS 'Authorization scopes: read, write, admin';
```

### 3.4 Key Rotation and Revocation Endpoints

#### Create New API Key
```http
POST /v1/api-keys
Authorization: Bearer <admin_token_or_existing_key_with_admin_scope>
Content-Type: application/json

{
  "name": "Production CI/CD",
  "scopes": ["read", "write"]
}
```

**Response (201 Created)**:
```json
{
  "id": "01HXYZ...",
  "key": "pk_live_7a9x3k_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b",
  "name": "Production CI/CD",
  "key_prefix": "7a9x3k",
  "scopes": ["read", "write"],
  "created_at": "2026-01-07T14:00:00Z"
}
```

> [!WARNING]
> The `key` field is returned ONLY in this response. Store it securely.

#### List API Keys
```http
GET /v1/api-keys
Authorization: Bearer <key>
```

**Response (200 OK)**:
```json
{
  "keys": [
    {
      "id": "01HXYZ...",
      "name": "Production CI/CD",
      "key_prefix": "7a9x3k",
      "scopes": ["read", "write"],
      "created_at": "2026-01-07T14:00:00Z",
      "last_used_at": "2026-01-07T14:25:00Z",
      "revoked": false
    }
  ]
}
```

#### Revoke API Key
```http
DELETE /v1/api-keys/{key_id}
Authorization: Bearer <key_with_admin_scope>
```

**Response (204 No Content)**: Key revoked successfully

**Response (404 Not Found)**:
```json
{
  "error": {
    "code": "KEY_NOT_FOUND",
    "message": "API key not found or already revoked"
  }
}
```

> [!NOTE]
> Returns 404 for both non-existent and already-revoked keys to prevent enumeration.

#### Rotate Key (Atomic Create + Revoke)
```http
POST /v1/api-keys/{key_id}/rotate
Authorization: Bearer <key_with_admin_scope>
```

**Response (201 Created)**:
```json
{
  "old_key_id": "01HXYZ...",
  "old_key_revoked_at": "2026-01-07T15:00:00Z",
  "new_key": {
    "id": "01HABC...",
    "key": "pk_live_9b2m4n_...",
    "key_prefix": "9b2m4n",
    "scopes": ["read", "write"],
    "created_at": "2026-01-07T15:00:00Z"
  }
}
```

---

## 4. Authentication Middleware

### 4.1 Header Format

```http
Authorization: Bearer pk_live_7a9x3k_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b
```

Alternative (for webhook callbacks accepting keys):
```http
X-API-Key: pk_live_7a9x3k_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b
```

### 4.2 Middleware Flow

```
┌──────────────────────────────────────────────────────────────────┐
│                    Authentication Middleware                      │
└──────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│ 1. Extract Key from Header                                        │
│    - Check Authorization: Bearer ...                              │
│    - Fallback: X-API-Key header                                   │
│    - Missing? → 401 UNAUTHORIZED                                  │
└──────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│ 2. Parse Key Format                                               │
│    - Validate pk_{env}_{prefix}_{secret} structure                │
│    - Invalid format? → 401 UNAUTHORIZED (same message as invalid) │
└──────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│ 3. Cache Lookup (Redis)                                           │
│    Key: "apikey:verified:{sha256(plaintext)}"                     │
│    Hit? → Load cached auth context                                │
│    Miss? → Continue to DB verification                            │
└──────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│ 4. Database Lookup                                                │
│    Query by key_prefix (WHERE revoked_at IS NULL)                 │
│    No rows? → 401 UNAUTHORIZED                                    │
│    Multiple rows? → Verify against each (rare collision case)     │
└──────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│ 5. Argon2id Verification (Timing-Safe)                           │
│    - Use constant-time comparison                                 │
│    - Invalid? → 401 UNAUTHORIZED                                  │
│    - Valid? → Update last_used_at (async)                         │
└──────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│ 6. Cache Result (Redis)                                           │
│    Store: key_id, user_id, scopes for 5 minutes                   │
└──────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│ 7. Inject Auth Context                                            │
│    context.WithValue(ctx, authContextKey, AuthContext{...})       │
└──────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                      next.ServeHTTP()
```

### 4.3 Error Responses

#### 401 Unauthorized
Missing, malformed, or invalid API key:
```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Invalid or missing API key"
  }
}
```

> [!IMPORTANT]
> **Timing Attack Prevention**: Always return the SAME error message regardless of whether:
> - Key format is invalid
> - Key prefix doesn't exist
> - Key hash doesn't match
> - Key is revoked
>
> Add artificial delay (~200ms) when rejecting to match successful Argon2 verification time.

#### 403 Forbidden
Valid key but insufficient scope:
```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "Insufficient permissions. Required scope: write"
  }
}
```

### 4.4 Logging Redaction Rules

| Field | Logged As | Example |
|-------|-----------|---------|
| Full key | **NEVER** | ❌ |
| Key prefix | `key_prefix` | `"7a9x3k"` |
| Key ID (ULID) | `key_id` | `"01HXYZ..."` |
| User ID | `user_id` | `"user_123"` |
| Scopes | `scopes` | `["read", "write"]` |

```go
// ❌ BAD - Never do this
logger.Info("auth success", slog.String("key", plaintextKey))

// ✅ GOOD - Log only identifiers
logger.Info("auth success", 
    slog.String("key_id", authCtx.KeyID),
    slog.String("key_prefix", authCtx.KeyPrefix),
    slog.String("user_id", authCtx.UserID),
)
```

---

## 5. Scopes Model

### 5.1 Scope Definitions

| Scope | Description | Allowed Actions |
|-------|-------------|-----------------|
| `read` | Read-only access | GET all resources, list links |
| `write` | Create/modify access | POST, PUT, PATCH links |
| `admin` | Full control | All above + DELETE + manage API keys |

### 5.2 Endpoint-to-Scope Mapping

| Endpoint | Method | Required Scope | Notes |
|----------|--------|----------------|-------|
| `GET /v1/links` | GET | `read` | List links |
| `GET /v1/links/{id}` | GET | `read` | Get single link |
| `POST /v1/links` | POST | `write` | Create link |
| `PUT /v1/links/{id}` | PUT | `write` | Update link |
| `PATCH /v1/links/{id}` | PATCH | `write` | Partial update |
| `DELETE /v1/links/{id}` | DELETE | `admin` | Soft-delete |
| `GET /v1/api-keys` | GET | `read` | List own keys |
| `POST /v1/api-keys` | POST | `admin` | Create new key |
| `DELETE /v1/api-keys/{id}` | DELETE | `admin` | Revoke key |
| `POST /v1/api-keys/{id}/rotate` | POST | `admin` | Rotate key |
| `GET /{short_code}` | GET | **None** | Public redirect |
| `GET /healthz` | GET | **None** | Health check |
| `GET /readyz` | GET | **None** | Readiness check |

### 5.3 Scope Check Middleware

```go
func RequireScope(required ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            authCtx := GetAuthContext(r.Context())
            if authCtx == nil {
                writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
                return
            }
            
            // Check if any required scope is present
            for _, req := range required {
                if slices.Contains(authCtx.Scopes, req) {
                    next.ServeHTTP(w, r)
                    return
                }
                // Admin implies all scopes
                if slices.Contains(authCtx.Scopes, "admin") {
                    next.ServeHTTP(w, r)
                    return
                }
            }
            
            writeError(w, http.StatusForbidden, "FORBIDDEN", 
                fmt.Sprintf("Insufficient permissions. Required scope: %s", required[0]))
        })
    }
}
```

---

## 6. Rate Limiting

### 6.1 Algorithm: Token Bucket

**Rationale**:
| Algorithm | Burst Handling | Implementation | Fairness |
|-----------|---------------|----------------|----------|
| Fixed Window | Poor (edge spikes) | Simple | Low |
| Sliding Window | Good | Complex | Medium |
| Leaky Bucket | Strict | Medium | High but no burst |
| **Token Bucket** | Good (controlled burst) | Medium | High |

Token Bucket allows controlled bursts while maintaining average rate limits.

### 6.2 Rate Limit Tiers

#### Per-API-Key Limits (API Endpoints)

| Tier | Requests/Minute | Burst | Use Case |
|------|-----------------|-------|----------|
| Free | 60 | 10 | Default for all keys |
| Pro | 600 | 50 | Future paid tier |
| Unlimited | ∞ | ∞ | Internal/admin |

#### Per-IP Limits (Redirect Endpoint)

| Limit | Value | Purpose |
|-------|-------|---------|
| Requests/second | 100 | Prevent DDoS on redirect path |
| Burst | 20 | Allow legitimate traffic spikes |

### 6.3 Redis Key Schema

```
Rate Limit Keys:
┌────────────────────────────────────────────────────────────────┐
│ Per-API-Key:                                                    │
│   Key:    ratelimit:apikey:{key_id}                            │
│   Type:   Hash                                                  │
│   Fields: tokens (float), last_update (unix_timestamp)          │
│   TTL:    120 seconds (auto-cleanup)                            │
├────────────────────────────────────────────────────────────────┤
│ Per-IP (Redirect):                                              │
│   Key:    ratelimit:ip:{ip_hash}                               │
│   Type:   Hash                                                  │
│   Fields: tokens (float), last_update (unix_timestamp)          │
│   TTL:    10 seconds (short-lived)                              │
└────────────────────────────────────────────────────────────────┘

IP Hashing:
  - Use SHA256(client_ip) to avoid storing raw IPs
  - Truncate to 16 chars for key brevity
```

### 6.4 Token Bucket Pseudocode

```go
type RateLimiter struct {
    redis  *redis.Client
    rate   float64       // tokens per second
    burst  int           // max tokens (bucket capacity)
}

func (rl *RateLimiter) Allow(key string) (allowed bool, retryAfter time.Duration) {
    ctx := context.Background()
    now := time.Now().Unix()
    
    // Lua script for atomic token bucket
    script := redis.NewScript(`
        local key = KEYS[1]
        local rate = tonumber(ARGV[1])
        local burst = tonumber(ARGV[2])
        local now = tonumber(ARGV[3])
        local ttl = tonumber(ARGV[4])
        
        -- Get current state
        local data = redis.call('HMGET', key, 'tokens', 'last_update')
        local tokens = tonumber(data[1]) or burst
        local last_update = tonumber(data[2]) or now
        
        -- Refill tokens based on elapsed time
        local elapsed = now - last_update
        tokens = math.min(burst, tokens + (elapsed * rate))
        
        -- Check if request is allowed
        local allowed = 0
        local retry_after = 0
        
        if tokens >= 1 then
            tokens = tokens - 1
            allowed = 1
        else
            -- Calculate when 1 token will be available
            retry_after = math.ceil((1 - tokens) / rate)
        end
        
        -- Update state
        redis.call('HMSET', key, 'tokens', tokens, 'last_update', now)
        redis.call('EXPIRE', key, ttl)
        
        return {allowed, retry_after, math.floor(tokens)}
    `)
    
    result, err := script.Run(ctx, rl.redis, 
        []string{key}, 
        rl.rate, rl.burst, now, 120,
    ).Slice()
    
    if err != nil {
        // Fail open on Redis errors (allow request)
        return true, 0
    }
    
    allowed = result[0].(int64) == 1
    retryAfterSec := result[1].(int64)
    remaining := result[2].(int64)
    
    return allowed, time.Duration(retryAfterSec) * time.Second
}
```

### 6.5 Rate Limit Headers

**Successful Requests (2xx/3xx/4xx)**:
```http
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1704639600
```

**Rate Limited (429)**:
```http
HTTP/1.1 429 Too Many Requests
Retry-After: 5
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1704639600
Content-Type: application/json

{
  "error": {
    "code": "RATE_LIMITED",
    "message": "Rate limit exceeded. Retry after 5 seconds."
  }
}
```

### 6.6 Configuration

```yaml
# config/rate_limit.yaml (or environment variables)
rate_limit:
  api:
    enabled: true
    default_requests_per_minute: 60
    default_burst: 10
  redirect:
    enabled: true
    requests_per_second: 100
    burst: 20
```

Environment variables:
```bash
RATE_LIMIT_API_ENABLED=true
RATE_LIMIT_API_RPM=60
RATE_LIMIT_API_BURST=10
RATE_LIMIT_REDIRECT_ENABLED=true
RATE_LIMIT_REDIRECT_RPS=100
RATE_LIMIT_REDIRECT_BURST=20
```

---

## 7. Observability

### 7.1 Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `penshort_auth_total` | Counter | `result=[success\|failure\|invalid_format]` | Authentication attempts |
| `penshort_auth_cache_hits_total` | Counter | - | Auth context cache hits |
| `penshort_rate_limit_total` | Counter | `result=[allowed\|rejected]`, `type=[api\|redirect]` | Rate limit decisions |
| `penshort_rate_limit_tokens` | Gauge | `key_id` | Current token count (sampled) |

**Prometheus Format**:
```
# HELP penshort_auth_total Total authentication attempts
# TYPE penshort_auth_total counter
penshort_auth_total{result="success"} 1523
penshort_auth_total{result="failure"} 42
penshort_auth_total{result="invalid_format"} 7

# HELP penshort_rate_limit_total Total rate limit decisions
# TYPE penshort_rate_limit_total counter
penshort_rate_limit_total{result="allowed",type="api"} 45210
penshort_rate_limit_total{result="rejected",type="api"} 127
penshort_rate_limit_total{result="allowed",type="redirect"} 892341
penshort_rate_limit_total{result="rejected",type="redirect"} 53
```

### 7.2 Logs

**Authentication Success**:
```json
{
  "level": "info",
  "msg": "authentication successful",
  "key_id": "01HXYZ...",
  "key_prefix": "7a9x3k",
  "user_id": "user_123",
  "scopes": ["read", "write"],
  "ip": "192.168.1.100",
  "endpoint": "POST /v1/links",
  "cache_hit": true,
  "request_id": "req_abc123"
}
```

**Authentication Failure**:
```json
{
  "level": "warn",
  "msg": "authentication failed",
  "reason": "invalid_key",
  "ip": "192.168.1.100",
  "endpoint": "POST /v1/links",
  "request_id": "req_def456"
}
```

> [!NOTE]
> Log `reason` is generic to prevent timing analysis. Possible values: `missing_key`, `invalid_format`, `invalid_key` (covers not found, revoked, wrong hash).

**Rate Limited**:
```json
{
  "level": "warn",
  "msg": "rate limit exceeded",
  "key_id": "01HXYZ...",
  "type": "api",
  "ip": "192.168.1.100",
  "endpoint": "POST /v1/links",
  "retry_after_seconds": 5,
  "request_id": "req_ghi789"
}
```

---

## 8. Security Considerations

### 8.1 Timing Attack Prevention

```go
// Always take constant time regardless of failure reason
func (m *AuthMiddleware) authenticate(key string) (*AuthContext, error) {
    startTime := time.Now()
    defer func() {
        elapsed := time.Since(startTime)
        // Ensure minimum processing time (~200ms)
        if elapsed < 200*time.Millisecond {
            time.Sleep(200*time.Millisecond - elapsed)
        }
    }()
    
    // ... authentication logic ...
}
```

### 8.2 Key Enumeration Prevention

- **Same error for all failures**: 401 with identical message
- **Prefix collision handled**: Query by prefix, verify each match
- **Revoked keys**: Treated as non-existent (404 on management endpoints)

### 8.3 Information Leakage Prevention

- **Logs**: Never log plaintext keys; only key_id and prefix
- **Traces**: Redact Authorization header in distributed traces
- **Error responses**: No details about why authentication failed

### 8.4 Log and Trace Protection

```go
// Middleware to redact sensitive headers before tracing
func RedactSensitiveHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Clone headers for tracing
        safeHeaders := make(http.Header)
        for k, v := range r.Header {
            if k == "Authorization" || k == "X-Api-Key" {
                safeHeaders[k] = []string{"[REDACTED]"}
            } else {
                safeHeaders[k] = v
            }
        }
        // Use safeHeaders for tracing span attributes
        next.ServeHTTP(w, r)
    })
}
```

---

## 9. Domain Model Changes

### 9.1 New Types

```go
// internal/model/apikey.go

package model

import "time"

// APIKey represents an API key entity.
type APIKey struct {
    ID         string     `json:"id"`
    UserID     string     `json:"user_id"`
    KeyHash    string     `json:"-"`              // Never serialize
    KeyPrefix  string     `json:"key_prefix"`
    Scopes     []string   `json:"scopes"`
    Name       string     `json:"name,omitempty"`
    RevokedAt  *time.Time `json:"revoked_at,omitempty"`
    LastUsedAt *time.Time `json:"last_used_at,omitempty"`
    CreatedAt  time.Time  `json:"created_at"`
}

// IsRevoked returns true if the key has been revoked.
func (k *APIKey) IsRevoked() bool {
    return k.RevokedAt != nil
}

// HasScope checks if the key has a specific scope.
func (k *APIKey) HasScope(scope string) bool {
    for _, s := range k.Scopes {
        if s == scope || s == "admin" {
            return true
        }
    }
    return false
}

// AuthContext holds authenticated request context.
type AuthContext struct {
    KeyID     string
    KeyPrefix string
    UserID    string
    Scopes    []string
}
```

### 9.2 Link Model Update

The existing `owner_id` column in the `links` table will be populated with the authenticated user's ID:

```go
// In link service, during creation
func (s *LinkService) Create(ctx context.Context, req CreateLinkRequest) (*Link, error) {
    authCtx := model.GetAuthContext(ctx)
    if authCtx == nil {
        return nil, ErrUnauthorized
    }
    
    link := &model.Link{
        OwnerID: authCtx.UserID,  // Now populated from auth context
        // ... other fields
    }
    // ...
}
```

---

## 10. Acceptance Criteria

### 10.1 API Key Management

- [ ] **AK-01**: Generate keys in format `pk_{env}_{prefix}_{secret}`
- [ ] **AK-02**: Plaintext key returned ONLY on creation (201)
- [ ] **AK-03**: Key hash stored using Argon2id
- [ ] **AK-04**: List keys returns only metadata (no hashes/secrets)
- [ ] **AK-05**: Revoked keys cannot authenticate
- [ ] **AK-06**: Rotation atomically creates new + revokes old

### 10.2 Authentication

- [ ] **AU-01**: Missing/invalid key returns 401
- [ ] **AU-02**: Same error message for all auth failures
- [ ] **AU-03**: Minimum 200ms response time for auth failures
- [ ] **AU-04**: Successful auth injects context with key_id, user_id, scopes
- [ ] **AU-05**: Verified keys cached in Redis for 5 minutes
- [ ] **AU-06**: last_used_at updated asynchronously

### 10.3 Authorization

- [ ] **AZ-01**: Endpoints enforce scope requirements
- [ ] **AZ-02**: Insufficient scope returns 403
- [ ] **AZ-03**: Admin scope grants all permissions
- [ ] **AZ-04**: Public endpoints (redirect, health) require no auth

### 10.4 Rate Limiting

- [ ] **RL-01**: API endpoints rate limited per key
- [ ] **RL-02**: Redirect endpoint rate limited per IP
- [ ] **RL-03**: Token bucket allows controlled bursts
- [ ] **RL-04**: 429 response includes Retry-After header
- [ ] **RL-05**: Rate limit headers on all responses
- [ ] **RL-06**: Redis failure → fail open (allow request)

### 10.5 Security

- [ ] **SC-01**: Plaintext keys never logged
- [ ] **SC-02**: Authorization header redacted in traces
- [ ] **SC-03**: No timing differences between failure modes
- [ ] **SC-04**: Revoked/non-existent keys indistinguishable

---

## 11. Test Plan

### 11.1 Unit Tests

| Test Suite | Coverage | Command |
|------------|----------|---------|
| Key generation | Format, uniqueness | `go test ./internal/auth -run TestKeyGeneration` |
| Argon2id hashing | Hash/verify roundtrip | `go test ./internal/auth -run TestArgon2` |
| Scope checking | HasScope logic | `go test ./internal/model -run TestScopes` |
| Token bucket | Refill, consume, burst | `go test ./internal/ratelimit -run TestTokenBucket` |

### 11.2 Integration Tests

| Test | Description | Command |
|------|-------------|---------|
| Auth middleware E2E | Full request cycle with Redis | `go test ./internal/middleware -run TestAuthMiddleware -tags=integration` |
| Rate limit Redis | Token bucket with real Redis | `go test ./internal/ratelimit -run TestRateLimitRedis -tags=integration` |
| API key CRUD | Create, list, revoke via API | `go test ./internal/handler -run TestAPIKeyHandlers -tags=integration` |

### 11.3 Concurrency Tests

```go
// Test concurrent rate limit accuracy
func TestRateLimitConcurrency(t *testing.T) {
    // Spawn 100 goroutines, each making 10 requests
    // Verify total allowed ≈ expected (burst + rate*time)
}

// Test concurrent key verification
func TestAuthConcurrency(t *testing.T) {
    // Spawn 50 goroutines authenticating same key
    // Verify no race conditions, cache consistency
}
```

**Command**: `go test ./... -race -tags=integration -run Concurrency`

### 11.4 Security Tests

| Test | Description |
|------|-------------|
| Timing analysis | Measure 1000 requests per failure mode; verify variance < 10% |
| Key enumeration | Verify same response for non-existent vs revoked |
| Header redaction | Confirm Authorization not in traces |

### 11.5 Load Tests (Manual)

```bash
# Using k6 or hey
hey -n 10000 -c 100 -H "Authorization: Bearer pk_live_..." http://localhost:8080/v1/links

# Expected: 
# - First burst of 10 succeeds immediately
# - Subsequent requests at ~1/second
# - 429s after burst exhausted
```

---

## 12. File Structure

```
internal/
├── auth/
│   ├── argon2.go           # Argon2id hashing wrapper
│   ├── argon2_test.go
│   ├── keygen.go           # Key generation logic
│   ├── keygen_test.go
│   └── context.go          # Auth context helpers
├── middleware/
│   ├── auth.go             # Authentication middleware
│   ├── auth_test.go
│   ├── ratelimit.go        # Rate limiting middleware
│   ├── ratelimit_test.go
│   └── scope.go            # Scope enforcement
├── model/
│   ├── apikey.go           # APIKey and AuthContext types
│   └── apikey_test.go
├── repository/
│   ├── apikey.go           # DB operations for api_keys
│   └── apikey_test.go
├── cache/
│   ├── auth.go             # Redis cache for auth contexts
│   └── ratelimit.go        # Redis rate limit state
├── handler/
│   ├── apikey.go           # API key management handlers
│   └── apikey_test.go
migrations/
└── 000003_api_keys.up.sql
└── 000003_api_keys.down.sql
```

---

## 13. Dependencies

```go
// go.mod additions
require (
    github.com/alexedwards/argon2id v1.0.0  // Argon2id wrapper
    // redis already present
)
```

---

## 14. Resolved Design Decisions

> [!NOTE]
> The following decisions were made to maintain simplicity while allowing future extensibility.

### 14.1 User Management: Lightweight Users Table

**Decision**: Create a minimal `users` table in Phase 3 to own API keys.

**Rationale**:
- Keeps system self-contained without external OAuth dependency
- Allows future expansion to team/organization ownership
- Simple migration path if external identity is added later

```sql
-- Migration: 000003_users.up.sql (created before api_keys)

CREATE TABLE users (
    id              TEXT PRIMARY KEY,           -- ULID
    email           TEXT UNIQUE NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE users IS 'Minimal user entity for key ownership (Phase 3)';
```

> [!TIP]
> For MVP, users can be created via admin API or seed script. Authentication of users themselves (login) is out of scope for Phase 3.

---

### 14.2 Rate Limit Tiers: Stored on API Key

**Decision**: Store `rate_limit_tier` on the `api_keys` table.

**Rationale**:
- Decouples rate limiting from user subscription logic (which doesn't exist yet)
- Allows per-key granularity (e.g., different limits for CI vs production keys)
- Simple to query during rate limit checks

**Schema Addition**:
```sql
ALTER TABLE api_keys ADD COLUMN rate_limit_tier TEXT NOT NULL DEFAULT 'free'
    CONSTRAINT chk_rate_limit_tier CHECK (rate_limit_tier IN ('free', 'pro', 'unlimited'));
```

**Tier Defaults**:
| Tier | Requests/Minute | Burst |
|------|-----------------|-------|
| `free` | 60 | 10 |
| `pro` | 600 | 50 |
| `unlimited` | No limit | No limit |

---

### 14.3 Webhook Keys: Shared Scopes Model

**Decision**: Webhooks use the same API key + scope model. Add `webhook` scope.

**Rationale**:
- Avoids separate authentication mechanism for webhooks
- Consistent developer experience
- Keys with `webhook` scope can configure webhook endpoints

**Updated Scopes**:
| Scope | Description |
|-------|-------------|
| `read` | Read-only access to resources |
| `write` | Create/modify links |
| `webhook` | Configure and manage webhooks (Phase 4) |
| `admin` | Full control (implies all scopes) |

**Constraint Update**:
```sql
ALTER TABLE api_keys DROP CONSTRAINT chk_scopes_valid;
ALTER TABLE api_keys ADD CONSTRAINT chk_scopes_valid CHECK (
    scopes <@ ARRAY['read', 'write', 'webhook', 'admin']::TEXT[]
);
```

---

## 15. References

- [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
- [RFC 6750: Bearer Token Usage](https://tools.ietf.org/html/rfc6750)
- [Token Bucket Algorithm](https://en.wikipedia.org/wiki/Token_bucket)
- [Argon2 RFC 9106](https://www.rfc-editor.org/rfc/rfc9106)
