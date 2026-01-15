#!/usr/bin/env bash
# Penshort E2E Test Runner
# Run: make test-e2e

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PENSHORT_BASE_URL="${PENSHORT_BASE_URL:-http://localhost:8080}"
export PENSHORT_BASE_URL

SKIP_COMPOSE="${E2E_SKIP_COMPOSE:-0}"
SKIP_DOWN="${E2E_SKIP_DOWN:-0}"

cleanup() {
  if [ "$SKIP_DOWN" = "1" ] || [ "$SKIP_COMPOSE" = "1" ]; then
    return
  fi
  docker compose down -v 2>/dev/null || true
}

trap cleanup EXIT

echo "Penshort E2E Test Runner"
echo "=========================="

echo "Base URL: $PENSHORT_BASE_URL"

if [ "$SKIP_COMPOSE" != "1" ]; then
  echo "[STEP] Starting Docker Compose stack"
  docker compose up -d --build

  echo "[STEP] Waiting for API readiness"
  ./scripts/wait-for-ready.sh
fi

echo "[STEP] Running E2E tests"
go test -v -tags=e2e ./tests/e2e/...

echo "[OK] E2E tests passed"
