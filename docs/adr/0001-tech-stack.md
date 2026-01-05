# ADR-0001: Technology Stack Selection

## Status

Accepted

## Date

2026-01-05

## Context

Penshort requires a technology stack that supports:
- High-performance redirect path (sub-50ms p95)
- Reliable data persistence for links, click events, and webhook delivery logs
- Fast caching for hot-path lookups
- Rate limiting with atomic operations
- Production-grade observability

We evaluated several options for each layer.

## Decision

### Primary Language: Go 1.22+

**Chosen over**: Node.js, Rust, Python

**Rationale**:
- Excellent HTTP server performance out of the box
- Strong concurrency model (goroutines) for webhook delivery
- Single binary deployment simplifies operations
- Rich ecosystem for PostgreSQL, Redis, and observability
- Team familiarity and hiring pool

### Database: PostgreSQL

**Chosen over**: MySQL, MongoDB, CockroachDB

**Rationale**:
- ACID compliance for link and API key data integrity
- Excellent JSON support for flexible analytics storage
- Mature migration tooling (golang-migrate)
- Strong ecosystem and community support
- Cost-effective managed options (Supabase, Neon, RDS)

### Cache & Rate Limiting: Redis

**Chosen over**: Memcached, In-memory, DynamoDB

**Rationale**:
- Atomic operations (INCR, EXPIRE) for rate limiting
- Pub/Sub potential for future real-time features
- Proven at scale for caching hot paths
- Simple operational model
- Lua scripting for complex atomics (token bucket)

### Why Not SQLite?

While SQLite is excellent for many use cases:
- No built-in connection pooling for concurrent writes
- Rate limiting requires Redis's atomic operations
- Horizontal scaling limited

## Consequences

### Positive

- **Performance**: Go + Redis delivers sub-50ms redirect latency
- **Reliability**: PostgreSQL ensures no data loss for critical entities
- **Operability**: All three technologies have mature managed offerings
- **Hiring**: Go and PostgreSQL skills are common among backend engineers

### Negative

- **Operational complexity**: Three services to manage (app, DB, Redis)
- **Cost**: Redis adds hosting cost vs. pure PostgreSQL solution
- **Learning curve**: Redis Lua scripts for complex rate limiting

### Neutral

- Go's error handling is verbose but explicit
- PostgreSQL requires migration discipline

## Alternatives Considered

### All-in-PostgreSQL (No Redis)

- **Pros**: Simpler ops, fewer moving parts
- **Cons**: Rate limiting performance concerns, redirect latency risk
- **Verdict**: Rejected for performance reasons on hot path

### Serverless (Lambda + DynamoDB)

- **Pros**: Auto-scaling, pay-per-use
- **Cons**: Cold start latency, vendor lock-in, complex local dev
- **Verdict**: Rejected for latency and developer experience

## References

- [Go HTTP performance benchmarks](https://www.techempower.com/benchmarks/)
- [Redis rate limiting patterns](https://redis.io/glossary/rate-limiting/)
- [PostgreSQL vs alternatives](https://www.postgresql.org/about/)
