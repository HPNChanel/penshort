# Security Response Playbook

This playbook defines how maintainers handle security vulnerability reports. It supplements [SECURITY.md](../SECURITY.md) with operational procedures.

## 1. Receiving Reports

Security reports should arrive via:

- **Primary email**: phucnguyen20031976@gmail.com
- **Alternate email**: lonelycoder0710@gmail.com

Do **not** discuss security reports in public issues, PRs, or discussion forums until coordinated disclosure.

### Expected Report Contents

Per SECURITY.md, reporters should include:

- Summary and impact assessment
- Affected component or endpoint
- Steps to reproduce
- Proof of concept (if available)
- Environment details (version, deployment type)

## 2. Acknowledgment

**Timeline**: Within 2 business days of receipt.

Send a response confirming:

- Report received
- Assigned tracking ID (internal, e.g., `SEC-2026-001`)
- Expected timeline for initial assessment
- Point of contact for follow-up

### Template

```
Subject: Re: [Security Report] - Received

Thank you for reporting this security issue to Penshort.

We have received your report and assigned it tracking ID SEC-YYYY-NNN.

We will provide an initial assessment within 7 days. If you have 
additional information, please reply to this email.

Thank you for helping keep Penshort secure.

— Penshort Security Team
```

## 3. Triage Steps

### Step 3.1: Verify Report

- [ ] Confirm the reported behavior exists
- [ ] Reproduce the issue in a controlled environment
- [ ] Identify affected versions and configurations

### Step 3.2: Assess Authenticity

- [ ] Is this a valid security issue (not a feature request or general bug)?
- [ ] Does the reporter have malicious intent indicators? (rare, but note)

### Step 3.3: Classify Severity

Use the following severity levels:

| Severity | Criteria | Response Target |
|----------|----------|-----------------|
| **Critical** | Remote code execution, authentication bypass, mass data exfiltration | 24–48 hours |
| **High** | Privilege escalation, significant information disclosure, API key compromise | 7 days |
| **Medium** | Limited scope vulnerabilities, denial of service vectors, rate limit bypass | 14 days |
| **Low** | Minimal impact issues, hardening recommendations, defense-in-depth gaps | 30 days |

### Severity Examples (Penshort-Specific)

**Critical:**
- SQL injection allowing data exfiltration
- Authentication bypass to any API key
- Remote code execution via webhook payloads

**High:**
- API key hash exposure
- Cross-user data access
- Webhook signature forgery

**Medium:**
- Rate limit bypass allowing abuse
- Internal ID enumeration
- Verbose error messages exposing internals

**Low:**
- Missing security headers
- Overly permissive CORS (if applicable)
- Theoretical attacks requiring unlikely conditions

## 4. Response Timeline

Aligned with [SECURITY.md](../SECURITY.md):

| Stage | Target |
|-------|--------|
| Acknowledgment | 2 business days |
| Initial assessment | 7 days |
| Fix or mitigation | 30 days (severity dependent) |

For Critical severity, aim for fix within 48 hours if feasible.

## 5. Fix Development

### Step 5.1: Create Private Fix

- Develop fix in a private branch or fork
- Do not reference the security issue in public commits
- Use generic commit messages until disclosure

### Step 5.2: Test Thoroughly

- [ ] Verify fix resolves the vulnerability
- [ ] Confirm no regression in related functionality
- [ ] Run full test suite
- [ ] Consider edge cases and bypass attempts

### Step 5.3: Prepare Release

- Draft changelog entry (keep generic until disclosure)
- Prepare security advisory (if CVE warranted)
- Coordinate release timing with reporter

## 6. Disclosure

### Coordinated Disclosure

Per SECURITY.md:

> Public disclosure will occur after a fix is released or 90 days after the initial report, whichever comes first, unless we agree on a different timeline.

### Disclosure Steps

1. **Release fix** — Tag release, publish binaries and images
2. **Publish advisory** — GitHub Security Advisory (if applicable)
3. **Notify reporter** — Thank them, share advisory link
4. **CVE assignment** — Request CVE if severity warrants (High/Critical)
5. **Public announcement** — Changelog, release notes, social media (optional)

### Advisory Template

```markdown
## Security Advisory: [Brief Title]

**Severity**: [Critical/High/Medium/Low]
**CVE**: [CVE-YYYY-NNNNN or "Pending" or "N/A"]
**Affected Versions**: [e.g., < 1.2.3]
**Fixed Version**: [e.g., 1.2.3]

### Description

[Brief description of the vulnerability without exploitation details]

### Impact

[What could an attacker achieve?]

### Mitigation

Upgrade to version X.Y.Z or later.

### Credit

Reported by [Reporter Name/Handle] (with permission)

### Timeline

- YYYY-MM-DD: Report received
- YYYY-MM-DD: Fix released
- YYYY-MM-DD: Advisory published
```

## 7. Post-Incident

### Retrospective

After each security incident:

1. **Document** — What happened, how it was found, how it was fixed
2. **Analyze** — Root cause, why it wasn't caught earlier
3. **Improve** — What process or code changes prevent recurrence?

### Update Documentation

- [ ] Update threat model in SECURITY.md if needed
- [ ] Add regression tests for the vulnerability class
- [ ] Update security scanning rules if applicable

### Communicate Learnings

- Share lessons learned with contributors (without sensitive details)
- Consider blog post for significant vulnerabilities (after disclosure)

## 8. Contact

Security response is coordinated by maintainers listed in [MAINTAINERS.md](../MAINTAINERS.md).

For questions about this playbook, contact the Lead Maintainer.
