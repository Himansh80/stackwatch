# StackWatch — Installation Guide

**Last updated:** 2026-08-25

This is the **shipped** installation guide for current StackWatch
(Tier 0–11 complete). Future tiers add features behind feature flags;
this guide tells you what's real right now.

Cross-references: [USER-GUIDE.md](USER-GUIDE.md) · [ARCHITECTURE.md](ARCHITECTURE.md) ·
[FAQ.md](FAQ.md) · [TROUBLESHOOTING.md](TROUBLESHOOTING.md) ·
[PRICING.md](PRICING.md)

---

## Onboarding checklist (15-minute TTFW)

If you want to go from zero to "first dashboard in 15 minutes", follow
this checklist in order. Every step has been timed; total is **12–18
minutes** on commodity hardware.

| Step | Time | What you'll have |
|------|------|------------------|
| 1. Provision a Linux VM (Ubuntu 22.04 LTS, 2 vCPU, 4 GB RAM) | 5 min | A clean host |
| 2. Install Postgres 16 (apt or docker) | 2 min | Database up |
| 3. Run the install.sh one-liner | 1 min | Three binaries deployed |
| 4. Open `https://<host>:8080/setup` and complete the wizard | 2 min | First admin account created |
| 5. Install the agent on your first monitored host | 1 min | One host emitting metrics |
| 6. Open the dashboard → Servers → click the new host | 1 min | Your first dashboard |
| 7. Add a second integration (Proxmox or TrueNAS) | 3 min | A second data source visible |
| 8. Set up one alert rule + one notification channel | 3 min | An end-to-end alert path |

If you hit a wall at any step, see [TROUBLESHOOTING.md](TROUBLESHOOTING.md).
The 15-minute claim assumes a Linux laptop or VM with internet access.

---

## Prerequisites

### Hardware

| Tier | Recommended host | Minimum | Disk |
|------|------------------|---------|------|
| Homelab (≤ 10 servers) | 2 vCPU, 4 GB RAM | 1 vCPU, 2 GB RAM | 20 GB |
| SMB (≤ 50 servers) | 4 vCPU, 8 GB RAM | 2 vCPU, 4 GB RAM | 100 GB |
| Mid (50–200 servers) | 8 vCPU, 16 GB RAM | 4 vCPU, 8 GB RAM | 500 GB |
| Enterprise (200+ servers, multi-region) | 16+ vCPU, 32 GB RAM | 8 vCPU, 16 GB RAM | 1 TB+ |

StackWatch itself is lightweight; budget for the agent fleet and the
metric retention. At 50 hosts with 90-day metric retention (Pro tier),
expect ~50 GB of metrics storage.

### Software

- **Linux x86_64** server (Ubuntu 22.04 LTS or Debian 12 recommended)
  — also tested on RHEL 9, Fedora 39, Amazon Linux 2023
- **macOS 13+** for local development
- **PostgreSQL 14+** (we test on 16) on the same host or another host
- **Go 1.23+** (to build from source)
- **Docker 24+** (optional, for the Docker install path)
- **kubectl** (optional, for the Kubernetes install path)
- **No frontend build step needed** — the static dashboard lives at
  `web/public/`, served by the api-gateway on port 8080.
- Ports **8080** (api-gateway), **8085** (web-terminal),
  **8088** (truenas-connector) accessible

### Network

- Outbound HTTPS to your package manager (apt/yum)
- Outbound HTTPS to `github.com` (binary downloads if you curl-pipe)
- Inbound 8080/8085/8088 from your browser and agents
- Outbound to your monitored hosts (Proxmox 8006, TrueNAS 80/443,
  SSH 22 on monitored servers)

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
| `DATABASE_URL` | yes (api-gateway) | `postgres://ios:***@<host>:5432/ios?sslmode=disable` |
| `JWT_SECRET` | yes | min 32 chars; same across api-gateway replicas |
| `IOS_SECRET_KEY` | yes | bcrypt + JWT secret (legacy alias) |
| `INSTALL_MODE` | yes | `cloud` or `self-hosted` (auto-detected in setup) |
| `ALLOWED_ORIGINS` | optional | CORS allowlist (default `*`) |
| `LOG_LEVEL` | optional | `debug` / `info` / `warn` / `error` (default `info`) |
| `BRAND_NAME` | optional | OEM re-branding (Tier 4 OEM) |
| `BRAND_PRIMARY_COLOR` | optional | OEM theming (Tier 4 OEM) |

Example `.env`:

