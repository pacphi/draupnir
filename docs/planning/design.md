# Draupnir: Per-Instance Agent

## Design Document — February 2026

---

## 1. What Draupnir Is

**Draupnir** is a lightweight, statically-compiled Go binary that runs on each Sindri-managed instance. It bridges the instance to the [Mimir](https://github.com/pacphi/mimir) control plane over a persistent WebSocket connection, providing:

- **Heartbeat** — periodic liveness signals
- **Metrics** — real-time system resource reporting (CPU, memory, disk, network)
- **Remote terminal** — PTY session allocation for browser-based shell access
- **Command dispatch** — one-off command execution initiated from Mimir
- **Event streaming** — lifecycle events (deploy, connect, disconnect, error)

It is distributed as a [Sindri extension](https://github.com/pacphi/sindri) — installed via `sindri extension install draupnir` and configured through environment variables.

---

## 2. Why Go

The agent is a Go binary rather than a Node.js package for concrete reasons:

| Factor | Go agent | Node.js agent |
|---|---|---|
| Deployment size | ~8 MB static binary | ~50+ MB with runtime |
| Runtime dependency | None | Node.js must be installed |
| Startup time | < 50ms | ~500ms |
| Memory footprint | ~5 MB idle | ~40 MB idle |
| Cross-compilation | Native (CGO_ENABLED=0) | Requires bundling |
| Extension portability | Single binary download | Package install |

A single static binary is ideal for the Sindri extension install model: download, chmod, run.

---

## 3. Agent Architecture

```
draupnir/
├── cmd/agent/
│   └── main.go               # Entrypoint: config loading, startup, signal handling
├── internal/
│   ├── config/               # Environment-based configuration
│   ├── registration/         # POST registration to Mimir API on boot
│   ├── heartbeat/            # Periodic heartbeat sender
│   ├── metrics/              # gopsutil-based system metrics collector
│   ├── terminal/             # PTY session manager (creack/pty)
│   └── websocket/            # gorilla/websocket client + reconnect logic
├── pkg/protocol/
│   └── messages.go           # Go structs for the shared WebSocket protocol
└── extension/
    ├── extension.yaml        # Sindri extension definition
    └── install.sh            # Platform-aware binary installer
```

### Startup Sequence

```
main.go
  1. Load config from environment variables
  2. POST /api/v1/instances/register → Mimir (with retry)
  3. Open WebSocket connection to Mimir
  4. Start heartbeat goroutine (default 30s)
  5. Start metrics goroutine (default 60s)
  6. Enter message dispatch loop (terminal:create, command:dispatch, ...)
  7. Block on OS signal (SIGTERM/SIGINT)
  8. Graceful shutdown: close PTY sessions, flush pending messages, close WS
```

---

## 4. WebSocket Protocol

All messages use a JSON envelope defined in both Go (`pkg/protocol/messages.go`) and TypeScript (`@mimir/protocol` in the mimir repo).

### Envelope

```json
{
  "protocol_version": "1.0",
  "type": "<message_type>",
  "session_id": "<optional>",
  "payload": { ... }
}
```

`protocol_version` enables forward/backward compatibility checks. Both sides should silently ignore unknown fields (additive changes don't require a version bump; breaking changes do).

### Message Types

**Outbound (Draupnir → Mimir)**

| Type | Payload | Interval |
|---|---|---|
| `registration` | `RegistrationPayload` | On connect |
| `heartbeat` | `HeartbeatPayload` | Every 30s |
| `metrics` | `MetricsPayload` | Every 60s |
| `terminal:output` | `TerminalOutputPayload` | On PTY data |
| `terminal:closed` | `TerminalClosedPayload` | On PTY exit |
| `command:result` | `CommandResultPayload` | On completion |
| `event` | `EventPayload` | On occurrence |

**Inbound (Mimir → Draupnir)**

| Type | Payload | Description |
|---|---|---|
| `terminal:create` | `TerminalCreatePayload` | Allocate new PTY session |
| `terminal:close` | session_id | Close a PTY session |
| `terminal:input` | `TerminalInputPayload` | Send keystrokes to PTY |
| `terminal:resize` | `TerminalResizePayload` | Resize PTY dimensions |
| `command:dispatch` | `CommandDispatchPayload` | Execute a command |

Full type definitions: `pkg/protocol/messages.go`.

---

## 5. Configuration

All configuration is via environment variables — no config file, no flags.

| Variable | Required | Default | Description |
|---|---|---|---|
| `SINDRI_CONSOLE_URL` | yes | — | Mimir base URL |
| `SINDRI_CONSOLE_API_KEY` | yes | — | Authentication key |
| `SINDRI_INSTANCE_ID` | no | hostname | Unique instance identifier |
| `SINDRI_PROVIDER` | no | — | `fly`, `docker`, `k8s`, `e2b`, `devpod` |
| `SINDRI_REGION` | no | — | Geographic region |
| `SINDRI_AGENT_HEARTBEAT` | no | `30` | Heartbeat interval (seconds) |
| `SINDRI_AGENT_METRICS` | no | `60` | Metrics interval (seconds) |
| `SINDRI_AGENT_TAGS` | no | — | Comma-separated `key=value` labels |
| `SINDRI_LOG_LEVEL` | no | `info` | `debug`, `info`, `warn`, `error` |

---

## 6. Sindri Extension Packaging

Draupnir is distributed as a Sindri v3 extension:

```yaml
# extension/extension.yaml
metadata:
  name: draupnir
  version: 1.0.0
  description: Sindri instance agent for mimir fleet management
  category: management
  homepage: https://github.com/pacphi/draupnir

install:
  method: script
  script:
    path: install.sh
    timeout: 120

validate:
  commands:
    - name: sindri-agent
      versionFlag: --version
      expectedPattern: "\\d+\\.\\d+\\.\\d+"
```

> **Note:** The copy registered in the Sindri extension registry (`v3/extensions/draupnir/extension.yaml`) uses `category: devops` (to satisfy Sindri's schema enum) and a GitHub `url:` for the install script rather than a local `path:`.

### Install Script Logic (`extension/install.sh`)

1. Detect OS and architecture (`uname -s`, `uname -m`)
2. Resolve the latest release tag from GitHub API
3. Download the appropriate static binary (`sindri-agent-{os}-{arch}`)
4. Verify SHA256 checksum
5. Install to `~/.local/bin/sindri-agent` with `chmod +x`

### Cross-Compilation Targets

| OS | Arch | Binary name |
|---|---|---|
| Linux | amd64 | `sindri-agent-linux-amd64` |
| Linux | arm64 | `sindri-agent-linux-arm64` |
| macOS | amd64 | `sindri-agent-darwin-amd64` |
| macOS | arm64 (M-series) | `sindri-agent-darwin-arm64` |

Built with `CGO_ENABLED=0` for fully static binaries — no glibc dependency on Linux.

---

## 7. Release Automation

On `git tag v*`:

1. CI validates tag format (`v<semver>`)
2. Runs full test suite with race detector
3. Cross-compiles all 4 targets
4. Creates GitHub Release with binaries + SHA256 checksums
5. Triggers `update-extension.yml` — opens a PR against `pacphi/sindri` updating `v3/extensions/draupnir/extension.yaml` to the new version

This makes draupnir releases self-propagating into the Sindri extension registry with zero manual steps.

---

## 8. Implementation Status (as of February 2026)

| Component | Status |
|---|---|
| WebSocket client with reconnect | ✅ Complete |
| Registration with Mimir | ✅ Complete |
| Heartbeat (configurable interval) | ✅ Complete |
| System metrics (CPU, memory, disk, network) | ✅ Complete (`gopsutil/v4`) |
| PTY session manager (multi-session) | ✅ Complete (`creack/pty`) |
| Command dispatch + result streaming | ✅ Complete |
| Event emission (lifecycle events) | ✅ Complete |
| Environment-based configuration | ✅ Complete |
| Sindri extension definition + install.sh | ✅ Complete |
| Cross-compilation CI (4 targets) | ✅ Complete |
| Release workflow + SHA256 checksums | ✅ Complete |
| Automated draupnir→sindri PR on release | ✅ Complete |
| Unit tests with race detector | ✅ Complete |
| `protocol_version` field in envelope | ✅ Complete (v1.0) |

---

## 9. Development Workflow

```bash
# Build for current platform
make build

# Cross-compile all 4 targets
make build-all

# Run tests (with race detector)
make test

# Format + vet + lint
make fmt
make vet
make lint

# CI gate (vet + fmt-check + test + build-all)
make ci
```

---

## 10. Related Projects

| Repository | Role |
|---|---|
| [sindri](https://github.com/pacphi/sindri) | CLI tool — installs draupnir as an extension |
| [mimir](https://github.com/pacphi/mimir) | Fleet management control plane — receives draupnir's WebSocket connections |
| **draupnir** (this repo) | Per-instance agent — bridges each instance to mimir |
