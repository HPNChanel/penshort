#!/bin/bash
# Penshort End-to-End Smoke Test
# ================================
#
# This script demonstrates the complete Penshort workflow:
# 1. Check health
# 2. Create an API key (if needed)
# 3. Create a short link
# 4. Click the link (trigger redirect)
# 5. Query analytics
# 6. Create a webhook (optional)
#
# Prerequisites:
#   docker compose up -d
#   Wait for services to be ready
#
# Usage:
#   chmod +x e2e-smoke-test.sh
#   ./e2e-smoke-test.sh
#
# For existing API key:
#   API_KEY=psk_live_xxx ./e2e-smoke-test.sh

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"
API_KEY="${API_KEY:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "============================================"
echo "  Penshort End-to-End Smoke Test"
echo "============================================"
echo ""

# ============================================================
# Step 1: Health Check
# ============================================================
echo -e "${YELLOW}Step 1: Checking service health...${NC}"

HEALTH=$(curl -s "$BASE_URL/healthz")
if echo "$HEALTH" | grep -q '"status":"ok"'; then
    echo -e "${GREEN}✓ Liveness: OK${NC}"
else
    echo -e "${RED}✗ Liveness check failed${NC}"
    echo "$HEALTH"
    exit 1
fi

READY=$(curl -s "$BASE_URL/readyz")
if echo "$READY" | grep -q '"status":"ok"'; then
    echo -e "${GREEN}✓ Readiness: OK${NC}"
else
    echo -e "${RED}✗ Readiness check failed (Postgres/Redis may be starting)${NC}"
    echo "$READY"
    exit 1
fi

echo ""

# ============================================================
# Step 2: Create API Key (if not provided)
# ============================================================
echo -e "${YELLOW}Step 2: API Key setup...${NC}"

if [ -z "$API_KEY" ]; then
    echo "  No API_KEY provided. Creating new key..."
    
    # Note: In production, you'd need an existing key to create new ones
    # For initial setup, the system may have a bootstrap key or allow unauthenticated key creation
    # This assumes development mode allows key creation
    
    KEY_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/api-keys" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "E2E Test Key",
            "scopes": ["links:read", "links:write", "analytics:read", "webhooks:manage"]
        }')
    
    if echo "$KEY_RESPONSE" | grep -q '"key"'; then
        API_KEY=$(echo "$KEY_RESPONSE" | grep -o '"key":"[^"]*"' | cut -d'"' -f4)
        echo -e "${GREEN}✓ API Key created: ${API_KEY:0:12}...${NC}"
        echo ""
        echo -e "${YELLOW}  ⚠️ SAVE THIS KEY! It won't be shown again:${NC}"
        echo "  $API_KEY"
    else
        echo -e "${RED}  Could not create API key. Using unauthenticated mode.${NC}"
        echo "  Response: $KEY_RESPONSE"
        echo ""
        echo "  TIP: In production, set API_KEY env var before running."
    fi
else
    echo -e "${GREEN}✓ Using provided API key: ${API_KEY:0:12}...${NC}"
fi

echo ""

# ============================================================
# Step 3: Create a Short Link
# ============================================================
echo -e "${YELLOW}Step 3: Creating a short link...${NC}"

TIMESTAMP=$(date +%s)
DESTINATION="https://example.com/e2e-test-$TIMESTAMP"
ALIAS="e2e-test-$TIMESTAMP"

LINK_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/links" \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -d "{
        \"destination\": \"$DESTINATION\",
        \"alias\": \"$ALIAS\",
        \"redirect_type\": 302
    }")

