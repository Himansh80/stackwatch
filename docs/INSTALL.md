# StackWatch — Installation Guide

**Last updated:** 2026-08-17

This is the **shipped** installation guide for current StackWatch (Tier
0/1/2/3 complete). Future tiers add features behind feature flags; this
guide tells you what's real right now.

---

## Prerequisites

- Linux x86_64 server (Ubuntu 22.04+ recommended) — 2 CPU, 4 GB RAM minimum
- PostgreSQL 14+ (we test on 16) on the same host or another host
- Go 1.23+ (to build from source)
- **No frontend build step needed** for the current shipped API tier —
  the static dashboard lives at `web/public/` (planned §0.5 React rebuild
  is on the Tier 4+ roadmap)
- Ports 8080 (api-gateway), 8085 (web-terminal), 8088 (truenas-connector-sidecar)
  accessible

---

## Production install (systemd + binaries)

### 1. Clone the repo

```bash
git clone https://github.com/Himanshu7613/stackwatch.git
cd stackwatch
```

### 2. Build

```bash
# All three binaries (api-gateway, web-terminal, truenas-connector)
GOMAXPROCS=1 nice -n 15 go build -p=1 -o bin/api-gateway-linux ./cmd/api-gateway
GOMAXPROCS=1 nice -n 15 go build -p=1 -o bin/web-terminal-linux ./cmd/web-terminal
GOMAXPROCS=1 nice -n 15 go build -p=1 -o bin/truenas-connector-linux ./cmd/truenas-connector
```

Cross-compiling for delivery to a Linux host from a macOS/Windows dev:

```bash
for target in api-gateway web-terminal truenas-connector; do
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build -ldflags="-s -w" -o bin/${target}-linux ./cmd/${target}
done
```

### 3. Configure environment

Each service needs:

| Variable | Required | Purpose |
|----------|----------|---------|
| `DATABASE_URL` | yes (api-gateway) | `postgres://ios:<pw>@<host>:5432/ios?sslmode=disable` |
| `JWT_SECRET` | yes | min 32 chars; same across api-gateway replicas |
| `IOS_SECRET_KEY` | yes | bcrypt + JWT secret (legacy alias) |
| `INSTALL_MODE` | yes | `cloud` or `self-hosted` (auto-detected in setup) |
| `ALLOWED_ORIGINS` | optional | CORS allowlist (default `*`) |
| `LOG_LEVEL` | optional | `debug` / `info` / `warn` / `error` (default `info`) |

Example `.env`:

```bash
DATABASE_URL=postgres://ios:3d5cb43fba1f82283a2ba02c79e116cf@192.168.0.116:5432/ios?sslmode=disable
JWT_SECRET=$(head -c 64 /dev/urandom | base64)
INSTALL_MODE=cloud
ALLOWED_ORIGINS=https://stackwatch.example.com
```

### 4. Create database

```bash
# On the Postgres host
sudo -u postgres psql <<'SQL'
CREATE USER ios PASSWORD '<change-me>';
CREATE DATABASE ios OWNER ios;
SQL

# Run migrations in order
for m in migrations/*.sql; do
  PGPASSWORD=<pw> psql -h <db-host> -U ios -d ios -f "$m"
done
```

### 5. Run

```bash
# Install binaries to a stable path
sudo install -m 0755 bin/api-gateway-linux       /opt/stackwatch/bin/
sudo install -m 0755 bin/web-terminal-linux      /opt/stackwatch/bin/
sudo install -m 0755 bin/truenas-connector-linux /opt/stackwatch/bin/

# Create systemd units (templates in scripts/)
sudo install scripts/*.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now api-gateway truenas-connector web-terminal
```

### 6. Verify

```bash
curl -sm 3 http://127.0.0.1:8080/health
# → {"db":"ok","mode":"cloud","status":"ok","version":"0.1.0-tier0.5"}

# Then setup wizard at /setup (in browser)
# Or run setup/state machine-mode:
curl http://127.0.0.1:8080/api/v1/setup/status
```

---

## Architecture summary

StackWatch runs as **three Linux binaries + Postgres**:

```
                    :8080 ────── api-gateway (REST API, JWT, gateway)
    Browser  ━━━━━▷
    Agents   ━━━━━▷

                    :8085 ────── web-terminal (WebSocket ↔ SSH PTY bridge)

                    :8088 ────── truenas-connector (JSON-RPC over WebSocket sidecar)
                                ↕ wss://<truenas-host>/api/current
```

Each service is independent. They share only the Postgres database for
metadata (users / tenants / api-keys / connection state).

---

## Reverse proxy (nginx or Caddy)

Recommended: terminate TLS at an edge proxy. Don't enable TLS in the
binaries themselves in production.

```caddyfile
# /etc/caddy/Caddyfile
stackwatch.example.com {
  reverse_proxy localhost:8080
}
```

```nginx
# /etc/nginx/sites-available/stackwatch
server {
  listen 443 ssl http2;
  server_name stackwatch.example.com;
  ssl_certificate /etc/letsencrypt/live/stackwatch.example.com/fullchain.pem;
  ssl_certificate_key /etc/letsencrypt/live/stackwatch.example.com/privkey.pem;
  location / { proxy_pass http://localhost:8080; proxy_set_header Host $host; }
}
```

---

## Adding Proxmox

Once the gateway is up:

```bash
curl -X POST http://localhost:8080/api/v1/proxmox/hosts \
  -H "Authorization: Bearer ${JWT}" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"homelab-proxmox",
    "base_url":"https://192.168.0.107:8006",
    "api_token":"root@pam!stackwatch-token=UUID"
  }'
```

Returns the host ID. From there, all 78 Proxmox endpoints work (VMs,
LXC, storage, network, firewall, disks, ZFS, users+tokens, tasks,
ISCSI, pools, backup, certificates+ACME).

---

## Adding TrueNAS

```bash
curl -X POST http://localhost:8080/api/v1/truenas/hosts \
  -H "Authorization: Bearer ${JWT}" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"homelab-truenas",
    "base_url":"https://192.168.0.112",
    "api_token":"<truenas-api-key>",
    "verify_tls":false
  }'
```

The api-gateway talks to the local **truenas-connector** sidecar
which speaks JSON-RPC over WebSocket to TrueNAS middleware.

---

## Development install (Go-only, in-place)

```bash
go run ./cmd/api-gateway       # uses default SQLite fallback (dev)
DATABASE_URL=postgres://ios:test@localhost:5432/ios?sslmode=disable go run ./cmd/api-gateway
```

For hot-reload during development, install `air` or `realize`:

```bash
go install github.com/air-verse/air@latest
air   # watches and rebuilds
```

---

## Troubleshooting install

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md).

Top three failures:
- `permission denied`: `sudo chmod +x /opt/stackwatch/bin/*`
- `connection refused` from db: check `DATABASE_URL` and Postgres is listening
- `invalid signature` on JWT: every secret-rotating redeploy invalidates tokens
