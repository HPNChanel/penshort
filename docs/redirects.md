# Redirect Behavior

When a user visits a short link, Penshort resolves and redirects to the destination URL.

## Happy Path

```
GET /{short_code}
```

```bash
curl -I http://localhost:8080/abc123
```

Response:
```
HTTP/1.1 302 Found
Location: https://example.com/destination
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Referrer-Policy: strict-origin-when-cross-origin
Cache-Control: private, max-age=0
```

## Resolution Flow

```
┌─────────────┐     ┌───────────┐     ┌────────────┐
│   Request   │────▶│   Redis   │────▶│  Redirect  │
│ /{shortCode}│     │   Cache   │     │   (fast)   │
└─────────────┘     └───────────┘     └────────────┘
                          │
                     cache miss
                          │
                          ▼
                    ┌───────────┐     ┌────────────┐
                    │ PostgreSQL│────▶│  Backfill  │
                    │  Fallback │     │   Cache    │
                    └───────────┘     └────────────┘
```

- **Cache hit**: ~5ms response (p50)
- **Cache miss**: ~50ms response, then backfills cache

## Error Responses

### 404 Not Found

Link doesn't exist or is disabled:

```json
{
  "error": "Link not found",
  "code": "LINK_NOT_FOUND"
}
```

### 410 Gone

Link has expired (time-based or click limit):

```json
{
  "error": "Link has expired",
  "code": "LINK_EXPIRED"
}
```

## Security Headers

Every redirect response includes:

| Header | Value | Purpose |
|--------|-------|---------|
| `X-Content-Type-Options` | `nosniff` | Prevent MIME sniffing |
| `X-Frame-Options` | `DENY` | Prevent clickjacking |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Control referrer |
| `Cache-Control` | `private, max-age=0` | No browser caching |

## Rate Limiting

Redirect requests are rate-limited per IP to prevent abuse:

- **Default**: 1000 requests/minute per IP
- When exceeded: `429 Too Many Requests` with `Retry-After` header

See [Rate Limiting](rate-limiting.md) for details.

## Analytics Recording

Every successful redirect:
1. Increments click counter (async)
2. Records click event (async) including:
   - Timestamp
   - Referrer (sanitized)
   - User-Agent (truncated)
   - Country code (if available)
   - Visitor hash (for unique counting)
3. Triggers webhooks (if configured)

No latency added to redirect — all recording is fire-and-forget.
