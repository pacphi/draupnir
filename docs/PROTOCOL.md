# WebSocket Protocol

Draupnir communicates with the Mimir control plane over a persistent WebSocket connection using a JSON envelope format.

## Protocol Version

Current: **1.0**

Every message includes a `protocol_version` field in the envelope. This enables the control plane to handle agents running different versions gracefully.

## Envelope Format

```json
{
  "protocol_version": "1.0",
  "type": "<message_type>",
  "session_id": "<optional>",
  "payload": { ... }
}
```

## Message Types

### Outbound (Agent -> Mimir)

| Type | Description | Interval |
|------|-------------|----------|
| `heartbeat` | Agent liveness signal | Every 30s (configurable) |
| `metrics` | System resource snapshot | Every 60s (configurable) |
| `terminal:output` | PTY output bytes | On data |
| `terminal:closed` | PTY session ended | On close |
| `command:result` | Command execution result | On completion |
| `event` | Lifecycle event | On occurrence |
| `registration` | Initial registration | On connect |
| `llm_usage:batch` | Batch of LLM API token usage records | Every 30s (configurable) |

### Inbound (Mimir -> Agent)

| Type | Description |
|------|-------------|
| `terminal:create` | Allocate a new PTY session |
| `terminal:close` | Close a PTY session |
| `terminal:input` | Send keystrokes to PTY |
| `terminal:resize` | Resize PTY dimensions |
| `command:dispatch` | Execute a command |

## LLM Usage Batch Payload

The `llm_usage:batch` message carries a batch of LLM API call records captured by the Tier 1 proxy, Tier 2 eBPF interceptor, or Ollama detector:

```json
{
  "records": [
    {
      "provider": "anthropic",
      "model": "claude-sonnet-4-20250514",
      "operation": "chat",
      "inputTokens": 1500,
      "outputTokens": 500,
      "cacheReadTokens": 200,
      "cacheWriteTokens": 0,
      "costUsd": 0.0120,
      "captureTier": "proxy",
      "ts": 1709481600000
    }
  ]
}
```

Field naming follows [OpenTelemetry GenAI Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/gen-ai/) (`gen_ai.*` namespace).

## Payload Schemas

See `pkg/protocol/messages.go` for the canonical Go type definitions, or `@mimir/protocol` (in the mimir repo) for the TypeScript equivalents.

## Versioning Policy

- **Additive fields** (new optional fields in payloads) do not require a version bump
- **Breaking changes** (removed fields, changed semantics, new required fields) require incrementing `protocol_version`
- Both sides should handle unknown fields gracefully (ignore them)
