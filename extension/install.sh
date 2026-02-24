#!/usr/bin/env bash
# install.sh — Platform-aware binary installer for the draupnir agent.
#
# Downloads the correct draupnir binary from the latest GitHub release
# and installs it to ~/.local/bin/draupnir.
#
# Environment:
#   DRAUPNIR_VERSION  — override version (default: latest release)
#   INSTALL_DIR       — override install directory (default: ~/.local/bin)

set -euo pipefail

REPO="pacphi/draupnir"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
BINARY_NAME="draupnir"

# ── Detect platform ──────────────────────────────────────────────────────────

detect_platform() {
  local os arch

  case "$(uname -s)" in
    Linux*)  os="linux" ;;
    Darwin*) os="darwin" ;;
    *)       echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
  esac

  case "$(uname -m)" in
    x86_64|amd64)  arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *)             echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
  esac

  echo "${os}-${arch}"
}

# ── Resolve version ──────────────────────────────────────────────────────────

resolve_version() {
  if [ -n "${DRAUPNIR_VERSION:-}" ]; then
    echo "$DRAUPNIR_VERSION"
    return
  fi

  local latest
  latest=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' \
    | sed -E 's/.*"v([^"]+)".*/\1/')

  if [ -z "$latest" ]; then
    echo "Failed to determine latest version" >&2
    exit 1
  fi

  echo "$latest"
}

# ── Main ─────────────────────────────────────────────────────────────────────

main() {
  local platform version url

  platform=$(detect_platform)
  version=$(resolve_version)
  url="https://github.com/${REPO}/releases/download/v${version}/${BINARY_NAME}-${platform}"

  echo "Installing draupnir agent v${version} (${platform})..."

  mkdir -p "$INSTALL_DIR"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "${INSTALL_DIR}/${BINARY_NAME}"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "${INSTALL_DIR}/${BINARY_NAME}" "$url"
  else
    echo "Neither curl nor wget found. Please install one and retry." >&2
    exit 1
  fi

  chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

  echo "Installed: ${INSTALL_DIR}/${BINARY_NAME}"
  echo ""

  # Verify
  if "${INSTALL_DIR}/${BINARY_NAME}" --version >/dev/null 2>&1; then
    echo "Version: $("${INSTALL_DIR}/${BINARY_NAME}" --version)"
  fi

  # PATH hint
  case ":$PATH:" in
    *":${INSTALL_DIR}:"*) ;;
    *) echo "Note: Ensure ${INSTALL_DIR} is in your PATH" ;;
  esac
}

main "$@"
