#!/bin/bash
set -e
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"verify@stackwatch.io","password":"TestPassword!2026Stack"}' \
  | python3 -c 'import sys,json;print(json.loads(sys.stdin.read())["token"])')
echo "token len: ${#TOKEN}"

# Pick the credential that contains the real root password
CREDID=$(curl -s http://127.0.0.1:8080/api/v1/terminal/credentials \
  -H "Authorization: Bearer $TOKEN" \
  | python3 -c 'import sys,json
d=json.loads(sys.stdin.read())["credentials"]
# Find the real-password credential for the local SSH probe
for c in d:
  if c.get("name") == "real-root-pass": print(c["id"]); break
else:
  print(d[0]["id"])')
echo "CREDID=$CREDID"

# Create a fresh connection for this test
NEW=$(curl -s -X POST http://127.0.0.1:8080/api/v1/terminal/connections \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"tier3-fs-probe","host":"127.0.0.1","port":22,"user":"root","auth_method":"password"}')
echo "create resp: $NEW"
CID=$(echo "$NEW" | python3 -c 'import sys,json;print(json.loads(sys.stdin.read())["id"])')
echo "CID=$CID"

# Set credential_id on this connection
code=$(curl -s -o /tmp/r -w "%{http_code}" -X PUT \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/auth" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"auth_method\":\"password\",\"credential_id\":\"$CREDID\"}")
echo "$code PUT .../auth"
cat /tmp/r | head -c 200; echo

# Now /fs should work
code=$(curl -s -o /tmp/r -w "%{http_code}" -X GET \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs?path=/tmp" \
  -H "Authorization: Bearer $TOKEN")
echo "$code GET .../fs?path=/tmp"
cat /tmp/r | head -c 400; echo

code=$(curl -s -o /tmp/r -w "%{http_code}" -X GET \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs/read?path=/etc/hostname" \
  -H "Authorization: Bearer $TOKEN")
echo "$code GET .../fs/read /etc/hostname"
cat /tmp/r | head -c 200; echo

code=$(curl -s -o /tmp/r -w "%{http_code}" -X GET \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs/stat?path=/etc" \
  -H "Authorization: Bearer $TOKEN")
echo "$code GET .../fs/stat /etc"
cat /tmp/r | head -c 200; echo

code=$(curl -s -o /tmp/r -w "%{http_code}" -X POST \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs/mkdir" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path":"/tmp/tier3realprobe"}')
echo "$code POST .../fs/mkdir"
cat /tmp/r | head -c 200; echo

# Write
code=$(curl -s -o /tmp/r -w "%{http_code}" -X POST \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs/write" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path":"/tmp/tier3realprobe/hello.txt","content":"hello from tier3 probe","encoding":"utf8"}')
echo "$code POST .../fs/write"
cat /tmp/r | head -c 200; echo

# Read back
code=$(curl -s -o /tmp/r -w "%{http_code}" -X GET \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs/read?path=/tmp/tier3realprobe/hello.txt" \
  -H "Authorization: Bearer $TOKEN")
echo "$code GET .../fs/read (roundtrip)"
cat /tmp/r | head -c 200; echo

# Rename
code=$(curl -s -o /tmp/r -w "%{http_code}" -X POST \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs/rename" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path":"/tmp/tier3realprobe/hello.txt","new_path":"/tmp/tier3realprobe/renamed.txt"}')
echo "$code POST .../fs/rename"
cat /tmp/r | head -c 200; echo

# Delete
code=$(curl -s -o /tmp/r -w "%{http_code}" -X DELETE \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs?path=/tmp/tier3realprobe/renamed.txt" \
  -H "Authorization: Bearer $TOKEN")
echo "$code DELETE .../fs"
cat /tmp/r | head -c 200; echo

# Delete the dir
code=$(curl -s -o /tmp/r -w "%{http_code}" -X DELETE \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs?path=/tmp/tier3realprobe" \
  -H "Authorization: Bearer $TOKEN")
echo "$code DELETE .../fs (dir)"
cat /tmp/r | head -c 200; echo

# Cleanup: delete the test connection
code=$(curl -s -o /tmp/r -w "%{http_code}" -X DELETE \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID" \
  -H "Authorization: Bearer $TOKEN")
echo "$code DELETE .../connections/$CID (cleanup)"
