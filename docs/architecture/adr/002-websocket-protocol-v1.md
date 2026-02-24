# 002. WebSocket protocol v1 envelope

**Status:** Accepted
**Date:** 2026-02-23

## Context

Draupnir and Mimir communicate over a persistent WebSocket connection. Both sides need to exchange structured messages across a range of concerns: liveness signals, system metrics, PTY I/O, command dispatch, and lifecycle events. These concerns have different schemas and different cardinalities (one registration, periodic heartbeats, many terminal messages).

Two design questions had to be resolved:

1. **How to multiplex message types over a single WebSocket connection?**
2. **How to handle forward/backward compatibility as the protocol evolves?**

Options for multiplexing considered:

| Option | Notes |
|--------|-------|
| Sub-protocol per concern (separate WebSocket connections) | Cleaner isolation but multiple connections per instance; complicates reconnect logic |
| JSON envelope with a `type` discriminator field | Single connection; type field routes to correct handler; widely understood |
| Binary framing (protobuf, msgpack) | Smaller on the wire; harder to debug; more tooling overhead |

## Decision

Use a single WebSocket connection with a JSON envelope containing four top-level fields:

```json
{
  "protocol_version": "1.0",
  "type": "<message_type>",
  "session_id": "<optional>",
  "payload": { ... }
}
```

- `protocol_version` is a string (`"1.0"`) included in every message. Both sides log a warning if they receive a version they do not recognize, but continue processing if the message is otherwise well-formed.
- `type` is a namespaced string (`terminal:create`, `command:dispatch`, etc.) that determines how `payload` is decoded.
- `session_id` is optional and used only for messages scoped to a specific PTY session.
- `payload` is an opaque JSON object whose schema is defined by `type`.

**Versioning policy:**
- Additive changes (new optional fields in `payload`) do not require a `protocol_version` bump.
- Breaking changes (removed fields, changed semantics, new required fields, changed `type` string) require incrementing `protocol_version`.
- Both sides must ignore unknown fields in `payload` (Go's `json.Unmarshal` does this by default; TypeScript implementations must use permissive deserialization).

The canonical Go type definitions live in `pkg/protocol/messages.go`. The TypeScript equivalents live in `@mimir/protocol` in the mimir repository.

## Consequences

**Positive:**
- A single WebSocket connection per instance simplifies reconnect logic, load balancing, and observability.
- JSON is human-readable and trivially debuggable with `SINDRI_LOG_LEVEL=debug`.
- The `protocol_version` field provides a forward-compatibility escape hatch without requiring a connection upgrade mechanism.
- `session_id` cleanly scopes PTY messages without requiring separate connections per terminal session.

**Negative / trade-offs:**
- JSON serialization overhead is negligible for control messages and metrics, but PTY output bytes are Base64-encoded in `payload`, adding ~33% overhead compared to a binary framing format. For interactive terminal sessions, this is acceptable; for bulk file transfers it would not be.
- Protocol changes must be coordinated across two repositories (this repo and mimir) in two languages. The canonical Go structs in `pkg/protocol/messages.go` serve as documentation but cannot be automatically compiled into TypeScript.

**Follow-on constraints:**
- Any change to the protocol must update `pkg/protocol/messages.go` first and be accompanied by a corresponding change to mimir's TypeScript types in the same or a preceding release.
- If terminal throughput becomes a bottleneck, a future ADR should evaluate binary framing (WebSocket binary frames with msgpack or a custom header) as `protocol_version: "2.0"`.
