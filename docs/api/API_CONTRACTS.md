# API Contracts

> Status: Production  
> Owner: Penshort Maintainers  
> Last Updated: 2026-01-17

## Overview

This document defines how Penshort maintains API contracts, ensuring consistency between the OpenAPI specification and implementation.

## OpenAPI Specification

### Location

```
docs/api/openapi.yaml
```

### Update Process

Penshort follows a **spec-first** approach:

1. **Design**: Update `openapi.yaml` with proposed changes
2. **Review**: PR includes spec changes for API-affecting code
3. **Implement**: Code changes match the updated spec
4. **Validate**: Contract tests verify spec-implementation alignment

### Versioning

- Version embedded in spec: `info.version: 1.0.0`
- Path prefix: `/api/v1/`
- Breaking changes require new major version path (`/api/v2/`)

---

## Contract Testing

### Approach

| Decision | Choice |
|----------|--------|
| Generate server stubs | **No** (too heavy for project size) |
| Validate responses in CI | **Yes** (using `kin-openapi` library) |
| Test location | `tests/contract/` |
| Schema source of truth | `docs/api/openapi.yaml` |

### What Contract Tests Assert

| Test Category | Assertion |
|--------------|-----------|
| **Endpoint Existence** | All OpenAPI paths respond with non-404 for valid requests |
| **Required Fields** | Response bodies contain all `required` schema fields |
| **Error Schema** | 4xx/5xx responses match `ErrorResponse` schema |
| **Content-Type** | JSON endpoints return `application/json` header |

### Running Contract Tests

```bash
make test-contract
```

---

## Error Response Schema

All error responses follow this standard schema:

```json
{
  "error": "Human-readable error message",
  "code": "MACHINE_READABLE_CODE"
}
```

### Standard Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `UNAUTHORIZED` | 401 | Missing or invalid API key |
| `FORBIDDEN` | 403 | Valid key but insufficient permissions |
| `NOT_FOUND` | 404 | Resource does not exist |
| `INVALID_DESTINATION` | 400 | Invalid URL format |
| `INVALID_ALIAS` | 400 | Alias format invalid |
| `ALIAS_TAKEN` | 409 | Alias already in use |
| `LINK_NOT_FOUND` | 404 | Short link does not exist |
| `LINK_EXPIRED` | 410 | Link has expired |
| `RATE_LIMITED` | 429 | Rate limit exceeded |
| `INTERNAL_ERROR` | 500 | Unexpected server error |

---

## Backward Compatibility Rules (v1)

### Allowed Changes (Non-Breaking)

✅ Add new optional fields to response bodies  
✅ Add new endpoints  
✅ Add new optional query parameters  
✅ Add new error codes  
✅ Extend enum values (with client consideration)

### Prohibited Changes (Breaking)

❌ Remove or rename existing fields  
❌ Change field types  
❌ Change required/optional status of request fields  
❌ Remove endpoints  
❌ Change URL paths  
❌ Change authentication requirements

### Deprecation Process

1. **Announce**: Add `deprecated: true` in OpenAPI spec
2. **Document**: Add deprecation notice to CHANGELOG
3. **Timeline**: Minimum 2 minor versions before removal
4. **Headers**: Add `Deprecation` header to affected endpoints
5. **Remove**: Only in new major version (v2)

---

## Rate Limit Headers

All authenticated endpoints return these headers:

| Header | Description |
|--------|-------------|
| `X-RateLimit-Limit` | Requests allowed per minute |
| `X-RateLimit-Remaining` | Requests remaining in window |
| `X-RateLimit-Reset` | Unix timestamp when limit resets |
| `Retry-After` | Seconds to wait (on 429 only) |

---

## See Also

- [OpenAPI Spec](./openapi.yaml)
- [PR Checklist](./PR_CHECKLIST.md)
- [Test Strategy](../testing/TEST_STRATEGY.md)
