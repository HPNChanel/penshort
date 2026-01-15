# End-to-End Examples

Complete workflow examples for Penshort.

## Quick Smoke Test

Run the full workflow in one script:

```bash
# Start services
docker compose up -d

# Wait for ready
curl -fsS http://localhost:8080/readyz

# Run end-to-end test
chmod +x docs/examples/e2e/e2e-smoke-test.sh
./docs/examples/e2e/e2e-smoke-test.sh
```

The script will:
1. Check health endpoints
2. Bootstrap an API key (if API_KEY is not set)
3. Create a short link
4. Test redirect (302)
5. Generate test clicks
6. Query analytics
7. Cleanup (disable link)

## Manual Step-by-Step

### 1. Start Services

```bash
docker compose up -d
curl -fsS http://localhost:8080/readyz
```

### 2. Bootstrap an Admin API Key

```bash
export DATABASE_URL="postgres://penshort:penshort@localhost:5432/penshort?sslmode=disable"
API_KEY=$(go run ./scripts/bootstrap-api-key.go -database-url "$DATABASE_URL" -format plain)
```

Save the returned key; it is shown only once.

### 3. Create Short Link

```bash
curl -X POST http://localhost:8080/api/v1/links \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"destination": "https://example.com", "alias": "my-link"}'
```

### 4. Test Redirect

```bash
curl -I http://localhost:8080/my-link
# Expected: 302 Found with Location header
```

### 5. Query Analytics

```bash
curl -H "Authorization: Bearer $API_KEY" \
  "http://localhost:8080/api/v1/links/{link_id}/analytics"
```

## With Existing API Key

```bash
API_KEY=pk_live_xxx ./docs/examples/e2e/e2e-smoke-test.sh
```
