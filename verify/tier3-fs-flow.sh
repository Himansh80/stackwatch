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
echo "CID=$CID"

echo "=== POST/PUT/DELETE on connections/$CID ==="
# Update with description
code=$(curl -s -o /tmp/r -w "%{http_code}" -X PUT \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"notes":"updated by tier3-probe"}')
echo "$code PUT .../connections/$CID"
cat /tmp/r | head -c 200; echo

# Update verification mode
code=$(curl -s -o /tmp/r -w "%{http_code}" -X PUT \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/verification" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"mode":"insecure"}')
echo "$code PUT .../verification"
cat /tmp/r | head -c 200; echo

# Update auth (set credential_id)
code=$(curl -s -o /tmp/r -w "%{http_code}" -X PUT \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/auth" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"credential_id":"77add4ae-c72d-4886-bb39-6eaf8d0a1006"}')
echo "$code PUT .../auth"
cat /tmp/r | head -c 200; echo

# Try /fs with the credential_id now attached
code=$(curl -s -o /tmp/r -w "%{http_code}" -X GET \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs?path=/" \
  -H "Authorization: Bearer $TOKEN")
echo "$code GET .../fs?path=/"
cat /tmp/r | head -c 200; echo

# Read a known file
code=$(curl -s -o /tmp/r -w "%{http_code}" -X GET \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs/read?path=/etc/hostname" \
  -H "Authorization: Bearer $TOKEN")
echo "$code GET .../fs/read?path=/etc/hostname"
cat /tmp/r | head -c 200; echo

# Stat
code=$(curl -s -o /tmp/r -w "%{http_code}" -X GET \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs/stat?path=/etc" \
  -H "Authorization: Bearer $TOKEN")
echo "$code GET .../fs/stat"
cat /tmp/r | head -c 200; echo

# mkdir new dir
code=$(curl -s -o /tmp/r -w "%{http_code}" -X POST \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs/mkdir" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path":"/tmp/tier3probe"}')
echo "$code POST .../fs/mkdir"
cat /tmp/r | head -c 200; echo

# write a file
code=$(curl -s -o /tmp/r -w "%{http_code}" -X POST \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs/write" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path":"/tmp/tier3probe/hello.txt","content":"hello from tier3 probe"}')
echo "$code POST .../fs/write"
cat /tmp/r | head -c 200; echo

# read it back
code=$(curl -s -o /tmp/r -w "%{http_code}" -X GET \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs/read?path=/tmp/tier3probe/hello.txt" \
  -H "Authorization: Bearer $TOKEN")
echo "$code GET .../fs/read (roundtrip)"
cat /tmp/r | head -c 200; echo

# rename
code=$(curl -s -o /tmp/r -w "%{http_code}" -X POST \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs/rename" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path":"/tmp/tier3probe/hello.txt","new_path":"/tmp/tier3probe/hello-renamed.txt"}')
echo "$code POST .../fs/rename"
cat /tmp/r | head -c 200; echo

# delete
code=$(curl -s -o /tmp/r -w "%{http_code}" -X DELETE \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs?path=/tmp/tier3probe/hello-renamed.txt" \
  -H "Authorization: Bearer $TOKEN")
echo "$code DELETE .../fs"
cat /tmp/r | head -c 200; echo

# Delete the dir we made
code=$(curl -s -o /tmp/r -w "%{http_code}" -X DELETE \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs?path=/tmp/tier3probe" \
  -H "Authorization: Bearer $TOKEN")
echo "$code DELETE .../fs (dir)"
cat /tmp/r | head -c 200; echo
