# 003. Environment-based configuration (no config file)

**Status:** Accepted
**Date:** 2026-02-23

## Context

Draupnir needs access to credentials (the Mimir API key) and runtime parameters (heartbeat interval, instance identity, tags). The configuration mechanism must work across all supported deployment environments: Fly.io secrets, Docker environment variables, Kubernetes Secrets, systemd `EnvironmentFile`, and plain shell exports in a developer terminal.

Options considered:

| Option | Notes |
|--------|-------|
| Environment variables only | Universally supported across all deployment environments; secrets injected without hitting disk |
| YAML/TOML config file | Familiar for server software; requires file path conventions and file permissions management; awkward in containers |
| CLI flags | Verbose for long-running processes; not well-suited for secrets |
| Mixed (env + file + flags) | Maximum flexibility; significantly more complex to document and test |

## Decision

All configuration is through environment variables. There is no configuration file, no CLI flags (beyond `--version` and `--help`), and no layered precedence resolution.

All variables use the `SINDRI_` prefix to namespace them and signal their relationship to the Sindri ecosystem. Variables fall into two categories:

- **Required:** `SINDRI_CONSOLE_URL`, `SINDRI_CONSOLE_API_KEY` — agent refuses to start without these
- **Optional with defaults:** all others — sensible defaults are chosen so the agent works in a minimal environment without explicit configuration

The `internal/config` package is the sole place where environment variables are read. No other package reads from `os.Getenv` directly.

## Consequences

**Positive:**
- Works identically in every deployment environment — no special file provisioning or path management.
- Secrets (the API key) never touch the filesystem from the agent's perspective; the runtime injects them into the process environment.
- Configuration is trivially visible (`env | grep SINDRI_`) and easy to override for debugging.
- Single source of truth: `internal/config/config.go` is the complete configuration reference.
- No file permissions, file ownership, or config format parsing to worry about.

**Negative / trade-offs:**
- Environment variables are process-wide and visible to all goroutines, which is fine here but would be a problem in a multi-tenant context (not applicable to Draupnir — one agent per instance).
- Values that change frequently (e.g., rotating API keys) require a process restart to pick up; there is no hot-reload. Restarting the agent is acceptable — the control plane handles reconnects gracefully.
- Long or complex values (many tags, long URLs) are somewhat awkward as environment variables compared to structured YAML. The `SINDRI_AGENT_TAGS` comma-separated format is a pragmatic compromise.

**Follow-on constraints:**
- Any new configuration option must be added to `internal/config/config.go` and documented in [docs/CONFIGURATION.md](../../CONFIGURATION.md). Adding options in other packages is not permitted.
- The `SINDRI_` prefix must be maintained for all future variables to avoid collisions with other tools running in the same environment.
- If the number of variables grows substantially (>20), or if structured/nested configuration becomes necessary, this decision should be revisited in a new ADR.
