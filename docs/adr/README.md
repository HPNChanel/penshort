# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for the Penshort project.

## What is an ADR?

An ADR is a document that captures an important architectural decision made along with its context and consequences.

## Index

| ID | Title | Status | Date |
|----|-------|--------|------|
| [ADR-0001](0001-tech-stack.md) | Technology Stack Selection | Accepted | 2026-01-05 |

## Template

When creating a new ADR, use the following template:

```markdown
# ADR-XXXX: Title

## Status

Proposed | Accepted | Deprecated | Superseded by [ADR-XXXX]

## Context

What is the issue that we're seeing that is motivating this decision?

## Decision

What is the change that we're proposing and/or doing?

## Consequences

What becomes easier or more difficult to do because of this change?

### Positive

- ...

### Negative

- ...

### Neutral

- ...
```

## Naming Convention

- Files: `XXXX-short-title.md` (e.g., `0001-tech-stack.md`)
- Sequential numbering, zero-padded to 4 digits
