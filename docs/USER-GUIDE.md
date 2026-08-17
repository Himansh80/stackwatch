# StackWatch — User Guide

**Last updated:** 2026-08-17

This guide covers **what is actually shipped right now**: Tier 0 (auth +
tenants + users + api-keys), Tier 1 (Proxmox full replacement), Tier 2
(TrueNAS SCALE 25.10 sidecar), and Tier 3 (browser SSH terminal).

Future tiers expand this guide as features ship.

---

## What StackWatch does

StackWatch is a **single-binary self-hosted platform** that replaces:

| Replaces | With |
|----------|------|
| Proxmox UI (web interface only) | Tier 1 — full replacement, 78 endpoints |
| TrueNAS UI (web interface only) | Tier 2 — JSON-RPC over WebSocket, 17 endpoints |
| Termius / SecureCRT / MobaXterm | Tier 3 — WebSocket terminal + SFTP, 33 endpoints |
| Datadog | Tier 6 — in progress (Netdata/Prom/Grafana/Loki integrations) |
| Cockpit / Server Admin | Tier 4 — pending (services, processes, network, logs) |
| Portainer | Tier 5 — pending (containers, compose, watchtower) |
| PagerDuty + OpsGenie | Tier 8 — partial (rule evaluator + 11 channels) |

Pricing per server starts at **$0 (self-host)** and tops at **$8/server/month**
for SaaS. See [PRICING.md](PRICING.md).

---

## Tier 0 — Authentication

### Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \\
  -H "Content-Type: application/json" \\
  -d '{"email":"you@example.com","password":"YourPass!2026Stack"}'
```

Returns `{ "token": "<JWT>", "user": {...} }`. The JWT is valid for 24 hours.

Use it on subsequent calls:

```bash
curl -H "Authorization: Bearer ${TOKEN}" \\
  http://localhost:8080/api/v1/auth/me
```

### Sign up

Sign-up is admin-gated by default. To create another user:

```bash
curl -X POST http://localhost:8080/api/v1/users \\
  -H "Authorization: Bearer ${TOKEN}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "email":"newadmin@example.com",
    "full_name":"New Admin",
    "role":"super_admin",
    "password":"InitialPass!2026Stack"
  }'
```

`role` is one of `viewer` / `operator` / `admin` / `super_admin`.
`super_admin` bypasses tenant scoping (for multi-tenant operations).

### Forgot password

```bash
curl -X POST http://localhost:8080/api/v1/auth/forgot \\
  -H "Content-Type: application/json" \\
  -d '{"email":"you@example.com"}'
```

If the user exists, returns a reset token (in production, this is emailed).

### Reset password

```bash
curl -X POST http://localhost:8080/api/v1/auth/reset \\
  -H "Content-Type: application/json" \\
  -d '{"token":"<reset-token>", "new_password":"NewPass!2026Stack"}'
```

### Change your password (when logged in)

```bash
curl -X POST http://localhost:8080/api/v1/auth/change-password \\
  -H "Authorization: Bearer ${TOKEN}" \\
  -H "Content-Type: application/json" \\
  -d '{"old_password":"<current>", "new_password":"<new>"}'
```

### Refresh token

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \\
  -H "Content-Type: application/json" \\
  -d '{"token":"<existing-still-valid-jwt>"}'
```

Returns a fresh JWT. Useful for long-lived sessions.

### Sign out

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \\
  -H "Authorization: Bearer ${TOKEN}"
```

(Server-side token revocation is a future tier; JWTs expire naturally after 24h.)

### API Keys

For service-to-service auth (e.g. agent registration):

```bash
# Create
curl -X POST http://localhost:8080/api/v1/api-keys \\
  -H "Authorization: Bearer ${TOKEN}" \\
  -H "Content-Type: application/json" \\
  -d '{"name":"monitoring-agent","scopes":["read","write"]}'
# → {"id":"...","token":"sw_<6hex>_<base64url>","prefix":"sw_xxxxxx_"}
# ⚠️ Copy the token NOW — it is never shown again.

# Use it
curl -H "Authorization: Bearer sw_xxxxxx_<base64url>" \\
  http://localhost:8080/api/v1/servers
```

Revoke when done:

```bash
curl -X POST http://localhost:8080/api/v1/api-keys/${KEY_ID}/revoke \\
  -H "Authorization: Bearer ${TOKEN}"
```

---

## Tier 1 — Proxmox

### Register a Proxmox host

```bash
curl -X POST http://localhost:8080/api/v1/proxmox/hosts \\
  -H "Authorization: Bearer ${TOKEN}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "name":"homelab",
    "base_url":"https://192.168.0.107:8006",
    "api_token":"root@pam!stackwatch-token=<uuid>"
  }'
```

The token gets auto-prefixed with `PVEAPIToken=` internally. Returns
`{"host":{"id":"<uuid>",...}}`.

### List VMs

```bash
HOST_ID="<uuid>"
curl -H "Authorization: Bearer ${TOKEN}" \\
  "http://localhost:8080/api/v1/proxmox/hosts/${HOST_ID}/vms"
# → {"vms":[...], "total":N}
```

### Create + Start + Stop a VM

```bash
# Create
curl -X POST -H "Authorization: Bearer ${TOKEN}" -H "Content-Type: application/json" \\
  -d '{"vmid":9000,"name":"test-vm","memory":2048,"cores":2,"sockets":1,"net0":"virtio,bridge=vmbr0","scsi0":"local-lvm:32"}' \\
  "http://localhost:8080/api/v1/proxmox/hosts/${HOST_ID}/nodes/router/qemu"

# Start
curl -X POST -H "Authorization: Bearer ${TOKEN}" \\
  "http://localhost:8080/api/v1/proxmox/hosts/${HOST_ID}/nodes/router/qemu/9000/status/start"
