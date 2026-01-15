# Blog Outline: Penshort Trade-offs

Purpose: Required post documenting key architectural trade-offs for the project.

> **Evidence Requirement**: Per [PROJECT.txt](../PROJECT.txt), this blog post is a required deliverable demonstrating production-ready decision-making.

---

## Post Metadata

| Field | Value |
|-------|-------|
| **Target Length** | 2,000 - 3,000 words |
| **Audience** | Developers evaluating URL shorteners, Go backend engineers |
| **Tone** | Technical but accessible, honest about trade-offs |
| **Publication** | Dev.to, Hashnode, or personal blog |

---

## Outline Structure

### 1. Introduction: Why Penshort is Developer-First

**Key points:**
- The problem with generic URL shorteners for developer workflows
- API keys, analytics, webhooks as first-class features
- Target use cases: SaaS integrations, marketing automation, developer tools

**Evidence to include:**
- Comparison table: Penshort vs Bitly vs TinyURL (feature focus)

---

### 2. Storage: PostgreSQL vs Redis Roles

**Trade-off:** PostgreSQL as source of truth vs Redis as cache layer

**Key points:**
- PostgreSQL: Links, users, API keys, analytics (durable storage)
- Redis: Short code → destination mapping cache (performance)
- Why not Redis-only? Durability, complex queries, transactional integrity
- Why not Postgres-only? Redirect latency requirements

**Evidence to include:**
- Architecture diagram showing data flow
- Latency comparison: cache hit vs cache miss
- Cache invalidation strategy (write-through vs write-behind)

---

### 3. Redirect Path Performance

**Trade-off:** Speed vs accuracy in redirect handling

**Key points:**
- Target: < 10ms p99 for cache hits
- Cache backfill strategy on cache miss
- Async analytics recording (don't block redirect)
- 301 vs 302 decision and SEO implications

**Evidence to include:**
- Benchmark results (see [BENCHMARKS.md](../BENCHMARKS.md))
- Flame graph or profiling data (optional)
- Cache hit rate metrics from demo deployment

---

### 4. Rate Limiting Strategy

**Trade-off:** Security vs user experience

**Key points:**
- Per API key limits (protect infrastructure, fair usage)
- Per IP limits for redirect endpoint (abuse prevention)
- Token bucket vs sliding window (chose sliding window for predictability)
- Redis-backed for distributed rate limiting

**Evidence to include:**
- Rate limit headers example
- Behavior when limit exceeded
- Configuration options exposed to users

---

### 5. Webhook Delivery Reliability

**Trade-off:** Guaranteed delivery vs system complexity

**Key points:**
- At-least-once delivery guarantee
- Exponential backoff with jitter
- Webhook signing (HMAC-SHA256) for verification
- Delivery state tracking (pending, success, failed, exhausted)

**Evidence to include:**
- Retry schedule (e.g., 1m, 5m, 15m, 1h, 4h)
- Webhook payload example with signature
- Failure recovery scenarios

---

### 6. Click Event Durability

**Trade-off:** Real-time analytics vs data durability

**Key points:**
- In-memory queue with periodic flush to PostgreSQL
- Trade-off: Some events may be lost on crash
- Why not message queue? Simplicity over complexity for v1
- Future consideration: Kafka/NATS for high-volume scenarios

**Evidence to include:**
- Queue depth metric (`penshort_analytics_queue_depth`)
- Flush interval configuration
- Data loss window calculation

---

### 7. Security Baseline

**Trade-off:** Defense in depth vs implementation complexity

**Key points:**
- API key hashing (bcrypt) — never stored in plain text
- Log redaction — no secrets in logs
- Webhook signing — clients can verify authenticity
- Input validation — prevent injection attacks

**Evidence to include:**
- Security scanning results (govulncheck, gosec, trivy)
- Example of redacted log output
- Rate limiting as security measure

---

## Conclusion Template

**Summary of decisions:**
- Chose [X] because [reason], accepting [trade-off]
- Would reconsider [Y] when [condition]

**Lessons learned:**
- What worked well
- What we'd do differently

**Call to action:**
- Try Penshort: [demo link]
- Star on GitHub: [repo link]
- Feedback welcome: [issue tracker]

---

## Evidence Checklist

| Section | Evidence Type | Status |
|---------|---------------|--------|
| Architecture | Diagram | [ ] |
| Redirect performance | Benchmark numbers | [ ] |
| Cache effectiveness | Hit rate metrics | [ ] |
| Rate limiting | Configuration examples | [ ] |
| Webhook reliability | Retry schedule | [ ] |
| Security | Scan results | [ ] |

---

## See Also

- [BENCHMARKS.md](../BENCHMARKS.md) — Performance data
- [docs/deployment.md](./deployment.md) — Deployment guide
- [docs/demo-deployment.md](./demo-deployment.md) — Demo setup
