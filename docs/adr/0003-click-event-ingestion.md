# ADR-0003: Click Event Ingestion Pipeline

## Status

Accepted

## Date

2026-01-09

## Context

Penshort needs reliable click analytics without impacting redirect latency.
We require:
- Non-blocking click capture on the redirect path.
- Durable persistence to PostgreSQL with retries.
- Correct analytics aggregation for time-bounded queries.
- Minimal operational complexity (no Kafka or external stream processor).

## Decision

### Queue and Persistence

- Use Redis Streams (`stream:click_events`) with a consumer group (`analytics_workers`) for click event queuing.
- The redirect handler publishes click events asynchronously with a short timeout.
- A worker consumes from the stream, batch inserts into PostgreSQL, and retries failures.
- Messages are ACKed only after both raw event insertion and daily stats update succeed.
- Pending messages are periodically reclaimed with `XAUTOCLAIM` to avoid stuck events.

### Aggregation Strategy

- Daily aggregates are recomputed for affected `(link_id, date)` buckets from `click_events`.
- Aggregation updates are idempotent to ensure accuracy across retries and partial failures.

### Deployment Shape

- The analytics worker runs in the API process by default to reduce operational overhead.
- A unique consumer ID is generated per process to allow horizontal scaling.

## Consequences

### Positive

- Redirect latency stays low (fire-and-forget enqueue).
- At-least-once event processing with retries and pending reclaim.
- Aggregations remain correct even if a batch is retried.

### Negative

- Recomputing daily stats reads from `click_events`, which can be heavier on high-volume days.
- Running workers in the API process couples ingestion to API uptime.

### Neutral

- Redis Streams are simple to operate but require monitoring for lag and pending depth.

## Alternatives Considered

### Synchronous Inserts on Redirect

- **Pros**: Simplest and strongest consistency.
- **Cons**: Adds latency to redirect path; risks timeouts under load.
- **Verdict**: Rejected to protect redirect performance.

### Kafka or External Stream Processor

- **Pros**: High throughput, durable streaming.
- **Cons**: Overkill for current scale; added operational complexity.
- **Verdict**: Rejected for v1 scope.
