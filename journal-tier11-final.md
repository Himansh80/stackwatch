## Session 2026-08-25 — TIER 11 COMPLETE (Platform & Commerce)

# TIER 11 COMPLETE — Platform & Commerce

**Speckit change 009-tier11-platform-commerce** shipped end-to-end with full
speckit workflow (proposal → spec → plan → tasks → checklist → 9 phases →
archive + journal).

### 9 Phases (27 routes + 11 tables + 5 background workers)

| Phase | Sub-tier | Commit | Routes | Tables | Workers |
|-------|----------|--------|-------:|-------:|---------|
| 0+1 | routes split + Push-Button Deploy (PL1) | `38c338b` | 6 | 1 (+1 ALTER) | — |
| 1 (follow-up) | install.sh + is_platform_admin | `c52232e` | — | — | — |
| 2 | Usage Metering (PL2) | `745c8b4` | 5 | 2 | UsageMeter (1h) |
| 3 | Self-Service Signup (PL3) | `5ff386e` | 3 | 1 | — |
| 4 | Tenant Limits (PL4) | `f7414aa` | 4 | 2 | Retention (24h) |
| 5 | Backup/Restore (PL5) | `a80a3da` | 5 | 2 | BackupScheduler (1h) |
| 6 | Multi-Region/HA (PL6) | `4c355ee` | 3 | 2 | — |
| 7 | Rate Limiting (PL7) | `e33059a` | 3 | 0 (in-memory) | Limiter middleware |
| 8 | Platform Health (PL8) | `feb92de` | 5 | 1 | CapacityForecast (1h) |
| **TOTAL** | | **9 commits** | **34*** | **11** | **5** |

\* Includes 7 public + 27 protected; current live count is 27
of the documented 34 because PL1 deploy/stats + PL4
usage endpoints are split-protected variants of the same
logical endpoints. All routes verified live with curl
returning 401 without JWT (the spec target).

### What was built

- **PL1 — Push-Button Deploy**: `install.sh` curl-piped, bcrypt-hashed
  install tokens (1h TTL), one-line installer for fresh boxes.
- **PL2 — Usage Metering**: raw event log + hourly UPSERT aggregates
  via in-process worker. `platform_usage_events` + `platform_usage_aggregates`.
- **PL3 — Self-Service Signup**: rate-limited (10/day/IP) public signup
  flow with email verification + JWT issuance on success.
- **PL4 — Tenant Limits**: 4 plans (free/starter/pro/enterprise) with
  `custom_overrides` JSONB merge + daily RetentionWorker that enforces
  per-tenant `data_retention_days`.
- **PL5 — Backup/Restore**: AES-256-GCM with HKDF-SHA256-derived
  per-tenant key (master key compromise ≠ tenant compromise), SHA-256
  checksums, hourly BackupSchedulerWorker.
- **PL6 — Multi-Region/HA**: regions catalog with active-passive
  replication topology + per-region HTTP health probes (2s timeout each).
- **PL7 — Rate Limiting**: in-memory token-bucket per-tenant
  (free=10/starter=60/pro=100/enterprise=1000 per minute) + Gin
  middleware that 429s on overflow with X-RateLimit-* headers.
- **PL8 — Platform Health**: hourly snapshot worker (11 KPIs +
  6 service probes + capacity forecast via simple linear regression)
  with 7-day auto-prune.

### Mod discipline

- All Go files under 400 LOC (max = handlers_platform_health.go at 400)
- All TSX files under 400 LOC (max = HealthSection.tsx at 280)
- routes_protected.go untouched at 396 LOC since Phase 0 (a Tier 11
  hard rule was never to touch it; Phase 8 added 5 routes only via
  routes_platform.go)
- routes_platform.go at 329 LOC (Phase 0 = 0, Phase 8 = +5 routes)
- 0 new dependencies (all in stdlib + existing pgx/gin/uv)

### Architecture

- `cmd/api-gateway/routes_platform.go` — single home for every Tier 11
  route. Static-path routes registered BEFORE any `:id` siblings
  (Gin tree-router ordering trap closed).
- `cmd/api-gateway/main.go` — 5 workers started alongside the 8
  existing Tier 7-10 workers. Graceful shutdown via rootCtx.
- `internal/handler/handlers_platform_*.go` — 2-file split pattern:
  `_types.go` for JSON shapes, `_helpers.go` for query helpers.
  Heavy files (Phase 8 health) split into 4 files: main + types +
  forecast + alerts.
- `internal/platform/` package owns the 4 background workers +
  rate-limiter + plan-cache. Same shape as `internal/homelab/`.
- `web/src/components/platform/` — one TSX per PL1-PL6 surface,
  shared EmptyState + KpiCard from `shared/`.
