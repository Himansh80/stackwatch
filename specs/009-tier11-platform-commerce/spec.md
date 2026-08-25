# Tier 11 Specification — Platform & Commerce

## 1. Goals (measurable)

| Metric | Target |
|--------|--------|
| Tier 11 routes live on `.115` | ≥ 35 |
| Tier 11 DB tables on `.116` | 8 new |
| All files < 400 LOC | Yes |
| Plans enforced at API layer | Free, Pro, Business, Enterprise |
| Signup → first heartbeat | < 60s |
| Backup cadence | Daily + on-demand |
| Backup retention | 30 days |
| Rate limit per tenant | 60 req/min default, configurable per plan |
| Rate limit per IP | 600 req/hour |
| Multi-region | v1: single-region + read replica support flag |

## 2. Sub-features

### PL1 — Push-Button Deploy

**Tables:** `deploy_install_tokens`, `deploy_servers` (already covered
by Tier 0 `servers` table; this phase adds the install-token lifecycle).

**Routes (4):**
- `POST /api/v1/platform/deploy/install-token` — create install token
  (admin only; one-time use; expires in 1 hour)
- `GET /api/v1/platform/deploy/install-script?token=X&backend=Y` — public
  endpoint that returns the install script with the token + backend
  baked in (Datadog-style one-liner)
- `POST /api/v1/agents/register` — agent registers itself (existing
  Tier 0 endpoint; this phase adds install-token validation)
- `GET /api/v1/platform/deploy/stats` — admin: count of installs in
  last 24h / 7d / 30d

**UI:** `DeploySection.tsx` — pre-canned curl snippet + copy button +
  per-OS tabs (Linux/macOS/Windows).

### PL2 — Usage Metering

**Tables:** `usage_events`, `usage_daily_rollups`

**Routes (3):**
- `POST /api/v1/usage/events` — internal: meter an event (called by
  other handlers). Body: `{event_type, quantity}`. Idempotent on
  (tenant_id, day, event_type).
- `GET /api/v1/usage/summary?from=X&to=Y` — tenant: aggregate usage
  by event_type for a date range.
- `GET /api/v1/admin/usage/all?tenant_id=X&from=Y&to=Z` — super-admin:
  cross-tenant usage view.

**Internal helpers:**
- `meterEvent(pool, tenantID, eventType, quantity)` — exported helper
  used by every billable handler.

**Worker:** `UsageRollupWorker` (hourly) — rolls up raw `usage_events`
into `usage_daily_rollups` for fast aggregation.

### PL3 — Self-Service Signup

**Routes (3):**
- `POST /api/v1/public/signup` — already exists from Tier 0; this phase
  adds captcha + IP rate limit + email verification flow.
- `POST /api/v1/public/verify-email?token=X` — confirm email; activates
  tenant.
- `POST /api/v1/public/resend-verification` — resend verification email.

**Email integration:** Resend (HTTP API) primary, SMTP fallback, dev-mode
logs to console.

### PL4 — Tenant Limits

**Tables:** `plan_limits` (one row per plan with limit JSONB).

**Routes (2):**
- `GET /api/v1/admin/plans` — list plans + limits
- `PATCH /api/v1/admin/plans/:plan` — update limits (super-admin only)

**Implementation:** Middleware on every mutation route that reads
`plan_limits[tenant.plan]` and returns 402 Payment Required if
quantity would exceed. Limits checked:
- `servers.max`
- `metrics.daily_max`
- `seats.max`
- `retention_days`
- `api.rate_per_minute`

**Internal helper:** `enforceLimit(pool, tenantID, limitType, current)` —
exported helper used by handlers.

### PL5 — Backup/Restore

**Tables:** `platform_backups`

**Routes (3):**
- `GET /api/v1/admin/backups` — list backups (with status, size, type)
- `POST /api/v1/admin/backups` — create backup on-demand (async)
- `POST /api/v1/admin/backups/:id/restore` — restore from backup
  (super-admin only, returns 202 with confirmation prompt)

