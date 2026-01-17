# Penshort

> Developer-focused URL shortener for API-first workflows

Penshort is a specialized URL shortener built for developers who need programmable, API-driven link management.

## Key Features

| Feature | Description |
|---------|-------------|
| **API Keys** | Per-user authentication with scoped permissions |
| **Rate Limiting** | Per-key and per-IP limits with clear 429 responses |
| **Analytics** | Click events with totals, uniques, and breakdowns |
| **Webhooks** | Signed delivery on click with retries and backoff |
| **Expiration** | Time-based link expiry |
| **Cache-Backed** | Redis-powered redirects for fast lookups |

## Quick Start

```bash
# Clone and start
git clone https://github.com/HPNChanel/penshort.git
cd penshort
make up

# Verify it's running
curl -fsS http://localhost:8080/readyz

# Bootstrap an admin API key (local)
export DATABASE_URL="postgres://penshort:penshort@localhost:5432/penshort?sslmode=disable"
API_KEY=$(go run ./scripts/bootstrap-api-key.go -database-url "$DATABASE_URL" -format plain)

# Create your first short link
curl -X POST http://localhost:8080/api/v1/links \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"destination": "https://example.com"}'
```

## Documentation

- [Quickstart Guide](quickstart.md)
- [API Overview](api-overview.md)
- [Security](security.md)
- [Testing & Verification](https://github.com/HPNChanel/penshort/blob/main/docs/testing/TEST_STRATEGY.md)

## Verification

Run the full verification pipeline locally before contributing:

```bash
make verify
```

This validates: environment, dependencies, migrations, lint, tests, E2E, docs, and security scans.

## Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| API | Go 1.22+ | Core service |
| Database | PostgreSQL 16 | System of record |
| Cache | Redis 7 | Redirect caching, rate limiting |

## License

MIT License. See [LICENSE](https://github.com/HPNChanel/penshort/blob/main/LICENSE).

---

[GitHub](https://github.com/HPNChanel/penshort) | [Releases](https://github.com/HPNChanel/penshort/releases) | [Issues](https://github.com/HPNChanel/penshort/issues)
