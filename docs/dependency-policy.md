# Dependency Update Policy

This document defines how Penshort manages dependencies, including routine updates and security patches.

## Overview

Penshort maintains a conservative dependency posture:

- **Minimal dependencies** — Only add what provides clear value
- **Prompt updates** — Keep dependencies current to reduce security debt
- **Careful upgrades** — Test thoroughly before merging major updates

## Dependency Categories

| Category | Examples | Update Approach |
|----------|----------|-----------------|
| **Direct** | chi, pgx, go-redis | Active management, prompt updates |
| **Indirect** | Transitive dependencies | Monitor via tooling, update as needed |
| **Tooling** | golangci-lint, air, migrate | Update quarterly or as needed |
| **CI/CD** | GitHub Actions | Pin versions, update quarterly |

## Routine Update Cadence

### Weekly

- **Dependabot PRs** — Review and merge patch-level updates for direct dependencies
- **CI Check** — Ensure `govulncheck` passes on main branch

### Monthly

- **Full Audit** — Review all dependencies:
  ```bash
  go list -m -mod=mod all
  go mod tidy
  govulncheck ./...
  ```
- **Minor Updates** — Evaluate and merge minor version bumps
- **Documentation** — Update go.mod/go.sum and test

### Quarterly

- **Major Versions** — Evaluate major version upgrades for key dependencies
- **Go Toolchain** — Assess upgrading Go version (after .1 or .2 patch release)
- **Action Versions** — Update GitHub Actions to latest stable
- **Tooling** — Update linters and development tools

## Security Patch Policy

Security vulnerabilities in dependencies are prioritized by severity.

### Severity-Based Response

| Severity | Response Time | Action |
|----------|---------------|--------|
| **Critical** | 48 hours | Immediate patch, expedited release |
| **High** | 7 days | Priority patch in next release cycle |
| **Medium** | 14 days | Include in scheduled update |
| **Low** | 30 days | Address in next monthly audit |

### Critical CVE Process

1. **Detection** — Via Dependabot alert, govulncheck, or external report
2. **Assessment** — Verify exploitability in Penshort's usage
3. **Patch** — Update dependency, test, merge
4. **Release** — Cut patch release if in production-affecting path
5. **Communicate** — Note in CHANGELOG.md, security advisory if warranted

### When Patching Isn't Immediate

If a dependency patch isn't available:

1. **Document** — Track in internal issue
2. **Mitigate** — Implement workarounds if possible
3. **Monitor** — Watch upstream for fix
4. **Consider alternatives** — If fix is delayed, evaluate replacement

## Tooling

### Automated Scanning

| Tool | Purpose | Frequency |
|------|---------|-----------|
| **Dependabot** | Dependency update PRs | Continuous |
| **govulncheck** | Go vulnerability database | Every CI run |
| **Trivy** | Container image scanning | Every Docker build |
| **gosec** | Go security linting | Every CI run |

### Configuration

Dependabot is configured in `.github/dependabot.yml`:

```yaml
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
    commit-message:
      prefix: "deps"
    labels:
      - "dependencies"

  - package-ecosystem: "docker"
    directory: "/"
    schedule:
      interval: "weekly"
    commit-message:
      prefix: "deps(docker)"
    labels:
      - "dependencies"

  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "monthly"
    commit-message:
      prefix: "deps(actions)"
    labels:
      - "dependencies"
```

## Breaking Changes in Dependencies

### Before Major Version Bump

1. **Review changelog** — Understand breaking changes
2. **Assess impact** — Map changes to Penshort code
3. **Create branch** — Isolate upgrade work
4. **Update code** — Adapt to API changes
5. **Test thoroughly** — Run full test suite, integration tests
6. **Document** — Note in CHANGELOG.md under "Changed"

### Go Version Upgrades

- Wait for `.1` or `.2` patch release before upgrading in production
- Test in CI first by adding new Go version to matrix
- Update `go.mod` and Dockerfile simultaneously
- Verify all platforms build successfully

## Dependency Addition Guidelines

Before adding a new dependency:

1. **Justify** — Does it provide significant value over stdlib?
2. **Evaluate** — License compatible? Actively maintained? Security history?
3. **Minimize** — Can we use a smaller, focused package instead?
4. **Document** — Note in PR why this dependency was added

### Rejection Criteria

- License incompatible with MIT
- Unmaintained (no commits in 12+ months, unaddressed issues)
- Excessive transitive dependencies
- Known unpatched vulnerabilities

## Monitoring

### Alerts

- GitHub Security Advisories (enabled)
- Dependabot alerts (enabled)
- govulncheck failures (CI enforced)

### Review

- Monthly: Review open Dependabot PRs
- Quarterly: Audit dependency tree health

## See Also

- [SECURITY.md](../SECURITY.md) — Security policy
- [CONTRIBUTING.md](../CONTRIBUTING.md) — How to contribute
- [docs/releasing.md](./releasing.md) — Release process
