# Security Policy

## Supported versions

| Version | Support status |
|---------|---------------|
| `1.x` (latest) | Active — security fixes applied |
| Pre-release (`-alpha`, `-beta`, `-rc`) | Not supported — upgrade to latest stable |

Only the latest stable release receives security fixes. If you are running an older `1.x` version, upgrade to the current release before reporting.

## Reporting a vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Report vulnerabilities through [GitHub Security Advisories](https://github.com/pacphi/draupnir/security/advisories/new). This keeps the report confidential until a fix is available.

Please include:
- A description of the vulnerability and its potential impact
- Steps to reproduce or a proof-of-concept (if available)
- The draupnir version affected
- Your environment (OS, architecture, how the agent is deployed)

### Response timeline

| Milestone | Target |
|-----------|--------|
| Acknowledgment | Within 48 hours |
| Initial assessment | Within 5 business days |
| Fix for critical issues | Within 14 days |
| Fix for non-critical issues | Next scheduled release |

You will be credited in the release notes unless you request otherwise.

## Threat model

Draupnir is a per-instance agent. Understanding its trust boundaries helps scope what constitutes a security issue:

**In scope:**
- Authentication bypass against the Mimir API key check
- PTY session escape allowing access outside the intended shell environment
- Credential leakage (API key, instance ID) through logs or error messages
- WebSocket message injection enabling unauthorized command execution
- Unsafe handling of inbound `command:dispatch` payloads

**Out of scope (by design):**
- The agent does not expose any inbound network listener — all communication is outbound WebSocket to Mimir. There is no port to attack from outside the instance.
- The agent runs with the same OS privileges as the user who started it. Privilege escalation within the host OS is outside the agent's control.
- Issues in Mimir (the control plane) or Sindri (the CLI) should be reported to those repositories.

## Dependency vulnerabilities

Run `make audit` to check for known vulnerabilities in Go dependencies:

```bash
make audit   # requires govulncheck (installed automatically if missing)
```

Dependencies are reviewed and updated regularly. The release workflow includes a vulnerability check step.
