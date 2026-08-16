# StackWatch — Installation Guide

**Last updated:** 2026-08-16 (Tier 0)

## Prerequisites

- Linux x86_64 server (Ubuntu 22.04+ recommended) — 2 CPU, 4 GB RAM minimum
- PostgreSQL 14+ (we test on 16)
- Go 1.23+ (to build from source)
- Node.js 20+ (for the web frontend)
- Ports 8080 (api-gateway) and 5173 (dev frontend) accessible

## Production install (systemd + binaries)

### 1. Clone the repo

```bash
git clone https://github.com/Himanshu7613/stackwatch.git
cd stackwatch
```

### 2. Build

```bash
# Backend
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/api-gateway-linux ./cmd/api-gateway

# Frontend
cd web && npm install && npm run build && cd ..
```

### 3. Configure environment

Create `/etc/stackwatch/stackwatch.env`:

```bash
DATABASE_URL=postgres://ios:STRONG_PASSWORD@db.internal:5432/ios?sslmode=disable
JWT_SECRET=GENERATE_A_RANDOM_64_CHAR_SECRET_HERE_PLEASE_ROTATE
HTTP_ADDR=:8080
```

Generate a JWT secret:

```bash
openssl rand -hex 32
```

### 4. Create database

```bash
PGPASSWORD=DB_ADMIN_PASSWORD psql -h DB_HOST -U DB_ADMIN -d postgres -c "CREATE DATABASE ios;"
PGPASSWORD=DB_PASSWORD psql -h DB_HOST -U ios -d ios -f migrations/000_init_schema.sql
```

### 5. Run

```bash
# Copy binaries
sudo mkdir -p /opt/stackwatch/bin /opt/stackwatch/logs
sudo cp bin/api-gateway-linux /opt/stackwatch/bin/
sudo chmod +x /opt/stackwatch/bin/api-gateway-linux

# Start (replace the env file path with your own)
sudo systemctl edit stackwatch-api-gateway --stdin <<EOF
[Service]
EnvironmentFile=/etc/stackwatch/stackwatch.env
ExecStart=/opt/stackwatch/bin/api-gateway-linux
Restart=always
RestartSec=10
User=root
WorkingDirectory=/opt/stackwatch
StandardOutput=append:/opt/stackwatch/logs/api-gateway.log
StandardError=append:/opt/stackwatch/logs/api-gateway.log
EOF

# Enable + start
sudo systemctl daemon-reload
sudo systemctl start stackwatch-api-gateway
sudo systemctl status stackwatch-api-gateway
```

### 6. Verify

```bash
curl http://localhost:8080/health
# {"db":"ok","status":"ok","version":"0.1.0-tier0"}
```

## Development install

```bash
# Backend (auto-reloads on file change using air or just rebuild)
go run ./cmd/api-gateway

# Frontend (Vite dev server with hot reload)
cd web && npm run dev
# Open http://localhost:5173 — proxies /api to :8080
```

## Reverse proxy (nginx)

```nginx
location /api {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}
```

## Troubleshooting

- **`db_url_set: false`** — DATABASE_URL env var is empty
- **`invalid token`** — JWT_SECRET differs between sessions (rotate the secret)
- **`403` on /auth/me after signup** — wait a moment; user needs to be created first (handled)
- **Database migrations** — `000_init_schema.sql` is idempotent (CREATE IF NOT EXISTS), safe to re-run
