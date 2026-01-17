# API Consistency PR Checklist

Use this checklist when reviewing PRs that affect the API surface.

## Required Checks

### OpenAPI Spec

- [ ] `docs/api/openapi.yaml` updated if endpoints/schemas changed
- [ ] New endpoints have complete request/response schemas
- [ ] Examples provided for new request/response bodies

### Error Handling

- [ ] Error responses use `ErrorResponse` schema (`error`, `code`)
- [ ] New error codes documented in `API_CONTRACTS.md`
- [ ] 4xx errors have descriptive messages

### Backward Compatibility (v1)

- [ ] No fields removed from existing responses
- [ ] No field types changed
- [ ] No required fields added to request bodies
- [ ] Breaking changes documented and versioned (v2)

### Testing

- [ ] Contract tests added/updated for new endpoints
- [ ] Error paths tested (400, 401, 404, etc.)
- [ ] E2E coverage for critical flows

### Documentation

- [ ] CHANGELOG updated for user-facing changes
- [ ] README updated if new features added

---

## Quick Reference

**Spec Location**: `docs/api/openapi.yaml`

**Error Response Format**:
```json
{"error": "Message", "code": "ERROR_CODE"}
```

**Common Error Codes**: `UNAUTHORIZED`, `NOT_FOUND`, `INVALID_DESTINATION`, `ALIAS_TAKEN`, `RATE_LIMITED`

---

## See Also

- [API Contracts](./API_CONTRACTS.md)
- [OpenAPI Spec](./openapi.yaml)
