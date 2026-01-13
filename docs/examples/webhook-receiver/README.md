# Penshort Webhook Receiver Example

A minimal Go server demonstrating how to receive and verify Penshort webhooks.

## Quick Start

```bash
# Set your webhook secret (from Penshort webhook creation)
export PENSHORT_WEBHOOK_SECRET="whsec_your_secret_here"

# Run the server
go run main.go
```

Server starts on `http://localhost:9000/webhook`.

## Testing Locally with Docker

If Penshort runs in Docker, use `host.docker.internal` to reach your local receiver:

```bash
# Create webhook pointing to local receiver
curl -X POST http://localhost:8080/api/v1/webhooks \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "http://host.docker.internal:9000/webhook",
    "events": ["click"]
  }'
```

## What It Does

1. **Listens** on port 9000 for POST requests
2. **Verifies** the `X-Penshort-Signature` header using HMAC-SHA256
3. **Validates** timestamp is within ±5 minutes (prevents replay attacks)
4. **Logs** the event details

## Example Output

```
2026/01/13 08:30:00 Starting webhook receiver on :9000
2026/01/13 08:30:15 ✓ Received click event for my-link
2026/01/13 08:30:15   Link ID:  01HQXK5M7Y...
2026/01/13 08:30:15   Time:     2026-01-13T08:30:15Z
2026/01/13 08:30:15   Referrer: twitter.com
2026/01/13 08:30:15   Country:  US
```

## Security Notes

- Always verify signatures before processing
- Use HTTPS in production
- Store secrets securely (env vars, secrets manager)
