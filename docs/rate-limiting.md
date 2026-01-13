# Rate Limiting

Penshort applies rate limits to protect the service and ensure fair usage.

## Types of Rate Limits

| Type | Scope | Default |
|------|-------|---------|
| **API** | Per API key | Tier-based (see below) |
| **Redirect** | Per IP | 1000 req/min |

## API Rate Limits by Tier

| Tier | Requests/minute | Burst |
|------|-----------------|-------|
| Free | 60 | 10 |
| Standard | 600 | 100 |
| Premium | 6000 | 1000 |

## Response Headers

Every API response includes rate limit headers:

| Header | Description | Example |
|--------|-------------|---------|
| `X-RateLimit-Limit` | Max requests per minute | `600` |
| `X-RateLimit-Remaining` | Requests left in window | `542` |
| `X-RateLimit-Reset` | Unix timestamp when limit resets | `1705142460` |

## Rate Limited Response

When you exceed the limit, you receive `429 Too Many Requests`:

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 45
X-RateLimit-Limit: 600
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1705142460
Content-Type: application/json

{
  "error": {
    "code": "RATE_LIMITED",
    "message": "Rate limit exceeded. Retry after 45 seconds."
  }
}
```

## Handling Rate Limits

### Basic Retry Logic

```javascript
async function fetchWithRetry(url, options, maxRetries = 3) {
  for (let attempt = 0; attempt < maxRetries; attempt++) {
    const response = await fetch(url, options);
    
    if (response.status === 429) {
      const retryAfter = parseInt(response.headers.get('Retry-After')) || 60;
      console.log(`Rate limited. Waiting ${retryAfter}s...`);
      await new Promise(r => setTimeout(r, retryAfter * 1000));
      continue;
    }
    
    return response;
  }
  throw new Error('Max retries exceeded');
}
```

### Proactive Rate Limit Checking

```javascript
function checkRemainingQuota(response) {
  const remaining = parseInt(response.headers.get('X-RateLimit-Remaining'));
  const resetTime = parseInt(response.headers.get('X-RateLimit-Reset'));
  
  if (remaining < 10) {
    const waitMs = (resetTime * 1000) - Date.now();
    console.log(`Low quota (${remaining}). Consider waiting ${waitMs}ms`);
  }
}
```

## Redirect Rate Limiting

Public redirect endpoints (`GET /{shortCode}`) are rate limited per IP:

- **Limit**: 1000 requests/minute
- **Scope**: Client IP (respects `X-Forwarded-For`)
- **Fail behavior**: Returns 429 with `Retry-After`

This prevents abuse while allowing legitimate traffic.

## Best Practices

1. **Monitor headers** — Track `X-RateLimit-Remaining` proactively
2. **Implement backoff** — Respect `Retry-After` header
3. **Batch operations** — Use list endpoints instead of individual calls
4. **Cache responses** — Reduce unnecessary API calls
5. **Request higher tier** — Contact us if default limits are insufficient

## Algorithm

Rate limiting uses a **token bucket algorithm** backed by Redis:

- Tokens refill continuously at `requests_per_minute / 60` per second
- Burst allows temporary spikes up to `burst` tokens
- State is shared across all API instances