```

### Real-world Proxmox quirks we know about

(Listed in journal.md and tested live — these come up.)

- VM type filter is `?type=vm` (not `?type=qemu`)
- Task log endpoint returns `[{t,n}, ...]` (flat array, not wrapped)
- ACME endpoint `/cluster/acme/info` may 501 on older Proxmox versions
- ISCSI may 501 on the host even if the kernel supports it

See [docs/FEATURES.md](FEATURES.md#-tier-1--proxmox-done) for the
complete feature list.

---

## Tier 2 — TrueNAS SCALE

Proxied through the **truenas-connector** sidecar (port 8088) which
authenticates via JSON-RPC over WebSocket.

### Register a TrueNAS host

```bash
curl -X POST http://localhost:8080/api/v1/truenas/hosts \\
  -H "Authorization: Bearer ${TOKEN}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "name":"bareilly-nas",
    "base_url":"https://192.168.0.112",
    "api_token":"<truenas-api-key>"
  }'
```

### List pools, datasets, snapshots

```bash
HOST_ID="<uuid>"
curl -X POST -H "Authorization: Bearer ${TOKEN}" -H "Content-Type: application/json" \\
  -d "{\"host_id\":\"${HOST_ID}\"}" \\
  "http://localhost:8081/pools/list"

curl -X POST -H "Authorization: Bearer ${TOKEN}" -H "Content-Type: application/json" \\
  -d "{\"host_id\":\"${HOST_ID}\"}" \\
  "http://localhost:8081/datasets/list"

curl -X POST -H "Authorization: Bearer ${TOKEN}" -H "Content-Type: application/json" \\
  -d "{\"host_id\":\"${HOST_ID}\", \"filters\":[[\"pool\",\"=\",\"tank\"]]}" \\
  "http://localhost:8081/snapshots/list"
```

### Real-world TrueNAS quirks we work around

- Old `/api/v2.0/*` REST API is **removed** on SCALE 25.10; use
  JSON-RPC over `wss://<host>/api/current` only
- Middleware rate-limits login → we cache the WS connection per host
- Empty `filters` array must be wrapped in nested lists; `null` filters
  trip SCALE middleware with `[EINVAL] filters: ...`
- `auth.login` returns `true` (or `false`), not a token. Cookie session
  is established by the WS upgrade

---

## Tier 3 — Browser SSH Terminal

### Create a connection

```bash
curl -X POST http://localhost:8080/api/v1/terminal/connections \\
  -H "Authorization: Bearer ${TOKEN}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "name":"my-server",
    "host":"192.168.0.107",
    "port":22,
    "username":"root",
    "auth_type":"key",
    "key_id":"<sshkey-uuid>"
  }'
```

`auth_type` is `password` or `key`. Passwords go in the encrypted
credentials vault at `/api/v1/terminal/credentials`.

### Add an SSH key

```bash
# Generate one
ssh-keygen -t ed25519 -f ~/.ssh/stackwatch -N "" -C "stackwatch-agent"

# Upload public key
PUB_KEY=$(cat ~/.ssh/stackwatch.pub)
curl -X POST http://localhost:8080/api/v1/terminal/keys \\
  -H "Authorization: Bearer ${TOKEN}" \\
  -H "Content-Type: application/json" \\
  -d "{\"name\":\"stackwatch\",\"public_key\":\"${PUB_KEY}\"}"
# Returns key id; pair with your private key in your ~/.ssh/config
```

### Open a terminal session

WebSocket at `ws://localhost:8085/api/v1/ws?connection=<connection-uuid>`.
Compatible with **xterm.js** on the front-end. Streams real SSH PTY bytes
both directions.

### SFTP file browser

```bash
# List directory
CONN_ID="<connection-uuid>"
curl -H "Authorization: Bearer ${TOKEN}" \\
  "http://localhost:8080/api/v1/terminal/connections/${CONN_ID}/fs?path=/etc"
# → {"entries":[...],"path":"/etc"}

# Read file
curl -H "Authorization: Bearer ${TOKEN}" \\
  "http://localhost:8080/api/v1/terminal/connections/${CONN_ID}/fs/read?path=/etc/hostname"

# Write file
curl -X POST -H "Authorization: Bearer ${TOKEN}" -H "Content-Type: application/json" \\
  -d '{"path":"/tmp/foo","content":"hello"}' \\
  "http://localhost:8080/api/v1/terminal/connections/${CONN_ID}/fs/write"
```

### Trust on first connect (MITM protection)

```bash
curl -X POST http://localhost:8080/api/v1/terminal/known-hosts/trust \\
  -H "Authorization: Bearer ${TOKEN}" \\
  -H "Content-Type: application/json" \\
  -d '{"host":"192.168.0.107","port":22,"fingerprint":"SHA256:..."}'
```

After trusting, new connections skip the prompt. Without trust, SFTP /
terminal refuses to connect.

---

## Dashboard

The static dashboard at `https://stackwatch.example.com/` is a single-page
app that talks to all three binaries. **Tier 4+** will replace it with
a real React frontend; right now it's a thin explorer over the API.

---

## Roles

| Role | What they can do |
|------|------------------|
| `viewer` | Read-only across all endpoints in their tenant |
| `operator` | + Write on alert rules, dashboards, status pages |
| `admin` | + Create/delete users, API keys, hosts |
| `super_admin` | + Bypass tenant scoping, manage all tenants |

---

## Where to get help

- [FAQ.md](FAQ.md)
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- [GitHub Issues](https://github.com/Himanshu7613/stackwatch/issues)
- Email: `support@stackwatch.io`

## Roadmap

See [FEATURES.md](FEATURES.md) for the full Tier 0-13 roadmap.
