# Tier 11 Specification — Platform & Commerce

## 1. Goals (measurable)

| Metric | Target |
|--------|--------|
| Tier 11 routes live on `.115` | ≥ 45 |
| Tier 11 DB tables on `.116` | 8 (plus 1 ALTER on users for `is_platform_admin`) |
| All files < 400 LOC | Yes (every Go + TSX) |
| One-command deploy script | Working (curl \| bash) |
| Self-service signup rate limit | 10/day per IP |
| Free tier server cap | 3 (configurable via env) |
| Retention enforcement | Daily cron at 03:00 UTC |
| Multi-region replication | Active-passive (cloud mode only) |
| Rate limit per tenant | Based on plan tier |
| Platform health dashboard | Real-time (1s poll) |
| BACKUP_ENCRYPTION | AES-256-GCM with tenant-derived key |

## 2. Sub-features

### PL1 — One-Command Deploy

**Tables:** `platform_deploy_tokens`

**Routes (4):**
- `POST /api/v1/platform/deploy/install-token` — generate a one-time install token (cloud mode only)
- `GET  /api/v1/platform/deploy/install.sh` — return the canonical install script
- `GET  /api/v1/platform/deploy/status?token=X` — check install progress
- `POST /api/v1/platform/deploy/mode` — switch INSTALL_MODE (cloud|self_hosted) — super_admin only

**Install script flow:**
1. User runs `curl -fsSL https://stackwatch.smarthomelab.fun/install.sh | bash`
2. Script downloads binary + `.env` template
3. Script prompts for INSTALL_MODE
4. Script writes systemd unit + starts service
5. Service hits `/api/v1/platform/deploy/install-token` (auto-generates)
6. Service hits `/api/v1/platform/deploy/status?token=X` to mark active

### PL2 — Usage Metering

**Tables:** `platform_usage_events`, `platform_usage_aggregates`

**Routes (5):**
- `POST /api/v1/platform/usage/event` — record a usage event (server.created, alert.fired, api.call, storage.gb, dashboard.panel.rendered)
- `GET  /api/v1/platform/usage/current?tenant_id=X&period=month` — current period usage
- `GET  /api/v1/platform/usage/history?tenant_id=X&periods=12` — historical usage (chart data)
- `GET  /api/v1/platform/usage/summary` — super_admin only — all tenants summary
- `GET  /api/v1/platform/usage/export?tenant_id=X&format=csv` — export raw events

**Background worker:** `internal/platform/usage_meter.go` — aggregates raw
events into hourly buckets every hour.

### PL3 — Self-Service Signup

**Tables:** `platform_signups`

**Routes (4):**
- `POST /api/v1/platform/signup` — create new tenant + admin user (cloud mode only, rate-limited 10/day/IP)
- `POST /api/v1/platform/signup/verify` — verify email token
- `POST /api/v1/platform/signup/resend` — resend verification email
- `GET  /api/v1/platform/signup/check-email?email=X` — is email already taken?

**Note:** `INSTALL_MODE=self_hosted` rejects this endpoint with 403.

### PL4 — Tenant Limits

**Tables:** `platform_tenant_limits`, `platform_plan_definitions`

**Routes (5):**
- `GET    /api/v1/platform/limits/definitions` — return all plan definitions (free/starter/pro/enterprise)
- `GET    /api/v1/platform/limits/me` — get my plan + current usage
- `PATCH  /api/v1/platform/limits/me` — change my plan (super_admin only)
- `POST   /api/v1/platform/limits/check` — dry-run: would this operation exceed my limit?
- `GET    /api/v1/platform/limits/usage` — current usage vs limits (for billing warnings)

**Limits enforced** (background `internal/platform/retention.go` + middleware):
- Free: 3 servers, 7-day data retention, 14-day metric retention, 1GB storage
- Starter: 10 servers, 30-day data, 30-day metrics, 10GB
- Pro: 50 servers, 90-day data, 90-day metrics, 100GB
- Enterprise: unlimited everything

### PL5 — Backup/Restore

