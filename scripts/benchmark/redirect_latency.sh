#!/bin/bash
# =============================================================================
# Redirect Latency Benchmark
# =============================================================================
# Requires: hey (https://github.com/rakyll/hey)
#   Install: go install github.com/rakyll/hey@latest
#
# Usage: ./redirect_latency.sh [shortcode] [base_url]
#   Default shortcode: bench
#   Default base_url: http://localhost:8080
# =============================================================================

set -e

SHORTCODE="${1:-bench}"
BASE_URL="${2:-http://localhost:8080}"
URL="${BASE_URL}/${SHORTCODE}"

echo "============================================="
echo "Penshort Redirect Latency Benchmark"
echo "============================================="
echo "Target: ${URL}"
echo "Date: $(date -Iseconds)"
echo ""

# Check if hey is installed
if ! command -v hey &> /dev/null; then
    echo "ERROR: 'hey' is not installed"
    echo "Install with: go install github.com/rakyll/hey@latest"
    exit 1
fi

# Warm-up run
echo ">>> Warm-up (100 requests)..."
hey -n 100 -c 10 -q 50 "${URL}" > /dev/null 2>&1 || true

# Main benchmark
echo ">>> Main benchmark (10,000 requests, 100 concurrent)..."
echo ""

hey -n 10000 -c 100 "${URL}"

echo ""
echo "============================================="
echo "Benchmark complete"
echo "============================================="
