# Releasing Penshort

This document describes the release process for Penshort OSS.

## Versioning (SemVer)

Penshort follows [Semantic Versioning 2.0.0](https://semver.org/):

```
MAJOR.MINOR.PATCH (e.g., 1.2.3)
```

| Bump | When to Use |
|------|-------------|
| **MAJOR** | Breaking changes (see below) |
| **MINOR** | New features, backward-compatible additions |
| **PATCH** | Bug fixes, security patches, documentation |

### Pre-release Versions

Use pre-release identifiers for testing:

- `v1.0.0-alpha.1` — Early development, unstable
- `v1.0.0-beta.1` — Feature complete, testing phase
- `v1.0.0-rc.1` — Release candidate, final testing

---

## What Constitutes a Breaking Change

> [!CAUTION]
> Breaking changes require a **MAJOR** version bump. Never ship breaking changes in MINOR or PATCH releases.

### API Breaking Changes

- Removing or renaming endpoints
- Changing response schema (removing fields, changing types)
- Changing request payload requirements
- Modifying authentication/authorization requirements
- Reducing rate limits below documented values

### Configuration Breaking Changes

- Renaming environment variables
- Removing configuration options
- Changing default values with behavioral impact

### Database Breaking Changes

- Migrations requiring manual data transformation
- Schema changes that prevent rollback
- Changes to stored data formats

### Behavioral Breaking Changes

- Changing redirect status codes (301 ↔ 302)
- Modifying analytics counting methodology
- Altering webhook payload signatures
- Changing retry/backoff behavior

---

## Changelog Strategy

We use [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format with **manual curation**.

### Why Manual Changelog

- Clearer, human-readable entries
- Focus on user impact, not commit history
- Better grouping of related changes
- Editorial control over release notes

### Changelog Categories

| Category | Content |
|----------|---------|
| **Added** | New features |
| **Changed** | Changes in existing functionality |
| **Deprecated** | Features to be removed in future |
| **Removed** | Removed features |
| **Fixed** | Bug fixes |
| **Security** | Vulnerability fixes |

### Writing Good Changelog Entries

```markdown
### Added
- Rate limiting per API key with configurable quotas (#123)

### Fixed
- Webhook retry now respects exponential backoff correctly (#145)
```

**Do:**
- Start with a verb (Add, Fix, Change, Remove)
- Reference issue/PR numbers
- Focus on user impact

**Don't:**
- Copy commit messages verbatim
- Include internal refactoring details
- Use technical jargon users won't understand

---

## Docker Image Tagging

Images are published to GitHub Container Registry: `ghcr.io/hpnchanel/penshort`

| Tag | Example | Purpose |
|-----|---------|---------|
| `latest` | `:latest` | Latest stable release |
| `vX.Y.Z` | `:v1.2.3` | Immutable release tag |
| `vX.Y` | `:v1.2` | Latest patch in minor version |
| `sha-XXXXXX` | `:sha-abc1234` | Commit-pinned (CI builds) |
| `edge` | `:edge` | Latest main branch (unstable) |

### Production Recommendations

```yaml
# Pin to specific version for production
image: ghcr.io/hpnchanel/penshort:v1.2.3

# Or use minor tag for automatic patch updates
image: ghcr.io/hpnchanel/penshort:v1.2
```

---

## Cross-Platform Binaries

Release binaries are built for:

| OS | Architecture | Filename |
|----|--------------|----------|
| Linux | amd64 | `penshort_linux_amd64` |
| Linux | arm64 | `penshort_linux_arm64` |
| macOS | amd64 (Intel) | `penshort_darwin_amd64` |
| macOS | arm64 (Apple Silicon) | `penshort_darwin_arm64` |
| Windows | amd64 | `penshort_windows_amd64.exe` |

Binaries are attached as assets to each GitHub Release.

---

## How to Cut a Release

### Prerequisites

- [ ] All CI checks passing on `main`
- [ ] No critical/high severity security issues open
- [ ] `CHANGELOG.md` updated with release entries

### Step-by-Step Process

#### 1. Prepare Changelog

Move entries from `[Unreleased]` to a new version section:

```markdown
## [1.2.0] - 2026-01-15

### Added
- New webhook retry configuration options

### Fixed
- Rate limiter now correctly resets at window boundary
```

#### 2. Update Version References

If version is embedded in code (optional for Go):

```go
// internal/version/version.go
const Version = "1.2.0"
```

#### 3. Create Release Commit

```bash
git add CHANGELOG.md
git commit -m "chore: prepare release v1.2.0"
git push origin main
```

#### 4. Create and Push Tag

```bash
git tag -a v1.2.0 -m "Release v1.2.0"
git push origin v1.2.0
```

#### 5. Monitor Release Workflow

The GitHub Actions release workflow will automatically:

1. Run full CI validation
2. Build cross-platform binaries
3. Build and push Docker images
4. Create GitHub Release with assets
5. Extract changelog for release notes

#### 6. Verify Release

- [ ] GitHub Release page shows correct version and notes
- [ ] All binary assets are attached
- [ ] Docker image pulls successfully:
  ```bash
  docker pull ghcr.io/hpnchanel/penshort:v1.2.0
  ```
- [ ] Health check passes:
  ```bash
  docker run --rm ghcr.io/hpnchanel/penshort:v1.2.0 --version
  ```

#### 7. Update Unreleased Section

Add a fresh `[Unreleased]` section to `CHANGELOG.md`:

```markdown
## [Unreleased]

### Added
- (none yet)
```

#### 8. Announce (Optional)

- Post to project discussions/Discord
- Tweet/post about significant releases

---

## Release-Ready Repo Checklist

Before considering the repository release-ready:

### Documentation

- [ ] `README.md` with clear installation and usage
- [ ] `CHANGELOG.md` following Keep a Changelog
- [ ] `CONTRIBUTING.md` with contribution guidelines
- [ ] `SECURITY.md` with vulnerability reporting process
- [ ] `docs/` with API and deployment documentation

### CI/CD

- [ ] Lint, test, build passing on all PRs
- [ ] Security scanning (govulncheck, trivy, gosec)
- [ ] Release workflow triggered by version tags
- [ ] Docker images published to registry

### Code Quality

- [ ] Test coverage > 60%
- [ ] No critical linter warnings
- [ ] No known high/critical vulnerabilities

### Operational Readiness

- [ ] Health check endpoints (`/healthz`, `/readyz`)
- [ ] Structured logging (JSON format option)
- [ ] Prometheus metrics endpoint
- [ ] Graceful shutdown handling

---

## Hotfix Process

For urgent fixes to released versions:

1. **Create hotfix branch** from the release tag:
   ```bash
   git checkout -b hotfix/v1.2.1 v1.2.0
   ```

2. **Apply fix** and update changelog

3. **Tag and release**:
   ```bash
   git tag -a v1.2.1 -m "Hotfix: critical bug description"
   git push origin v1.2.1
   ```

4. **Cherry-pick to main** if applicable:
   ```bash
   git checkout main
   git cherry-pick <commit-sha>
   ```

---

## See Also

- [CHANGELOG.md](../CHANGELOG.md) — Release history
- [CONTRIBUTING.md](../CONTRIBUTING.md) — Contribution guidelines
- [Deployment](./deployment.md) — Production deployment guide
- [Demo Deployment](./demo-deployment.md) — Public demo setup
