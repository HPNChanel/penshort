# API Overview

Penshort provides a REST API for link management and analytics.

**Base URL**: `http://localhost:8080/api/v1`

## Authentication

All API endpoints (except health checks and redirects) require an API key:

```bash
curl -H "Authorization: Bearer pk_live_..." http://localhost:8080/api/v1/links
```

See [Authentication](../authentication.md) for key management.

## Endpoints Summary

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/links` | Create a short link |
| `GET` | `/api/v1/links` | List links |
| `GET` | `/api/v1/links/{id}` | Get link details |
| `PATCH` | `/api/v1/links/{id}` | Update link |
| `DELETE` | `/api/v1/links/{id}` | Delete link |
| `GET` | `/api/v1/links/{id}/analytics` | Link analytics |
| `POST` | `/api/v1/webhooks` | Create webhook |
| `GET` | `/api/v1/webhooks` | List webhooks |
| `GET` | `/api/v1/webhooks/{id}` | Get webhook |
| `POST` | `/api/v1/api-keys` | Create API key |
| `GET` | `/api/v1/api-keys` | List API keys |
| `GET` | `/:short_code` | Redirect (public) |
| `GET` | `/healthz` | Health check |
| `GET` | `/readyz` | Readiness check |

## Create Link

```bash
curl -X POST http://localhost:8080/api/v1/links \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "destination": "https://example.com",
    "alias": "my-link",
    "expires_at": "2026-12-31T23:59:59Z"
  }'
```

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `destination` | string | Yes | Target URL |
| `alias` | string | No | Custom short code |
| `redirect_type` | int | No | 301 or 302 (default 302) |
| `expires_at` | string | No | Expiration timestamp (RFC3339) |

### Response

```json
{
  "id": "01HXYZ...",
  "short_code": "my-link",
  "short_url": "http://localhost:8080/my-link",
  "destination": "https://example.com",
  "redirect_type": 302,
  "status": "active",
  "click_count": 0,
  "created_at": "2026-01-15T10:00:00Z",
  "updated_at": "2026-01-15T10:00:00Z"
}
```

## Analytics

```bash
curl -H "Authorization: Bearer $API_KEY" \
  http://localhost:8080/api/v1/links/01HXYZ.../analytics
```

## Error Responses

All errors follow this format:

```json
{
  "error": "Link not found",
  "code": "LINK_NOT_FOUND"
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `LINK_NOT_FOUND` | 404 | Link doesn't exist |
| `LINK_EXPIRED` | 410 | Link has expired |
| `INVALID_REQUEST` | 400 | Malformed request body |
| `UNAUTHORIZED` | 401 | Missing or invalid API key |
| `RATE_LIMITED` | 429 | Too many requests |

## Rate Limiting

API responses include rate limit headers:

```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1705312800
```

When rate limited:

```
HTTP/1.1 429 Too Many Requests
Retry-After: 60
```

## OpenAPI Specification

Full API specification: [openapi.yaml](../api/openapi.yaml)
