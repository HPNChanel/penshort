# Triage Guide

This guide helps maintainers triage incoming issues effectively.

## Triage Process

### Step 1: Review New Issues

Check issues labeled `needs-triage` at least every 3 business days.

[View Untriaged Issues →](https://github.com/HPNChanel/penshort/issues?q=is%3Aissue+is%3Aopen+label%3Aneeds-triage)

### Step 2: Validate Issue

For each issue, verify:

| Check | Action if Failed |
|-------|------------------|
| Is this a duplicate? | Close with `duplicate` label, link to original |
| Is this a security issue? | Close, redirect to SECURITY.md privately |
| Is it in scope? | Close with `wontfix`, explain why |
| Does it have enough detail? | Add `waiting-response`, request specifics |

### Step 3: Apply Labels

Apply exactly one label from each applicable category:

1. **Type** (required): `bug`, `enhancement`, `documentation`, or `question`
2. **Priority** (required): `priority/critical`, `priority/high`, `priority/medium`, or `priority/low`
3. **Area** (if applicable): `area/api`, `area/database`, `area/cache`, etc.
4. **Effort** (if known): `good-first-issue`, `help-wanted`, or `complex`

### Step 4: Mark Triaged

Remove `needs-triage` and add `triaged` label.

---

## Priority Guidelines

### Critical (`priority/critical`)

- Security vulnerabilities
- Data loss or corruption
- Complete service unavailability
- Blocks all users

**Response**: Same day acknowledgment, expedited fix.

### High (`priority/high`)

- Major feature broken
- Significant performance degradation
- Blocks significant use cases
- Affects production deployments

**Response**: Review within 24 hours.

### Medium (`priority/medium`)

- Partial feature breakage
- Workaround available
- Affects specific configurations
- Documentation gaps causing confusion

**Response**: Standard triage timeline (3 days).

### Low (`priority/low`)

- Minor inconvenience
- Edge cases
- Nice-to-have improvements
- Cosmetic issues

**Response**: Include in regular triage cycle.

---

## Good First Issue Criteria

Before applying `good-first-issue`, verify ALL of these:

- [ ] **Clear scope**: Issue has defined acceptance criteria
- [ ] **Limited surface**: Changes touch ≤2 files or 1 package
- [ ] **Accessible**: No deep domain knowledge required
- [ ] **Testable**: Existing tests, or test addition is trivial
- [ ] **Guided**: You've added file pointers or approach hints

### When Adding `good-first-issue`

Add a comment with:

```markdown
## Getting Started

**Files to look at:**
- `internal/foo/bar.go` - main logic here
- `internal/foo/bar_test.go` - add tests here

**Approach suggestion:**
[Brief description of how to approach this]

**Questions?**
Comment here before starting — we're happy to help!
```

---

## Handling Common Scenarios

### Security Reports in Issues

```markdown
Thank you for reporting this. Security issues should be reported privately.

Please email the details to phucnguyen20031976@gmail.com as described in [SECURITY.md](../SECURITY.md).

I'm closing this issue to prevent public disclosure. We'll follow up via email.
```

Then close immediately.

### Duplicate Issues

```markdown
This appears to be a duplicate of #123.

Closing in favor of the original issue. Please add any additional context there.
```

### Out of Scope

```markdown
Thank you for the suggestion! After review, this falls outside Penshort's current scope.

Penshort is focused on [specific scope]. For [suggested feature], you might consider [alternative].

Closing as `wontfix`, but feel free to discuss further in GitHub Discussions.
```

### Needs More Information

```markdown
Thanks for opening this issue! To help us investigate, could you provide:

- [ ] Version or commit hash
- [ ] Steps to reproduce
- [ ] Expected vs actual behavior
- [ ] Relevant logs (with secrets redacted)

Adding `waiting-response` — we'll follow up once we have more details.
```

---

## Stale Issue Handling

Issues are automatically marked stale after 30 days of inactivity (via `.github/workflows/stale.yml`).

### Exemptions

These labels exempt issues from staleness:

- `priority/critical`
- `priority/high`
- `security`
- `good-first-issue`
- `help-wanted`

### Manual Intervention

If an issue is important but lacks activity:

1. Comment with an update
2. Ping relevant people if assigned
3. Consider reducing priority

---

## Metrics to Track

### Weekly

- Untriaged issues count
- Average time to first response
- Issues closed vs opened

### Monthly

- `good-first-issue` conversion rate
- Stale issue trends
- SLA compliance

---

## See Also

- [CONTRIBUTING.md](../CONTRIBUTING.md) — Full triage process documentation
- [MAINTAINERS.md](../MAINTAINERS.md) — Maintainer expectations
- [labels.yml](./labels.yml) — Label definitions
