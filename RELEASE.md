# Release Process

Quick reference for maintainers cutting a new release.

## Prerequisites

- [ ] All CI checks passing on `main`
- [ ] `CHANGELOG.md` updated with release entries (moved from `[Unreleased]`)
- [ ] No open critical/high security issues

## Steps

### 1. Update Changelog

Move entries from `[Unreleased]` to a new version section:

```markdown
## [1.2.0] - 2026-01-15

### Added
- New feature description (#123)

### Fixed
- Bug fix description (#456)
```

### 2. Commit Changes

```bash
git add CHANGELOG.md
git commit -m "chore: prepare release v1.2.0"
git push origin main
```

### 3. Create Tag

```bash
git tag -a v1.2.0 -m "Release v1.2.0"
git push origin v1.2.0
```

### 4. Monitor Workflow

The [release workflow](.github/workflows/release.yml) will automatically:

1. ✅ Validate changelog and run CI
2. ✅ Build binaries for Linux, macOS, Windows (amd64/arm64)
3. ✅ Build and push Docker image to `ghcr.io`
4. ✅ Create GitHub Release with assets

### 5. Verify

- [ ] GitHub Release page shows correct version
- [ ] All binary assets attached
- [ ] Docker image pulls: `docker pull ghcr.io/hpnchanel/penshort:v1.2.0`

## Hotfix

For urgent fixes to a released version:

```bash
git checkout -b hotfix/v1.2.1 v1.2.0
# Apply fix, update changelog
git tag -a v1.2.1 -m "Hotfix: description"
git push origin v1.2.1
# Cherry-pick to main if needed
```

## Version Bumping

| Change Type | Version Bump | Example |
|-------------|--------------|---------|
| Breaking changes | MAJOR | 1.x.x → 2.0.0 |
| New features | MINOR | 1.2.x → 1.3.0 |
| Bug fixes | PATCH | 1.2.3 → 1.2.4 |

## See Also

- [Detailed Releasing Guide](docs/releasing.md) — Full versioning rules and breaking change definitions
- [CHANGELOG.md](CHANGELOG.md) — Release history
