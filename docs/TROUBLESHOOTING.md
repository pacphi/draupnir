# Troubleshooting

This guide covers the most common failure modes for the Draupnir agent. Start with the general diagnostic approach, then jump to the relevant section.

## General diagnostic approach

1. **Set `SINDRI_LOG_LEVEL=debug`** — this is the single most useful step. Debug logging shows WebSocket frames, registration attempts, reconnect timing, and goroutine events.
2. **Run the agent in the foreground** — avoid starting via systemd or init systems while debugging; run `draupnir` directly in a terminal so logs go to stdout/stderr.
3. **Confirm the binary** — run `draupnir --version` to confirm the binary is present and shows the expected version.

---

## Agent won't start

### Symptom: `SINDRI_CONSOLE_URL is required` (or similar)

The required environment variables are not set.

```
fatal: SINDRI_CONSOLE_URL is required
```

**Fix:** Set the required variables before starting the agent:

```bash
export SINDRI_CONSOLE_URL="https://mimir.example.com"
export SINDRI_CONSOLE_API_KEY="your-api-key"
draupnir
```

See [docs/CONFIGURATION.md](CONFIGURATION.md) for the full variable reference.

---

### Symptom: `command not found: draupnir`

The binary is not in `$PATH`.

**Fix:**

```bash
# Check if installed
ls ~/.local/bin/draupnir

# Add to PATH (add to ~/.bashrc or ~/.zshrc for persistence)
export PATH="$HOME/.local/bin:$PATH"
```

Or reinstall via Sindri:

```bash
sindri extension install draupnir
```

---

## Cannot connect to Mimir

### Symptom: Registration fails repeatedly

```
error: registration failed: connection refused
error: registration failed: no such host
```

**Diagnosis checklist:**

1. Confirm `SINDRI_CONSOLE_URL` is correct (no trailing slash, correct scheme):

   ```bash
   curl -v "${SINDRI_CONSOLE_URL}/api/v1/instances/register"
   ```

   Expect a `401 Unauthorized` or `400 Bad Request` — anything other than `connection refused` or DNS failure means the URL is reachable.

2. Check network connectivity from the instance to the Mimir host. On cloud providers, verify security group / firewall rules allow outbound HTTPS (port 443).

3. Confirm Mimir is running: check its status page or health endpoint.

---

### Symptom: `401 Unauthorized` during registration

```
error: registration failed: 401 Unauthorized
```

**Fix:** The `SINDRI_CONSOLE_API_KEY` is missing, expired, or incorrect. Generate a new key from the Mimir dashboard and update the environment variable.

---

### Symptom: WebSocket upgrade fails after successful registration

The agent registers successfully over REST but fails to open the WebSocket connection.

```
info: registration successful
error: websocket dial failed: unexpected HTTP status 403
```

**Likely causes:**
- The API key does not have WebSocket permissions (check Mimir RBAC settings)
- A reverse proxy or load balancer in front of Mimir is not passing `Upgrade: websocket` headers

**Fix for reverse proxy:** ensure the proxy passes `Connection: Upgrade` and `Upgrade: websocket` headers and does not buffer the WebSocket connection.

---

### Symptom: Repeated reconnect loop

The agent connects, exchanges a few messages, then disconnects and reconnects in a tight loop.

**Set debug logging to see the close reason:**

```bash
SINDRI_LOG_LEVEL=debug draupnir
```

Look for the `close code` in the log output:

| Close code | Meaning | Action |
|------------|---------|--------|
| `1000` | Normal closure from Mimir | Mimir deliberately closed the connection — check Mimir logs |
| `1001` | Going away (server restart) | Normal; reconnect is expected |
| `1008` | Policy violation | Check API key validity and protocol version compatibility |
| `1011` | Internal server error | Check Mimir logs for a server-side error |

---

## PTY / terminal session issues

### Symptom: `terminal:create` received but PTY allocation fails

```
error: failed to start PTY: operation not permitted
```

**Likely cause:** The agent is running in a container without PTY support. Docker containers need `--tty` (or `tty: true` in Compose) and must not have `no-new-privileges` set if the PTY requires it.

**Fix for Docker:**

```bash
docker run --tty --interactive ...
```

Or in `docker-compose.yml`:

```yaml
services:
  agent:
    tty: true
```

---

### Symptom: PTY session opens but terminal output is garbled

The terminal dimensions were not set correctly, causing line-wrapping corruption.

**Fix:** Ensure the Mimir frontend sends a `terminal:resize` message immediately after `terminal:create` with the correct columns and rows. The agent resizes the PTY in response to `terminal:resize` messages at any time.

---

### Symptom: PTY session hangs after the shell exits

The agent did not send `terminal:closed` and the session appears stuck in Mimir.

**Likely cause:** The shell process exited but the PTY was not reaped. With debug logging, confirm the agent logs `PTY session closed` and sends `terminal:closed`.

If the issue is reproducible, please [open an issue](https://github.com/pacphi/draupnir/issues) with the debug log output.

---

## Metrics not appearing in Mimir

### Symptom: Metrics are absent or stale

**Check the collection interval:**

```bash
echo "Current interval: ${SINDRI_AGENT_METRICS:-60}s"
```

The default is 60 seconds. Metrics will not appear until the first interval elapses after startup.

---

### Symptom: Disk or network metrics show zero on macOS

Some `gopsutil` counters on macOS require elevated privileges for accurate readings.

- **Disk I/O counters:** require root for per-disk statistics; per-partition usage is available without root
- **Network I/O counters:** available without root on macOS 10.14+

For production use, Linux (amd64 or arm64) provides the most complete metrics. macOS is primarily supported for local development.

---

## Build and development issues

### Symptom: `golangci-lint` not found

The lint step is skipped with a warning. Install it:

```bash
# macOS
brew install golangci-lint

# Linux (official installer)
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Or via Go (slower)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

---

### Symptom: `make ci` fails with format errors

```
✗ Unformatted files: internal/metrics/collector.go
  Run: make fmt
```

**Fix:** Run `make fmt` then re-run `make ci`.

---

### Symptom: Race condition detected in tests

```
WARNING: DATA RACE
```

The tests run with `-race` enabled by default. If a race is detected, it indicates a real concurrency bug. Do not disable `-race` — fix the underlying issue. Races in `internal/terminal/` are most likely due to unsynchronized PTY session map access.

---

## Getting more help

- Check [GitHub Issues](https://github.com/pacphi/draupnir/issues) for known problems
- Open a new issue with: draupnir version, OS/arch, log output at `debug` level, and reproduction steps
- For Mimir-side issues (control plane errors, fleet view problems), refer to the [mimir](https://github.com/pacphi/mimir) repository
