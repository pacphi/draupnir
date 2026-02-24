# Configuration Reference

Draupnir is configured entirely through environment variables. There is no configuration file and no command-line flags — this keeps the binary self-contained and makes it straightforward to inject configuration in any runtime environment (systemd, Docker, Fly.io, Kubernetes, etc.).

## Quick reference

| Variable | Required | Default | Type |
|----------|----------|---------|------|
| [`SINDRI_CONSOLE_URL`](#sindri_console_url) | **yes** | — | URL string |
| [`SINDRI_CONSOLE_API_KEY`](#sindri_console_api_key) | **yes** | — | string |
| [`SINDRI_INSTANCE_ID`](#sindri_instance_id) | no | hostname | string |
| [`SINDRI_PROVIDER`](#sindri_provider) | no | — | enum |
| [`SINDRI_REGION`](#sindri_region) | no | — | string |
| [`SINDRI_AGENT_HEARTBEAT`](#sindri_agent_heartbeat) | no | `30` | integer (seconds) |
| [`SINDRI_AGENT_METRICS`](#sindri_agent_metrics) | no | `60` | integer (seconds) |
| [`SINDRI_AGENT_TAGS`](#sindri_agent_tags) | no | — | string |
| [`SINDRI_LOG_LEVEL`](#sindri_log_level) | no | `info` | enum |

---

## Variable reference

### `SINDRI_CONSOLE_URL`

The base URL of the Mimir control plane. The agent appends `/api/v1/instances/register` for the initial REST registration and then establishes a WebSocket connection.

- **Required:** yes
- **Format:** `https://<host>` or `http://<host>` (no trailing slash)
- **Example:** `https://mimir.example.com`

---

### `SINDRI_CONSOLE_API_KEY`

Authentication key used in the `Authorization: Bearer <key>` header for both the registration request and the WebSocket upgrade. Obtain this from the Mimir dashboard.

- **Required:** yes
- **Sensitivity:** treat as a secret; do not log or expose in process listings

---

### `SINDRI_INSTANCE_ID`

Stable identifier for this instance as it appears in the Mimir fleet view. Must be unique across all instances connected to the same Mimir deployment.

- **Required:** no
- **Default:** system hostname (`os.Hostname()`)
- **Constraints:** must be a non-empty string; should be stable across restarts for the same instance

---

### `SINDRI_PROVIDER`

Deployment provider where this instance is running. Used by Mimir to display provider-specific metadata and routing information. Has no effect on agent behavior.

- **Required:** no
- **Valid values:** `fly`, `docker`, `k8s`, `e2b`, `devpod`
- **Example:** `SINDRI_PROVIDER=fly`

---

### `SINDRI_REGION`

Geographic region of this instance. Used by Mimir for display and optional routing decisions. Has no effect on agent behavior.

- **Required:** no
- **Example:** `SINDRI_REGION=iad` (Fly.io region code), `SINDRI_REGION=us-east-1` (AWS)

---

### `SINDRI_AGENT_HEARTBEAT`

Interval in seconds between heartbeat messages sent to Mimir. The heartbeat is a liveness signal; Mimir uses its absence to detect disconnected instances.

- **Required:** no
- **Default:** `30`
- **Type:** positive integer (seconds)
- **Minimum recommended:** `10` (lower values increase network traffic without benefit)

---

### `SINDRI_AGENT_METRICS`

Interval in seconds between system metrics snapshots sent to Mimir. Each snapshot includes CPU usage, memory usage, disk usage, and network I/O.

- **Required:** no
- **Default:** `60`
- **Type:** positive integer (seconds)
- **Note:** metrics collection uses [gopsutil/v4](https://github.com/shirou/gopsutil); on macOS, some disk and network counters require root privileges for full accuracy

---

### `SINDRI_AGENT_TAGS`

Arbitrary key=value labels attached to this instance's registration payload. Mimir surfaces these for filtering and grouping in the fleet view.

- **Required:** no
- **Format:** comma-separated `key=value` pairs
- **Example:** `SINDRI_AGENT_TAGS=env=production,team=platform,app=api`
- **Constraints:** keys and values must not contain commas or `=` characters

---

### `SINDRI_LOG_LEVEL`

Controls the verbosity of the agent's structured log output (written to stderr).

- **Required:** no
- **Default:** `info`
- **Valid values:**

| Level | Output |
|-------|--------|
| `debug` | All messages including internal WebSocket frames and retry attempts |
| `info` | Startup, connection events, registration, and periodic summaries |
| `warn` | Recoverable errors (failed metrics read, reconnect attempts) |
| `error` | Fatal errors only |

Use `debug` when diagnosing connection or protocol issues. Switch back to `info` or `warn` in production to reduce log volume.

---

## Example: minimal configuration

```bash
export SINDRI_CONSOLE_URL="https://mimir.example.com"
export SINDRI_CONSOLE_API_KEY="sk-..."
draupnir
```

## Example: full configuration

```bash
export SINDRI_CONSOLE_URL="https://mimir.example.com"
export SINDRI_CONSOLE_API_KEY="sk-..."
export SINDRI_INSTANCE_ID="api-prod-01"
export SINDRI_PROVIDER="fly"
export SINDRI_REGION="iad"
export SINDRI_AGENT_HEARTBEAT="15"
export SINDRI_AGENT_METRICS="30"
export SINDRI_AGENT_TAGS="env=production,team=platform"
export SINDRI_LOG_LEVEL="info"
draupnir
```

## Example: `.env` file (for local development)

```bash
# .env — do not commit to version control
SINDRI_CONSOLE_URL=http://localhost:8080
SINDRI_CONSOLE_API_KEY=dev-key-local
SINDRI_INSTANCE_ID=local-dev
SINDRI_PROVIDER=docker
SINDRI_LOG_LEVEL=debug
```

Load with: `set -a && source .env && set +a && draupnir`
