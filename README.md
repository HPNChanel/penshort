# Penshort

Penshort is a developer-focused URL shortener for API-first workflows. It is not a general-purpose consumer shortener.

**Status**: Production-ready Q1 2026 — Link CRUD, redirects, analytics, webhooks, API keys, rate limiting.

## Quick Start

```bash
# Start services
docker compose up -d

# Wait for ready
curl -fsS http://localhost:8080/readyz

# Create your first short link
curl -X POST http://localhost:8080/api/v1/links \
  -H "Content-Type: application/json" \
  -d '{"destination": "https://example.com"}'

# Test redirect
curl -I http://localhost:8080/{short_code}
```

📖 **[Full Quickstart Guide →](docs/quickstart.md)**

## Documentation

| Topic | Description |
|-------|-------------|
| [Quickstart](docs/quickstart.md) | Get running in <10 minutes |
| [Authentication](docs/authentication.md) | API key management |
| [Link Management](docs/links.md) | Create, update, delete links |
| [Redirects](docs/redirects.md) | Redirect behavior and caching |
| [Analytics](docs/analytics.md) | Click statistics and breakdowns |
| [Webhooks](docs/webhooks.md) | Signed delivery, retries, verification |
| [Rate Limiting](docs/rate-limiting.md) | Limits, headers, handling 429s |
| [Deployment](docs/deployment.md) | Docker/Compose production setup |
| [API Reference](docs/api/openapi.yaml) | OpenAPI 3.0 specification |

### Examples

- [curl examples](docs/examples/curl/all-endpoints.sh) — All API endpoints
- [Webhook receiver (Go)](docs/examples/webhook-receiver/) — Signature verification
- [TypeScript client](docs/examples/clients/typescript.ts) — Minimal typed client

## Project Goals

- Build a specialized shortener for developer workflows: API keys, analytics, webhooks, rate limits, and expiration policies.
- Provide reliable, cache-backed redirects with clear operational behavior.
- Keep the system small-team friendly and easy to operate.

## Stack

- **Go** for the API service
- **PostgreSQL** as the system of record
- **Redis** for cache and rate limiting

## Features

- **Link management**: create, update, disable, and redirect (301/302) with optional custom aliases
- **Expiration policies**: time-based and optional click-count based
- **Analytics**: click events (timestamp, referrer, user-agent, region), unique vs total, query by link and time range
- **Webhooks**: signed delivery on click, retries with exponential backoff, delivery state tracking
- **Authentication and rate limiting**: API keys per user/team, per-key limits, per-IP limit for redirect
- **Ops**: health/readiness endpoints, Prometheus metrics

## Roadmap

- ✅ Milestone 1: Core redirect and link CRUD with Redis caching
- ✅ Milestone 2: API keys and rate limiting
- ✅ Milestone 3: Click events and analytics queries
- ✅ Milestone 4: Webhooks with signing and retries
- ✅ Milestone 5: Ops endpoints, monitoring basics, and delivery evidence

## Phase 1: Local development

### Prerequisites
- Go 1.22+
- Docker + Docker Compose
- `migrate` CLI (golang-migrate) for local migrations

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

### Configuration
Required:
- `DATABASE_URL` (PostgreSQL connection string)
- `REDIS_URL` (Redis connection string)

Optional:
- `APP_ENV` (default: `development`)
- `APP_PORT` (default: `8080`)
- `LOG_LEVEL` (default: `info`)
- `LOG_FORMAT` (default: `json`)
- `READ_TIMEOUT` (default: `5s`)
- `WRITE_TIMEOUT` (default: `10s`)
- `SHUTDOWN_TIMEOUT` (default: `30s`)

### Migrations
```bash
make migrate
make migrate-down
```

### Testing
```bash
make test
make lint
```

### Integration smoke test (local)
```bash
make up
curl -fsS http://localhost:8080/healthz
curl -fsS http://localhost:8080/readyz
```

### Trade-offs and TODOs (Phase 1)
- Readiness checks validate connectivity only; schema-level checks come later.
- Integration coverage is smoke-level; add migration + cache behavior tests in Phase 2.
- Local migrations require the `migrate` CLI; consider a Dockerized migration target.

## Repository docs
- [CONTRIBUTING.md](CONTRIBUTING.md) - how to propose changes and submit PRs.
- [SECURITY.md](SECURITY.md) - how to report security issues.
- [GOVERNANCE.md](GOVERNANCE.md) - maintainership and decision process.
- [docs/adr/](docs/adr/) - architecture decision records.

## License
MIT License. See [LICENSE](LICENSE).
