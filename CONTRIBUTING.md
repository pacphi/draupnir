# Contributing to Draupnir

Thank you for your interest in contributing. Draupnir is a lightweight Go agent — contributions that keep it small, correct, and dependency-light are most welcome.

## Prerequisites

| Tool | Minimum version | Install |
|------|----------------|---------|
| Go | 1.25 | [go.dev/dl](https://go.dev/dl/) |
| golangci-lint | latest | [golangci-lint.run](https://golangci-lint.run/welcome/install/) |
| make | any | OS package manager |

## Getting started

```bash
git clone https://github.com/pacphi/draupnir.git
cd draupnir
make hooks      # install pre-commit hooks
make build      # build for current platform → dist/sindri-agent
make test       # run unit tests with race detector
```

## Development workflow

```bash
make fmt        # format all Go source files (gofmt)
make vet        # run go vet
make lint       # run golangci-lint (falls back to go vet if not installed)
make audit      # run govulncheck vulnerability scan
make build-all  # cross-compile linux/amd64, linux/arm64, darwin/amd64, darwin/arm64
make ci         # full gate: vet + fmt-check + test + build-all
```

Build output goes to `dist/`. The binaries are statically compiled (`CGO_ENABLED=0`) — no glibc dependency.

## Pre-commit hooks

Running `make hooks` configures Git to use `.githooks/pre-commit`, which runs `go fmt`, `go vet`, and `golangci-lint` before every commit. The CI pipeline enforces the same checks, so the hook catches failures locally.

## Project structure

```
cmd/agent/          # binary entrypoint — startup, signal handling, goroutine wiring
internal/
  config/           # environment-based configuration loading
  heartbeat/        # periodic liveness signal sender
  metrics/          # gopsutil-based system resource collector
  registration/     # initial registration POST to Mimir on startup
  terminal/         # PTY session manager (creack/pty)
  websocket/        # gorilla/websocket client with reconnect logic
pkg/protocol/       # shared WebSocket message types (used by Mimir too)
extension/          # Sindri extension definition and install script
```

**Adding a new internal package:** create a directory under `internal/`, add a `_test.go` file alongside the implementation, and wire the package into `cmd/agent/main.go`. Keep each package's public surface minimal — the only cross-package types live in `pkg/protocol/`.

## Commit message convention

This project uses [Conventional Commits](https://www.conventionalcommits.org/). The release workflow derives the changelog automatically from commit messages.

```
<type>(<scope>): <subject>

Types: feat, fix, chore, docs, refactor, test, ci, perf
```

Examples:

```
feat(terminal): support PTY resize during active session
fix(websocket): handle reconnect after TLS handshake failure
docs: add CONTRIBUTING.md
chore(deps): update gopsutil to v4.26.1
```

Breaking changes must include a `!` after the type or a `BREAKING CHANGE:` footer:

```
feat(protocol)!: add required session_token field to registration payload
```

## Pull request guidelines

- Open an issue first for non-trivial changes so design can be discussed before code is written.
- Keep PRs focused — one logical change per PR.
- All tests must pass (`make ci`).
- Do not add new third-party dependencies without discussion. The agent's small footprint is deliberate.

## Running tests

```bash
make test                        # all packages, race detector enabled
go test ./internal/metrics/...  # single package
go test -run TestCollector ./internal/metrics/...  # single test
```

## Related projects

| Repository | Role |
|------------|------|
| [sindri](https://github.com/pacphi/sindri) | CLI that installs draupnir as an extension |
| [mimir](https://github.com/pacphi/mimir) | Control plane that receives draupnir's WebSocket connections |
