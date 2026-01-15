# Fresh Clone Reproducibility Specification

> Status: Production
> Owner: Penshort Maintainers
> Last Updated: 2026-01-15

## Overview

This document specifies exactly what a contributor needs to run Penshort from a fresh clone. The goal is deterministic, reproducible builds that fail fast with actionable errors.

## Prerequisites

### Required Software

| Software | Minimum Version | Check Command | Installation |
|----------|-----------------|---------------|--------------|
| Go | 1.22+ | `go version` | https://golang.org/dl/ |
| Docker | 24.0+ | `docker version` | https://docs.docker.com/get-docker/ |
| Docker Compose | v2.20+ | `docker compose version` | Included with Docker Desktop |
| Git | 2.40+ | `git version` | https://git-scm.com/ |

### Required for `make verify`

| Tool | Purpose | Install |
|------|---------|---------|
| golangci-lint | Lint | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` |
| migrate | Migrations | `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest` |
| govulncheck | Dependency scan | `go install golang.org/x/vuln/cmd/govulncheck@latest` |
| gosec | SAST | `go install github.com/securego/gosec/v2/cmd/gosec@latest` |
| gitleaks | Secret scan | `go install github.com/zricethezav/gitleaks/v8@latest` |

## Supported Operating Systems

| OS | Version | Status | Notes |
|----|---------|--------|-------|
| Linux | Ubuntu 22.04+ | Primary | Tested in CI |
| Linux | Debian 12+ | Supported | Docker required |
| macOS | 13+ | Supported | Docker Desktop required |
| Windows | 10/11 (WSL2) | Supported | Use Ubuntu 22.04 |
| Windows | Native | Limited | PowerShell scripts provided |

## Quick Start (5 minutes)

```bash
git clone https://github.com/HPNChanel/penshort.git
cd penshort

make doctor
make up
curl -fsS http://localhost:8080/readyz
```

## Canonical Command Matrix

| Command | Purpose | Requires Docker | Time |
|---------|---------|-----------------|------|
| `make doctor` | Diagnose environment issues | No | <1 min |
| `make setup` | Download dependencies | No | <1 min |
| `make up` | Start API + PostgreSQL + Redis | Yes | <1 min |
| `make down` | Stop all services | Yes | <10 sec |
| `make test-unit` | Run unit tests | No | <30 sec |
| `make test-integration` | Run integration tests | Yes | <2 min |
| `make test-e2e` | Run end-to-end tests | Yes | <10 min |
| `make docs-check` | Validate docs examples | Yes | <1 min |
| `make security` | Security scans | No | <5 min |
| `make verify` | Full verification pipeline | Yes | <15 min |

## Environment Variables

### Required (production)

| Variable | Description | Example |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://user:pass@host:5432/penshort?sslmode=disable` |
| `REDIS_URL` | Redis connection string | `redis://host:6379` |

### Optional (defaults)

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | `development` | Environment name |
| `APP_PORT` | `8080` | API server port |
| `LOG_LEVEL` | `info` | Log level |
| `LOG_FORMAT` | `json` | Log format |
| `READ_TIMEOUT` | `5s` | HTTP read timeout |
| `WRITE_TIMEOUT` | `10s` | HTTP write timeout |
| `SHUTDOWN_TIMEOUT` | `30s` | Graceful shutdown timeout |
| `WEBHOOK_ALLOW_INSECURE` | `false` | Allow HTTP/localhost webhook targets for local testing |

## Verification Steps

```bash
make verify
```

Expected output (success):

```
[STEP] Doctor
[STEP] Start dependencies (postgres, redis)
[STEP] Apply migrations
[STEP] Lint
[STEP] Unit tests
[STEP] Integration tests
[STEP] E2E smoke tests
[STEP] Docs examples validation
[STEP] Security checks

[OK] Verify complete
```

## Troubleshooting

### Docker Not Starting

**Symptom**: `Cannot connect to Docker daemon`

**Solution**:
```bash
# Linux
sudo systemctl start docker
sudo usermod -aG docker $USER

# macOS/Windows
# Open Docker Desktop
```

### Port Already in Use

**Symptom**: `bind: address already in use`

**Solution**:
```bash
# Linux/macOS
lsof -i :8080

# Windows
netstat -ano | findstr :8080
```

### Database Connection Failed

**Symptom**: `failed to connect to postgres`

**Solution**:
```bash
docker compose ps
docker compose logs postgres
make down
make up
```

## CI/CD Environment

GitHub Actions mirrors local verification by running `make verify`.
