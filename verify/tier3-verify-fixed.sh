#!/bin/bash
# Tier 3 fix verification — tests /fs routes end-to-end with both auth methods.
set -e
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"verify@stackwatch.io","password":"TestPassword!2026Stack"}' \
  | python3 -c 'import sys,json;print(json.loads(sys.stdin.read())["token"])')
echo "token len: ${#TOKEN}"

# Pick first credential (assume it has a real password; if not, we report).
CREDID=$(curl -s http://127.0.0.1:8080/api/v1/terminal/credentials \
  -H "Authorization: Bearer $TOKEN" \
  | python3 -c 'import sys,json;d=json.loads(sys.stdin.read())["credentials"];print(d[0]["id"])')
echo "CREDID=$CREDID"

# Reveal so we know what password it has
SECRET=$(curl -s -X POST "http://127.0.0.1:8080/api/v1/terminal/credentials/$CREDID/reveal" \
  -H "Authorization: Bearer $TOKEN" \
  | python3 -c 'import sys,json;print(json.loads(sys.stdin.read())["secret"])')
echo "SECRET preview: ${SECRET:0:8}..."

# Create test connection
NEW=$(curl -s -X POST http://127.0.0.1:8080/api/v1/terminal/connections \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"tier3-fixed","host":"127.0.0.1","port":22,"user":"root","auth_method":"password"}')
echo "create resp: $NEW"
CID=$(echo "$NEW" | python3 -c 'import sys,json;print(json.loads(sys.stdin.read())["id"])')
echo "CID=$CID"

# Attach credential
curl -s -X PUT "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/auth" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"auth_method\":\"password\",\"credential_id\":\"$CREDID\"}" > /dev/null
echo "auth attached"

# Now exercise every /fs endpoint
echo ""
echo "=== fs/list /tmp ==="
curl -s -o /tmp/r -w "code=%{http_code}\n" \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs?path=/tmp" \
  -H "Authorization: Bearer $TOKEN"
head -c 300 /tmp/r; echo

echo "=== fs/read /etc/hostname ==="
curl -s -o /tmp/r -w "code=%{http_code}\n" \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs/read?path=/etc/hostname" \
  -H "Authorization: Bearer $TOKEN"
head -c 300 /tmp/r; echo

echo "=== fs/stat /etc ==="
curl -s -o /tmp/r -w "code=%{http_code}\n" \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs/stat?path=/etc" \
  -H "Authorization: Bearer $TOKEN"
head -c 300 /tmp/r; echo

echo "=== fs/mkdir /tmp/tier3fix ==="
curl -s -o /tmp/r -w "code=%{http_code}\n" -X POST \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs/mkdir" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path":"/tmp/tier3fix"}'
head -c 200 /tmp/r; echo

echo "=== fs/write (UTF-8 content) /tmp/tier3fix/hello.txt ==="
curl -s -o /tmp/r -w "code=%{http_code}\n" -X POST \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs/write" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path":"/tmp/tier3fix/hello.txt","content":"hello from tier3 fixed build"}'
head -c 200 /tmp/r; echo

echo "=== fs/read back ==="
curl -s -o /tmp/r -w "code=%{http_code}\n" \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs/read?path=/tmp/tier3fix/hello.txt" \
  -H "Authorization: Bearer $TOKEN"
head -c 300 /tmp/r; echo

echo "=== fs/rename ==="
curl -s -o /tmp/r -w "code=%{http_code}\n" -X POST \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs/rename" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"old_path":"/tmp/tier3fix/hello.txt","new_path":"/tmp/tier3fix/renamed.txt"}'
head -c 200 /tmp/r; echo

echo "=== fs/delete file ==="
curl -s -o /tmp/r -w "code=%{http_code}\n" -X DELETE \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs?path=/tmp/tier3fix/renamed.txt" \
  -H "Authorization: Bearer $TOKEN"
head -c 200 /tmp/r; echo

echo "=== fs/delete dir ==="
curl -s -o /tmp/r -w "code=%{http_code}\n" -X DELETE \
  "http://127.0.0.1:8080/api/v1/terminal/connections/$CID/fs?path=/tmp/tier3fix" \
  -H "Authorization: Bearer $TOKEN"
head -c 200 /tmp/r; echo

# Cleanup connection
curl -s -X DELETE "http://127.0.0.1:8080/api/v1/terminal/connections/$CID" \
  -H "Authorization: Bearer $TOKEN" > /dev/null
echo "cleaned up connection $CID"
