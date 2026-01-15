# CI Gates Specification

> Status: Production
> Owner: Penshort Maintainers
> Last Updated: 2026-01-15

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
1. Doctor checks
2. Docker Compose dependencies
3. Migrations
4. Lint
5. Unit tests
6. Integration tests
7. E2E smoke tests
8. Docs examples validation
9. Security scans

**Failure Action**: PR blocked, requires fix.

## GitHub Pages Deploy

Runs only on `main` branch pushes and publishes the `site/` directory.
