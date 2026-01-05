# Penshort — Product Requirements Document v1

> **Version**: 1.0.0  
> **Last Updated**: 2026-01-05  
> **Status**: Approved for Implementation

---

## 1. Executive Summary

Penshort is a **developer-focused URL shortener** optimized for API-first workflows. Unlike generic consumer shorteners, Penshort provides programmatic control over short links, detailed analytics, webhook notifications, and team-scoped API key management — all built for integration into developer toolchains.

### Target Release

**Production-Ready Q1 2026** — A fully functional URL shortener with:
- Sub-50ms redirect latency (p95)
- 99.5% availability
- Zero click event data loss
- Signed webhook delivery with retries

---

## 2. Personas

| Persona | Description | Primary Jobs-to-be-Done |
|---------|-------------|------------------------|
| **Solo Developer** | Individual developer building internal tools, side projects, or MVPs | Programmatic link creation, quick analytics checks, API-first integration |
| **DevOps/Platform Engineer** | Team member managing developer experience tooling | Bulk link management, monitoring integrations, webhook notifications for automation |
| **API Consumer (CI/CD)** | Automated pipelines generating short links | Machine-to-machine auth via API keys, rate-limit-aware retries, structured error responses |
| **Team Lead / Tech Manager** | Oversees developer tooling for a team | Usage visibility across team, API key lifecycle management, audit trail |

---

## 3. Core Features

### 3.1 Link Management

| Attribute | Specification |
|-----------|---------------|
| **Create** | `destination` (required, HTTPS/HTTP), `alias` (optional, 3-50 chars, alphanumeric + hyphen), `redirect_type` (301/302, default: 302), `expires_at` (ISO8601), `max_clicks` (optional) |
| **Read** | Returns full link object including click count, creation date, status |
| **Update** | Mutable: `destination`, `redirect_type`, `expires_at`, `max_clicks`, `enabled` |
| **Delete** | Soft delete (set `deleted_at`), redirect returns 410 Gone |
| **List** | Paginated (cursor-based), filterable by `status`, `created_after`, `created_before` |

**API Endpoint**: `POST/GET/PATCH/DELETE /api/v1/links`

### 3.2 Redirect Behavior

| Scenario | Behavior |
|----------|----------|
| Happy path | Redis lookup → 301/302 redirect within 50ms p95 |
| Cache miss | PostgreSQL fallback → backfill Redis → redirect |
| Expired (time) | Return `410 Gone` with `{"error": "link_expired", "code": "LINK_EXPIRED"}` |
| Expired (clicks) | Same as time expiration |
| Disabled | Return `404 Not Found` |
| Not found | Return `404 Not Found` |

**API Endpoint**: `GET /{short_code}`

### 3.3 Analytics

| Metric | Definition |
|--------|------------|
| **Total clicks** | Count of all click events for a link |
| **Unique clicks** | Distinct visitors per day, keyed by `SHA256(IP + User-Agent + daily_salt)` |
| **Breakdown fields** | `timestamp`, `referrer`, `user_agent_family`, `country_code` (optional) |

**API Endpoint**: `GET /api/v1/links/{id}/analytics?from=...&to=...`

### 3.4 Webhooks

| Aspect | Specification |
|--------|---------------|
| **Trigger** | Click event fires webhook |
| **Payload** | `{"event": "click", "link_id": "...", "short_code": "...", "timestamp": "...", "visitor": {...}}` |
| **Signing** | HMAC-SHA256 in `X-Penshort-Signature` header |
| **Format** | `t={timestamp},v1={signature}` |
| **Clock skew** | ±5 minutes tolerance |
| **Retry policy** | Exponential backoff: 1s, 2s, 4s, 8s, 16s (5 attempts max) |
| **States** | `pending`, `delivered`, `failed`, `dead_letter` |

**API Endpoints**: `POST/GET/PATCH/DELETE /api/v1/webhooks`

