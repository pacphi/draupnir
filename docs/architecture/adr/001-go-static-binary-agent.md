# 001. Go static binary agent

**Status:** Accepted
**Date:** 2026-02-23

## Context

Draupnir needs to run on each instance managed by Sindri — environments that include Fly.io VMs, Docker containers, Kubernetes pods, E2B sandboxes, and DevPod workspaces. These environments vary significantly in what is pre-installed.

The original agent prototype existed inside the Sindri repository as a Node.js process. Node.js requires a runtime to be present, which adds ~50 MB to the container image and ~500 ms to startup time. More importantly, it creates a hard dependency on the Node.js version bundled into the Sindri base image, making the agent's runtime tied to Sindri's image build cycle.

Alternative languages considered:

| Language | Pros | Cons |
|----------|------|------|
| **Go** | Static binary, fast startup, excellent concurrency primitives, native cross-compilation | Larger binary than C/Zig; GC pauses (acceptable at this scale) |
| **Rust** | Even smaller binary, no GC | Longer compile times, unfamiliar to the team at the time; `creack/pty` ecosystem better in Go |
| **Node.js** | Shared codebase with Mimir (TypeScript) | Runtime dependency; slow startup; large deployment footprint |
| **Python** | Familiar | Runtime dependency; poor cross-compilation story |

## Decision

Implement Draupnir as a Go binary compiled with `CGO_ENABLED=0` (fully static, no glibc dependency on Linux).

The binary is cross-compiled to four targets at release time:
- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`

It is distributed as a Sindri extension (downloaded and installed by `install.sh`) and as standalone binaries attached to each GitHub Release.

## Consequences

**Positive:**
- No runtime dependencies. The install model is: download binary, `chmod +x`, run.
- Binary size is approximately 8 MB — two orders of magnitude smaller than a Node.js equivalent.
- Startup time is under 50 ms.
- Idle memory footprint is approximately 5 MB.
- Cross-compilation is native to the Go toolchain (`GOOS`/`GOARCH` environment variables).
- `CGO_ENABLED=0` avoids glibc version mismatches on older Linux distros.

**Negative / trade-offs:**
- The agent and the Mimir control plane (TypeScript) cannot share code. The shared WebSocket protocol types must be kept in sync manually across two repos and two languages (`pkg/protocol/messages.go` in this repo and `@mimir/protocol` in the mimir repo).
- Contributors need Go experience. The TypeScript team working on Mimir cannot easily contribute Go code to Draupnir.

**Follow-on constraints:**
- The `pkg/protocol/messages.go` struct definitions are the canonical source of truth for the wire format. Any protocol change must be reflected there first, then propagated to Mimir's TypeScript types.
- Adding a CGO dependency would break the static build and the current cross-compilation setup. New dependencies must be pure Go.
