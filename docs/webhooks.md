# Webhooks

Receive notifications when clicks occur on your short links.

## Create a Webhook

```bash
curl -X POST http://localhost:8080/api/v1/webhooks \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "target_url": "https://your-server.com/webhook",
    "event_types": ["click"],
    "name": "My Webhook"
  }'
```

Response (secret shown only once):

```json
{
  "id": "01HQWH...",
  "target_url": "https://your-server.com/webhook",
  "event_types": ["click"],
  "enabled": true,
  "secret": "4f8d2e1b9c7a5f3d...",
  "created_at": "2026-01-13T08:00:00Z"
}
```

## Webhook Payload

```json
{
  "event_type": "click",
  "event_id": "01HQXK5M7Y...",
  "timestamp": "2026-01-13T08:30:00Z",
  "data": {
    "short_code": "abc123",
    "link_id": "01HQXK5M7Y...",
    "referrer": "https://twitter.com/...",
    "country_code": "US"
  }
}
```

## Headers and Signature

Each delivery includes:

- `X-Penshort-Signature`
- `X-Penshort-Timestamp` (unix seconds)
- `X-Penshort-Delivery-Id`

### Signature Format

The canonical string is:

```
{timestamp}.{payloadJSON}
```

The signature is HMAC-SHA256 over the canonical string.

Because Penshort stores only a hash of the secret, the signing key is:

```
sha256(secret)
```

### Example (Go)

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "strconv"
)

func verifySignature(secret, signature string, timestamp int64, payload []byte) bool {
    sum := sha256.Sum256([]byte(secret))
    canonical := strconv.FormatInt(timestamp, 10) + "." + string(payload)
    mac := hmac.New(sha256.New, sum[:])
    mac.Write([]byte(canonical))
    expected := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(expected))
}
```

## Retry Policy

Retries use exponential backoff with jitter:

| Attempt | Delay |
|---------|-------|
| 1 | 1 minute |
| 2 | 5 minutes |
| 3 | 30 minutes |
| 4 | 2 hours |
| 5 | 12 hours |

After 5 failed attempts, the delivery moves to `exhausted` state.

### Delivery States

| State | Description |
|-------|-------------|
| `pending` | Queued for delivery |
| `success` | Delivered (2xx response) |
| `failed` | Last attempt failed, will retry |
| `exhausted` | All retries failed |

## Manage Webhooks

### List Webhooks

```bash
curl -H "Authorization: Bearer $API_KEY" \
  http://localhost:8080/api/v1/webhooks
```

### Update Webhook

```bash
curl -X PATCH http://localhost:8080/api/v1/webhooks/{id} \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}'
```

### Rotate Secret

```bash
curl -X POST -H "Authorization: Bearer $API_KEY" \
  http://localhost:8080/api/v1/webhooks/{id}/rotate-secret
```

### View Deliveries

```bash
curl -H "Authorization: Bearer $API_KEY" \
  "http://localhost:8080/api/v1/webhooks/{id}/deliveries?status=failed"
```

### Retry a Delivery

```bash
curl -X POST -H "Authorization: Bearer $API_KEY" \
  http://localhost:8080/api/v1/webhooks/{id}/deliveries/{delivery_id}/retry
```

## Local Development

Webhook targets are validated strictly by default (HTTPS only, public IPs only, port 443). For local testing,
set `WEBHOOK_ALLOW_INSECURE=true` to allow `http://localhost`, `http://127.0.0.1`, or `http://host.docker.internal`.
Do not enable this setting in production.

## Best Practices

1. Verify signatures before processing
2. Respond within 5 seconds
3. Handle duplicate deliveries idempotently
4. Use HTTPS for production endpoints
5. Monitor `exhausted` deliveries
