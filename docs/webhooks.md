# Webhooks

Receive real-time notifications when clicks occur on your short links.

## Create a Webhook

```bash
curl -X POST http://localhost:8080/api/v1/webhooks \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://your-server.com/webhook",
    "events": ["click"],
    "link_id": "01HQXK5M7Y..."
  }'
```

Response (⚠️ secret shown **only once**):
```json
{
  "id": "01HQWH...",
  "url": "https://your-server.com/webhook",
  "events": ["click"],
  "link_id": "01HQXK5M7Y...",
  "enabled": true,
  "secret": "whsec_a1b2c3d4e5f6...",
  "created_at": "2026-01-13T08:00:00Z"
}
```

> Omit `link_id` to receive events for all your links.

---

## Webhook Payload

```json
{
  "event": "click",
  "link_id": "01HQXK5M7Y...",
  "short_code": "abc123",
  "timestamp": "2026-01-13T08:30:00Z",
  "visitor": {
    "referrer": "twitter.com",
    "user_agent": "Mozilla/5.0...",
    "country_code": "US"
  }
}
```

---

## Signature Verification

Every webhook request includes an `X-Penshort-Signature` header for verification.

### Header Format

```
X-Penshort-Signature: t=1705142400,v1=abc123def456...
```

| Part | Description |
|------|-------------|
| `t` | Unix timestamp when signature was created |
| `v1` | HMAC-SHA256 signature |

### Verification Steps

1. **Extract timestamp and signature** from header
2. **Build signed payload**: `{timestamp}.{request_body}`
3. **Compute expected signature**: `HMAC-SHA256(secret, signed_payload)`
4. **Compare signatures** using constant-time comparison
5. **Check timestamp** is within ±5 minutes (prevents replay attacks)

### Example (Go)

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "math"
    "strconv"
    "strings"
    "time"
)

func VerifySignature(header, body, secret string) bool {
    parts := strings.Split(header, ",")
    if len(parts) != 2 {
        return false
    }
    
    timestamp := strings.TrimPrefix(parts[0], "t=")
    signature := strings.TrimPrefix(parts[1], "v1=")
    
    // Check timestamp (±5 min tolerance)
    ts, _ := strconv.ParseInt(timestamp, 10, 64)
    if math.Abs(float64(time.Now().Unix()-ts)) > 300 {
        return false
    }
    
    // Compute expected signature
    signedPayload := timestamp + "." + body
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write([]byte(signedPayload))
    expected := hex.EncodeToString(mac.Sum(nil))
    
    return hmac.Equal([]byte(signature), []byte(expected))
}
```

### Example (Node.js)

```javascript
const crypto = require('crypto');

function verifySignature(header, body, secret) {
  const [tPart, v1Part] = header.split(',');
  const timestamp = tPart.replace('t=', '');
  const signature = v1Part.replace('v1=', '');
  
  // Check timestamp (±5 min tolerance)
  const now = Math.floor(Date.now() / 1000);
  if (Math.abs(now - parseInt(timestamp)) > 300) {
    return false;
  }
  
  // Compute expected signature
  const signedPayload = `${timestamp}.${body}`;
  const expected = crypto
    .createHmac('sha256', secret)
    .update(signedPayload)
    .digest('hex');
  
  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(expected)
  );
}
```

---

## Retry Policy

Failed deliveries are retried with exponential backoff:

| Attempt | Delay | Cumulative |
|---------|-------|------------|
| 1 | Immediate | 0s |
| 2 | 1s | 1s |
| 3 | 2s | 3s |
| 4 | 4s | 7s |
| 5 | 8s | 15s |
| 6 (final) | 16s | 31s |

After 5 failed retries, the delivery moves to `exhausted` state.

### Delivery States

| State | Description |
|-------|-------------|
| `pending` | Queued for delivery |
| `delivered` | Successfully delivered (2xx response) |
| `failed` | Last attempt failed, will retry |
| `exhausted` | All retries failed |

---

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

---

## Best Practices

1. **Verify signatures** — Always validate before processing
2. **Respond quickly** — Return 200 within 5 seconds
3. **Idempotency** — Handle duplicate deliveries gracefully
4. **Use HTTPS** — Required for production endpoints
5. **Monitor failures** — Set up alerts for `exhausted` deliveries
