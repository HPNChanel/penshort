# Blog Outline: Penshort Trade-offs

Purpose: Required post documenting key trade-offs for the project.

## Post goals
- Explain major storage, caching, and reliability decisions.
- Capture what was chosen and why.

## Sections (draft)
1. Why Penshort is developer-first (API keys, analytics, webhooks)
2. PostgreSQL vs Redis roles (source of truth vs cache)
3. Redirect path performance and cache backfill strategy
4. Rate limiting strategy (per API key, optional per IP)
5. Webhook delivery reliability (retry, backoff, delivery state)
6. Click event durability and queueing trade-offs
7. Security baseline (key hashing, log redaction, webhook signing)

## Evidence to include
- Benchmarks or latency notes for redirect path
- Failure modes and mitigations
- Operational metrics and alerts
