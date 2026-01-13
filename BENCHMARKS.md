# Penshort Benchmarks

> Performance benchmarks and targets for the Penshort URL shortener.

---

## Methodology

### Test Environment

| Component | Specification |
|-----------|---------------|
| Hardware  | Reference: 4 vCPU, 8GB RAM |
| OS        | Linux (Ubuntu 22.04 LTS) |
| Database  | PostgreSQL 16 |
| Cache     | Redis 7 |
| Network   | Localhost (eliminate network variance) |

### Tools

- **Load testing**: [`hey`](https://github.com/rakyll/hey)
- **Profiling**: `go tool pprof`
- **Tracing**: OpenTelemetry (optional)

### Procedure

1. Start fresh Docker Compose stack: `docker compose up -d`
2. Run migrations: `make migrate`
3. Create benchmark link via API
4. Warm-up: 100 requests at low concurrency
5. Main run: 10,000 requests at 100 concurrent connections
6. Record results

---

## Performance Targets

### Redirect Latency (Primary Metric)

| Percentile | Target | Critical |
|------------|--------|----------|
| p50        | < 10ms | < 25ms   |
| p95        | < 25ms | < 50ms   |
| p99        | < 50ms | < 100ms  |

### Throughput

| Metric | Target |
|--------|--------|
| Requests/sec (cached) | > 5,000 |
| Requests/sec (cache miss) | > 1,000 |

### Analytics Ingest

| Metric | Target |
|--------|--------|
| Queue drain rate | > 1,000 events/sec |
| Ingest lag (p99) | < 30 seconds |

---

## Benchmark Results

### Redirect Latency

| Date | Version | p50 | p95 | p99 | RPS | Notes |
|------|---------|-----|-----|-----|-----|-------|
| _YYYY-MM-DD_ | _vX.Y.Z_ | _Xms_ | _Xms_ | _Xms_ | _X_ | _Initial baseline_ |

### How to Run

```bash
# Ensure stack is running
docker compose up -d

# Create test link
curl -X POST http://localhost:8080/api/v1/links \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"short_code": "bench", "destination": "https://example.com"}'

# Run benchmark
./scripts/benchmark/redirect_latency.sh bench
```

---

## Reproducing Results

To ensure reproducibility:

1. **Use identical hardware** or document deviations
2. **Cold start**: Stop all services, then restart
3. **Consistent data**: Use fresh database with only benchmark link
4. **Multiple runs**: Report average of 3 runs
5. **Document**: Record exact commit hash and configuration

---

## Profiling

### CPU Profile

```bash
# Start with profiling enabled
go run -race ./cmd/api -cpuprofile=cpu.prof

# Analyze
go tool pprof cpu.prof
```

### Memory Profile

```bash
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

---

## Known Bottlenecks

| Area | Issue | Mitigation |
|------|-------|------------|
| Cache miss | PostgreSQL lookup | Pre-warm cache on startup |
| High cardinality | Large link table | Use database indexes, partitioning |
| Analytics burst | Redis stream growth | Monitor queue depth, scale workers |

---

## Changelog

| Date | Change |
|------|--------|
| 2026-01-13 | Initial benchmarks skeleton created |
