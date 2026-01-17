# CI Gates Specification

> Status: Production
> Owner: Penshort Maintainers
> Last Updated: 2026-01-16

## Overview

CI mirrors local verification by running `make verify` in a single job. This keeps CI behavior identical to local runs.

## Gate Summary

| Gate | Time Budget | Blocks Release | Pass Criteria |
|------|-------------|----------------|---------------|
| `verify` | 30 min | Yes | `make verify` succeeds |
| `deploy-pages` | 5 min | Main only | Site deploys successfully |

## Verify Gate

**Purpose**: Run the full verification pipeline.

**Command**:

```bash
make verify
```

**Includes**:

| Step | Description | Failure Impact |
|------|-------------|----------------|
| 1. Doctor | Check prerequisites | Blocks all subsequent steps |
| 2. Dependencies | Start Postgres/Redis | Database tests will fail |
| 3. Migrations | Apply schema changes | Integration tests will fail |
| 4. Lint | Code quality checks | PR rejected |
| 5. Unit tests | Fast isolated tests | PR rejected |
| 6. Integration tests | Database/cache tests | PR rejected |
| 7. Contract tests | OpenAPI schema validation | PR rejected |
| 8. E2E tests | Full HTTP flows | PR rejected |
| 9. Docs validation | Example verification | PR rejected |
| 10. Security scans | gosec/gitleaks/govulncheck | PR rejected |

**Failure Action**: PR blocked, requires fix.

## GitHub Pages Deploy

Runs only on `main` branch pushes and publishes the `site/` directory.

## CI Troubleshooting

### Common Failures

| Symptom | Cause | Fix |
|---------|-------|-----|
| `make verify` timeout | Docker service slow | Increase timeout or check Docker health |
| `migrate` fails | Migration syntax error | Check migration files for errors |
| `golangci-lint` fails | Lint violations | Run `make lint` locally, fix issues |
| `go test` fails | Test assertion failed | Run specific test locally to debug |
| `gosec` fails | Security issue detected | Review gosec output, fix or add exclusion |
| `gitleaks` fails | Secret detected | Remove secret, rotate if exposed |
| `govulncheck` fails | Vulnerable dependency | Run `go get -u ./...` and test |

### Reproduce CI Locally

```bash
# Exact CI reproduction
make verify

# If failures occur, run steps individually:
make doctor
make up
make migrate
make lint
make test-unit
make test-integration
make test-e2e
make docs-check
make security
```

### CI Environment Details

| Setting | Value |
|---------|-------|
| Runner | `ubuntu-latest` |
| Go Version | 1.22 |
| Timeout | 30 minutes |
| Docker | Compose v2 |

## Adding New Gates

New CI gates should:

1. Have a corresponding `make` target
2. Work locally without CI-specific config
3. Be documented in this file
4. Have clear pass/fail criteria

