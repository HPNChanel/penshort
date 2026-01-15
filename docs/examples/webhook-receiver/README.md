# Penshort Webhook Receiver Example

A minimal Go server demonstrating how to receive and verify Penshort webhooks.

## Quick Start

```bash
# Set your webhook secret (from Penshort webhook creation)
export PENSHORT_WEBHOOK_SECRET="your_secret_here"

# Run the server
go run main.go
```

Server starts on `http://localhost:9000/webhook`.

## Testing Locally with Docker

If Penshort runs in Docker, use `host.docker.internal` to reach your local receiver:

```bash
curl -X POST http://localhost:8080/api/v1/webhooks \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "target_url": "http://host.docker.internal:9000/webhook",
    "event_types": ["click"],
    "name": "Local Receiver"
  }'
```

For local testing, set `WEBHOOK_ALLOW_INSECURE=true` on the Penshort API to allow HTTP and localhost targets.

## What It Does

1. Listens on port 9000 for POST requests
2. Verifies `X-Penshort-Signature` using HMAC-SHA256
3. Validates `X-Penshort-Timestamp` is within 5 minutes
4. Logs the event details

Note: Penshort signs using `sha256(secret)` as the HMAC key.

## Example Output

```
2026/01/13 08:30:00 Starting webhook receiver on :9000
2026/01/13 08:30:15 Received click event for my-link
2026/01/13 08:30:15   Delivery ID: 01HQXK5M7Y...
2026/01/13 08:30:15   Link ID:     01HQXK5M7Y...
2026/01/13 08:30:15   Timestamp:   2026-01-13T08:30:15Z
2026/01/13 08:30:15   Referrer:    twitter.com
2026/01/13 08:30:15   Country:     US
```

## Security Notes

- Always verify signatures before processing
- Use HTTPS in production
- Store secrets securely (env vars, secrets manager)
