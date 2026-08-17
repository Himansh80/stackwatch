# StackWatch — Testing

**Last updated:** 2026-08-17

What's testable TODAY, with concrete steps and expected outputs.

---

## Tier 0 — End-to-end (auth + tenants + users + api-keys + refresh)

### Health check (smoke test)

```bash
curl -sm 3 http://localhost:8080/health
# Expected: {"db":"ok","mode":"cloud","status":"ok","version":"0.1.0-tier0.5"}
```

### Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"verify@stackwatch.io","password":"TestPassword!2026Stack"}'
# Expected: HTTP 200, JSON with "token" (400+ chars)
```

### Sign up a new tenant (admin only)

```bash
TOKEN="<login-token>"
curl -X POST http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "email":"newuser@example.com",
    "role":"operator",
    "password":"TestPass!2026Stack"
  }'
# Expected: HTTP 201, JSON with user id and tenant id
```

### Issue an API key, then use it

```bash
# Create
curl -X POST http://localhost:8080/api/v1/api-keys \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"name":"smoke-test","scopes":["read"]}'
# Expected: HTTP 201, JSON with token starting "sw_<6hex>_"

# Use it (replace KEY below)
curl -H "Authorization: Bearer sw_<6hex>_..." \
  http://localhost:8080/api/v1/auth/me
# Expected: HTTP 200, JSON with user profile

# Revoke
curl -X POST http://localhost:8080/api/v1/api-keys/<key-id>/revoke \
  -H "Authorization: Bearer ${TOKEN}"
# Expected: HTTP 200, JSON {revoked:true}
```

### Refresh a token

```bash
# New token from /auth/login
ORIGINAL="<jwt>"

curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d "{\"token\":\"${ORIGINAL}\"}"
# Expected: HTTP 200, JSON with new token

# Use it
curl -H "Authorization: Bearer <new-jwt>" \
  http://localhost:8080/api/v1/auth/me
# Expected: HTTP 200
```

---

## Tier 1 — Proxmox

### Register the Proxmox host

```bash
curl -X POST http://localhost:8080/api/v1/proxmox/hosts \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"homelab",
    "base_url":"https://192.168.0.107:8006",
    "api_token":"root@pam!stackwatch-token=<uuid>"
  }'
# Expected: HTTP 201, JSON with host id

HOST_ID="<id-from-above>"
```

### Test the host's connectivity

```bash
curl -H "Authorization: Bearer ${TOKEN}" \
  "http://localhost:8080/api/v1/proxmox/hosts/${HOST_ID}/test"
# Expected: HTTP 200, JSON {"ok":true,"version":"..."}
```

### List VMs

```bash
curl -H "Authorization: Bearer ${TOKEN}" \
  "http://localhost:8080/api/v1/proxmox/hosts/${HOST_ID}/vms"
# Expected: HTTP 200, JSON {"vms":[...], "total": N}
# (Real-world on .107: 15 VMs)
```

### List cluster tasks (recent)

```bash
curl -H "Authorization: Bearer ${TOKEN}" \
  "http://localhost:8080/api/v1/proxmox/hosts/${HOST_ID}/cluster/tasks"
# Expected: HTTP 200, JSON {"tasks":[...], "total": N}
```

---

## Tier 2 — TrueNAS

### Register the TrueNAS host

```bash
curl -X POST http://localhost:8080/api/v1/truenas/hosts \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"bareilly-nas",
    "base_url":"https://192.168.0.112",
    "api_token":"<truenas-api-key>"
  }'
# Expected: HTTP 201
```

### List pools

```bash
HOST_ID="<id>"

# Direct against sidecar:
curl -X POST http://localhost:8088/pools/list \
  -H "Content-Type: application/json" \
  -d "{\"host_id\":\"${HOST_ID}\"}"
# Expected: HTTP 200, JSON {"pools":[...], "total": N}
```

---

## Tier 3 — Terminal

### Create a connection

```bash
curl -X POST http://localhost:8080/api/v1/terminal/connections \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"homelab-test",
    "host":"192.168.0.107",
    "port":22,
    "username":"root",
    "auth_type":"key",
    "key_id":"<ssh-key-uuid>"
  }'
```

### List connections

```bash
curl -H "Authorization: Bearer ${TOKEN}" \
  http://localhost:8080/api/v1/terminal/connections
# Expected: HTTP 200, JSON {"connections":[...]}
```

### SSH-key upload (for key auth)

```bash
PUB=$(cat ~/.ssh/id_ed25519.pub)
curl -X POST http://localhost:8080/api/v1/terminal/keys \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"workstation\",\"public_key\":\"${PUB}\"}"
```

---

## Automated verifier scripts

The maintainer keeps these in `C:\Users\himan\AppData\Local\Temp\`:

| File | Tier | Checks | Last run |
|------|------|--------|----------|
| `hermes-verify-tier0-complete-2026-08-17.py` | 0 | 17 | 15/15 clean |
| `hermes-verify-tier0-1-2-live-2026-08-17.py` | 0+1+2 | 27 | 3/3 clean |

Each script:
- Logs in fresh with a real admin account
- Runs every check as a separate `curl`
- Prints `PASS` / `FAIL` per check
- Exits non-zero on any failure
- Writes `.proof.txt` for audit

To use them yourself (assumes you've cloned the repo and have Go 1.23+):

```bash
# 1. Get your Linux + Windows test hosts reachable
# 2. Set the env vars the verifier expects:
export STACKWATCH_GATEWAY_HOST=192.168.0.115
export STACKWATCH_TEST_EMAIL=you@example.com
export STACKWATCH_TEST_PASSWORD=YourPass!2026Stack

# 3. Run
python hermes-verify-tier0-complete-2026-08-17.py
python hermes-verify-tier0-1-2-live-2026-08-17.py
```

---

## CI gates (each PR must pass)

```bash
go build ./...                  # must succeed, zero output
go vet ./...                    # zero output
gofmt -l .                      # zero files returned
golangci-lint run ./...         # configured in .golangci.yml
bash scripts/file-size-check.sh # no file > 500 lines
go test ./internal/... -count=1 -race -timeout 60s   # all tests pass
```

The CI runs on every PR via `.github/workflows/backend-ci.yml`.

---

## Manual GUI smoke test (browser)

1. Open `https://stackwatch.example.com/login`
2. Log in with the test account
3. **Servers** page: should show at least the API host itself
4. **Terminal** page: open a new connection
5. (Tier 1) **Proxmox** → node detail: should show VMs

---

## Known gaps (Tier 0-3 only)

| Feature | Status |
|---------|--------|
| Frontend React app | Not started (§0.5 Vite+React+TS scaffold is planned for Tier 4.0+) |
| SAML/OIDC SSO | Tier 9 |
| Audit log chain (tamper-evident) | Tier 9 |
| Row-Level Security (RLS) | Tier 9 |
| Dashboard polish (Datadog-style) | Tier 12 |

---

## What I need from you

If you find a bug or have a question not covered above, file an issue:

```
https://github.com/Himanshu7613/stackwatch/issues/new
```

Include: `curl` command you ran, expected vs actual response, your
`stackwatch version` (`curl /api/v1/setup/state` returns this).
