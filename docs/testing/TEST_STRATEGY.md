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
make test-e2e
make docs-check
make security
make verify
```
