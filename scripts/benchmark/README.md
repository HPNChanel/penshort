# Benchmark Scripts

Scripts for measuring Penshort performance metrics.

## Prerequisites

Install [`hey`](https://github.com/rakyll/hey):

```bash
go install github.com/rakyll/hey@latest
```

## Available Benchmarks

### `redirect_latency.sh`

Measures redirect endpoint latency (p50/p95/p99).

```bash
# Default: http://localhost:8080/bench
./redirect_latency.sh

# Custom shortcode and base URL
./redirect_latency.sh mylink https://penshort.example.com
```

**Setup:**
1. Create a benchmark link via API:
   ```bash
   curl -X POST http://localhost:8080/api/v1/links \
     -H "Authorization: Bearer $API_KEY" \
     -H "Content-Type: application/json" \
     -d '{"short_code": "bench", "destination": "https://example.com"}'
   ```
2. Run the benchmark script

## Interpreting Results

Key metrics from `hey` output:
- **Latency distribution**: p50, p90, p95, p99 latencies
- **Requests/sec**: Throughput capacity
- **Status codes**: Should be 3xx (redirect) or 2xx

**Performance targets** (see `BENCHMARKS.md`):
- p50: < 10ms
- p99: < 50ms
- Throughput: > 5,000 req/s on reference hardware