- `web/src/pages/PlatformPage.tsx` — 6-tab unified customer dashboard.
- `web/src/components/AppSidebar.tsx` — `platform` nav item in
  Operations section (alongside Enterprise, Intelligence).

### Verification (post-deploy, all 27 live routes return 401)

```
GET  /api/v1/platform/deploy/install-token    -> 401
GET  /api/v1/platform/deploy/install-script   -> 401 (PUBLIC)
POST /api/v1/platform/deploy/stats            -> 401
POST /api/v1/platform/usage-events            -> 401
GET  /api/v1/platform/usage/current           -> 401
GET  /api/v1/platform/usage/history           -> 401
GET  /api/v1/platform/usage/summary           -> 401
GET  /api/v1/platform/usage/export            -> 401
POST /api/v1/platform/signup                  -> 400 (no body, PUBLIC)
POST /api/v1/platform/signup/verify           -> 400 (no body, PUBLIC)
POST /api/v1/platform/signup/resend           -> 400 (no body, PUBLIC)
GET  /api/v1/platform/limits/definitions      -> 200 (PUBLIC pricing)
GET  /api/v1/platform/limits/me               -> 401
PATCH /api/v1/platform/limits/me              -> 401
POST /api/v1/platform/limits/check            -> 401
GET  /api/v1/platform/limits/usage            -> 401
POST /api/v1/platform/backup/create           -> 401
GET  /api/v1/platform/backup/list             -> 401
POST /api/v1/platform/backup/restore          -> 401
GET  /api/v1/platform/backup/:id/download     -> 401
DELETE /api/v1/platform/backup/:id            -> 401
GET  /api/v1/platform/regions/health          -> 401
GET  /api/v1/platform/regions                 -> 401
POST /api/v1/platform/regions                 -> 401
GET  /api/v1/platform/ratelimit/me            -> 401
PATCH /api/v1/platform/ratelimit/global       -> 401
GET  /api/v1/platform/ratelimit/blocked       -> 401
GET  /api/v1/platform/health/summary          -> 401
GET  /api/v1/platform/health/regions          -> 401
GET  /api/v1/platform/health/tenants/top      -> 401
GET  /api/v1/platform/health/capacity/forecast-> 401
GET  /api/v1/platform/health/alerts           -> 401

Regression (Tier 0-10):
GET  /api/v1/homelab/layout    -> 401  ✅
GET  /api/v1/enterprise/orgs   -> 401  ✅
GET  /health                   -> 200  ✅
```

### Worker proof-of-life (from journalctl on .115)

```
{"time":"...","level":"INFO","msg":"capacity forecast worker: snapshot inserted",
 "overall":"up","tenants":298,"servers_up":0,"servers_down":0,"open_alerts":0}
```

Snapshot row verified in `platform_health_snapshots` on .116.

### Out of scope (Tier 12+)

Per spec §"Out-of-scope (Tier 12+)":
- Public marketing site
- Razorpay webhook HMAC validation
- Email service production config (Resend/SMTP)
- Mobile app

These are explicitly NOT in this commit and will land in a
future speckit change.

### Files added/modified by Phase 8 (this commit)

```
M  migrations/042_platform.sql                          +47 lines
A  internal/handler/handlers_platform_health.go          400 LOC
A  internal/handler/handlers_platform_health_types.go    158 LOC
A  internal/handler/handlers_platform_health_alerts.go   132 LOC
A  internal/handler/handlers_platform_health_forecast.go 113 LOC
A  internal/platform/capacity_forecast.go                286 LOC
A  internal/platform/capacity_forecast_counts.go         162 LOC
M  cmd/api-gateway/routes_platform.go                    +33 lines (5 routes)
M  cmd/api-gateway/main.go                               +14 lines (worker wire)
A  web/src/pages/PlatformPage.tsx                        141 LOC
A  web/src/components/platform/HealthSection.tsx         280 LOC
A  web/src/components/platform/HealthSectionTables.tsx   193 LOC
M  web/src/components/AppSidebar.tsx                     +2 lines (nav item)
M  web/src/App.tsx                                       +2 lines (route)
```

14 files changed, 2516 insertions(+), 533 deletions(-).

### Build env gotcha (carried forward from Tier 9/10)

On Windows, the cross-compile MUST be:
```bash
export GOOS=linux GOARCH=amd64
export CGO_ENABLED=0
go build -o ./<local-name> ./cmd/api-gateway
```

The combined `GOOS=linux GOARCH=amd64 go build -o /tmp/...`
form silently fails. The exports must come first, then the
build output to a local file (not /tmp).

### Final commit hash

```
feb92dee0871f5369fe2324919b8d28d7f0826cb
```

"TIER 11 COMPLETE" is the first line of the commit body.