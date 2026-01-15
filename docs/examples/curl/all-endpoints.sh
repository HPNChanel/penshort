#!/usr/bin/env bash
# Penshort API - curl Examples
#
# Prerequisites:
#   docker compose up -d
#   curl -fsS http://localhost:8080/readyz
#   API key in $API_KEY (see docs/quickstart.md)

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
API_KEY="${API_KEY:-your_api_key_here}"

if command -v jq >/dev/null 2>&1; then
  JQ="jq"
else
  JQ="cat"
fi

# ============================================================
# Health Checks (no auth required)
# ============================================================

echo "=== Health Checks ==="

curl -s "$BASE_URL/healthz" | $JQ
curl -s "$BASE_URL/readyz" | $JQ

# ============================================================
# Links - CRUD Operations
# ============================================================

echo "=== Link Operations ==="

curl -X POST "$BASE_URL/api/v1/links" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "destination": "https://example.com/hello-world"
  }' | $JQ

curl -X POST "$BASE_URL/api/v1/links" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "destination": "https://example.com/campaign",
    "alias": "my-campaign",
    "redirect_type": 302,
    "expires_at": "2026-12-31T23:59:59Z"
  }' | $JQ

LINK_ID="01HQXK5M7Y..."  # Replace with actual ID
curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/links/$LINK_ID" | $JQ

curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/links?limit=10&status=active" | $JQ

curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/links?created_after=2026-01-01T00:00:00Z&limit=20" | $JQ

curl -X PATCH "$BASE_URL/api/v1/links/$LINK_ID" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "destination": "https://example.com/updated-path",
    "enabled": true
  }' | $JQ

curl -X PATCH "$BASE_URL/api/v1/links/$LINK_ID" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}' | $JQ

curl -X DELETE -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/links/$LINK_ID"

# ============================================================
# Redirect (no auth required)
# ============================================================

echo "=== Redirect ==="

curl -L "$BASE_URL/my-campaign"

curl -I "$BASE_URL/my-campaign"

# ============================================================
# Analytics
# ============================================================

echo "=== Analytics ==="

curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/links/$LINK_ID/analytics" | $JQ

curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/links/$LINK_ID/analytics?from=2026-01-01&to=2026-01-31" | $JQ

curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/links/$LINK_ID/analytics?include=daily,referrers" | $JQ

# ============================================================
# Webhooks
# ============================================================

echo "=== Webhooks ==="

curl -X POST "$BASE_URL/api/v1/webhooks" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "target_url": "https://your-server.com/webhook",
    "event_types": ["click"],
    "name": "Example Webhook"
  }' | $JQ

curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/webhooks" | $JQ

WEBHOOK_ID="01HQWH..."  # Replace with actual ID
curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/webhooks/$WEBHOOK_ID" | $JQ

curl -X PATCH "$BASE_URL/api/v1/webhooks/$WEBHOOK_ID" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}' | $JQ

curl -X POST -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/webhooks/$WEBHOOK_ID/rotate-secret" | $JQ

curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/webhooks/$WEBHOOK_ID/deliveries?status=failed" | $JQ

DELIVERY_ID="01HQWD..."
curl -X POST -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/webhooks/$WEBHOOK_ID/deliveries/$DELIVERY_ID/retry" | $JQ

curl -X DELETE -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/webhooks/$WEBHOOK_ID"

# ============================================================
# API Keys
# ============================================================

echo "=== API Keys ==="

curl -X POST "$BASE_URL/api/v1/api-keys" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "CI Pipeline",
    "scopes": ["read", "write"]
  }' | $JQ

curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/api-keys" | $JQ

KEY_ID="01HQAK..."  # Replace with actual ID
curl -X POST -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/api-keys/$KEY_ID/rotate" | $JQ

curl -X DELETE -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/api-keys/$KEY_ID"

echo "=== Done ==="
