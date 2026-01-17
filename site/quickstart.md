# Quickstart

Get Penshort running in under 5 minutes.

## Prerequisites

- Docker 24.0+ ([install](https://docs.docker.com/get-docker/))
- Docker Compose v2+ (included with Docker Desktop)
- Go 1.22+ (for bootstrap key)

## Step 1: Clone and Start

```bash
git clone https://github.com/HPNChanel/penshort.git
cd penshort
make up
```

This starts:
- Penshort API on `http://localhost:8080`
- PostgreSQL on `localhost:5432`
- Redis on `localhost:6379`

## Step 2: Verify It's Running

```bash
curl -fsS http://localhost:8080/healthz
curl -fsS http://localhost:8080/readyz
```

## Step 3: Bootstrap an Admin API Key (Local)

```bash
export DATABASE_URL="postgres://penshort:penshort@localhost:5432/penshort?sslmode=disable"
API_KEY=$(go run ./scripts/bootstrap-api-key.go -database-url "$DATABASE_URL" -format plain)
```

## Step 4: Create Your First Link

```bash
curl -X POST http://localhost:8080/api/v1/links \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"destination": "https://example.com"}'
```

## Step 5: Test the Redirect

```bash
# Replace 'abc123' with your actual short_code from the response
curl -I http://localhost:8080/abc123
```

Expected response:

```http
HTTP/1.1 302 Found
Location: https://example.com
X-Penshort-Link-Id: 01HQXY...
```

## Next Steps

- [Authentication](../authentication.md) - API keys and scopes
- [Link Management](../links.md) - Full CRUD operations
- [Analytics](../analytics.md) - Track clicks
- [Webhooks](../webhooks.md) - Real-time notifications

## Cleanup

```bash
make down
make down-clean
```

---

## Manual Installation

If you prefer not to use Docker:

### Requirements
- Go 1.22+
- PostgreSQL 16+
- Redis 7+

### Steps

```bash
git clone https://github.com/HPNChanel/penshort.git
cd penshort

cp .env.example .env

export DATABASE_URL="postgres://user:pass@localhost:5432/penshort?sslmode=disable"
migrate -path migrations -database "$DATABASE_URL" up

make dev
```
