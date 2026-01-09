---
trigger: always_on
---

Penshort Project Rules & Implementation Guide

This document is the project-specific rulebook and single reference zone for implementing Penshort from scratch. It defines scope, priorities, architectural expectations, quality gates, and delivery evidence. When ambiguity exists, follow the principles and “decision defaults” in this file.

## 1) Product Intent (Non-Negotiable)
Penshort is not a generic URL shortener. It is a specialized shortener optimized for developer workflows:
- API keys and team/user access
- Analytics suitable for developer usage (queryable, time-bounded)
- Webhooks with signing, retries, and delivery state
- Rate limiting and reliable redirects
- Expiration policies (time-based and optional click-count-based)

## 2) Project Success Criteria (Evidence-Driven)
A release is considered “done” only when these are demonstrably true:
- Public deployment with minimum monitoring
- At least 10 real users (or at minimum 10 real API keys actively used)
- One blog post documenting key trade-offs (e.g., PostgreSQL vs Redis, rate limiting strategy, webhook reliability)
- Redirect path is fast and cache-backed
- No leakage of secrets in logs, errors, or analytics payloads

## 3) Scope Definition

### 3.1 Functional Requirements (In Scope)
A. Link management
- Create short link (custom alias optional)
- Redirect with configurable 301/302
- Expiration policies:
  - time-based expiration
  - optional click-count expiration

B. Analytics
- Click events captured with:
  - timestamp
  - referrer
  - user-agent
  - optional country/region
  - unique vs total
- API for querying analytics by:
  - link
  - time range (from/to or window)

C. Webhooks
- Trigger on click
- Webhook signing so clients can verify authenticity
- Retry with backoff
- Persist delivery state and attempts

D. Authentication & rate limiting
- API keys per user/team
- Rate limiting per API key (and/or per IP for redirect)

E. Admin/Ops (minimum)
- Health endpoints
- Basic dashboard OR admin endpoints (keep simple)

### 3.2 Non-Functional Requirements (Production-Ready Q1 Bar)
- Redirect performance: Redis cache for short_code → destination mapping
- Reliability: do not lose click events; use an internal queue if needed for webhook dispatch
- Security baseline:
  - never store API keys in plaintext
  - never log secrets
  - ensure minimal principle-of-least-privilege in configs

### 3.3 Explicit Non-Goals (Out of Scope Unless Required)
- Consumer-grade link previews and social cards
- Complex RBAC beyond simple user/team and API-key scoping
- A full-featured UI dashboard (basic endpoints are acceptable)
- Enterprise SSO
- Over-engineered event streaming (Kafka, etc.) unless justified by evidence

## 4) System Design Defaults (Decision Rules)

### 4.1 Storage and Caching Defaults
- Source of truth: PostgreSQL for durable entities (users/teams, API keys, links, webhook config, delivery logs, click events).
- Redis is mandatory for:
  - redirect hot-path mapping cache
  - rate limiting state (token bucket / leaky bucket / fixed window with safeguards)
  - optional short-lived queues or idempotency keys (if needed)

Redirect path must succeed with:
1) Redis lookup for short_code → destination (+ metadata like redirect type, expiry)
2) Fallback to PostgreSQL if cache miss, then backfill Redis
3) Expired or invalid codes return appropriate response (do not redirect)

### 4.2 Click Event Reliability
Default requirement: click events must not be lost.
- If synchronous insert into PostgreSQL is acceptable for throughput, do it.
- If redirect latency becomes an issue, use a durable internal queue pattern:
  - enqueue minimal click payload quickly (Redis + persistence strategy)
  - process asynchronously to PostgreSQL
  - define lossless or at least-at-least-once semantics and document trade-offs

### 4.3 Webhook Delivery Semantics
- Webhooks trigger on click events.
- Must be signed (HMAC-based signing is a safe default).
- Delivery must record:
  - event id
  - target URL
  - timestamp(s)
  - attempt count
  - last status code / error
  - next retry time
  - final state: pending/succeeded/failed-deadletter
- Retries with backoff:
  - exponential backoff with jitter is the default
  - cap retries and record dead-letter state

### 4.4 Unique vs Total Definition (Default)
- Total clicks = total recorded click events.
- Unique clicks default = unique visitor approximation over a time window per link.
  - Use a stable-but-privacy-aware key (e.g., hash of IP + user-agent + day salt).
  - Do not store raw IP if you can avoid it; if storing, define retention and access rules.

### 4.5 Expiration Rules
- Time-based: if now > expires_at, link is expired.
- Click-count-based (optional): if total_clicks >= max_clicks, link is expired.
- Expiration check must be enforceable at redirect time.

## 5) Security Rules (Hard Requirements)
- API keys:
  - never store the raw key after creation
  - store only a hash (strong, slow hash preferred for secrets) and optionally a key prefix for identification
  - show the raw key only once at creation
