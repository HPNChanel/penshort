# Authentication

Penshort uses API keys for authentication. Keys are required for all API endpoints except redirects and health checks.

## API Key Format

Keys follow this format:

```
pk_<env>_<prefix>_<secret>
```

Example:

```
pk_live_7a9f3c_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b
```

- `env`: `live` or `test`
- `prefix`: 6 hex characters (visible identifier)
- `secret`: 32 hex characters

## Using API Keys

Include the key in either header:

```bash
curl -H "Authorization: Bearer pk_live_..." \
  http://localhost:8080/api/v1/links
```

```bash
curl -H "X-API-Key: pk_live_..." \
  http://localhost:8080/api/v1/links
```

## Bootstrapping the First Key (Local)

There is no unauthenticated key creation. For local development, bootstrap a key directly into the database:

```bash
export DATABASE_URL="postgres://penshort:penshort@localhost:5432/penshort?sslmode=disable"
API_KEY=$(go run ./scripts/bootstrap-api-key.go -database-url "$DATABASE_URL" -format plain)
```

The key is printed once. Store it securely.

## Key Management

### Create a Key

```bash
curl -X POST http://localhost:8080/api/v1/api-keys \
  -H "Authorization: Bearer $EXISTING_ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "CI Pipeline",
    "scopes": ["read", "write"]
  }'
```

Response (key shown only once):

```json
{
  "id": "01HQXK...",
  "key": "pk_live_7a9f3c_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b",
  "name": "CI Pipeline",
  "key_prefix": "7a9f3c",
  "scopes": ["read", "write"],
  "rate_limit_tier": "free",
  "created_at": "2026-01-13T08:00:00Z"
}
```

### List Keys

```bash
curl -H "Authorization: Bearer $API_KEY" \
  http://localhost:8080/api/v1/api-keys
```

### Revoke a Key

```bash
curl -X DELETE -H "Authorization: Bearer $API_KEY" \
  http://localhost:8080/api/v1/api-keys/{key_id}
```

### Rotate a Key

```bash
curl -X POST -H "Authorization: Bearer $API_KEY" \
  http://localhost:8080/api/v1/api-keys/{key_id}/rotate
```

## Scopes

| Scope | Permissions |
|-------|-------------|
| `read` | List and get links and API keys |
| `write` | Create, update, delete links |
| `webhook` | Manage webhooks and deliveries |
| `admin` | Full access (implies all scopes) |

## Rate Limits

Rate limits are applied per API key based on tier:

| Tier | Requests/minute | Burst |
|------|-----------------|-------|
| Free | 60 | 10 |
| Pro | 600 | 50 |
| Unlimited | 0 (no limit) | 0 |

See [Rate Limiting](rate-limiting.md) for headers and 429 handling.

## Security Best Practices

1. Never commit keys to version control
2. Use environment variables for key storage
3. Rotate keys regularly
4. Use minimal scopes for each key
5. Revoke immediately if a key is compromised
