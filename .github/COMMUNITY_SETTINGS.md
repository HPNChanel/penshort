# GitHub Community Settings

This document describes GitHub-specific configuration for the Penshort repository that cannot be represented as files and must be configured through the GitHub web interface or API.

## Discussions

### Enable Discussions

1. Go to **Settings → General → Features**
2. Check **Discussions**
3. Click **Set up discussions**

### Recommended Categories

| Category | Format | Description |
|----------|--------|-------------|
| **Announcements** | Announcement | Project news and releases (maintainers only) |
| **Q&A** | Question/Answer | Get help using Penshort |
| **Ideas** | Open-ended | Feature suggestions and brainstorming |
| **Show and Tell** | Open-ended | Share what you've built with Penshort |

### Moderation Settings

- Enable **Mark as Answer** in Q&A category
- Set default sort to **Top** for announcements
- Consider enabling **Require approval** for first-time posters (spam prevention)

---

## Branch Protection

### Main Branch Rules

Go to **Settings → Branches → Add rule** for `main`:

| Setting | Value |
|---------|-------|
| Require pull request before merging | ✅ |
| Required approvals | 1 |
| Dismiss stale reviews | ✅ |
| Require status checks | CI (lint, test, build) |
| Require linear history | ✅ (recommended) |
| Include administrators | ✅ |

---

## Security Settings

### Security Tab Configuration

1. **Private vulnerability reporting**: Enable
   - Go to **Settings → Security → Private vulnerability reporting**
   - Enable to allow reporters to use GitHub's private disclosure

2. **Dependabot alerts**: Enable
   - Go to **Settings → Security → Dependabot alerts**
   - Enable for all vulnerability types

3. **Secret scanning**: Enable
   - Go to **Settings → Security → Secret scanning**
   - Enable push protection if available

---

## Issue Forms Auto-Labeling

Our issue templates (`.github/ISSUE_TEMPLATE/*.yml`) automatically apply labels:

| Template | Auto-Label |
|----------|------------|
| Bug report | `bug` |
| Feature request | `enhancement` |

To add `needs-triage` automatically, create a GitHub Action:

```yaml
# .github/workflows/auto-label.yml (optional, for auto-adding needs-triage)
name: Auto Label
on:
  issues:
    types: [opened]
jobs:
  add-triage-label:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/github-script@v7
        with:
          script: |
            await github.rest.issues.addLabels({
              owner: context.repo.owner,
              repo: context.repo.repo,
              issue_number: context.issue.number,
              labels: ['needs-triage']
            });
```

---

## Repository Settings Checklist

### General

- [ ] Description set: "Developer-focused URL shortener for API-first workflows"
- [ ] Website URL set (if deployed)
- [ ] Topics added: `url-shortener`, `api`, `golang`, `redis`, `postgresql`

### Features

- [ ] Issues enabled
- [ ] Discussions enabled
- [ ] Projects enabled (optional)
- [ ] Wiki disabled (docs in repo)

### Pull Requests

- [ ] Allow merge commits: ✅
- [ ] Allow squash merging: ✅ (recommended default)
- [ ] Allow rebase merging: ✅
- [ ] Auto-delete head branches: ✅

---

## Labels Sync

To sync labels from `labels.yml` to the repository:

```bash
# Using GitHub CLI
cat .github/labels.yml | yq -r '.[] | "gh label create \"\(.name)\" --description \"\(.description)\" --color \"\(.color)\" --force"' | sh

# Or use github-label-sync (npm package)
npx github-label-sync --labels .github/labels.yml HPNChanel/penshort
```

---

## CODEOWNERS (Optional)

If code ownership enforcement is desired, create `.github/CODEOWNERS`:

```
# Default owners for everything
* @HPNChanel

# Specific paths (examples)
# /internal/webhook/ @webhook-specialist
# /docs/ @docs-team
```

---

## See Also

- [CONTRIBUTING.md](../CONTRIBUTING.md) — Contribution guidelines
- [MAINTAINERS.md](../MAINTAINERS.md) — Maintainer responsibilities
- [SECURITY.md](../SECURITY.md) — Security policy
