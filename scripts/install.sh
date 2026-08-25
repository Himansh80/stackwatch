#!/bin/bash
# StackWatch Agent Installer (PL1 / Phase 1)
#
# Usage (curl | bash one-liner — Datadog-style):
#
#   curl -fsSL https://stackwatch.smarthomelab.fun/install.sh | sudo bash -s -- \
#       --backend https://stackwatch.smarthomelab.fun \
#       --token swi_xxx...
#
# Or with env vars:
#
#   INSTALL_MODE=cloud BACKEND=https://stackwatch.smarthomelab.fun \
#   TOKEN=swi_xxx... curl -fsSL https://stackwatch.smarthomelab.fun/install.sh | sudo bash
#
# What this does (idempotent — safe to re-run):
#   1. Detect OS (Ubuntu / Debian / RHEL / Fedora / macOS)
#   2. Check for root / sudo
#   3. Write /opt/stackwatch/.env (mode + backend URL + token)
#   4. Download the api-gateway binary into /opt/stackwatch/bin
#   5. Install a systemd unit at /etc/systemd/system/stackwatch.service
#      (launchd plist on macOS — skipped with a warning for now)
#   6. Enable + start the service
#   7. Echo post-install instructions (admin URL, post-install verify)
#
# Exit codes:
#   0  — installed and service is active
#   10 — not root and no sudo
#   20 — unsupported OS
#   30 — binary download failed (HTTP / curl exit)
#   40 — systemd unit install failed (daemon-reload / enable / start)
#   50 — token missing or malformed (must start with `swi_`)
set -euo pipefail

# --- Defaults (override via CLI flags or env vars) ---
INSTALL_MODE="${INSTALL_MODE:-cloud}"
BACKEND="${BACKEND:-}"
TOKEN="${TOKEN:-}"
BINARY_PATH="/opt/stackwatch/bin/stackwatch-agent"
BINARY_TMP="/tmp/stackwatch-agent.install"
ENV_FILE="/opt/stackwatch/.env"
SERVICE_FILE="/etc/systemd/system/stackwatch.service"

# --- Parse args ---
while [[ $# -gt 0 ]]; do
    case "$1" in
        --mode)    INSTALL_MODE="$2"; shift 2 ;;
        --backend) BACKEND="$2"; shift 2 ;;
        --token)   TOKEN="$2"; shift 2 ;;
        -h|--help)
            sed -n '2,30p' "$0"; exit 0 ;;
        *)
            echo "Unknown arg: $1" >&2; exit 1 ;;
    esac
done

# --- Helpers ---
log() { printf '\033[1;34m[install]\033[0m %s\n' "$*" >&2; }
warn() { printf '\033[1;33m[install]\033[0m %s\n' "$*" >&2; }
fail() { printf '\033[1;31m[install]\033[0m %s\n' "$*" >&2; exit "${2:-1}"; }

# --- 1. Root check ---
if [[ "$EUID" -ne 0 ]]; then
    if command -v sudo >/dev/null 2>&1; then
        log "Re-executing via sudo..."
        exec sudo -E --preserve-env=INSTALL_MODE,BACKEND,TOKEN -- bash "$0" "$@"
    fi
    fail "This installer needs root (no sudo available either)" 10
fi

# --- 2. Token sanity ---
if [[ -z "$TOKEN" ]]; then
    fail "Missing --token swi_xxx... (generate one in the StackWatch admin UI)" 50
fi
if [[ "$TOKEN" != swi_* ]]; then
    fail "Token must start with 'swi_'" 50
fi
if [[ -z "$BACKEND" ]]; then
    fail "Missing --backend https://stackwatch.smarthomelab.fun" 50
fi

# --- 3. OS detect ---
os="unknown"
if [[ -f /etc/os-release ]]; then
    . /etc/os-release
    case "$ID" in
        ubuntu|debian) os="debian" ;;
        rhel|centos|rocky|almalinux) os="rhel" ;;
        fedora) os="fedora" ;;
        amzn) os="rhel" ;; # Amazon Linux → RHEL family
    esac
elif [[ "$(uname -s)" == "Darwin" ]]; then
    os="macos"
fi

if [[ "$os" == "unknown" ]]; then
    fail "Unsupported OS. Supported: Ubuntu/Debian, RHEL/CentOS/Rocky/Alma, Fedora, Amazon Linux, macOS" 20
fi

log "Detected OS: $os"
log "Install mode: $INSTALL_MODE  Backend: $BACKEND  Token: ${TOKEN:0:8}…"

# --- 4. Install mode validation ---
case "$INSTALL_MODE" in
    cloud|self_hosted) ;;
    *)
        fail "INSTALL_MODE must be 'cloud' or 'self_hosted' (got: $INSTALL_MODE)" 50 ;;
esac

# --- 5. Prepare dirs ---
mkdir -p /opt/stackwatch/bin
mkdir -p /opt/stackwatch/logs

# --- 6. Write .env ---
log "Writing $ENV_FILE"
umask 077
cat > "$ENV_FILE" <<EOF
# StackWatch Agent config — written by install.sh
INSTALL_MODE=$INSTALL_MODE
BACKEND_URL=$BACKEND
INSTALL_TOKEN=$TOKEN
LOG_DIR=/opt/stackwatch/logs
EOF
chmod 600 "$ENV_FILE"

# --- 7. Download binary ---
log "Downloading StackWatch agent binary from $BACKEND ..."
if ! curl -fsSL --connect-timeout 15 --max-time 120 \
        "$BACKEND/installers/linux/amd64/stackwatch-agent" \
        -o "$BINARY_TMP"; then
    fail "Binary download failed (HTTP error from $BACKEND)" 30
fi
chmod +x "$BINARY_TMP"
install -m 0755 "$BINARY_TMP" "$BINARY_PATH"
rm -f "$BINARY_TMP"

# --- 8. Systemd unit (skip on macOS) ---
if [[ "$os" == "macos" ]]; then
    warn "macOS detected — skipping systemd install. Run the binary directly:"
    warn "  $BINARY_PATH --config $ENV_FILE"
    log "Install complete."
    exit 0
fi

if ! command -v systemctl >/dev/null 2>&1; then
    fail "systemctl not found — non-systemd Linux not supported by this installer yet" 40
fi

log "Writing systemd unit $SERVICE_FILE"
cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=StackWatch Agent (api-gateway)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
EnvironmentFile=$ENV_FILE
ExecStart=$BINARY_PATH --config $ENV_FILE
Restart=on-failure
RestartSec=5
LimitNOFILE=65536
StandardOutput=append:/opt/stackwatch/logs/agent.log
StandardError=append:/opt/stackwatch/logs/agent.err.log

[Install]
WantedBy=multi-user.target
EOF

log "systemctl daemon-reload + enable + start"
systemctl daemon-reload
systemctl enable stackwatch.service
systemctl restart stackwatch.service
sleep 2

if ! systemctl is-active --quiet stackwatch.service; then
    warn "Service did not become active — last 20 log lines:"
    journalctl -u stackwatch.service -n 20 --no-pager >&2 || true
    fail "Service failed to start" 40
fi

log "StackWatch agent installed and active."
log "  Backend:   $BACKEND"
log "  Mode:      $INSTALL_MODE"
log "  Logs:      journalctl -u stackwatch.service -f"
log "  Status:    systemctl status stackwatch.service"
log "  Disable:   systemctl disable --now stackwatch.service"