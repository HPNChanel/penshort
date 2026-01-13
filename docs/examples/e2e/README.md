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
1. ✅ Check health endpoints
2. ✅ Create an API key
3. ✅ Create a short link
4. ✅ Test redirect (302)
5. ✅ Generate test clicks
6. ✅ Query analytics
7. ✅ Cleanup (disable link)

## Manual Step-by-Step

### 1. Start Services

```bash
docker compose up -d
curl -fsS http://localhost:8080/readyz
```

### 2. Create API Key

```bash
curl -X POST http://localhost:8080/api/v1/api-keys \
  -H "Content-Type: application/json" \
  -d '{"name": "My Key", "scopes": ["links:read", "links:write", "analytics:read"]}'
```

Save the returned key — it's shown only once!

### 3. Create Short Link

```bash
export API_KEY="psk_live_your_key_here"

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
API_KEY=psk_live_xxx ./docs/examples/e2e/e2e-smoke-test.sh
```
