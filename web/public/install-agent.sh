#!/bin/bash
# StackWatch Agent Installer
# Works for BOTH cloud and self-hosted — just set BACKEND env var.
#
# Usage:
#   curl -fsSL https://<backend>/install-agent.sh | sudo BACKEND=https://<backend> EMAIL=<you@example.com> bash
#
# Env vars:
#   BACKEND     - required - the api-gateway URL (cloud or self-hosted)
#   EMAIL       - required - your account email
#   TAGS        - optional - comma-separated tags to apply to this server
#   HOSTNAME    - optional - override the detected hostname

set -e

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
log()  { echo -e "${BLUE}[$(date +%T)]${NC} $*" >&2; }
ok()   { echo -e "${GREEN}[✓]${NC} $*" >&2; }
warn() { echo -e "${YELLOW}[!]${NC} $*" >&2; }
die()  { echo -e "${RED}[✗]${NC} $*" >&2; exit 1; }

[ -z "$BACKEND" ] && die "BACKEND env var is required. Example: BACKEND=https://stackwatch.smarthomelab.fun"
[ -z "$EMAIL" ] && die "EMAIL env var is required"

BACKEND="${BACKEND%/}"  # strip trailing slash
log "Installing StackWatch agent for $EMAIL → $BACKEND"

# 1. Detect OS
if [ -f /etc/os-release ]; then
  . /etc/os-release
  OS=$ID
else
  die "Cannot detect OS (no /etc/os-release)"
fi
ok "Detected OS: $OS"

# 2. Detect arch
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) die "Unsupported arch: $ARCH" ;;
esac
ok "Detected arch: $ARCH"

# 3. Check for existing agent
if pgrep -f ios-agent >/dev/null 2>&1; then
  warn "Existing ios-agent process found — will stop + replace"
  systemctl stop ios-agent 2>/dev/null || true
  pkill -f ios-agent 2>/dev/null || true
  sleep 1
fi

# 4. Install binary
INSTALL_DIR=/opt/ios-agent
mkdir -p "$INSTALL_DIR"

# Try to download the latest binary from the backend
log "Fetching agent binary from $BACKEND..."
DOWNLOAD_URL="$BACKEND/api/v1/download/ios-agent-linux-$ARCH"
TMP_BIN=$(mktemp)
if ! curl -fsSL -o "$TMP_BIN" "$DOWNLOAD_URL"; then
  die "Failed to download agent from $DOWNLOAD_URL. Is the backend reachable?"
fi
chmod +x "$TMP_BIN"
mv "$TMP_BIN" "$INSTALL_DIR/ios-agent"
ok "Binary installed at $INSTALL_DIR/ios-agent"

# 5. Authenticate — get API key from backend
log "Registering this host with backend..."
# First, we need an API key. For cloud/self-hosted, the user creates this in the dashboard.
# The one-liner pattern (Datadog-style): user pastes BACKEND + INGEST_KEY, we use that directly.
if [ -n "$INGEST_KEY" ]; then
  API_KEY="$INGEST_KEY"
  ok "Using provided INGEST_KEY"
else
  # Try login via email + ask for password (interactive)
  die "No INGEST_KEY provided. Please generate one in the StackWatch dashboard (Servers → Add Server) and re-run with INGEST_KEY=<key>"
fi

# 6. Register the server
HOSTNAME_VAL="${HOSTNAME:-$(hostname)}"
IP_ADDR=$(hostname -I 2>/dev/null | awk '{print $1}' || echo "")
TAGS_JSON="[]"
if [ -n "$TAGS" ]; then
  TAGS_JSON=$(echo "$TAGS" | tr ',' '\n' | jq -R . | jq -s .)
fi

REGISTER_BODY=$(cat <<EOF
{
  "hostname": "$HOSTNAME_VAL",
  "ip_address": "$IP_ADDR",
  "tags": $TAGS_JSON
}
EOF
)

log "POST $BACKEND/api/v1/agents/register"
REGISTER_RESP=$(curl -fsSL -X POST "$BACKEND/api/v1/agents/register" \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d "$REGISTER_BODY")
SERVER_UUID=$(echo "$REGISTER_RESP" | jq -r '.server_id // .id // empty')
[ -z "$SERVER_UUID" ] && die "Failed to register server: $REGISTER_RESP"
ok "Registered as server $SERVER_UUID"

# 7. Write config
CONFIG_FILE="$INSTALL_DIR/.env"
cat > "$CONFIG_FILE" <<EOF
BACKEND_URL=$BACKEND
API_KEY=$API_KEY
SERVER_UUID=$SERVER_UUID
LISTEN_ADDR=0.0.0.0:9101
EOF
chmod 600 "$CONFIG_FILE"
ok "Config written to $CONFIG_FILE"

# 8. Create systemd service
SERVICE_FILE=/etc/systemd/system/ios-agent.service
cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=StackWatch Agent
After=network.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=$CONFIG_FILE
ExecStart=$INSTALL_DIR/ios-agent -backend \$BACKEND_URL -api-key \$API_KEY -server-uuid \$SERVER_UUID -listen \$LISTEN_ADDR
Restart=always
RestartSec=10
User=root

[Install]
WantedBy=multi-user.target
EOF
ok "Systemd unit installed"

# 9. Start service
systemctl daemon-reload
systemctl enable ios-agent
systemctl start ios-agent
sleep 2
if systemctl is-active --quiet ios-agent; then
  ok "Service started successfully"
else
  warn "Service may not be running. Check: journalctl -u ios-agent -n 30"
fi

# 10. Verify
log "Verifying..."
sleep 3
if curl -fsS http://127.0.0.1:9101/health >/dev/null 2>&1; then
  ok "Agent is healthy at http://127.0.0.1:9101/health"
else
  warn "Agent health check failed — check logs"
fi

echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  StackWatch agent installed and running!              ║${NC}"
echo -e "${GREEN}║                                                        ║${NC}"
echo -e "${GREEN}║  Backend:    $BACKEND${NC}"
echo -e "${GREEN}║  Server ID:  $SERVER_UUID${NC}"
echo -e "${GREEN}║  Listen:     0.0.0.0:9101                               ║${NC}"
echo -e "${GREEN}║                                                        ║${NC}"
echo -e "${GREEN}║  View in dashboard: $BACKEND/app.html                  ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════╝${NC}"
