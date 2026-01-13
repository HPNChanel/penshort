# Link Management

Create, manage, and organize short links programmatically.

## Create a Link

```bash
curl -X POST http://localhost:8080/api/v1/links \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "destination": "https://example.com/long/path",
    "alias": "my-custom-alias",
    "redirect_type": 302,
    "expires_at": "2026-12-31T23:59:59Z"
  }'
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `destination` | string | ✅ | Target URL (http/https, max 2048 chars) |
| `alias` | string | ❌ | Custom short code (3-50 chars, alphanumeric + hyphen) |
| `redirect_type` | int | ❌ | 301 (permanent) or 302 (temporary, default) |
| `expires_at` | string | ❌ | Expiration time (ISO8601) |

### Response

```json
{
  "id": "01HQXK5M7Y...",
  "short_code": "my-custom-alias",
  "short_url": "http://localhost:8080/my-custom-alias",
  "destination": "https://example.com/long/path",
  "redirect_type": 302,
  "expires_at": "2026-12-31T23:59:59Z",
  "status": "active",
  "click_count": 0,
  "created_at": "2026-01-13T08:00:00Z",
  "updated_at": "2026-01-13T08:00:00Z"
}
```

## Get a Link

```bash
curl -H "Authorization: Bearer $API_KEY" \
  http://localhost:8080/api/v1/links/{id}
```

## List Links

```bash
curl -H "Authorization: Bearer $API_KEY" \
  "http://localhost:8080/api/v1/links?limit=20&status=active"
```

### Query Parameters

| Param | Description |
|-------|-------------|
| `cursor` | Pagination cursor from previous response |
| `limit` | Items per page (1-100, default 20) |
| `status` | Filter: `active`, `expired`, `disabled` |
| `created_after` | Filter by creation date (RFC3339) |
| `created_before` | Filter by creation date (RFC3339) |

### Pagination

Responses include a `pagination` object:

```json
{
  "data": [...],
  "pagination": {
    "next_cursor": "eyJpZCI6IjAxSFFYSzVNN1kiLCJjIjoiMjAyNi0wMS0xM1QwODowMDowMFoifQ",
    "has_more": true
  }
}
```

Use `next_cursor` in subsequent requests to fetch more pages.

## Update a Link

```bash
curl -X PATCH http://localhost:8080/api/v1/links/{id} \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "destination": "https://example.com/new-path",
    "enabled": false
  }'
```

### Updatable Fields

| Field | Description |
|-------|-------------|
| `destination` | Change target URL |
| `redirect_type` | Change 301/302 |
| `expires_at` | Change/set expiration |
| `enabled` | Enable/disable link |

## Delete a Link

```bash
curl -X DELETE -H "Authorization: Bearer $API_KEY" \
  http://localhost:8080/api/v1/links/{id}
```

Deletion is **soft** — the link returns 410 Gone on redirect.

## Link Status

| Status | Description |
|--------|-------------|
| `active` | Link is working |
| `expired` | Past `expires_at` or click limit reached |
| `disabled` | Manually disabled via `enabled: false` |

## Error Codes

| Code | HTTP | Description |
|------|------|-------------|
| `INVALID_JSON` | 400 | Request body is not valid JSON |
| `INVALID_DESTINATION` | 400 | URL is malformed or not http/https |
| `INVALID_ALIAS` | 400 | Alias format invalid (must be 3-50 chars, alphanumeric + hyphen) |
| `INVALID_REDIRECT_TYPE` | 400 | Redirect type must be 301 or 302 |
| `ALIAS_TAKEN` | 409 | Alias already in use |
| `URL_TOO_LONG` | 400 | Destination exceeds 2048 characters |
| `EXPIRES_IN_PAST` | 422 | Expiry date must be in the future |
| `LINK_NOT_FOUND` | 404 | Link doesn't exist |
| `LINK_EXPIRED` | 409 | Cannot update expired link |
| `MISSING_ID` | 400 | Link ID is required in path |

## 301 vs 302 Redirects

| Type | Use Case |
|------|----------|
| **302** (default) | Tracking links, temporary campaigns, A/B testing |
| **301** | Permanent URL migrations, SEO juice transfer |

> ⚠️ Browsers cache 301 redirects aggressively. Use 302 if you might change the destination.
