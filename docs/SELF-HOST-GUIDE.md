# StackWatch — Self-Hosting Guide

**Last updated:** 2026-08-17

Run StackWatch on your own hardware. This is the "Tier 3" plan from
PRICING.md but the code is identical — no commercial license needed
for internal use.

---

## Minimum requirements

- Linux x86_64 (Ubuntu 22.04+, RHEL 9+, Debian 12+)
- 2 vCPU, 4 GB RAM minimum (8 GB recommended for ≥20 servers)
- 50 GB disk for the platform + 10 GB/year per agent for metrics
- Postgres 14+ (separate host recommended for production)
- Internet: not required once installed (local dashboard)

---

## Install — Linux (systemd)

```bash
# 1. Install Postgres
sudo apt install -y postgresql-16
sudo -u postgres psql -c "CREATE USER ios PASSWORD '<change-me>';"
sudo -u postgres psql -c "CREATE DATABASE ios OWNER ios;"

# 2. Build platform
git clone https://github.com/Himanshu7613/stackwatch.git
cd stackwatch
go build -o bin/api-gateway-linux ./cmd/api-gateway

# 3. Run setup wizard (browser-friendly)
/opt/stackwatch/bin/api-gateway-linux --setup-mode
# → opens http://localhost:8080/setup on first visit
# → creates admin account, JWT secret, telemetry opt-in

# 4. Run as systemd
sudo cp scripts/api-gateway.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now api-gateway
```

---

## Install — Docker (Tier 4.0 ships this)

Currently hand-rolled Docker Compose is the only option. Tier 4 will ship
an official one.

```bash
docker compose up -d
```

Coming with `docker-compose.yml`, `Dockerfile`, `install.sh`. For now use
systemd.

---

## TLS termination

Recommended: put Caddy, nginx, or HAProxy in front.

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

## Migrations

Each migration is a single SQL file in `migrations/`. Apply in order:

```bash
PGPASSWORD=<pw> psql -h <host> -U ios -d ios -f migrations/000_init_schema.sql
PGPASSWORD=<pw> psql -h <host> -U ios -d ios -f migrations/001_proxmox_schema.sql
# ... etc.
```

A migration runner is on the Tier 4.0 backlog (`bin/migrate`).

---

## Backup

- **Postgres**: nightly `pg_dump` to your snapshot storage
- **Secrets**: never stored at rest on the platform binary
- **Audit logs**: same DB, included in pg_dump

---

## Upgrades

1. Stop service: `sudo systemctl stop api-gateway`
2. Replace binary: `sudo cp bin/api-gateway-linux.new /opt/stackwatch/bin/`
3. Run migrations: `bin/migrate up`
4. Start service: `sudo systemctl start api-gateway`

---

## Active Directory / SSO integration

Tier 9 ships SAML + OIDC. For today, if your AD requires SAML, use a
SAML-to-OIDC bridge (e.g. `auth-proxy` or `mod_auth_mellon` in front of
the dashboard endpoint).

---

## Air-gap / offline

StackWatch has zero runtime dependencies on the internet after install.
The only network call is the optional telemetry beacon (`OTEL_EXPORTER_OTLP_ENDPOINT`).
Set it to `""` to disable.

---

## High availability

Single-replica for now. Multi-replica + leader election lands in Tier 5+.

---

## What "good" looks like

| Servers | CPU | RAM | Disk |
|---------|-----|-----|------|
| 1-5     | 1 vCPU | 1 GB | 5 GB  |
| 20      | 2 vCPU | 4 GB | 20 GB |
| 50      | 4 vCPU | 8 GB | 50 GB |
| 100+    | 8 vCPU | 16 GB | 100 GB |

Numbers grow linearly with `count(servers) × metrics_per_server × retention`.
Tier 6 introduces Prometheus-style retention rollups (1h/6h/1d) to flatten
this.
