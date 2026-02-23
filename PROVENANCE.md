# Provenance

Draupnir originated as the instance-agent component within the [Sindri](https://github.com/pacphi/sindri) project. As the fleet-management architecture crystallized into distinct runtime boundaries — CLI, control plane, and per-instance agent — draupnir was established as an independent project to own its release lifecycle, cross-compilation targets, and protocol evolution.

## Lineage

| | |
|---|---|
| **Parent project** | [pacphi/sindri](https://github.com/pacphi/sindri) |
| **Original path** | `v3/console/agent/` |
| **Reference commits** | `1c2170f6..f049cd6` |
| **Established** | 2026-02-23 |

## Ecosystem

Draupnir is one of three complementary projects:

| Repository | Role |
|---|---|
| [sindri](https://github.com/pacphi/sindri) | CLI tool and extension ecosystem — provisions and configures instances |
| [mimir](https://github.com/pacphi/mimir) | Fleet management control plane — orchestrates, observes, and administers instances at scale |
| **draupnir** (this repo) | Lightweight per-instance agent — bridges each instance to the mimir control plane via WebSocket |
