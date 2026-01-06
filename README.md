# Penshort

Penshort is a developer-focused URL shortener for API-first workflows. It is not a general-purpose consumer shortener.

Status: Phase 1 service skeleton (config, health checks, logging, CI, Docker Compose).

## Project goals
- Build a specialized shortener for developer workflows: API keys, analytics, webhooks, rate limits, and expiration policies.
- Provide reliable, cache-backed redirects with clear operational behavior.
- Keep the system small-team friendly and easy to operate.

## Planned stack
- Go for the API service
- PostgreSQL as the system of record
- Redis for cache and rate limiting

## Feature scope (planned)
- Link management: create, update, disable, and redirect (301/302) with optional custom aliases.
- Expiration policies: time-based and optional click-count based.
- Analytics: click events (timestamp, referrer, user-agent, optional region), unique vs total, query by link and time range.
- Webhooks: signed delivery on click, retries with backoff, and delivery state.
- Authentication and rate limiting: API keys per user/team, per-key limits, optional per-IP limit for redirect.
- Admin/ops: health and readiness endpoints, minimal admin surface.

## Non-functional goals (planned)
- Fast redirects backed by Redis cache for short_code to destination mapping.
- Reliability: do not lose click events; webhook delivery is tracked.
- Security baseline: API keys never stored in plaintext; logs do not contain secrets.

## Roadmap outline
- Milestone 1: Core redirect and link CRUD with Redis caching.
- Milestone 2: API keys and rate limiting.
- Milestone 3: Click events and analytics queries.
- Milestone 4: Webhooks with signing and retries.
- Milestone 5: Ops endpoints, monitoring basics, and delivery evidence.

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