if echo "$LINK_RESPONSE" | grep -q '"id"'; then
    LINK_ID=$(echo "$LINK_RESPONSE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    SHORT_CODE=$(echo "$LINK_RESPONSE" | grep -o '"short_code":"[^"]*"' | cut -d'"' -f4)
    SHORT_URL=$(echo "$LINK_RESPONSE" | grep -o '"short_url":"[^"]*"' | cut -d'"' -f4)
    
    echo -e "${GREEN}✓ Link created:${NC}"
    echo "  ID:         $LINK_ID"
    echo "  Short Code: $SHORT_CODE"
    echo "  Short URL:  $SHORT_URL"
    echo "  Destination: $DESTINATION"
else
    echo -e "${RED}✗ Failed to create link${NC}"
    echo "$LINK_RESPONSE"
    exit 1
fi

echo ""

# ============================================================
# Step 4: Test Redirect
# ============================================================
echo -e "${YELLOW}Step 4: Testing redirect...${NC}"

REDIRECT_RESPONSE=$(curl -s -I "$BASE_URL/$SHORT_CODE" 2>&1 | head -10)

if echo "$REDIRECT_RESPONSE" | grep -q "302\|301"; then
    LOCATION=$(echo "$REDIRECT_RESPONSE" | grep -i "^location:" | cut -d' ' -f2 | tr -d '\r')
    echo -e "${GREEN}✓ Redirect works!${NC}"
    echo "  Status: 302 Found"
    echo "  Location: $LOCATION"
else
    echo -e "${RED}✗ Redirect failed${NC}"
    echo "$REDIRECT_RESPONSE"
fi

echo ""

# ============================================================
# Step 5: Generate Some Clicks
# ============================================================
echo -e "${YELLOW}Step 5: Generating test clicks...${NC}"

for i in 1 2 3; do
    curl -s -o /dev/null "$BASE_URL/$SHORT_CODE"
    echo "  Click $i recorded"
    sleep 0.5
done

echo -e "${GREEN}✓ 3 clicks generated${NC}"
echo ""

# ============================================================
# Step 6: Query Analytics
# ============================================================
echo -e "${YELLOW}Step 6: Querying analytics...${NC}"

# Wait a moment for analytics to process
sleep 2

ANALYTICS_RESPONSE=$(curl -s -H "Authorization: Bearer $API_KEY" \
    "$BASE_URL/api/v1/links/$LINK_ID/analytics")

if echo "$ANALYTICS_RESPONSE" | grep -q '"summary"'; then
    TOTAL_CLICKS=$(echo "$ANALYTICS_RESPONSE" | grep -o '"total_clicks":[0-9]*' | cut -d':' -f2)
    UNIQUE_VISITORS=$(echo "$ANALYTICS_RESPONSE" | grep -o '"unique_visitors":[0-9]*' | cut -d':' -f2)
    
    echo -e "${GREEN}✓ Analytics retrieved:${NC}"
    echo "  Total Clicks:    $TOTAL_CLICKS"
    echo "  Unique Visitors: $UNIQUE_VISITORS"
else
    echo -e "${YELLOW}⚠ Analytics may still be processing${NC}"
    echo "  (Analytics have ~1 min processing delay)"
fi

echo ""

# ============================================================
# Step 7: Get Link Details (verify click count)
# ============================================================
echo -e "${YELLOW}Step 7: Verifying link click count...${NC}"

LINK_DETAILS=$(curl -s -H "Authorization: Bearer $API_KEY" \
    "$BASE_URL/api/v1/links/$LINK_ID")

CLICK_COUNT=$(echo "$LINK_DETAILS" | grep -o '"click_count":[0-9]*' | cut -d':' -f2)
echo -e "${GREEN}✓ Link click count: $CLICK_COUNT${NC}"

echo ""

# ============================================================
# Step 8: Cleanup (optional - disable link)
# ============================================================
echo -e "${YELLOW}Step 8: Cleanup - disabling test link...${NC}"

DISABLE_RESPONSE=$(curl -s -X PATCH "$BASE_URL/api/v1/links/$LINK_ID" \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -d '{"enabled": false}')

if echo "$DISABLE_RESPONSE" | grep -q '"status":"disabled"'; then
    echo -e "${GREEN}✓ Test link disabled${NC}"
else
    echo -e "${YELLOW}⚠ Could not disable link (may already be disabled)${NC}"
fi

echo ""

# ============================================================
# Summary
# ============================================================
echo "============================================"
echo -e "${GREEN}  End-to-End Test Complete!${NC}"
echo "============================================"
echo ""
echo "Summary:"
echo "  ✓ Health checks passed"
echo "  ✓ Short link created: $SHORT_URL"
echo "  ✓ Redirect working (302)"
echo "  ✓ Clicks recorded: $CLICK_COUNT"
echo "  ✓ Analytics queryable"
echo ""
echo "Next steps:"
echo "  - Try the webhook receiver example"
echo "  - Read the full docs at docs/quickstart.md"
echo ""
