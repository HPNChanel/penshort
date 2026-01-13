#!/bin/bash
# Penshort API - Complete curl Examples
# =====================================
#
# Prerequisites:
#   docker compose up -d
#   curl -fsS http://localhost:8080/readyz
#
# Usage: Run individual commands or source entire file for reference

BASE_URL="http://localhost:8080"
API_KEY="your_api_key_here"  # Replace with actual key

# ============================================================
# Health Checks (no auth required)
# ============================================================

echo "=== Health Checks ==="

# Liveness probe
curl -s "$BASE_URL/healthz" | jq .
# Expected: {"status":"ok"}

# Readiness probe
curl -s "$BASE_URL/readyz" | jq .
# Expected: {"status":"ok","checks":{"postgres":"ok","redis":"ok"}}

# ============================================================
# Links - CRUD Operations
# ============================================================

echo "=== Link Operations ==="

# Create a link (minimal)
curl -X POST "$BASE_URL/api/v1/links" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "destination": "https://example.com/hello-world"
  }' | jq .
# Expected: 201 Created with link object

# Create a link (with custom alias and expiry)
curl -X POST "$BASE_URL/api/v1/links" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "destination": "https://example.com/campaign",
    "alias": "my-campaign",
    "redirect_type": 302,
    "expires_at": "2026-12-31T23:59:59Z"
  }' | jq .
# Expected: 201 Created with custom short_code "my-campaign"

# Get a link by ID
LINK_ID="01HQXK5M7Y..."  # Replace with actual ID
curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/links/$LINK_ID" | jq .
# Expected: 200 OK with link object

# List links (with pagination)
curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/links?limit=10&status=active" | jq .
# Expected: 200 OK with data array and pagination

# List links (with date filter)
curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/links?created_after=2026-01-01T00:00:00Z&limit=20" | jq .

# Update a link
curl -X PATCH "$BASE_URL/api/v1/links/$LINK_ID" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "destination": "https://example.com/updated-path",
    "enabled": true
  }' | jq .
# Expected: 200 OK with updated link

# Disable a link
curl -X PATCH "$BASE_URL/api/v1/links/$LINK_ID" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}' | jq .

# Delete a link (soft delete)
curl -X DELETE -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/links/$LINK_ID"
# Expected: 204 No Content

# ============================================================
# Redirect (no auth required)
# ============================================================

echo "=== Redirect ==="

# Test redirect (follow redirects)
curl -L "http://localhost:8080/my-campaign"
# Expected: Follows redirect to destination

# Test redirect (see headers only)
curl -I "http://localhost:8080/my-campaign"
# Expected: 301/302 with Location header

# ============================================================
# Analytics
# ============================================================

echo "=== Analytics ==="

# Get analytics (default 7 days)
curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/links/$LINK_ID/analytics" | jq .

# Get analytics (custom date range)
curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/links/$LINK_ID/analytics?from=2026-01-01&to=2026-01-31" | jq .

# Get analytics (specific breakdowns only)
curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/links/$LINK_ID/analytics?include=daily,referrers" | jq .

# ============================================================
# Webhooks
# ============================================================

echo "=== Webhooks ==="

# Create a webhook
curl -X POST "$BASE_URL/api/v1/webhooks" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://your-server.com/webhook",
    "events": ["click"]
  }' | jq .
# Expected: 201 Created with secret (shown only once!)

# Create a webhook for specific link
curl -X POST "$BASE_URL/api/v1/webhooks" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://your-server.com/webhook",
    "events": ["click"],
    "link_id": "01HQXK5M7Y..."
  }' | jq .

# List webhooks
curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/webhooks" | jq .

# Get webhook details
WEBHOOK_ID="01HQWH..."  # Replace with actual ID
curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/webhooks/$WEBHOOK_ID" | jq .

# Update webhook
curl -X PATCH "$BASE_URL/api/v1/webhooks/$WEBHOOK_ID" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}' | jq .

# Rotate webhook secret
curl -X POST -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/webhooks/$WEBHOOK_ID/rotate-secret" | jq .
# Expected: 200 OK with new secret

# List webhook deliveries
curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/webhooks/$WEBHOOK_ID/deliveries?status=failed" | jq .

# Retry a failed delivery
DELIVERY_ID="01HQWD..."
curl -X POST -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/webhooks/$WEBHOOK_ID/deliveries/$DELIVERY_ID/retry" | jq .

# Delete webhook
curl -X DELETE -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/webhooks/$WEBHOOK_ID"
# Expected: 204 No Content

# ============================================================
# API Keys
# ============================================================

echo "=== API Keys ==="

# Create an API key
curl -X POST "$BASE_URL/api/v1/api-keys" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "CI Pipeline",
    "scopes": ["links:read", "links:write"]
  }' | jq .
# Expected: 201 Created with raw key (shown only once!)

# List API keys
curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/api-keys" | jq .

# Rotate an API key
KEY_ID="01HQAK..."  # Replace with actual ID
curl -X POST -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/api-keys/$KEY_ID/rotate" | jq .
# Expected: 200 OK with new key

# Revoke an API key
curl -X DELETE -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/api-keys/$KEY_ID"
# Expected: 204 No Content

echo "=== Done ==="
