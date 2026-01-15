#!/bin/bash
# Penshort Wait for Ready Script
# Waits for the API to become ready
# Used by E2E tests and CI

set -e

MAX_ATTEMPTS=${1:-30}
ATTEMPT=0
URL=${PENSHORT_URL:-http://localhost:8080}

echo "Waiting for $URL/readyz to become ready..."

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    ATTEMPT=$((ATTEMPT + 1))
    
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$URL/readyz" 2>/dev/null || echo "000")
    
    if [ "$HTTP_CODE" = "200" ]; then
        echo "Ready after $ATTEMPT attempt(s)"
        exit 0
    fi
    
    echo "  Attempt $ATTEMPT/$MAX_ATTEMPTS (status: $HTTP_CODE)"
    sleep 2
done

echo "Failed to become ready after $MAX_ATTEMPTS attempts"
exit 1
