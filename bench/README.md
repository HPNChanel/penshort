# Penshort Performance Benchmarks

> Load testing and performance validation for the Penshort URL shortener.

## Prerequisites

- [k6](https://k6.io/docs/get-started/installation/) (primary tool)
- [hey](https://github.com/rakyll/hey) (optional, for quick ad-hoc tests)
- Docker Compose (for running against local stack)
- API key (run `go run ./scripts/bootstrap-api-key.go`)

### Install k6

```bash
# macOS
brew install k6

# Windows (winget)
winget install k6 --source winget

# Windows (choco)
choco install k6

# Linux (Debian/Ubuntu)
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
  --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" \
  | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update && sudo apt-get install k6
```

## Quick Start

```bash
# 1. Start services
docker compose up -d

# 2. Wait for ready
./scripts/wait-for-ready.sh

# 3. Bootstrap API key
export API_KEY=$(go run ./scripts/bootstrap-api-key.go -format plain)

# 4. Setup test data (1000 links)
./bench/data/setup-links.sh "$API_KEY"

# 5. Run benchmarks
k6 run bench/scripts/redirect-latency.js

# Or use make targets
make bench-redirect
make bench-api
make bench-all
```

## Benchmark Scripts

| Script | Description | Duration |
|--------|-------------|----------|
| `redirect-latency.js` | Redirect performance (cache hit/miss) | ~2 min |
| `redirect-ratelimit.js` | IP rate limiting stress test | ~1 min |
| `api-create-link.js` | Link creation under API rate limits | ~3 min |
| `worker-ingest.js` | Analytics event processing throughput | ~2 min |

## Running Individual Benchmarks

```bash
# Redirect latency (cache hit vs miss)
k6 run bench/scripts/redirect-latency.js

# With custom base URL
k6 run --env BASE_URL=http://localhost:8080 bench/scripts/redirect-latency.js

# With JSON output for CI
k6 run --out json=results/redirect.json bench/scripts/redirect-latency.js

# Dry run (1 VU, 10 iterations)
k6 run --vus 1 --iterations 10 bench/scripts/redirect-latency.js
```

## Windows Users

Use the PowerShell runner:

```powershell
.\bench\run-benchmark.ps1 -Script redirect-latency.js
.\bench\run-benchmark.ps1 -Script api-create-link.js -BaseURL http://localhost:8080
```

## Interpreting Results

### Key Metrics

| Metric | Description | Target |
|--------|-------------|--------|
| `http_req_duration` p95 | 95th percentile latency | <25ms (cache hit) |
| `http_req_duration` p99 | 99th percentile latency | <50ms (cache hit) |
| `http_req_failed` | Error rate | <1% |
| `http_reqs` | Requests per second | >5000 (cached) |

### Threshold Failures

k6 exits with code 99 if thresholds fail. Check the summary for which thresholds failed:

```
✗ http_req_duration{scenario:cache_hit}..............: avg=15ms p95=35ms
  ✗ p95<25
```

## Directory Structure

```
bench/
├── README.md                 # This file
├── scripts/
│   ├── redirect-latency.js   # Redirect benchmark (cache hit/miss)
│   ├── redirect-ratelimit.js # Rate limiting stress test
│   ├── api-create-link.js    # API throughput under rate limits
│   ├── worker-ingest.js      # Analytics worker throughput
│   └── util/
│       ├── common.js         # Shared k6 utilities
│       └── thresholds.js     # Centralized threshold definitions
├── data/
│   ├── setup-links.sh        # Pre-populate test links
│   └── codes.txt             # Generated short codes (gitignored)
├── results/                  # JSON outputs (gitignored)
└── run-benchmark.ps1         # Windows PowerShell runner
```

## CI Integration

Benchmarks run:
- **Nightly**: Full suite, results uploaded as artifacts
- **Release branches**: Quality gate (fail if >20% regression)
- **PR (optional)**: With `[benchmark]` label, informational only

See `.github/workflows/benchmark-nightly.yml` for workflow definition.

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `k6: command not found` | Install k6 (see Prerequisites) |
| Connection refused | Ensure `docker compose up -d` and services are ready |
| 401 Unauthorized | Set `API_KEY` environment variable |
| Low RPS | Check Docker resource limits, ensure localhost testing |
| Flaky results | Run 3x and take median, avoid background processes |
