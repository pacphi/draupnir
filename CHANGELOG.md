# Changelog

All notable changes to Draupnir are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) conventions and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

This file is automatically updated by the release workflow on each tagged release. Do not edit it by hand — use [Conventional Commits](https://www.conventionalcommits.org/) in your commit messages and the changelog will reflect your changes at the next release.

<!-- git-cliff-unreleased-start -->
## [Unreleased]
<!-- git-cliff-unreleased-end -->

## [1.0.0] — 2026-02-23

### Added

- Initial release of Draupnir as a standalone project, extracted from the Sindri monorepo
- WebSocket client with automatic reconnect logic (`internal/websocket`)
- Registration with Mimir control plane on startup (`internal/registration`)
- Configurable heartbeat sender — default 30 s (`internal/heartbeat`)
- System metrics collection via `gopsutil/v4`: CPU, memory, disk, network (`internal/metrics`)
- Multi-session PTY manager for remote terminal access (`internal/terminal`)
- Command dispatch and result streaming
- Lifecycle event emission (deploy, connect, disconnect, error)
- Environment-based configuration with nine variables (`internal/config`)
- WebSocket protocol v1.0 envelope with `protocol_version` field (`pkg/protocol`)
- Sindri extension definition and platform-aware installer script (`extension/`)
- Cross-compilation to four targets: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`
- GitHub Actions release workflow with SHA256 checksums
- Automated PR to update the Sindri extension registry on release
- Unit tests with race detector across all packages
- Pre-commit hooks for format, vet, and lint checks

[Unreleased]: https://github.com/pacphi/draupnir/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/pacphi/draupnir/releases/tag/v1.0.0
