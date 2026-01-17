# Penshort Benchmarks

> Performance benchmarks and targets for the Penshort URL shortener.

---

## Tools

| Tool | Purpose | When to Use |
|------|---------|-------------|
| **k6** (primary) | Load testing, scenario-based benchmarks | Structured testing, CI integration |
| **hey** (ad-hoc) | Quick HTTP benchmarking | Development, quick checks |
| **go test -bench** | Micro-benchmarks | Unit-level performance |

### Why k6?

1. **Scenario flexibility**: Model cache hit/miss, rate limit ramp-up
2. **Native thresholds**: Built-in pass/fail gates (no text parsing)
3. **JSON output**: CI artifact integration, trend analysis
4. **Windows support**: Native binary for cross-platform use

---

## Quick Start

```bash
# Install k6 (macOS)
brew install k6

# Install k6 (Windows)
winget install k6 --source winget

# Run redirect latency benchmark
make bench-redirect

# Run all benchmarks
make bench-all
```

See `bench/README.md` for detailed instructions.

---

## Benchmark Suite

| Benchmark | Target Metric | Threshold |
|-----------|---------------|-----------|
| Redirect (cache hit) | p95 latency | <25ms |
| Redirect (cache miss) | p95 latency | <100ms |
| API create link | p95 latency | <200ms |
| Rate limit rejection | p95 latency | <20ms |
| Worker drain | throughput | >1000 events/sec |

### Running Benchmarks

```bash
# Individual benchmarks
make bench-redirect      # Redirect latency (cache hit/miss)
make bench-api           # API create link throughput
make bench-ratelimit     # Rate limiting stress test
make bench-worker        # Analytics worker throughput

# All benchmarks
make bench-all

# Windows (PowerShell)
.\bench\run-benchmark.ps1 -Script redirect-latency.js
```

---

## Test Environment

| Component | Specification |
|-----------|---------------|
| Hardware  | Reference: 4 vCPU, 8GB RAM |
| OS        | Linux (Ubuntu 22.04 LTS) / Docker |
| Database  | PostgreSQL 16 |
| Cache     | Redis 7 |
| Network   | Localhost (eliminate network variance) |

---

## Performance Targets

### Redirect Latency

| Scenario | p50 | p95 | p99 |
|----------|-----|-----|-----|
| Cache hit | <5ms | <15ms | <30ms |
| Cache miss | <25ms | <75ms | <150ms |

### Throughput

| Metric | Target |
|--------|--------|
| Redirects/sec (cached) | >5,000 |
| Redirects/sec (uncached) | >1,000 |
| API creates/sec | >100 |

### Analytics Worker

| Metric | Target |
|--------|--------|
| Queue drain rate | >1,000 events/sec |
| Ingest lag (p99) | <30 seconds |

---

## CI Integration

| Trigger | Scope | Failure Mode |
|---------|-------|--------------|
| Nightly (2 AM UTC) | Full suite | Informational |
| Release branch | Full suite | Quality gate |
| PR with `benchmark` label | Subset | Informational |

**Threshold philosophy**: Detect **regression** (20% tolerance), not absolute targets.

See `.github/workflows/benchmark-nightly.yml` for workflow definition.

---

## Threshold Methodology

1. **Collect baseline**: Run benchmarks 30 times over 1 week
2. **Calculate median**: Use median of p95 values
3. **Set threshold**: `threshold = median × 1.2` (20% tolerance)
4. **Monitor regression**: Alert if >20% slower than baseline

Thresholds are defined in `bench/scripts/util/thresholds.js`.

---

## Profiling

### CPU Profile

```bash
go run -race ./cmd/api -cpuprofile=cpu.prof
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
| High cardinality | Large link table | Database indexes, partitioning |
| Analytics burst | Redis stream growth | Monitor queue depth, scale workers |

---

## Changelog

| Date | Change |
|------|--------|
| 2026-01-17 | Migrated to k6 benchmark suite, added CI integration |
| 2026-01-13 | Initial benchmarks skeleton created |
