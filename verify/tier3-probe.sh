#!/bin/bash
set -e
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"verify@stackwatch.io","password":"TestPassword!2026Stack"}' \
  | python3 -c 'import sys,json;print(json.loads(sys.stdin.read())["token"])')
echo "token len: ${#TOKEN}"

CID=$(curl -s http://127.0.0.1:8080/api/v1/terminal/connections \
  -H "Authorization: Bearer $TOKEN" \
  | python3 -c 'import sys,json;print(json.loads(sys.stdin.read())["connections"][0]["id"])')
CREDID=$(curl -s http://127.0.0.1:8080/api/v1/terminal/credentials \
  -H "Authorization: Bearer $TOKEN" \
  | python3 -c 'import sys,json;print(json.loads(sys.stdin.read())["credentials"][0]["id"])')
echo "CID=$CID  CREDID=$CREDID"

echo "=== Tier 3 sub-routes ==="
declare -a routes=(
  "GET|/api/v1/terminal/keys/c2350087-95da-4072-a0cb-92b3e3fec342"
  "GET|/api/v1/terminal/connections/$CID"
  "POST|/api/v1/terminal/connections/$CID/test"
  "GET|/api/v1/terminal/connections/$CID/verification"
  "GET|/api/v1/terminal/connections/$CID/auth"
  "GET|/api/v1/terminal/connections/$CID/fs"
  "GET|/api/v1/terminal/connections/$CID/fs/stat"
  "GET|/api/v1/terminal/credentials/$CREDID"
  "POST|/api/v1/terminal/credentials/$CREDID/reveal"
)
for r in "${routes[@]}"; do
  m="${r%%|*}"
  p="${r#*|}"
  if [ "$m" = "POST" ]; then
    code=$(curl -s -o /tmp/r -w "%{http_code}" -X POST \
      "http://127.0.0.1:8080${p}" \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" -d '{}')
  else
    code=$(curl -s -o /tmp/r -w "%{http_code}" -X $m \
      "http://127.0.0.1:8080${p}" \
      -H "Authorization: Bearer $TOKEN")
  fi
  body=$(cat /tmp/r | head -c 220)
  printf "%s %-6s %s\n    %s\n" "$code" "$m" "$p" "$body"
done
