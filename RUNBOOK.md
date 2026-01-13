# Penshort Operations Runbook

> **Purpose**: First-line debugging and recovery procedures for on-call engineers.
> **Last Updated**: 2026-01-11

---

## Quick Reference

| Check | Command |
|-------|---------|
| Liveness | `curl http://localhost:8080/healthz` |
| Readiness | `curl http://localhost:8080/readyz` |
| Metrics | `curl http://localhost:8080/metrics` |
| Logs | `docker logs penshort-api` |

---

## 1. Redirect Errors

### Symptoms
- Users report 404/410/500 on short links
- Metric: `penshort_redirect_cache_misses_total` spiking
- Error logs with `"path":"/{shortcode}"`

### Triage

```bash
# Check error rate
curl -s http://localhost:8080/metrics | grep redirect

# Check recent errors in logs
docker logs penshort-api 2>&1 | grep -E '"status":(4|5)' | tail -20

# Look up specific link (requires admin API key)
curl -H "Authorization: Bearer $ADMIN_KEY" \
  "http://localhost:8080/api/v1/admin/links?q=abc123"
```

### Resolution

| Error | Cause | Action |
|-------|-------|--------|
| 404 | Link not found | Verify link exists in DB: `SELECT * FROM links WHERE short_code = 'xxx'` |
| 410 | Link expired | Check `expires_at` or click limit reached |
| 500 | Internal error | Check Redis/Postgres connectivity via `/readyz` |

---

## 2. Queue Backlog

### Symptoms
- Metric: `penshort_analytics_queue_depth > 10000`
- Metric: `penshort_webhook_queue_depth > 1000`
- High ingest lag: `penshort_analytics_ingest_lag_seconds_sum` increasing

### Triage

```bash
# Check current queue depths
curl -s http://localhost:8080/metrics | grep queue_depth

# Check worker health in logs
docker logs penshort-api 2>&1 | grep -E "(worker|batch)" | tail -20

# Check Redis queue directly
redis-cli XLEN penshort:click_events
```

### Resolution

| Cause | Action |
|-------|--------|
| DB write slow | Check Postgres connections, run `EXPLAIN ANALYZE` on slow queries |
| Worker crashed | Restart container: `docker restart penshort-api` |
| Burst traffic | Monitor; let queue drain if DB is healthy |

**Emergency**: If queue > 50,000 for > 10 min:
1. Check DB connection pool exhaustion
2. Consider scaling workers (if horizontal scaling available)

---

## 3. Database Slow (PostgreSQL)

### Symptoms
- `/readyz` returns `{"postgres": "error: ..."}` or times out
- High latency across all API endpoints
- Connection timeout errors in logs

### Triage

```bash
# Check readiness
curl -s http://localhost:8080/readyz | jq .

# Direct DB check
psql $DATABASE_URL -c "SELECT 1"

# Check active queries
psql $DATABASE_URL -c "SELECT pid, state, query, now() - query_start AS duration 
FROM pg_stat_activity WHERE state = 'active' ORDER BY duration DESC LIMIT 10"

# Check for locks
psql $DATABASE_URL -c "SELECT * FROM pg_locks WHERE granted = false LIMIT 10"
```

### Resolution

| Cause | Action |
|-------|--------|
| Connection exhaustion | Increase `max_connections` or reduce app pool size |
| Long-running query | Kill it: `SELECT pg_cancel_backend(pid)` |
| Lock contention | Identify blocking query, optimize or restart |
| Missing index | Run `EXPLAIN ANALYZE` on slow queries |

**Enable slow query logging**:
```sql
ALTER SYSTEM SET log_min_duration_statement = '1000';
SELECT pg_reload_conf();
```

---

## 4. Redis Down

### Symptoms
- `/readyz` returns `{"redis": "error: ..."}` with 503
- 100% cache miss rate
- Redirect latency spike (falls back to Postgres)

### Triage

```bash
# Check readiness
curl -s http://localhost:8080/readyz | jq .

# Direct Redis check
redis-cli -u $REDIS_URL ping

# Check memory
redis-cli -u $REDIS_URL INFO memory | grep used_memory_human

# Check slow commands
redis-cli -u $REDIS_URL SLOWLOG GET 10
```

### Resolution

| Cause | Action |
|-------|--------|
| Redis OOM | Check memory, increase limit, or flush expired keys |
| Network partition | Check firewall, security groups |
| Redis crashed | Restart: `docker restart redis` |
| Connection exhaustion | Increase `maxclients`, check pool config |

**Degraded Mode Behavior**:
- Redirects continue via Postgres (slower: 50ms → 200ms+)
- Rate limiting may be disabled (if Redis-backed)
- Analytics queue stops accepting new events

---

## 5. Webhook Failure Storm

### Symptoms
- Metric: `penshort_webhook_deliveries_total{status="failed"}` spiking
- Metric: `penshort_webhook_deliveries_total{status="exhausted"}` increasing
- Webhook queue building up

### Triage

