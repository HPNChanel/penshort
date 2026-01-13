# Authentication

Penshort uses API keys for authentication. Keys are required for all API endpoints except redirects and health checks.

## API Key Format

| Environment | Format | Example |
|-------------|--------|---------|
| Production | `psk_live_{32_chars}` | `psk_live_aBcD1234...` |
| Test | `psk_test_{32_chars}` | `psk_test_xYz98765...` |

## Using API Keys

Include the key in the `Authorization` header:

```bash
curl -H "Authorization: Bearer psk_live_your_key_here" \
  http://localhost:8080/api/v1/links
```

## Key Management

### Create a Key

```bash
curl -X POST http://localhost:8080/api/v1/api-keys \
  -H "Authorization: Bearer $EXISTING_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "CI Pipeline",
    "scopes": ["links:read", "links:write"]
  }'
```

Response (⚠️ key shown **only once**):
```json
{
  "id": "01HQXK...",
  "name": "CI Pipeline",
  "prefix": "psk_live",
  "key": "psk_live_aBcDeFgH1234567890abcdef12345678",
  "scopes": ["links:read", "links:write"],
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

Generates a new key while revoking the old one:

```bash
curl -X POST -H "Authorization: Bearer $API_KEY" \
  http://localhost:8080/api/v1/api-keys/{key_id}/rotate
```

## Scopes

| Scope | Permissions |
|-------|-------------|
| `links:read` | List and get links |
| `links:write` | Create, update, delete links |
| `analytics:read` | Query analytics |
| `webhooks:manage` | Full webhook access |

## Security Best Practices

1. **Never commit keys** to version control
2. **Use environment variables** for key storage
3. **Rotate keys** regularly (every 90 days recommended)
4. **Use minimal scopes** for each key
5. **Revoke immediately** if a key is compromised

## Rate Limits

Rate limits are applied per API key based on tier:

| Tier | Requests/minute | Burst |
|------|-----------------|-------|
| Free | 60 | 10 |
| Standard | 600 | 100 |
| Premium | 6000 | 1000 |

See [Rate Limiting](rate-limiting.md) for details on headers and handling 429 responses.
