#!/bin/bash
# =============================================================================
# Setup Test Links for Benchmarking
# =============================================================================
# Creates 1000 links for cache miss testing.
# Short codes are saved to /tmp/bench-codes.txt for use by k6 scripts.
#
# Usage:
#   ./setup-links.sh <API_KEY> [BASE_URL]
#   ./setup-links.sh sk_abc123 http://localhost:8080
#
# Prerequisites:
#   - Docker Compose stack running
#   - API key bootstrapped
#   - curl and jq installed
# =============================================================================

set -e

API_KEY="${1:?Missing API_KEY. Usage: $0 <API_KEY> [BASE_URL]}"
BASE_URL="${2:-http://localhost:8080}"
NUM_LINKS="${3:-1000}"
OUTPUT_FILE="/tmp/bench-codes.txt"

echo "============================================="
echo "Penshort Benchmark Data Setup"
echo "============================================="
echo "Base URL: ${BASE_URL}"
echo "Links to create: ${NUM_LINKS}"
echo "Output: ${OUTPUT_FILE}"
echo ""

# Check prerequisites
for cmd in curl jq; do
  if ! command -v "$cmd" &> /dev/null; then
    echo "ERROR: '$cmd' is not installed"
    exit 1
  fi
done

# Wait for service readiness
echo ">>> Checking service readiness..."
for i in $(seq 1 30); do
  if curl -sf "${BASE_URL}/readyz" > /dev/null 2>&1; then
    echo "Service is ready"
    break
  fi
  echo "Waiting for service... (${i}/30)"
  sleep 1
done

# Clear previous output
> "${OUTPUT_FILE}"

# Create links with progress
echo ""
echo ">>> Creating ${NUM_LINKS} links..."
success=0
failed=0

for i in $(seq 1 "$NUM_LINKS"); do
  response=$(curl -sf -X POST "${BASE_URL}/api/v1/links" \
    -H "Authorization: Bearer ${API_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"destination\": \"https://example.com/bench-${i}\"}" 2>/dev/null) || true

  if [ -n "$response" ]; then
    short_code=$(echo "$response" | jq -r '.short_code // empty')
    if [ -n "$short_code" ]; then
      echo "$short_code" >> "${OUTPUT_FILE}"
      ((success++))
    else
      ((failed++))
    fi
  else
    ((failed++))
  fi

  # Progress indicator every 100 links
  if [ $((i % 100)) -eq 0 ]; then
    echo "  Created ${success}/${i} links..."
  fi
done

echo ""
echo "============================================="
echo "Setup Complete"
echo "============================================="
echo "Created: ${success} links"
echo "Failed: ${failed} links"
echo "Output saved to: ${OUTPUT_FILE}"
echo ""
echo "To run benchmarks:"
echo "  k6 run bench/scripts/redirect-latency.js"