- Webhook signatures:
  - include timestamp and body digest to prevent replay and tampering
  - define allowed clock skew for verification
- Logs:
  - never include secrets, full tokens, raw API keys, or sensitive headers
  - avoid logging full destination URLs if they could contain secrets; if necessary, redact query parameters by default
- Input validation:
  - validate destination URL (scheme allowlist, length constraints)
  - protect against open redirect abuse (this is the product, but ensure URL is well-formed)
- Rate limiting:
  - implement on write APIs at minimum
  - consider redirect IP-based limit if abuse is likely
- Dependency policy:
  - minimize dependencies; prefer standard libraries and well-maintained packages
  - pin versions and track security updates

## 6) API & Behavior Expectations

### 6.1 Public Redirect Behavior
- `/{short_code}` performs redirect.
- Redirect type is configurable per link (301/302).
- If expired/disabled/not found:
  - return a clear, non-redirect response
  - do not leak internal IDs or configuration

### 6.2 Management API (Developer-Facing)
Must support:
- Create link (with optional custom alias)
- Retrieve link details
- Update link (destination, redirect type, expiration settings, enabled/disabled)
- Delete/disable (prefer soft delete)
- List links (paging)

### 6.3 Analytics API
Must support:
- Query by link and time range
- Return total clicks and unique clicks
- Optional breakdowns: referrer, user-agent family, region (only if implemented reliably)

### 6.4 Webhook Management
Must support:
- Configure webhook endpoint per user/team (or per link if chosen)
- Rotate signing secret
- View delivery logs/status

### 6.5 Admin/Ops Endpoints
Minimum:
- Health endpoint for liveness
- Readiness endpoint checking DB/Redis connectivity

## 7) Repository & Documentation Rules
This project must remain understandable and operable by a small team.

Required documentation artifacts:
- README: local dev, env vars, how to run migrations, how to test
- docs/adr: Architecture Decision Records for major choices (cache strategy, event capture, webhook retries)
- SECURITY.md: threat model highlights, secret handling, signing verification rules
- BLOG.md or /docs/blog-outline.md: outline for required trade-offs post (can be drafted early)

## 8) Implementation Workflow (How the Agent Should Work)
Always follow this loop:
1) Restate the target milestone and acceptance criteria.
2) Propose a minimal design aligned with the defaults above.
3) Implement in small, reviewable steps.
4) Add tests and operational checks as part of each step.
5) Produce “evidence”:
   - example API calls and expected outputs
   - performance notes for redirect path
   - logs showing redaction behavior
6) Record any significant decision in an ADR.

If forced to choose between speed and correctness:
- Redirect path correctness and security > feature breadth.
- Data durability for click events > fancy analytics breakdowns.

## 9) Quality Gates (Do Not Merge Without)
- Unit tests for core business logic (link creation, expiration evaluation, signature generation/verification)
- Integration tests for:
  - PostgreSQL migrations and core queries
  - Redis cache behavior and fallback logic
  - rate limiter behavior
- Static checks:
  - linting and formatting
  - basic security scanning where available
- Operational sanity:
  - health and readiness endpoints behave predictably
  - no secrets in logs under normal request flows

## 10) Performance Targets (Directional)
- Redirect path should be predominantly served from Redis.
- Cache miss should be rare after warm-up; implement backfill.
- Click capture must not meaningfully degrade redirect latency; if it does, move capture off-path with a reliable queue pattern.

## 11) Deployment Expectations
- Container-friendly configuration (12-factor):
  - env vars for DB/Redis/keys
  - structured logging
- Database migrations must be runnable in CI and deployment pipelines.
- Minimal monitoring:
  - request rate and error rate
  - latency (especially redirect)
  - queue depth / webhook retry counts
  - DB/Redis connectivity alerts

## 12) Milestones (Recommended Build Order)
Milestone 1: Core redirect + link CRUD
- Postgres schema, create link, resolve by short_code, redirect with 301/302
- Redis cache for short_code mapping

Milestone 2: Auth + API keys + rate limiting
- API key issuance, hashing, enforcement
- Rate limiting per API key on management APIs
- Optional IP-based rate limit on redirect

Milestone 3: Click events + analytics query
- Record click events
- Provide analytics API by time range + unique vs total

Milestone 4: Webhooks
- Signed delivery, retries, persisted status

Milestone 5: Ops/admin + deployment + evidence pack
- Health/readiness
- Monitoring basics
- Deployment docs and the required trade-offs blog draft

## 13) When Requirements Conflict
Use this priority order:
1) Security baseline and secret safety
2) Redirect correctness and performance path (Redis-backed)
3) Data durability for click events and webhook delivery states
4) Developer-focused workflows (API keys, queryable analytics)
5) UI/dashboard polish

## 14) Final Note
This file is authoritative for the project. If you (the agent) deviate from it, you must:
- document why in an ADR,
- explain the trade-off,
- and show evidence that the new approach better satisfies the success criteria.
