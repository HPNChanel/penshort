# Quickstart

Get Penshort running locally and create your first short link in under 10 minutes.

## Prerequisites

- Docker & Docker Compose
- curl (or any HTTP client)

## 1. Clone and Start

```bash
git clone https://github.com/penshort/penshort.git
cd penshort
docker compose up -d
```

## 2. Wait for Health Check

```bash
# Wait for all services to be ready
curl -fsS http://localhost:8080/readyz
```

Expected output:
```json
{"status":"ok","checks":{"postgres":"ok","redis":"ok"}}
```

## 3. Create Your First Short Link

```bash
curl -X POST http://localhost:8080/api/v1/links \
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

## 4. Test the Redirect

Open in browser or use curl:

```bash
curl -I http://localhost:8080/abc123
```

Expected:
```
HTTP/1.1 302 Found
Location: https://github.com/penshort/penshort
```

## 5. Check Analytics

```bash
curl http://localhost:8080/api/v1/links/{id}/analytics
```

Replace `{id}` with the link ID from step 3.

---

## Next Steps

- [Authentication](authentication.md) - Create API keys for production
- [Link Management](links.md) - Custom aliases, expiration, bulk operations
- [Webhooks](webhooks.md) - Get notified on clicks
- [Deployment](deployment.md) - Production setup

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `readyz` returns 503 | Wait 10s for Postgres/Redis to initialize |
| Port 8080 in use | Change `APP_PORT` in docker-compose.yml |
| Database errors | Run `docker compose down -v && docker compose up -d` |
