# Security Policy

## Reporting a vulnerability
Do not open public issues for security reports. Email:

- Primary: phucnguyen20031976@gmail.com
- Alternate: lonelycoder0710@gmail.com

Include:
- Summary and impact
- Affected component or endpoint
- Steps to reproduce
- Proof of concept (if possible)
- Environment details (version or commit, deployment type)

## Response targets
- Acknowledgment: within 2 business days
- Initial assessment: within 7 days
- Fix or mitigation target: within 30 days (severity dependent)

## Supported versions
Penshort is pre-release. Security fixes are handled on the main branch until a stable release exists.

## Threat model highlights
- API key leakage or misuse
- Authentication or authorization bypass
- Rate limit bypass enabling abuse
- SQL injection or data exfiltration
- Webhook signature forgery or replay
- Exposure of internal IDs or configuration

## Secret handling baseline (implementation requirement)
- Never store raw API keys after creation. Store a hash and a short prefix for identification.
- Never log secrets, tokens, or full API keys.
- Avoid logging full destination URLs when they include sensitive query parameters; redact by default.

## Webhook signing verification rules (baseline)
- Sign webhook payloads with HMAC using a shared secret.
- Include a timestamp and body digest in the signed data.
- Enforce a small clock skew window (for example, 5 minutes).
- Reject mismatched signatures, stale timestamps, and replayed payloads.

## Coordinated disclosure
We follow coordinated disclosure. Public disclosure will occur after a fix is released or 90 days after the initial report, whichever comes first, unless we agree on a different timeline.
