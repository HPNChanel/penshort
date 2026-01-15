# Quickstart

Get Penshort running locally and create your first short link in under 10 minutes.

## Prerequisites

- Docker & Docker Compose
- Go 1.22+
- curl (or any HTTP client)

## 1. Clone and Start

```bash
git clone https://github.com/penshort/penshort.git
cd penshort
docker compose up -d
```

## 2. Wait for Health Check

```bash
curl -fsS http://localhost:8080/readyz
```

Expected output:

```json
{"status":"ok","checks":{"postgres":"ok","redis":"ok"}}
```

## 3. Bootstrap an Admin API Key (Local)

Penshort requires an API key for all `/api/v1` endpoints. For local development, bootstrap a key directly into the database:

```bash
export DATABASE_URL="postgres://penshort:penshort@localhost:5432/penshort?sslmode=disable"
API_KEY=$(go run ./scripts/bootstrap-api-key.go -database-url "$DATABASE_URL" -format plain)
```

The key is printed once. Store it securely.

## 4. Create Your First Short Link

```bash
curl -X POST http://localhost:8080/api/v1/links \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"destination": "https://github.com/penshort/penshort"}'
```

Example response:

```json
{
  "id": "01HQXK5M7Y...",
  "short_code": "abc123",
  "short_url": "http://localhost:8080/abc123",
  "destination": "https://github.com/penshort/penshort",
  "redirect_type": 302,
  "status": "active",
  "click_count": 0,
  "created_at": "2026-01-13T08:00:00Z",
  "updated_at": "2026-01-13T08:00:00Z"
}
```

## 5. Test the Redirect

```bash
curl -I http://localhost:8080/abc123
```

Expected:

```
HTTP/1.1 302 Found
Location: https://github.com/penshort/penshort
```

## 6. Check Analytics

```bash
curl -H "Authorization: Bearer $API_KEY" \
  http://localhost:8080/api/v1/links/{id}/analytics
```

Replace `{id}` with the link ID from step 4.

---

## Next Steps

- [Authentication](authentication.md) - API key management
- [Link Management](links.md) - Custom aliases and updates
- [Webhooks](webhooks.md) - Click notifications
- [Deployment](deployment.md) - Production setup

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `readyz` returns 503 | Wait for Postgres/Redis to initialize |
| Port 8080 in use | Change `APP_PORT` in docker-compose.yml |
| Database errors | Run `docker compose down -v && docker compose up -d` |
| Webhook target rejected for localhost | Set `WEBHOOK_ALLOW_INSECURE=true` in dev (docker-compose sets it) |
