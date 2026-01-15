#!/usr/bin/env bash
# Penshort verification runner
# Runs doctor, starts dependencies, applies migrations, runs tests, docs checks, and security scans.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PENSHORT_BASE_URL="${PENSHORT_BASE_URL:-http://localhost:8080}"
DATABASE_URL="${DATABASE_URL:-postgres://penshort:penshort@localhost:5432/penshort?sslmode=disable}"
REDIS_URL="${REDIS_URL:-redis://localhost:6379}"

export PENSHORT_BASE_URL
export PENSHORT_URL="$PENSHORT_BASE_URL"
export DATABASE_URL
export REDIS_URL

step() {
  echo ""
  echo "[STEP] $1"
}

wait_for_health() {
  local service="$1"
  local max_attempts="${2:-30}"
  local attempt=0
  local id

  id=$(docker compose ps -q "$service")
  if [ -z "$id" ]; then
    echo "[FAIL] Service $service is not running"
    echo "  Fix: run 'docker compose up -d $service'"
    exit 1
  fi

  while [ "$attempt" -lt "$max_attempts" ]; do
    attempt=$((attempt + 1))
    local status
    status=$(docker inspect --format '{{.State.Health.Status}}' "$id" 2>/dev/null || echo "unknown")
    if [ "$status" = "healthy" ]; then
      echo "[OK] $service is healthy"
      return 0
    fi
    echo "  Waiting for $service health ($attempt/$max_attempts)"
    sleep 2
  done

  echo "[FAIL] $service did not become healthy"
  docker compose logs "$service" || true
  exit 1
}

cleanup() {
  if [ "${VERIFY_SKIP_DOWN:-0}" = "1" ]; then
    return
  fi
  docker compose down || true
}

trap cleanup EXIT

step "Doctor"
./scripts/doctor.sh

step "Start dependencies (postgres, redis)"
docker compose up -d postgres redis
wait_for_health postgres
wait_for_health redis

step "Apply migrations"
make migrate

step "Lint"
make lint

step "Start API"
docker compose up -d api
./scripts/wait-for-ready.sh

step "Unit tests"
make test-unit

step "Integration tests"
make test-integration

step "E2E smoke tests"
E2E_SKIP_COMPOSE=1 make test-e2e

step "Docs examples validation"
make docs-check

step "Security checks"
make security

echo ""
echo "[OK] Verify complete"
