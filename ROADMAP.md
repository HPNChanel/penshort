# Roadmap

This document outlines the planned features and improvements for Penshort. Items here are **candidates** — they represent directions under consideration, not commitments.

## Current Status

**Production-ready Q1 2026** — Core functionality complete:

- ✅ Link CRUD with custom aliases
- ✅ Redis-cached redirects (301/302)
- ✅ Expiration policies (time-based, click-count)
- ✅ API key authentication
- ✅ Per-key and per-IP rate limiting
- ✅ Click analytics with time-range queries
- ✅ Webhooks with signing and retries
- ✅ Health/readiness endpoints and Prometheus metrics

---

## vNext Candidates

The following features are under evaluation for future releases. Community feedback is welcome via [GitHub Discussions](https://github.com/HPNChanel/penshort/discussions) or Issues.

### High Priority

| Feature | Description | Complexity | Notes |
|---------|-------------|------------|-------|
| **Team/Organization Support** | Multi-user workspaces with shared link ownership, team-level API keys | High | Foundation for enterprise use |
| **Role-Based Access Control (RBAC)** | Granular permissions: admin, editor, viewer roles | High | Depends on Team support |
| **Web UI Dashboard** | Browser-based interface for link management, analytics viewing | High | Optional add-on, API-first remains core |

### Medium Priority

| Feature | Description | Complexity | Notes |
|---------|-------------|------------|-------|
| **Geo Enrichment** | IP-to-location resolution for analytics (country, region) | Medium | Privacy considerations, optional |
| **Advanced Analytics** | Custom date ranges, CSV export, trend analysis, comparison views | Medium | Builds on existing analytics |
| **Link Tags/Labels** | Organize links with custom taxonomy, filter by tag | Low | Quality-of-life improvement |
| **Bulk Operations** | Create, update, delete multiple links via single API call | Medium | Efficiency for high-volume users |

### Under Consideration

| Feature | Description | Complexity | Notes |
|---------|-------------|------------|-------|
| **Custom Domains** | User-provided short domains with SSL termination | High | Operational complexity |
| **UTM Builder** | Automatic UTM parameter management and templates | Low | Marketing use case |
| **Link Groups/Campaigns** | Logical grouping of links with shared settings | Medium | Builds on Tags feature |
| **Scheduled Links** | Activate/deactivate links at specific times | Low | Event marketing use case |
| **QR Code Generation** | Generate QR codes for short links | Low | Convenience feature |
| **A/B Testing** | Route traffic to multiple destinations with split ratios | High | Advanced use case |

---

## Not Planned

These features are explicitly **out of scope** for Penshort:

| Feature | Reason |
|---------|--------|
| **Consumer-facing shortener** | Penshort is developer/API-focused, not a general URL shortener |
| **Social media preview customization** | Link metadata is pass-through from destination |
| **Built-in email/notification service** | Use webhooks to integrate with your notification system |
| **Multi-tenancy with billing** | Penshort is self-hosted; monetization is out of scope |

---

## How Features Are Prioritized

1. **Alignment** — Does it fit Penshort's mission (developer-focused, API-first)?
2. **Demand** — Is there community interest (issues, discussions)?
3. **Complexity** — Can it be implemented without destabilizing core?
4. **Maintainability** — Can it be maintained long-term with current resources?

---

## Contributing to Roadmap

### Proposing Features

1. Open a [Feature Request](https://github.com/HPNChanel/penshort/issues/new?template=feature_request.yml)
2. Describe the problem you're solving
3. Explain how it fits Penshort's scope
4. Include acceptance criteria

### Contributing to Planned Features

1. Check if the feature has an open issue
2. Comment expressing interest to work on it
3. Wait for maintainer acknowledgment and scope agreement
4. Features marked `help-wanted` are ready for contribution

---

## Version Planning

| Version | Focus | Status |
|---------|-------|--------|
| **1.0** | Core functionality, production-ready | ✅ Complete |
| **1.x** | Stability, bug fixes, minor improvements | Current |
| **2.0** | Team/Org support, RBAC | Planning |
| **Future** | UI, advanced analytics, geo | Candidate |

---

## Updates

This roadmap is reviewed quarterly. Last updated: **2026-01-15**.

For the latest status, check:
- [GitHub Milestones](https://github.com/HPNChanel/penshort/milestones)
- [Project Board](https://github.com/HPNChanel/penshort/projects) (if enabled)
