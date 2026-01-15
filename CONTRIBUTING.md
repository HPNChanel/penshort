# Contributing to Penshort

Thanks for helping improve Penshort! This document explains how to contribute effectively and what to expect from maintainers.

## Table of Contents

- [Ways to Contribute](#ways-to-contribute)
- [Issue Triage Process](#issue-triage-process)
- [Label Taxonomy](#label-taxonomy)
- [Good First Issues](#good-first-issues)
- [Proposing Changes](#proposing-changes)
- [Pull Requests](#pull-requests)
- [Development Setup](#development-setup)
- [Response SLAs](#response-slas)
- [Code of Conduct](#code-of-conduct)
- [Changelog Guidelines](#changelog-guidelines)
- [Security Issues](#security-issues)

---

## Ways to Contribute

- **Report bugs** — Use the [bug report template](.github/ISSUE_TEMPLATE/bug_report.yml)
- **Request features** — Use the [feature request template](.github/ISSUE_TEMPLATE/feature_request.yml)
- **Improve documentation** — Fix typos, clarify explanations, add examples
- **Fix bugs** — Look for issues labeled `bug` + `help-wanted`
- **Implement features** — Check issues labeled `enhancement` + `help-wanted`
- **Review PRs** — Thoughtful code review is always welcome

---

## Issue Triage Process

All issues go through a triage process to ensure they're actionable:

### Lifecycle

```
Opened → needs-triage → triaged → [in-progress] → closed
                ↓
         waiting-response (if clarification needed)
                ↓
         stale (after 14 days without response)
```

### What Happens

1. **New Issue** — Automatically labeled `needs-triage`
2. **Triage** — Maintainer reviews within 3 business days:
   - Applies type, priority, and area labels
   - Requests clarification if needed
   - Assigns `triaged` label when ready for work
3. **Work Begins** — Contributor or maintainer picks up
4. **Resolution** — Issue closed when PR merges or issue becomes invalid

### Stale Issues

Issues marked `waiting-response` without activity for 14 days are labeled `stale`. After 7 more days, stale issues may be closed. You can always reopen by providing the requested information.

---

## Label Taxonomy

### Type Labels

| Label | Description |
|-------|-------------|
| `bug` | Something isn't working correctly |
| `enhancement` | New feature or improvement |
| `documentation` | Documentation changes only |
| `question` | Request for clarification |

### Priority Labels

| Label | Description |
|-------|-------------|
| `priority/critical` | Security issue or complete breakage |
| `priority/high` | Major functionality affected |
| `priority/medium` | Important but not blocking |
| `priority/low` | Nice to have, minor impact |

### Status Labels

| Label | Description |
|-------|-------------|
| `needs-triage` | Awaiting maintainer review |
| `triaged` | Reviewed and ready for work |
| `waiting-response` | Needs input from reporter |
| `stale` | No activity, may be closed |
| `wontfix` | Intentionally not addressing |
| `duplicate` | Duplicate of another issue |

### Area Labels

| Label | Description |
|-------|-------------|
| `area/api` | API endpoints and behavior |
| `area/database` | PostgreSQL and migrations |
| `area/cache` | Redis caching layer |
| `area/webhooks` | Webhook delivery and signing |
| `area/analytics` | Click tracking and statistics |
| `area/auth` | API keys and authentication |
| `area/ops` | Health checks, metrics, deployment |

### Effort Labels

| Label | Description |
|-------|-------------|
| `good-first-issue` | Suitable for new contributors |
| `help-wanted` | Open for external contribution |
| `complex` | Requires deep familiarity with codebase |

---

## Good First Issues

Issues labeled `good-first-issue` are specifically curated for new contributors.

### Qualification Criteria

An issue qualifies as a Good First Issue when **all** of these are true:

1. **Clear Scope** — Defined acceptance criteria, not open-ended
2. **Limited Surface** — Changes touch ≤2 files or 1 package
3. **No Deep Domain Knowledge** — Doesn't require understanding complex subsystems
4. **Testable** — Existing tests verify correctness, or test addition is straightforward
5. **Documented** — Maintainer provides setup hints, file pointers, or approach suggestions

### What You Get

- Explicit pointers to relevant files and functions
- Clear definition of "done"
- Priority review from maintainers
- Constructive feedback if first attempt needs adjustment

### Finding Good First Issues

[View Good First Issues →](https://github.com/HPNChanel/penshort/issues?q=is%3Aissue+is%3Aopen+label%3Agood-first-issue)

---

## Proposing Changes

### Before You Start

1. **Search existing issues** — Your idea may already be discussed
2. **Check the [Roadmap](ROADMAP.md)** — Feature may be planned or explicitly out of scope
3. **Open an issue first** — For anything beyond trivial fixes

### For Bug Fixes

- Open a bug report with reproduction steps
- Wait for triage if the fix is non-obvious
- Small, clearly-correct fixes can go straight to PR

### For New Features

- Open a feature request describing the **problem** first
- Propose your solution and alternatives considered
- **Wait for maintainer feedback** before investing time
- Large features may require an ADR in `docs/adr/`

---

## Pull Requests

### PR Guidelines

- **Use the PR template** — It helps maintainers review efficiently
- **One concern per PR** — Keep changes focused and small
- **Update docs** — When behavior changes, update relevant documentation
- **Add tests** — New code should include test coverage
- **Update CHANGELOG** — Add entry under `[Unreleased]`

### Commit Messages

Use conventional commit format:

```
type: short description

[optional body with more context]

[optional footer with issue references]
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `deps`

Examples:
```
feat: add per-key rate limit configuration
fix: webhook retry now respects backoff correctly
docs: clarify redirect caching behavior
```

### Review Process

1. CI must pass before review
2. At least one maintainer approval required for merge
3. Maintainers may request changes or ask questions
4. Once approved, maintainers will merge

---

## Development Setup

### Prerequisites

- Go 1.22+
- Docker + Docker Compose
- `migrate` CLI (golang-migrate)

### Quick Start

```bash
# Clone repository
git clone https://github.com/HPNChanel/penshort.git
cd penshort

# Start dependencies
make up

# Run API (with hot reload)
make dev

# Run tests
make test

# Run linter
make lint
```

### Before Submitting

- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] Changes are tested manually
- [ ] CHANGELOG.md updated

---

## Response SLAs

Maintainers commit to the following response times:

| Activity | Response Target |
|----------|-----------------|
| **New issues** | Acknowledgment within **3 business days** |
| **Pull requests** | First review within **5 business days** |
| **Questions/discussions** | Response within **7 business days** |
| **Security reports** | Per [SECURITY.md](SECURITY.md) — **2 business days** acknowledgment |

### What "Response" Means

- Issue triaged and labeled (not necessarily resolved)
- PR reviewed with feedback or approval (not necessarily merged)
- Question answered or directed to resources

### Managing Expectations

- Complex PRs may require multiple review cycles
- Holiday periods and vacations may cause delays
- Maintainers are volunteers — patience is appreciated

### If SLA Is Missed

If you haven't received a response within the SLA:

1. Comment on the issue/PR as a gentle reminder
2. For security issues, use the alternate email in SECURITY.md

---

## Code of Conduct

All contributors must follow [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

**Summary**: Be respectful, constructive, and inclusive. Harassment or abuse will not be tolerated.

---

## Changelog Guidelines

When making changes, update `CHANGELOG.md` under the `[Unreleased]` section:

| Category | Content |
|----------|---------|
| **Added** | New features |
| **Changed** | Changes to existing functionality |
| **Deprecated** | Features to be removed |
| **Removed** | Features that were removed |
| **Fixed** | Bug fixes |
| **Security** | Vulnerability fixes |

### Writing Good Entries

Write entries from a **user's perspective**:

✅ **Good**: "Add rate limiting per API key with configurable quotas"  
❌ **Bad**: "Refactor limiter module to support key-based configuration"

✅ **Good**: "Fix webhook retry honoring exponential backoff (#145)"  
❌ **Bad**: "Update retry logic in webhook package"

---

## Security Issues

**Do not open public issues for security vulnerabilities.**

Report security issues privately as described in [SECURITY.md](SECURITY.md):

- **Primary**: phucnguyen20031976@gmail.com
- **Alternate**: lonelycoder0710@gmail.com

---

## Release Process

Releases are managed by maintainers. See [docs/releasing.md](docs/releasing.md) for the full release process.

---

## For Maintainers

If you're a maintainer, these resources will help you:

- [Triage Guide](.github/TRIAGE.md) — Step-by-step issue triage process
- [Labels Configuration](.github/labels.yml) — Label taxonomy with colors
- [Community Settings](.github/COMMUNITY_SETTINGS.md) — GitHub repository settings guide
- [Security Playbook](.github/SECURITY_RESPONSE_PLAYBOOK.md) — Vulnerability response procedures

---

## See Also

- [MAINTAINERS.md](MAINTAINERS.md) — Who maintains this project
- [GOVERNANCE.md](GOVERNANCE.md) — Decision-making process
- [ROADMAP.md](ROADMAP.md) — Future direction
- [SECURITY.md](SECURITY.md) — Vulnerability reporting
- [docs/dependency-policy.md](docs/dependency-policy.md) — Dependency update policy
- [docs/](docs/) — Technical documentation
