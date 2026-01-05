# Penshort

Penshort is a developer-focused URL shortener for API-first workflows. It is not a general-purpose consumer shortener.

Status: Phase 0 OSS foundation only. Application code is not implemented yet.

## Project goals
- Build a specialized shortener for developer workflows: API keys, analytics, webhooks, rate limits, and expiration policies.
- Provide reliable, cache-backed redirects with clear operational behavior.
- Keep the system small-team friendly and easy to operate.

## Planned stack
- Go for the API service
- PostgreSQL as the system of record
- Redis for cache and rate limiting

## Feature scope (planned)
- Link management: create, update, disable, and redirect (301/302) with optional custom aliases.
- Expiration policies: time-based and optional click-count based.
- Analytics: click events (timestamp, referrer, user-agent, optional region), unique vs total, query by link and time range.
- Webhooks: signed delivery on click, retries with backoff, and delivery state.
- Authentication and rate limiting: API keys per user/team, per-key limits, optional per-IP limit for redirect.
- Admin/ops: health and readiness endpoints, minimal admin surface.

## Non-functional goals (planned)
- Fast redirects backed by Redis cache for short_code to destination mapping.
- Reliability: do not lose click events; webhook delivery is tracked.
- Security baseline: API keys never stored in plaintext; logs do not contain secrets.

## Roadmap outline
- Milestone 1: Core redirect and link CRUD with Redis caching.
- Milestone 2: API keys and rate limiting.
- Milestone 3: Click events and analytics queries.
- Milestone 4: Webhooks with signing and retries.
- Milestone 5: Ops endpoints, monitoring basics, and delivery evidence.

## Quickstart (placeholders)
This section will be filled once application code lands.
- Prerequisites: TBD
- Local development: TBD
- Configuration and environment variables: TBD
- Migrations: TBD
- Testing: TBD

## Repository docs
- [CONTRIBUTING.md](CONTRIBUTING.md) - how to propose changes and submit PRs.
- [SECURITY.md](SECURITY.md) - how to report security issues.
- [GOVERNANCE.md](GOVERNANCE.md) - maintainership and decision process.
- [docs/adr/](docs/adr/) - architecture decision records.

## License
MIT License. See [LICENSE](LICENSE).
