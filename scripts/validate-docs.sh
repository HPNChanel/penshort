#!/usr/bin/env bash
# Validate documentation examples against a running API

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

BASE_URL="${PENSHORT_BASE_URL:-http://localhost:8080}"
DATABASE_URL="${DATABASE_URL:-postgres://penshort:penshort@localhost:5432/penshort?sslmode=disable}"
API_KEY="${PENSHORT_DOCS_API_KEY:-}"

require_cmd() {
  local cmd="$1"
  local fix="$2"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "[FAIL] Missing required tool: $cmd"
    echo "  Fix: $fix"
    exit 1
  fi
}

require_cmd curl "Install curl (package manager or https://curl.se/download.html)"

if [ -z "$API_KEY" ]; then
  API_KEY=$(go run ./scripts/bootstrap-api-key.go -database-url "$DATABASE_URL" -name "docs-check" -scopes "admin" -format plain)
fi

if [ -z "$API_KEY" ]; then
  echo "[FAIL] Could not obtain API key for docs validation"
  echo "  Fix: set PENSHORT_DOCS_API_KEY or ensure DATABASE_URL is correct"
  exit 1
fi

step() {
  echo "[STEP] $1"
}

step "Health checks"
curl -fsS "$BASE_URL/healthz" >/dev/null
curl -fsS "$BASE_URL/readyz" >/dev/null

step "Create link"
TS=$(date +%s)
DEST="https://example.com/docs-$TS"
RESP=$(curl -fsS -X POST "$BASE_URL/api/v1/links" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d "{\"destination\":\"$DEST\"}")

LINK_ID=$(echo "$RESP" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
SHORT_CODE=$(echo "$RESP" | sed -n 's/.*"short_code":"\([^"]*\)".*/\1/p')

if [ -z "$LINK_ID" ] || [ -z "$SHORT_CODE" ]; then
  echo "[FAIL] Link creation response missing id or short_code"
  echo "$RESP"
  exit 1
fi

step "Get link"
curl -fsS -H "Authorization: Bearer $API_KEY" "$BASE_URL/api/v1/links/$LINK_ID" >/dev/null

step "Redirect"
CODE=$(curl -s -o /dev/null -w "%{http_code}" -I "$BASE_URL/$SHORT_CODE")
if [ "$CODE" != "301" ] && [ "$CODE" != "302" ]; then
  echo "[FAIL] Expected redirect status 301/302, got $CODE"
  exit 1
fi

echo "[OK] Docs examples validated"