**Worker:** `BackupWorker` (daily at 02:00 UTC):
1. Run `pg_dump --format=custom --file=/var/backups/stackwatch/daily-YYYYMMDD.dump`
2. Verify checksum + size > 0
3. INSERT row into `platform_backups` table with status='completed'
4. Prune backups older than 30 days
5. Optional: upload to S3 if `BACKUP_S3_BUCKET` env var set

### PL6 — Multi-Region / HA

**Tables:** `platform_regions`, `platform_replicas`

**Routes (3):**
- `GET /api/v1/admin/regions` — list configured regions
- `POST /api/v1/admin/regions` — add region (super-admin)
- `GET /api/v1/admin/replicas/status` — replication lag per replica

**v1 scope:**
- Single primary region (.116)
- Read-only replica in 1+ secondary regions (async streaming replication)
- Replica URL configurable via `READ_REPLICA_URL` env var
- Future: automatic failover (out of scope for v1)

### PL7 — Rate Limiting (per-IP + per-tenant)

**Implementation:** Sliding-window counter via `sync.Map` of
`(tenantID, route) → []request_timestamps`. Refactored out of Tier 0
`auth_login.go` into a reusable middleware.

**Limits:**
- Per-IP: 600 req/hour (free) / 6000 req/hour (pro+) / unlimited (enterprise)
- Per-tenant: 60 req/min (free) / 600 req/min (pro+) / 6000 req/min (business+)
- Per-route multipliers (e.g. /auth/* are 1/10 of normal limit)

**Worker:** `RateLimitCleanupWorker` (hourly) — prune stale entries
from the sync.Map.

### PL8 — Platform Health

**Tables:** `platform_health_samples` (time-series of service health)

**Routes (4):**
- `GET /api/v1/admin/health/services` — list services + last health check
- `GET /api/v1/admin/health/workers` — list background workers + last run
- `GET /api/v1/admin/health/queues` — depth of every queue (audit, sync, etc.)
- `GET /api/v1/public/status` — PUBLIC: minimal status page payload
  (services up/down + version + uptime). Used by status.smarthomelab.fun.

**Worker:** `PlatformHealthWorker` (60s) — samples every service +
worker + queue, inserts into `platform_health_samples`.

## 3. Files this change modifies

### Backend
- `migrations/042_platform.sql` (NEW) — all Tier 11 tables
- `cmd/api-gateway/routes_platform.go` (NEW) — mountPlatformRoutes
- `cmd/api-gateway/main.go` — wire 6 new workers
- 30+ new handler files under `internal/handler/handlers_platform_*.go`
- 6 new worker files under `internal/platform/` (new package)
- 2 new client files: `internal/client/stripe/` + `internal/client/resend/`

### Frontend
- `web/src/pages/AdminPage.tsx` (extend with 8 tabs)
- `web/src/pages/SettingsPage.tsx` (NEW) — billing + limits UI
- `web/src/components/admin/DeploySection.tsx` (NEW)
- `web/src/components/admin/UsageSection.tsx` (NEW)
- `web/src/components/admin/SignupSection.tsx` (NEW)
- `web/src/components/admin/LimitsSection.tsx` (NEW)
- `web/src/components/admin/BackupSection.tsx` (NEW)
- `web/src/components/admin/RegionsSection.tsx` (NEW)
- `web/src/components/admin/RateLimitSection.tsx` (NEW)
- `web/src/components/admin/HealthSection.tsx` (NEW)
- `web/src/components/admin/SettingsSection.tsx` (NEW)
- `web/src/components/PublicStatusPage.tsx` (NEW) — public status page

### Build / Deploy
- `Dockerfile` (NEW) — multi-stage build for self-hosted
- `docker-compose.yml` (NEW) — postgres + api-gateway
- `scripts/install.sh` (NEW) — one-liner installer for self-hosted
- `.env.example` (extend) — STRIPE_*, RAZORPAY_*, RESEND_*, S3_*, etc.
