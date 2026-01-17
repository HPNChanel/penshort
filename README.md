# Penshort

Penshort is a developer-focused URL shortener for API-first workflows. It is not a general-purpose consumer shortener.

**Status**: Production-ready Q1 2026 (link CRUD, redirects, analytics, webhooks, API keys, rate limiting).

## Installation

### Option 1: Docker (Recommended)

```bash
# Pull latest release
docker pull ghcr.io/hpnchanel/penshort:latest

# Or pin to specific version
docker pull ghcr.io/hpnchanel/penshort:v1.0.0

# Run with required environment variables
docker run -d \
  -p 8080:8080 \
  -e DATABASE_URL="postgres://user:pass@host:5432/penshort" \
  -e REDIS_URL="redis://host:6379" \
  ghcr.io/hpnchanel/penshort:latest
```

### Option 2: Download Binary

Download pre-built binaries from [GitHub Releases](https://github.com/HPNChanel/penshort/releases):

| OS | Architecture | Filename |
|----|--------------|----------|
| Linux | amd64 | `penshort_linux_amd64` |
| Linux | arm64 | `penshort_linux_arm64` |
| macOS | Intel | `penshort_darwin_amd64` |
| macOS | Apple Silicon | `penshort_darwin_arm64` |
| Windows | amd64 | `penshort_windows_amd64.exe` |

```bash
# Linux/macOS
curl -Lo penshort https://github.com/HPNChanel/penshort/releases/latest/download/penshort_linux_amd64
chmod +x penshort
./penshort
```

### Option 3: Build from Source

```bash
# Clone repository
git clone https://github.com/HPNChanel/penshort.git
cd penshort

# Build
go build -o penshort ./cmd/api

# Run
./penshort
```

**Requirements:** Go 1.22+, PostgreSQL, Redis

## Quick Start

```bash
# Start services
docker compose up -d

# Wait for ready
curl -fsS http://localhost:8080/readyz

# Bootstrap an admin API key (local)
export DATABASE_URL="postgres://penshort:penshort@localhost:5432/penshort?sslmode=disable"
API_KEY=$(go run ./scripts/bootstrap-api-key.go -database-url "$DATABASE_URL" -format plain)

# Create your first short link
curl -X POST http://localhost:8080/api/v1/links \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"destination": "https://example.com"}'

# Test redirect
curl -I http://localhost:8080/{short_code}
```

Full quickstart: `docs/quickstart.md`

## Verify

Run the full verification pipeline locally:

```bash
make verify
```

This runs: doctor checks, dependency startup, migrations, lint, unit tests, integration tests, E2E smoke tests, docs validation, and security scans.

## Documentation

| Topic | Description |
|-------|-------------|
| `docs/quickstart.md` | Get running in under 10 minutes |
| `docs/authentication.md` | API key management |
| `docs/links.md` | Create, update, delete links |
| `docs/redirects.md` | Redirect behavior and caching |
| `docs/analytics.md` | Click statistics and breakdowns |
| `docs/webhooks.md` | Signed delivery, retries, verification |
| `docs/rate-limiting.md` | Limits, headers, handling 429s |
| `docs/deployment.md` | Docker/Compose production setup |
| `docs/api/openapi.yaml` | OpenAPI specification |

### Examples

- `docs/examples/curl/all-endpoints.sh` - curl examples for core endpoints
- `docs/examples/webhook-receiver/` - webhook signature verification
- `docs/examples/e2e/` - end-to-end smoke scripts

## Features

- Link management: create, update, disable, and redirect (301/302) with custom aliases
- Expiration policies: time-based expiration
- Analytics: click events and aggregates, query by link and time range
- Webhooks: signed delivery on click with retries and delivery tracking
- Authentication and rate limiting: API keys per user/team, per-key limits, per-IP limit for redirect
- Ops: health/readiness endpoints, Prometheus metrics

## Stack

- Go for the API service
- PostgreSQL as the system of record
- Redis for cache and rate limiting

## Local Development

### Prerequisites
- Go 1.22+
- Docker + Docker Compose
- migrate CLI (golang-migrate) for local migrations

### Start services (Docker)
```bash
make up
```

Check endpoints:
```bash
curl -fsS http://localhost:8080/healthz
curl -fsS http://localhost:8080/readyz
```

### Run API without Docker
```bash
cp .env.example .env
make dev
```

### Migrations
```bash
make migrate
make migrate-down
```

### Testing
```bash
make test-unit
make test-integration
make test-e2e
```

### Troubleshooting

| Issue | Solution |
|-------|----------|
| `readyz` returns 503 | Wait for Postgres/Redis to initialize, check `docker compose logs` |
| Port 8080 in use | Change `APP_PORT` in docker-compose.yml or stop conflicting process |
| Port 5432 in use | Stop local Postgres: `sudo systemctl stop postgresql` or change port |
| Database connection errors | Run `docker compose down -v && docker compose up -d` |
| Webhook target rejected | Set `WEBHOOK_ALLOW_INSECURE=true` in dev (docker-compose sets this) |
| `make doctor` fails | Run the install command shown in the error message |
| `migrate` command not found | `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest` |
| `golangci-lint` not found | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` |
| E2E tests hang | Check webhook receiver can reach `host.docker.internal` |
| Security scan fails | Update dependencies: `go get -u ./... && go mod tidy` |

**Windows users**: Use PowerShell scripts directly (`.\scripts\verify.ps1`) or run via WSL2/Git Bash.

## Repository Docs

- `CONTRIBUTING.md` - how to propose changes and submit PRs
- `MAINTAINERS.md` - project maintainers and expectations
- `ROADMAP.md` - planned features and project direction
- `SECURITY.md` - how to report security issues
- `GOVERNANCE.md` - maintainership and decision process
- `docs/adr/` - architecture decision records
- `docs/dependency-policy.md` - dependency update cadence

## License

MIT License. See `LICENSE`.