```bash
DATABASE_URL=postgres://ios:***@192.168.0.116:5432/ios?sslmode=disable
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

### 5. systemd unit files

We ship `scripts/*.service`; here is the canonical `api-gateway.service`:

```ini
# /etc/systemd/system/api-gateway.service
[Unit]
Description=StackWatch api-gateway (REST API + JWT + gateway)
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=stackwatch
Group=stackwatch
WorkingDirectory=/opt/stackwatch
EnvironmentFile=/opt/stackwatch/.env
ExecStart=/opt/stackwatch/bin/api-gateway-linux
Restart=on-failure
RestartSec=5s
LimitNOFILE=65536

# Hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/stackwatch/data
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
```

Identical units exist for `web-terminal` (port 8085) and
`truenas-connector` (port 8088).

### 6. Run

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

### 7. Verify

```bash
curl -sm 3 http://127.0.0.1:8080/health
# → {"db":"ok","mode":"cloud","status":"ok","version":"0.1.0-tier11"}

# Then setup wizard at /setup (in browser)
# Or run setup/state machine-mode:
curl http://127.0.0.1:8080/api/v1/setup/status
```

---

## Docker install

A `docker-compose.yml` is shipped at the repo root for local
development and small self-hosted deployments.

### Prerequisites

- Docker Engine 24+
- Docker Compose v2 (`docker compose` not `docker-compose`)
- 4 GB RAM available to Docker

### One-command bring-up

```bash
git clone https://github.com/Himanshu7613/stackwatch.git
cd stackwatch
docker compose up -d
docker compose ps
# Expect 4 services: postgres, api-gateway, web-terminal, truenas-connector
```

The default `docker-compose.yml` exposes:

- `8080` → api-gateway (REST + dashboard)
- `8085` → web-terminal (WS)
- `8088` → truenas-connector (sidecar)
- `5432` → postgres (bind to `127.0.0.1` only by default)

### Verify

```bash
curl -sm 3 http://127.0.0.1:8080/health
# → {"db":"ok","mode":"cloud","status":"ok","version":"0.1.0-tier11"}
```

Open `http://127.0.0.1:8080/setup` in a browser and complete the
wizard. The default admin email + password is in `.env.example`.

### Production-grade Docker

For Docker in production, **do not** use the dev `docker-compose.yml`.
Instead:

1. Pin image tags (`stackwatch/api-gateway:v0.1.0-tier11.5`)
2. Mount `/opt/stackwatch/data` to a host volume (NOT a Docker volume)
3. Put Postgres on a managed service (RDS, Cloud SQL, or self-managed
   on a separate host) — **do not** run Postgres as a container on the
   same host as the api-gateway
4. Run `docker compose --profile prod up -d` (the prod profile
   excludes Postgres)
5. Front with nginx or Caddy for TLS termination

---

## Kubernetes install

A Helm chart is shipped at `deploy/helm/stackwatch/`. Tested on
Kubernetes 1.27+.

### Prerequisites

- `kubectl` configured for your cluster
- `helm` 3.13+
- An existing Postgres (in-cluster or managed)
- A wildcard TLS cert or cert-manager

### Install

```bash
helm repo add stackwatch https://charts.stackwatch.io
helm install stackwatch stackwatch/stackwatch \
  --namespace monitoring --create-namespace \
  --set database.url="postgres://ios:***@postgres:5432/ios" \
  --set jwt.secret="$(head -c 64 /dev/urandom | base64)" \
  --set ingress.host=stackwatch.example.com
```

### Verify

```bash
kubectl -n monitoring get pods
# Expect api-gateway, web-terminal, truenas-connector all Running
kubectl -n monitoring port-forward svc/api-gateway 8080:8080
curl -sm 3 http://127.0.0.1:8080/health
```

### Scaling

The api-gateway Deployment is stateless. To scale horizontally:

```bash
kubectl -n monitoring scale deploy/api-gateway --replicas=3
```

The Deployment's HPA is configured to scale on CPU > 70%. Multi-region
failover (Tier 11 PL6) requires a second cluster in a second region
and Postgres logical replication; see [ARCHITECTURE.md §Multi-region](ARCHITECTURE.md).

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

## macOS install (development)

```bash
brew install postgresql@16
brew services start postgresql@16
createdb ios
git clone https://github.com/Himanshu7613/stackwatch.git
cd stackwatch
make build
DATABASE_URL=postgres://$(whoami)@localhost:5432/ios?sslmode=disable \
  JWT_SECRET=$(head -c 64 /dev/urandom | base64) \
  INSTALL_MODE=cloud \
  ./bin/api-gateway-darwin
```

Then open `http://localhost:8080/setup`. macOS launchd plist support
is on the roadmap; for now, run the binaries in foreground.

---

## Adding Proxmox

Once the gateway is up:

```bash
curl -X POST http://localhost:8080/api/v1/proxmox/hosts \
  -H "Authorization: Bearer ***" \
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
  -H "Authorization: Bearer ***" \
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
DATABASE_URL=postgres://ios:***@localhost:5432/ios?sslmode=disable go run ./cmd/api-gateway
```

For hot-reload during development, install `air` or `realize`:

```bash
go install github.com/air-verse/air@latest
air   # watches and rebuilds
```

---

## Upgrading

```bash
# 1. Backup first
/opt/stackwatch/bin/backup-cli snapshot

# 2. Pull the new version
git pull origin main
make build

# 3. Run any new migrations
for m in migrations/*.new.sql; do
  PGPASSWORD=<pw> psql -h <db-host> -U ios -d ios -f "$m"
done

# 4. Restart in order: api-gateway, then sidecars
sudo systemctl restart api-gateway
sudo systemctl restart web-terminal
sudo systemctl restart truenas-connector
```

Migrations are **additive-only** (the schema never drops columns
between minor versions). A major-version bump (e.g. v0.x → v1.0) may
require manual steps; see `RELEASE-NOTES.md` for each release.

---

## Uninstalling

```bash
sudo systemctl disable --now api-gateway truenas-connector web-terminal
sudo rm /etc/systemd/system/{api-gateway,truenas-connector,web-terminal}.service
sudo rm -rf /opt/stackwatch
# Drop the database
sudo -u postgres psql -c "DROP DATABASE ios;"
sudo -u postgres psql -c "DROP USER ios;"
```

---

## Troubleshooting install

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md).

Top three failures:

- `permission denied`: `sudo chmod +x /opt/stackwatch/bin/*`
- `connection refused` from db: check `DATABASE_URL` and Postgres is listening
- `invalid signature` on JWT: every secret-rotating redeploy invalidates tokens

### Where to get help

- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) — common install failures
- [FAQ.md](FAQ.md) — operational questions
- GitHub Issues: https://github.com/Himanshu7613/stackwatch/issues
- Email: `support@stackwatch.io`