```bash
# Check failure rate
curl -s http://localhost:8080/metrics | grep webhook

# Check recent delivery logs
docker logs penshort-api 2>&1 | grep webhook | grep -E "(error|failed)" | tail -20

# List failed deliveries for specific webhook (requires API key)
curl -H "Authorization: Bearer $API_KEY" \
  "http://localhost:8080/api/v1/webhooks/{id}/deliveries?status=failed"
```

### Resolution

| Cause | Action |
|-------|--------|
| Endpoint 5xx | Contact endpoint owner |
| Endpoint timeout | Check if overloaded, increase timeout |
| Endpoint 4xx | Config issue (wrong URL, bad secret) |
| Certificate error | Verify TLS chain |
| All endpoints failing | Check outbound network |

**Retry a failed delivery**:
```bash
curl -X POST -H "Authorization: Bearer $API_KEY" \
  "http://localhost:8080/api/v1/webhooks/{id}/deliveries/{delivery_id}/retry"
```

---

## Administrative Commands

### Check System State

```bash
# Full health check
curl -s http://localhost:8080/readyz | jq .

# Key metrics summary
curl -s http://localhost:8080/metrics | grep -E "(queue_depth|cache_hits|deliveries|errors)"

# Admin stats (requires admin key)
curl -H "Authorization: Bearer $ADMIN_KEY" \
  "http://localhost:8080/api/v1/admin/stats"
```

### Link Operations

```bash
# Lookup link by shortcode or destination
curl -H "Authorization: Bearer $ADMIN_KEY" \
  "http://localhost:8080/api/v1/admin/links?q=abc123"

curl -H "Authorization: Bearer $ADMIN_KEY" \
  "http://localhost:8080/api/v1/admin/links?q=https://example.com"
```

### API Key Operations

```bash
# List keys for a user
curl -H "Authorization: Bearer $ADMIN_KEY" \
  "http://localhost:8080/api/v1/admin/api-keys?user_id=user123"

# Revoke a key
curl -X DELETE -H "Authorization: Bearer $ADMIN_KEY" \
  "http://localhost:8080/api/v1/api-keys/{key_id}"
```

---

## 6. Poison Message Handling (Dead-Letter Queue)

### Symptoms
- Metric: `penshort_analytics_event_processed{status="dead_lettered"}` increasing
- Same message ID appearing in logs repeatedly with parsing errors

### Behavior
The analytics worker automatically dead-letters messages that:
- Have missing or invalid payload format
- Fail JSON unmarshalling
- Fail payload validation (missing required fields)

Dead-lettered messages are moved to `stream:click_events:dlq` with metadata:
- `original_id`: Original Redis stream message ID
- `reason`: Category (invalid_format, unmarshal_error, validation_error)
- `detail`: Error description
- `dead_lettered_at`: Timestamp

### Triage

```bash
# Check DLQ depth
redis-cli XLEN stream:click_events:dlq

# View recent DLQ messages
redis-cli XRANGE stream:click_events:dlq - + COUNT 10

# Check dead-letter rate
curl -s http://localhost:8080/metrics | grep dead_lettered
```

### Resolution

| Cause | Action |
|-------|--------|
| Publisher bug | Fix publisher, messages already in DLQ are unrecoverable |
| Schema change | DLQ messages from old schema should be ignored |
| Malformed client | Investigate source, fix client |

**Reprocessing DLQ** (if message format has been fixed):
```bash
# Manual inspection - decide per-message
redis-cli XRANGE stream:click_events:dlq - + COUNT 100
# Manually republish valid messages or discard
```

---

## 7. Graceful Shutdown

### Behavior
On SIGTERM/SIGINT:
1. HTTP server stops accepting new connections
2. In-flight HTTP requests complete (up to shutdown timeout)
3. Analytics worker drains current batch
4. Resources (DB connections, Redis) are closed

### Verification

```bash
# Trigger graceful shutdown
docker compose kill -s SIGTERM api

# Verify in logs
docker logs penshort-api | grep -E "(shutting down|shutdown complete|stopping)"
```

### Troubleshooting

| Issue | Cause | Action |
|-------|-------|--------|
| Shutdown timeout | Long-running request or batch | Increase shutdown timeout in config |
| Data loss | Worker killed before batch ACK | Monitor pending messages, they will be reclaimed |

---

## Alerting Thresholds

| Alert | Condition | Severity |
|-------|-----------|----------|
| HighErrorRate | error_rate > 1% for 5m | Warning |
| CriticalErrorRate | error_rate > 5% for 2m | Critical |
| RedirectLatencyHigh | p95 > 100ms for 5m | Warning |
| QueueBacklog | analytics_queue_depth > 10000 for 10m | Warning |
| RedisDown | readyz{redis} != ok for 1m | Critical |
| PostgresDown | readyz{postgres} != ok for 1m | Critical |
| WebhookFailureStorm | webhook_failed_rate > 50% for 5m | Warning |
| PoisonMessageSpike | dead_lettered_rate > 10/min for 5m | Warning |

---

## Escalation Path

1. **First responder**: Follow runbook steps above
2. **If unresolved in 15 min**: Escalate to on-call engineer
3. **If data loss risk**: Escalate immediately to lead engineer
