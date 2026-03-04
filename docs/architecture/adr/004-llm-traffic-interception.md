# ADR 004: LLM Traffic Interception via Two-Tier Proxy and eBPF

- **Status:** Accepted
- **Date:** 2026-03-03
- **Deciders:** Core team
- **Relates to:** [Mimir ADR-0011](https://github.com/pacphi/mimir/docs/adr/0011-llm-token-cost-tracking.md)

## Context

Sindri instances run 13+ extensions that make LLM API calls to various providers (Anthropic, OpenAI, Google Gemini, Groq, Mistral, xAI, Cohere) and local Ollama inference. Mimir needs per-instance token usage data to track AI costs across the fleet. However:

- Extensions are third-party — modifying them is not an option
- Extensions call LLMs through 4 distinct patterns (CLI wrapping, HTTP proxy, direct HTTP, SDK calls)
- The `monitoring` extension only tracks Anthropic usage
- Draupnir is already deployed on every instance with a persistent WebSocket connection to Mimir

### Approaches Considered

| Approach | Pros | Cons | Verdict |
|----------|------|------|---------|
| Extension-dependent (ccm, monitoring) | Already exists | Fragile coupling; incomplete provider coverage | **Rejected** |
| **Tier 1: Env var proxy** | Simple; no kernel deps; ~800 LoC | Only catches apps that read `*_BASE_URL` env vars | **Ship first (MVP)** |
| **Tier 2: eBPF SSL uprobe** | Catches everything; no app cooperation | Kernel 5.8+; ~3000 LoC; BPF C code | **Ship second** |
| DNS interception | Universal | TLS CA injection required; fragile | Rejected |
| iptables MITM | Works for all | Root + CA cert distribution; complex | Rejected |

## Decision

### Two-tier architecture with graceful fallback

```
┌───────────────────────────────────────────────┐
│ Sindri Instance                                │
│                                                │
│  Extensions (claude-flow, ai-toolkit, etc.)    │
│         │ HTTPS (via *_BASE_URL env vars)      │
│         ▼                                      │
│  ┌── TIER 1: HTTP Proxy (:9090) ────────────┐ │
│  │  Forwards to real API, parses responses   │ │
│  │  Coverage: ~90% of traffic                │ │
│  └───────────────────────────────────────────┘ │
│                                                │
│  ┌── TIER 2: eBPF SSL Uprobe ───────────────┐ │
│  │  Hooks SSL_read/SSL_write (libssl.so)     │ │
│  │  Hooks crypto/tls (Go binaries)           │ │
│  │  Coverage: ~99% — catches hardcoded URLs  │ │
│  │  Requires: Linux 5.8+, BTF, CAP_BPF      │ │
│  └───────────────────────────────────────────┘ │
│                                                │
│  ┌── Ollama Detector ───────────────────────┐  │
│  │  Polls localhost:11434/api/ps             │  │
│  │  Reports tokens with $0 cost             │  │
│  └───────────────────────────────────────────┘  │
│                                                │
│  Reporter ──► WebSocket LLM_USAGE ──► Mimir   │
└───────────────────────────────────────────────┘
```

### Startup sequence

1. If `SINDRI_LLM_ADAPTER != "none"`:
   a. Start Tier 1 proxy on `:SINDRI_LLM_PROXY_PORT` (default 9090)
   b. Check `/sys/kernel/btf/vmlinux` — if present and kernel ≥ 5.8, load eBPF programs (Tier 2)
   c. If eBPF unavailable, log warning and rely on Tier 1 only
   d. Start Ollama detector (polls `localhost:11434` every 30s)
2. Reporter batches `UsageEvent`s and sends via WebSocket `llm_usage:batch` every 30s

### Token extraction per provider

Each provider has a specific JSON response format. Draupnir's parsers extract:

| Provider | Host | Input Path | Output Path |
|----------|------|-----------|-------------|
| Anthropic | `api.anthropic.com` | `.usage.input_tokens` | `.usage.output_tokens` |
| OpenAI | `api.openai.com` | `.usage.prompt_tokens` | `.usage.completion_tokens` |
| Google | `generativelanguage.googleapis.com` | `.usageMetadata.promptTokenCount` | `.usageMetadata.candidatesTokenCount` |
| Groq | `api.groq.com` | `.usage.prompt_tokens` | `.usage.completion_tokens` |
| Ollama | `localhost:11434` | `.prompt_eval_count` | `.eval_count` |

OpenAI-compatible format (Groq, Mistral, xAI, Together) reuses the OpenAI parser.

### New configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SINDRI_LLM_ADAPTER` | `auto` | `auto`, `proxy`, `ebpf`, `none` |
| `SINDRI_LLM_PROXY_PORT` | `9090` | Local proxy port |
| `SINDRI_LLM_REPORT_INTERVAL` | `30` | Seconds between usage batch sends |

### Env var injection (Sindri-side)

The Draupnir extension in Sindri sets `*_BASE_URL` env vars pointing to `localhost:9090` when `SINDRI_LLM_ADAPTER != 'none'`. This requires no changes to extensions — all major LLM SDKs respect their `*_BASE_URL` environment variable.

## Consequences

### Positive
- No modification of any extension required
- Tier 1 covers ~90% of traffic with minimal complexity (~800 LoC)
- Tier 2 provides full coverage for hardcoded URLs and already-running processes
- Graceful fallback: eBPF failure doesn't disable proxy
- Ollama tracking enables local-vs-cloud cost comparison

### Negative
- Tier 1 only works if extensions read `*_BASE_URL` env vars (most do)
- Tier 2 eBPF requires Linux 5.8+ with BTF — macOS and older kernels fall back to Tier 1
- eBPF stub is not yet wired to `cilium/ebpf` — Tier 2 needs `cilium/ebpf` dependency to be production-ready
- Pricing table in Go must be kept in sync with Mimir's `llm-pricing.ts`

### New files

```
internal/llm/
├── adapter.go      — Top-level lifecycle (starts Tier 1 + conditional Tier 2 + Ollama)
├── proxy.go        — Tier 1 HTTP reverse proxy
├── providers.go    — Per-provider response parsers
├── pricing.go      — Embedded LLM pricing table
├── reporter.go     — Batch + send via WebSocket LLM_USAGE channel
├── ollama.go       — Ollama auto-detection and monitoring
└── ebpf/
    ├── loader.go       — Linux: eBPF availability check + program loader stub
    └── loader_other.go — Non-Linux: no-op build tag
```

### Modified files
- `pkg/protocol/messages.go` — `MsgLLMUsageBatch`, `LLMUsageRecord`, `LLMUsageBatchPayload`
- `internal/config/config.go` — `LLMAdapter`, `LLMProxyPort`, `LLMReportInterval`
- `cmd/agent/main.go` — Start LLM adapter goroutine
