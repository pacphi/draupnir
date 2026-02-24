# Draupnir

[![License](https://img.shields.io/github/license/pacphi/draupnir)](LICENSE)
[![CI](https://github.com/pacphi/draupnir/actions/workflows/ci.yml/badge.svg)](https://github.com/pacphi/draupnir/actions/workflows/ci.yml)
[![Release](https://github.com/pacphi/draupnir/actions/workflows/release.yml/badge.svg)](https://github.com/pacphi/draupnir/actions/workflows/release.yml)
[![Test](https://github.com/pacphi/draupnir/actions/workflows/test.yml/badge.svg)](https://github.com/pacphi/draupnir/actions/workflows/test.yml)

Lightweight per-instance agent for [Sindri](https://github.com/pacphi/sindri) environments, connecting to the [Mimir](https://github.com/pacphi/mimir) control plane.

Draupnir runs on each managed instance, providing real-time metrics, remote terminal access, and command dispatch over a persistent WebSocket connection.

## Features

- System metrics collection (CPU, memory, disk, network)
- Heartbeat monitoring with configurable intervals
- Remote terminal sessions (PTY allocation)
- Command dispatch and result streaming
- Lifecycle event reporting
- Static binary — no runtime dependencies

## Quick Start

### Install via Sindri

```bash
sindri extension install draupnir
```

### Manual install

```bash
curl -fsSL https://raw.githubusercontent.com/pacphi/draupnir/main/extension/install.sh | bash
```

### Build from source

```bash
make build
```

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SINDRI_CONSOLE_URL` | yes | — | Mimir control plane URL |
| `SINDRI_CONSOLE_API_KEY` | yes | — | Authentication key |
| `SINDRI_INSTANCE_ID` | no | hostname | Unique instance identifier |
| `SINDRI_PROVIDER` | no | — | fly, docker, k8s, e2b, devpod |
| `SINDRI_REGION` | no | — | Geographic region |
| `SINDRI_AGENT_HEARTBEAT` | no | 30 | Heartbeat interval (seconds) |
| `SINDRI_AGENT_METRICS` | no | 60 | Metrics interval (seconds) |
| `SINDRI_AGENT_TAGS` | no | — | Comma-separated key=value labels |
| `SINDRI_LOG_LEVEL` | no | info | debug/info/warn/error |

## Supported Platforms

| OS | Architecture | Binary |
|----|-------------|--------|
| Linux | amd64 | `draupnir-linux-amd64` |
| Linux | arm64 | `draupnir-linux-arm64` |
| macOS | amd64 | `draupnir-darwin-amd64` |
| macOS | arm64 (M-series) | `draupnir-darwin-arm64` |

## CI

```bash
make ci    # vet, format-check, test, build-all
```

## Related Projects

- [sindri](https://github.com/pacphi/sindri) — CLI tool + extension ecosystem
- [mimir](https://github.com/pacphi/mimir) — Fleet management control plane

## License

[MIT](LICENSE)
