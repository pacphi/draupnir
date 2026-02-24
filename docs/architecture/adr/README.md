# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for Draupnir. ADRs capture significant decisions made during the design and evolution of the agent — not what was decided, but why.

## Format

Each ADR is a Markdown file named `NNN-short-title.md` (zero-padded three-digit index). It follows this structure:

```
# NNN. Title

**Status:** Accepted | Superseded by [NNN](NNN-...) | Deprecated
**Date:** YYYY-MM-DD

## Context
What problem or constraint prompted this decision?

## Decision
What was decided?

## Consequences
What are the trade-offs, follow-on work, or constraints introduced?
```

## Index

| # | Title | Status |
|---|-------|--------|
| [001](001-go-static-binary-agent.md) | Go static binary agent | Accepted |
| [002](002-websocket-protocol-v1.md) | WebSocket protocol v1 envelope | Accepted |
| [003](003-environment-based-configuration.md) | Environment-based configuration (no config file) | Accepted |
