# Security

Penshort is designed with security as a core principle. This page highlights key security decisions.

## API Key Protection

- API keys are hashed using Argon2id before storage.
- Keys follow the format `pk_<env>_<prefix>_<secret>`.

Example:

```
User provides: pk_live_7a9f3c_4f8d2e1b9c7a5f3d2e1b9c7a5f3d2e1b
We store: $argon2id$v=19$m=65536,t=3,p=2$salt$hash
```

## Webhook Security

Webhook deliveries include:

- `X-Penshort-Signature`
- `X-Penshort-Timestamp`
- `X-Penshort-Delivery-Id`

Signatures are HMAC-SHA256 of:

```
{timestamp}.{payloadJSON}
```

Penshort signs using `sha256(secret)` as the HMAC key.

## Logging Safety

Secrets are never logged. Logs include key prefixes only.

## Rate Limiting

### Per-Key Limits

| Tier | Requests/min | Burst |
|------|--------------|-------|
| Free | 60 | 10 |
| Pro | 600 | 50 |
| Unlimited | 0 (no limit) | 0 |

### Per-IP Limits (Redirects)

Redirects apply IP rate limits using Redis.

## Input Validation

- Only `http://` and `https://` destination URLs
- Maximum URL length: 2048 characters
- Alias format: 3-50 chars, alphanumeric + hyphen

## Vulnerability Reporting

Report security issues via `SECURITY.md` in the repository.
