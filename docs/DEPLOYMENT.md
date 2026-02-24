# Deployment

Draupnir is a single static binary. Deployment means getting the binary onto a managed instance and starting it with the correct environment variables. This document covers the standard distribution paths and common runtime environments.

## How Draupnir is distributed

Draupnir is published as a [Sindri extension](https://github.com/pacphi/sindri) and as standalone binaries attached to each [GitHub Release](https://github.com/pacphi/draupnir/releases).

### Method 1: Via Sindri (recommended)

When Sindri provisions an instance and the `draupnir` extension is enabled, it installs the agent automatically:

```bash
sindri extension install draupnir
```

This downloads the correct platform binary, verifies the SHA256 checksum, and installs it to `~/.local/bin/draupnir`. No further steps are needed — Sindri configures the environment variables from the instance's deployment configuration.

### Method 2: Installer script

For instances not managed by Sindri:

```bash
curl -fsSL https://raw.githubusercontent.com/pacphi/draupnir/main/extension/install.sh | bash
```

Override the version or install directory if needed:

```bash
DRAUPNIR_VERSION=1.2.0 INSTALL_DIR=/usr/local/bin \
  curl -fsSL .../install.sh | bash
```

### Method 3: Direct binary download

Download from [GitHub Releases](https://github.com/pacphi/draupnir/releases), choose the binary for your platform:

| Platform | Binary |
|----------|--------|
| Linux amd64 | `draupnir-linux-amd64` |
| Linux arm64 | `draupnir-linux-arm64` |
| macOS amd64 | `draupnir-darwin-amd64` |
| macOS arm64 (M-series) | `draupnir-darwin-arm64` |

Verify the checksum before use:

```bash
# Download binary and checksum
curl -fsSL https://github.com/pacphi/draupnir/releases/download/v1.0.0/draupnir-linux-amd64 -o draupnir
curl -fsSL https://github.com/pacphi/draupnir/releases/download/v1.0.0/checksums.txt -o checksums.txt

# Verify
sha256sum --check --ignore-missing checksums.txt

# Install
chmod +x draupnir
mv draupnir /usr/local/bin/draupnir
```

### Method 4: Build from source

```bash
git clone https://github.com/pacphi/draupnir.git
cd draupnir
make build         # current platform → dist/draupnir
make build-all     # all 4 targets → dist/draupnir-{os}-{arch}
```

---

## Startup sequence

Regardless of how it is installed, the agent follows this startup sequence on launch:

```
1. Load configuration from environment variables
2. POST /api/v1/instances/register → Mimir (retries on failure)
3. Open WebSocket connection to Mimir
4. Start heartbeat goroutine (default: every 30s)
5. Start metrics goroutine (default: every 60s)
6. Enter message dispatch loop
7. Block until SIGTERM or SIGINT
8. Graceful shutdown: close PTY sessions, flush pending messages, close WebSocket
```

The agent does not daemonize itself. Process supervision (systemd, runit, Docker restart policy, etc.) is the responsibility of the deployment environment.

---

## Running as a systemd service

Create `/etc/systemd/system/draupnir.service`:

```ini
[Unit]
Description=Draupnir — Sindri instance agent
Documentation=https://github.com/pacphi/draupnir
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=developer
EnvironmentFile=/etc/draupnir/env
ExecStart=/usr/local/bin/draupnir
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal
SyslogIdentifier=draupnir

# Hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full

[Install]
WantedBy=multi-user.target
```

Create `/etc/draupnir/env` (mode `0600`, owned by root):

```bash
SINDRI_CONSOLE_URL=https://mimir.example.com
SINDRI_CONSOLE_API_KEY=sk-...
SINDRI_INSTANCE_ID=my-instance-01
SINDRI_PROVIDER=fly
SINDRI_REGION=iad
```

Enable and start:

```bash
systemctl daemon-reload
systemctl enable draupnir
systemctl start draupnir
journalctl -u draupnir -f   # follow logs
```

---

## Running as a Docker sidecar

When the primary workload runs in a Docker container, the agent can run as a sidecar in the same Compose stack, sharing the host network namespace for accurate metrics:

```yaml
# docker-compose.yml
services:
  app:
    image: your-app:latest

  draupnir:
    image: ubuntu:24.04      # or any minimal Linux image
    network_mode: host        # share host network for accurate metrics
    tty: true                 # required for PTY session support
    volumes:
      - /usr/local/bin/draupnir:/usr/local/bin/draupnir:ro
    command: draupnir
    restart: unless-stopped
    environment:
      SINDRI_CONSOLE_URL: ${SINDRI_CONSOLE_URL}
      SINDRI_CONSOLE_API_KEY: ${SINDRI_CONSOLE_API_KEY}
      SINDRI_INSTANCE_ID: ${HOSTNAME}
      SINDRI_PROVIDER: docker
```

> **Note:** PTY sessions (`terminal:create`) require `tty: true`. Without it, the agent registers and sends metrics but remote terminal sessions will fail.

---

## Running on Fly.io

In a `fly.toml`, add the agent as a process alongside the main app:

```toml
[processes]
  app   = "/bin/start-app.sh"
  agent = "draupnir"
```

Set environment variables via Fly secrets:

```bash
fly secrets set SINDRI_CONSOLE_URL=https://mimir.example.com
fly secrets set SINDRI_CONSOLE_API_KEY=sk-...
```

The agent automatically picks up `FLY_REGION` and `FLY_APP_NAME` if you set:

```bash
SINDRI_PROVIDER=fly
SINDRI_REGION=${FLY_REGION}
SINDRI_INSTANCE_ID=${FLY_APP_NAME}-${FLY_MACHINE_ID}
```

---

## Running on Kubernetes

Deploy the agent as a sidecar container in the workload's Pod spec:

```yaml
spec:
  containers:
    - name: app
      image: your-app:latest

    - name: draupnir
      image: ubuntu:24.04
      command: ["/usr/local/bin/draupnir"]
      tty: true
      env:
        - name: SINDRI_CONSOLE_URL
          valueFrom:
            secretKeyRef:
              name: draupnir
              key: consoleUrl
        - name: SINDRI_CONSOLE_API_KEY
          valueFrom:
            secretKeyRef:
              name: draupnir
              key: apiKey
        - name: SINDRI_INSTANCE_ID
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: SINDRI_PROVIDER
          value: k8s
        - name: SINDRI_REGION
          value: us-east-1
      volumeMounts:
        - name: agent-binary
          mountPath: /usr/local/bin/draupnir
          subPath: draupnir
  volumes:
    - name: agent-binary
      configMap:
        name: draupnir-binary
        defaultMode: 0755
```

Create the secret:

```bash
kubectl create secret generic draupnir \
  --from-literal=consoleUrl=https://mimir.example.com \
  --from-literal=apiKey=sk-...
```

---

## Health indicators

The agent does not expose an HTTP health endpoint. Use these signals instead:

| Signal | Healthy indicator |
|--------|------------------|
| Exit code | `0` = clean shutdown; non-zero = error |
| Log output | `info: websocket connection established` within a few seconds of start |
| Mimir fleet view | Instance appears as "connected" after registration |
| Heartbeat | Instance last-seen timestamp updates every `SINDRI_AGENT_HEARTBEAT` seconds |

For process supervisors, monitor the process exit code and use `RestartSec` or equivalent to prevent tight restart loops on persistent failures.