**Tables:** `platform_backups`, `platform_backup_jobs`

**Routes (5):**
- `POST /api/v1/platform/backup/create` — trigger manual backup (returns job_id)
- `GET  /api/v1/platform/backup/list?tenant_id=X` — list backups for tenant
- `GET  /api/v1/platform/backup/download?backup_id=X` — download .tar.gz (encrypted)
- `POST /api/v1/platform/backup/restore` — restore from uploaded .tar.gz
- `GET  /api/v1/platform/backup/schedule?tenant_id=X` — get backup schedule

**Backup contents:**
- Postgres pg_dump (custom format, compressed)
- AES-256-GCM encrypted with tenant-derived key from `CREDENTIALS_MASTER_KEY`
- Stored in `/opt/stackwatch/backups/{tenant_id}/{backup_id}.tar.gz.enc`
- 7-day rolling retention (configurable)

### PL6 — Multi-Region/HA

**Tables:** `platform_regions`, `platform_region_replicas`

**Routes (3):**
- `GET  /api/v1/platform/regions` — list configured regions
- `POST /api/v1/platform/regions` — add region (cloud mode only)
- `GET  /api/v1/platform/regions/health` — per-region health check

**Architecture:** Active-passive Postgres replication. Reads from primary
region, writes to primary region only. Standby regions are read-only.

### PL7 — Rate Limiting

**Tables:** `platform_rate_limit_buckets` (in-memory)

**Routes (3):**
- `GET  /api/v1/platform/ratelimit/me` — get my current rate limit usage (X-RateLimit-* headers exposed)
- `PATCH /api/v1/platform/ratelimit/global` — super_admin only — adjust global limits
- `GET  /api/v1/platform/ratelimit/blocked` — super_admin only — currently blocked tenants

**Implementation:** Token-bucket per-tenant, in-memory (or Redis if available).
Headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`.

### PL8 — Platform Health

**Tables:** `platform_health_snapshots` (cached; auto-pruned at 7 days)

**Routes (5):**
- `GET /api/v1/platform/health/summary` — overall health (UP/DEGRADED/DOWN per region)
- `GET /api/v1/platform/health/regions` — per-region metrics (CPU, mem, disk, db_connections)
- `GET /api/v1/platform/health/tenants/top?n=10` — top tenants by usage
- `GET /api/v1/platform/health/capacity/forecast` — projected capacity exhaustion dates
- `GET /api/v1/platform/health/alerts` — active platform-level alerts (failed regions, exceeded limits, etc.)

**Background worker:** `internal/platform/capacity_forecast.go` — runs daily,
computes linear regression on metrics, returns 30/60/90-day projections.

## 3. Non-functional

- Every Go file < 400 LOC
- Every TSX file < 400 LOC
- 45 routes verified live with auth
- 8 tables created + indexes
- 5 background workers running
- Per-tenant isolation enforced on every query
- New role `platform_admin` (separate from `super_admin`)
- INSTALL_MODE env var: cloud | self_hosted (auto-detect)
- License key validation (cloud mode only)

## 4. Verification gates

- `go build ./cmd/api-gateway` exit 0
- `go vet ./cmd/api-gateway` exit 0
- `cd web && npm run type-check` exit 0
- `cd web && npm run lint` no new errors
- `cd web && npm run build` exit 0
- All 45 routes return 401 without auth
- All 45 routes return 200/201/202 with valid JWT
- Tier 0-10 routes still work (regression gate)
- /health returns 200
- Backup + restore round-trip works (manual test)

## 5. Deployment

- Binary built on Windows (`export GOOS=linux; export GOARCH=amd64`)
- Binary deployed to .115 via scp + systemctl restart
- Binary md5 verified
- All 5 workers started without panic

## 6. Out-of-scope gates

- NO Tier 12 features (marketing site, pricing page, etc.)
- NO Tier 13 features (mobile)
- NO email service integration beyond dev mode
- NO public marketing site (Tier 12)
- NO stripe webhook verification (Tier 11 covers basic Stripe; Razorpay
  webhook HMAC validation deferred to Tier 12)
