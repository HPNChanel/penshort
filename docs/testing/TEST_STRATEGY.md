# Penshort Test Strategy

> Status: Production
> Owner: Penshort Maintainers
> Last Updated: 2026-01-15

## Overview

This document defines the testing strategy for Penshort. The primary goal is a deterministic `make verify` that mirrors CI.

## Required Gates (make verify)

`make verify` runs these steps in order:

1. Environment readiness (doctor)
2. Dependencies start (Docker Compose)
3. Migrations apply
4. Lint
5. Unit tests
6. Integration tests
7. E2E smoke tests
8. Docs examples validation
9. Security checks (gosec + dependency scan + secret scan)

If any step fails, the command exits with a clear fix message.

## Test Pyramid

- Unit tests: fast, pure Go logic (no external services)
- Integration tests: real PostgreSQL and Redis
- E2E tests: full-stack HTTP flows

## Unit Tests

**Scope**: Pure Go logic with no external services.

**Command**:

```bash
make test-unit
```

**Notes**:
- Integration tests are skipped when `-short` is enabled.

## Integration Tests

**Scope**: Data access, cache behavior, and middleware behavior using real PostgreSQL and Redis.

**Command**:

```bash
make test-integration
```

**Requirements**:
- `DATABASE_URL` and `REDIS_URL` set
- Postgres and Redis running (use `make up`)

## Contract Tests

**Scope**: OpenAPI schema validation - verify API responses match documented spec.

**Location**: `tests/contract`

**Command**:

```bash
make contract  # or make test-contract
```

**What is validated**:
- OpenAPI spec syntax and structure
- Endpoint existence (documented paths respond)
- Error responses match ErrorResponse schema
- Response content-types are correct
- Required fields present in responses

**Notes**:
- Uses `kin-openapi` library for schema validation
- Tests skip gracefully if server not running
- Runs in CI after integration tests

## E2E Tests

**Scope**: Full HTTP flows against the running API.

**Location**: `tests/e2e`

**Command**:

```bash
make test-e2e
```

**Flows validated**:
- Create API key
- Create link
- Redirect
- Analytics ingestion
- Webhook delivery

**Notes**:
- The E2E runner can start Docker Compose automatically.
- Set `E2E_SKIP_COMPOSE=1` to use an existing stack.
- Local webhook delivery in E2E requires `WEBHOOK_ALLOW_INSECURE=true` (docker-compose sets this).

## Docs Examples Validation

**Scope**: Validate documented examples against a running server.

**Command**:

```bash
make docs-check
```

This runs `scripts/validate-docs.sh` and verifies:
- health endpoints
- link creation
- link retrieval
- redirect behavior

## Security Checks

**Scope**: Secrets detection, dependency vulnerabilities, and Go SAST.

**Command**:

```bash
make security
```

**Tools**:
- `gitleaks` for secrets scanning
- `govulncheck` for dependency vulnerabilities
- `gosec` for static analysis

## Benchmarks (Advisory)

Benchmarks are optional and run only if benchmark tests exist.

```bash
make bench
```

## Tooling Justification

| Tool | Purpose | Justification |
|------|---------|---------------|
| `golangci-lint` | Linting | Standard Go lint aggregator |
| `migrate` | Migrations | Required to apply schema changes |
| `govulncheck` | Dependency scan | Official Go vulnerability tool |
| `gosec` | SAST | Go-specific security scanner |
| `gitleaks` | Secret scan | Detects accidental secret leakage |

## Local Quick Reference

```bash
make doctor
make up
make migrate
make test-unit
make test-integration
make contract           # OpenAPI schema validation
make test-e2e
make docs-check
make security
make verify             # Runs all of the above
```

## Windows Usage (PowerShell)

All scripts have PowerShell equivalents:

```powershell
# Doctor check
.\scripts\doctor.ps1

# Full verification
.\scripts\verify.ps1

# E2E tests only
.\scripts\run-e2e.ps1

# Security scans
.\scripts\security.ps1

# Docs validation
.\scripts\validate-docs.ps1
```

> **Note**: `make` targets work on Windows with WSL2 or Git Bash. Native PowerShell users should use the scripts directly.

## CI/Local Alignment

CI runs exactly `make verify`. This ensures:

| Guarantee | Description |
|-----------|-------------|
| **Identical Steps** | Same 9-step pipeline locally and in CI |
| **Same Tool Versions** | CI installs tools at `@latest` matching local |
| **Same Environment** | Docker Compose services match CI |

If `make verify` passes locally, CI will pass.

## Example HTTP Responses

### Health Check (GET /healthz)

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"status":"ok"}
```

### Readiness Check (GET /readyz)

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"status":"ok","checks":{"database":"ok","redis":"ok"}}
```

### Create Link (POST /api/v1/links)

```http
HTTP/1.1 201 Created
Content-Type: application/json

{
  "id": "01HQXY...",
  "short_code": "abc123",
  "destination": "https://example.com",
  "redirect_type": 302,
  "created_at": "2026-01-16T12:00:00Z"
}
```

### Redirect (GET /{short_code})

```http
HTTP/1.1 302 Found
Location: https://example.com
X-Penshort-Link-Id: 01HQXY...
```

### Error Response (4xx/5xx)

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "validation_error",
  "message": "destination is required",
  "request_id": "req_01HQXY..."
}
```

## Failure Troubleshooting

| Failure | Likely Cause | Fix |
|---------|--------------|-----|
| `make doctor` fails | Missing tools | Run the install command shown |
| Postgres not healthy | Port conflict | `lsof -i :5432` or change port |
| Redis not healthy | Port conflict | `lsof -i :6379` or change port |
| E2E webhook timeout | Firewall blocking | Check `host.docker.internal` connectivity |
| Security scan fails | Known vulnerability | Update dependencies: `go get -u ./...` |

