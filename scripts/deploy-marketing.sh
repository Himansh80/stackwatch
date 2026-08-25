#!/bin/bash
# scripts/deploy-marketing.sh
#
# Tier 12 Phase 2 — Marketing site deploy.
#
# Syncs web/public/marketing/ to root@192.168.0.115:/opt/stackwatch/web/public/marketing/
# and verifies the page returns HTTP 200 from the local serve.py (port 8090).
#
# Usage:
#   bash scripts/deploy-marketing.sh                  # full deploy + verify
#   bash scripts/deploy-marketing.sh --no-verify      # deploy only
#   bash scripts/deploy-marketing.sh --dry-run        # rsync dry-run
#
# Exit codes:
#   0 — deploy succeeded and HTTP 200 confirmed
#   1 — rsync failed
#   2 — HTTP verification failed (non-200)
#   3 — host unreachable via ssh
#
# Requirements:
#   - sshpass OR ssh key for root@192.168.0.115
#   - rsync on both ends
#   - serve.py already running on .115 port 8090

set -euo pipefail

REMOTE_HOST="${REMOTE_HOST:-192.168.0.115}"
REMOTE_USER="${REMOTE_USER:-root}"
REMOTE_PORT="${REMOTE_PORT:-22}"
REMOTE_DIR="${REMOTE_DIR:-/opt/stackwatch/web/public/marketing}"
LOCAL_DIR="${LOCAL_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/web/public/marketing}"
SERVICE_URL="${SERVICE_URL:-http://127.0.0.1:8090/marketing/}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/id_rsa}"

VERIFY=1
DRY_RUN=""
for arg in "$@"; do
  case "$arg" in
    --no-verify)  VERIFY=0 ;;
    --dry-run)    DRY_RUN="--dry-run" ;;
    --help|-h)
      sed -n '2,30p' "$0"
      exit 0
      ;;
    *)            echo "Unknown arg: $arg"; exit 1 ;;
  esac
done

# Preflight checks -------------------------------------------------------
if [[ ! -d "$LOCAL_DIR" ]]; then
  echo "ERROR: Local marketing dir not found: $LOCAL_DIR" >&2
  exit 1
fi

for f in index.html styles.css script.js; do
  if [[ ! -f "$LOCAL_DIR/$f" ]]; then
    echo "ERROR: Required file missing: $LOCAL_DIR/$f" >&2
    exit 1
  fi
done

echo "== StackWatch marketing deploy =="
echo "Local:  $LOCAL_DIR"
echo "Remote: $REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR"
echo "Verify: $([[ $VERIFY -eq 1 ]] && echo "yes" || echo "no")"
echo

# Step 1 — rsync -------------------------------------------------------
echo ">>> Step 1: rsync to $REMOTE_HOST"
rsync -avz $DRY_RUN \
  --delete \
  --chmod=D755,F644 \
  -e "ssh -p $REMOTE_PORT -i $SSH_KEY -o StrictHostKeyChecking=accept-new -o ConnectTimeout=10" \
  "$LOCAL_DIR/" \
  "$REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR/"

if [[ -n "$DRY_RUN" ]]; then
  echo
  echo "Dry run complete. Re-run without --dry-run to apply."
  exit 0
fi

echo ">>> rsync OK"
echo

# Step 2 — verify HTTP 200 -----------------------------------------------
if [[ $VERIFY -eq 0 ]]; then
  echo "Skipping verification (--no-verify)."
  exit 0
fi

echo ">>> Step 2: HTTP verify on $REMOTE_HOST"
# We run curl from the remote box so we hit the local serve.py on :8090
# without needing to expose 8090 publicly.
REMOTE_STATUS=$(ssh -p "$REMOTE_PORT" -i "$SSH_KEY" \
  -o StrictHostKeyChecking=accept-new -o ConnectTimeout=10 \
  "$REMOTE_USER@$REMOTE_HOST" \
  "curl -s -o /dev/null -w '%{http_code}' --max-time 5 '$SERVICE_URL'" 2>/dev/null || echo "000")

echo "HTTP status: $REMOTE_STATUS"
echo "URL:         $SERVICE_URL"

if [[ "$REMOTE_STATUS" == "200" ]]; then
  echo
  echo "✓ Deploy verified — marketing page is live at $SERVICE_URL"
  exit 0
else
  echo
  echo "✗ Verification FAILED — expected 200, got $REMOTE_STATUS" >&2
  echo "  Hint: is serve.py running on $REMOTE_HOST port 8090?" >&2
  echo "  Try:  ssh $REMOTE_USER@$REMOTE_HOST 'systemctl status stackwatch-serve'" >&2
  exit 2
fi