### 3.5 Authentication & Rate Limiting

| Aspect | Specification |
|--------|---------------|
| **API Key format** | `psk_live_{32_random_chars}` (production) / `psk_test_{32_random_chars}` (test) |
| **Key storage** | Only prefix + bcrypt hash stored; raw key shown once at creation |
| **Rate limit (API)** | Default 100 req/s per API key; configurable |
| **Rate limit (redirect)** | Optional IP-based: 1000 req/min |
| **Algorithm** | Token bucket (Redis-backed) |
| **Headers** | `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` |

**API Endpoint**: `POST/GET/DELETE /api/v1/api-keys`

### 3.6 Admin/Ops Endpoints

| Endpoint | Purpose | Response |
|----------|---------|----------|
| `GET /health` | Liveness probe | `{"status": "ok"}` |
| `GET /ready` | Readiness probe | `{"status": "ready", "postgres": "ok", "redis": "ok"}` |
| `GET /metrics` | Prometheus metrics | Prometheus exposition format |

---

## 4. Required Metrics (Prometheus)

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `penshort_http_requests_total` | Counter | `method`, `path`, `status` | Total HTTP requests |
| `penshort_http_request_duration_seconds` | Histogram | `method`, `path` | Request latency |
| `penshort_redirect_cache_hits_total` | Counter | — | Redis cache hits |
| `penshort_redirect_cache_misses_total` | Counter | — | Redis cache misses |
| `penshort_webhook_deliveries_total` | Counter | `status` | Webhook delivery attempts |
| `penshort_rate_limit_exceeded_total` | Counter | `type` | Rate limit violations |
| `penshort_active_links` | Gauge | — | Number of active links |

---

## 5. Required Log Fields (Structured JSON)

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | ISO8601 | Event time |
| `level` | string | debug/info/warn/error |
| `message` | string | Human-readable message |
| `request_id` | string | Correlation ID |
| `method` | string | HTTP method |
| `path` | string | Request path (no query params) |
| `status` | int | HTTP status code |
| `duration_ms` | float | Request duration |
| `user_agent` | string | Truncated to 200 chars |
| `api_key_prefix` | string | First 8 chars only |

> ⚠️ **NEVER log**: full API keys, destination URLs with query params, raw IPs in production

---

## 6. Out-of-Scope for v1

| Excluded | Rationale |
|----------|-----------|
| Link previews / social cards | Developer-focused, not consumer |
| Full RBAC | Simple user/team + API key sufficient |
| Rich UI dashboard | API-first approach |
| Enterprise SSO | Post-v1 feature |
| Event streaming (Kafka) | PostgreSQL + Redis sufficient |
| Multi-region | Single region for v1 |
| QR code generation | Third-party can consume API |

---

## 7. Production-Ready Q1 Targets

| Category | Target | Measurement |
|----------|--------|-------------|
| **Latency** | p50 < 10ms, p95 < 50ms, p99 < 100ms | Prometheus histograms |
| **Availability** | 99.5% uptime | Monitoring |
| **Reliability** | Zero click event loss | Audit + testing |
| **Security** | No secrets in logs, hashed keys, signed webhooks | Automated scan |
| **Throughput** | 10,000 redirects/minute | Load test |

---

## 8. Success Criteria

A release is "done" when:

1. ✅ **Deployed publicly** with minimum monitoring
2. ✅ **10 real API keys** actively used
3. ✅ **One blog post** documenting trade-offs
4. ✅ **Redirect path** is fast and cache-backed
5. ✅ **No leakage** of secrets in logs, errors, or analytics

---

## Appendix: Priority Matrix

| Priority | Features |
|----------|----------|
| **P0** | Link CRUD, Redirect, Redis cache, API keys, Rate limiting |
| **P1** | Click analytics, Time expiration, Health endpoints |
| **P2** | Webhooks, Click-count expiration |
| **P3** | Prometheus metrics, Structured logging |
