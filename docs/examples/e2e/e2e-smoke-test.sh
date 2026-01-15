#!/usr/bin/env bash
# Penshort End-to-End Smoke Test
#
# Usage:
#   ./e2e-smoke-test.sh
#
# With existing API key:
#   API_KEY=pk_live_xxx ./e2e-smoke-test.sh

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
API_KEY="${API_KEY:-}"
DATABASE_URL="${DATABASE_URL:-postgres://penshort:penshort@localhost:5432/penshort?sslmode=disable}"

if [ -z "$API_KEY" ]; then
  API_KEY=$(go run ./scripts/bootstrap-api-key.go -database-url "$DATABASE_URL" -name "e2e-smoke" -scopes "admin" -format plain)
fi

if [ -z "$API_KEY" ]; then
  echo "Failed to obtain API key. Set API_KEY or DATABASE_URL."
  exit 1
fi

echo "============================================"
echo "  Penshort End-to-End Smoke Test"
echo "============================================"
echo ""

echo "Step 1: Health checks"
curl -fsS "$BASE_URL/healthz" >/dev/null
curl -fsS "$BASE_URL/readyz" >/dev/null

TIMESTAMP=$(date +%s)
DESTINATION="https://example.com/e2e-test-$TIMESTAMP"
ALIAS="e2e-test-$TIMESTAMP"

echo "Step 2: Create link"
LINK_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/links" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d "{\"destination\":\"$DESTINATION\",\"alias\":\"$ALIAS\",\"redirect_type\":302}")

LINK_ID=$(echo "$LINK_RESPONSE" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
SHORT_CODE=$(echo "$LINK_RESPONSE" | sed -n 's/.*"short_code":"\([^"]*\)".*/\1/p')

if [ -z "$LINK_ID" ] || [ -z "$SHORT_CODE" ]; then
  echo "Failed to create link"
  echo "$LINK_RESPONSE"
  exit 1
fi

echo "Step 3: Redirect"
REDIRECT_RESPONSE=$(curl -s -I "$BASE_URL/$SHORT_CODE" 2>&1 | head -10)
if ! echo "$REDIRECT_RESPONSE" | grep -q "302\|301"; then
  echo "Redirect failed"
  echo "$REDIRECT_RESPONSE"
  exit 1
fi

echo "Step 4: Generate clicks"
for i in 1 2 3; do
  curl -s -o /dev/null "$BASE_URL/$SHORT_CODE"
  sleep 0.2
done

sleep 1

echo "Step 5: Analytics"
ANALYTICS_RESPONSE=$(curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/v1/links/$LINK_ID/analytics")

if ! echo "$ANALYTICS_RESPONSE" | grep -q '"summary"'; then
  echo "Analytics response missing summary"
  echo "$ANALYTICS_RESPONSE"
  exit 1
fi

echo "Step 6: Disable link"
curl -s -X PATCH "$BASE_URL/api/v1/links/$LINK_ID" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}' >/dev/null

echo ""
echo "Smoke test complete."
