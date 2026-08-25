# Tier 11 Plan — Platform & Commerce

## Phase 0 — Routes split + foundation (mandatory before Phase 1)

Create `cmd/api-gateway/routes_platform.go` (~70 LOC) with stub
`mountPlatformRoutes(public, protected, admin *gin.RouterGroup, pool *db.Pool)`.

Why a 3-group signature: PL1/PL3/PL8 need public endpoints (signup,
install script, public status), PL2/PL4/PL5/PL7 need protected
(tenant scope), PL5/PL6/PL8 need admin (super-admin scope).

After Phase 0: `routes_protected.go` unchanged at 396 LOC, but a new
`routes_admin.go` (extract admin routes from protected) needed if any
admin route count >5.

## Phase 1 — Push-Button Deploy (PL1)

**Tables (1 new):**
- `deploy_install_tokens` (id, tenant_id, token_hash, expires_at, used_at)

**Routes (3 protected + 1 public):**
- POST /api/v1/platform/deploy/install-token (protected)
- GET /api/v1/platform/deploy/install-script (public)
- GET /api/v1/platform/deploy/stats (protected)

**Files (~12):**
- `internal/handler/handlers_platform_deploy.go` (~250 LOC, 3 routes)
- `internal/handler/handlers_platform_deploy_types.go` (~80 LOC)
- `internal/platform/deploy.go` (~150 LOC, token generator + validator)
- `web/src/components/admin/DeploySection.tsx` (~180 LOC)
- Plus 1-line wiring in routes_platform.go

**Tables:** +1 (1 total)

## Phase 2 — Usage Metering (PL2)

**Tables (2 new):**
- `usage_events` (id, tenant_id, event_type, quantity, ts)
- `usage_daily_rollups` (tenant_id, day, event_type, total_quantity)

**Routes (2 protected + 1 admin):**
- POST /api/v1/usage/events (protected, internal)
- GET /api/v1/usage/summary (protected, tenant scope)
- GET /api/v1/admin/usage/all (admin, super-admin scope)

**Files (~10):**
- `internal/handler/handlers_platform_usage.go` (~250 LOC, 3 routes)
- `internal/handler/handlers_platform_usage_types.go` (~80 LOC)
- `internal/platform/usage.go` (~150 LOC, meter helper + rollup worker)
- `internal/platform/usage_rollup.go` (~200 LOC, hourly rollup worker)
- `web/src/components/admin/UsageSection.tsx` (~180 LOC)

**Tables:** +2 (3 total)

## Phase 3 — Self-Service Signup + Email (PL3)

**No new tables** — extends existing users + tenants.

**Routes (3 public):**
- POST /api/v1/public/signup (already exists, extend with captcha + IP rate limit + email verification)
- POST /api/v1/public/verify-email (new)
- POST /api/v1/public/resend-verification (new)

**Files (~15):**
- `internal/client/resend/client.go` (~150 LOC, Resend API client)
- `internal/client/resend/dev.go` (~50 LOC, dev-mode logger)
- `internal/platform/signup.go` (~200 LOC, signup flow)
- `internal/platform/email.go` (~200 LOC, email templates)
- `internal/handler/handlers_platform_signup.go` (~200 LOC, 3 routes)
- `internal/handler/handlers_platform_signup_types.go` (~80 LOC)
- `internal/platform/ratelimit.go` (~200 LOC, in-process rate limiter for signup)
- `web/src/components/admin/SignupSection.tsx` (~180 LOC)
- Plus wiring

**Tables:** +0 (3 total)

## Phase 4 — Tenant Limits (PL4)

**Tables (1 new):**
- `plan_limits` (plan, limits_jsonb, updated_at)

**Routes (2 admin):**
- GET /api/v1/admin/plans
- PATCH /api/v1/admin/plans/:plan

**Files (~12):**
- `internal/handler/handlers_platform_limits.go` (~200 LOC, 2 routes)
- `internal/handler/handlers_platform_limits_types.go` (~80 LOC)
- `internal/platform/limits.go` (~200 LOC, enforceLimit middleware)
- `internal/platform/limits_seed.go` (~100 LOC, default plans)
- `web/src/components/admin/LimitsSection.tsx` (~180 LOC)

**Tables:** +1 (4 total)

## Phase 5 — Backup/Restore (PL5)

**Tables (1 new):**
- `platform_backups` (id, type, started_at, finished_at, status, size_bytes, path, checksum, error_message)

**Routes (3 admin):**
- GET /api/v1/admin/backups
- POST /api/v1/admin/backups (manual trigger)
- POST /api/v1/admin/backups/:id/restore

**Files (~10):**
- `internal/handler/handlers_platform_backup.go` (~250 LOC, 3 routes)
- `internal/handler/handlers_platform_backup_types.go` (~80 LOC)
- `internal/platform/backup.go` (~250 LOC, pg_dump wrapper + restore)
- `internal/platform/backup_worker.go` (~200 LOC, daily scheduler)
- `web/src/components/admin/BackupSection.tsx` (~200 LOC)

**Tables:** +1 (5 total)

## Phase 6 — Multi-Region / HA (PL6)

**Tables (2 new):**
- `platform_regions` (id, name, kind, primary_url, replica_url, is_active)
- `platform_replicas` (id, region_id, lag_seconds, last_synced_at, status)

**Routes (3 admin):**
- GET /api/v1/admin/regions
- POST /api/v1/admin/regions
- GET /api/v1/admin/replicas/status

**Files (~10):**
- `internal/handler/handlers_platform_regions.go` (~200 LOC, 3 routes)
- `internal/handler/handlers_platform_regions_types.go` (~80 LOC)
- `internal/platform/regions.go` (~200 LOC, region config + replicator worker)
- `web/src/components/admin/RegionsSection.tsx` (~180 LOC)

**Tables:** +2 (7 total)

## Phase 7 — Rate Limiting (PL7)

**No new tables** (in-memory state only).

**Files (~8):**
- `internal/platform/ratelimit.go` (extended, was created in Phase 3)
- `internal/platform/ratelimit_middleware.go` (~250 LOC, Gin middleware)
- `internal/platform/ratelimit_cleanup.go` (~150 LOC, hourly cleanup worker)
- `web/src/components/admin/RateLimitSection.tsx` (~200 LOC)

**Tables:** +0 (7 total)

## Phase 8 — Platform Health (PL8)

**Tables (1 new):**
- `platform_health_samples` (id, component, kind, status, latency_ms, sampled_at)

**Routes (3 admin + 1 public):**
- GET /api/v1/admin/health/services
- GET /api/v1/admin/health/workers
- GET /api/v1/admin/health/queues
- GET /api/v1/public/status

**Files (~10):**
- `internal/handler/handlers_platform_health.go` (~250 LOC, 4 routes)
- `internal/handler/handlers_platform_health_types.go` (~80 LOC)
- `internal/platform/health.go` (~200 LOC, sampling worker)
- `internal/platform/health_collectors.go` (~250 LOC, per-component collectors)
- `web/src/components/admin/HealthSection.tsx` (~200 LOC)
- `web/src/components/PublicStatusPage.tsx` (~150 LOC)

**Tables:** +1 (8 total)

## Phase 9 — Settings + Admin Page + Finalize

**No new tables.**

**Files:**
- `web/src/pages/AdminPage.tsx` (extend with 8 tabs — add: Deploy, Usage, Signup, Limits, Backup, Regions, RateLimit, Health)
- `web/src/pages/SettingsPage.tsx` (NEW, billing + limits UI for tenant)
- `web/src/components/admin/SettingsSection.tsx` (NEW, billing sub-tab)
- `web/src/App.tsx` (add `/admin` + `/settings` routes)
- `web/src/components/AppSidebar.tsx` (add Settings link + Admin link)
- `journal-tier11-final.md`
- Archive speckit change

## Total (estimated)

- **8 DB tables**
- **~25-30 routes** (mix of public/protected/admin)
- **~70 files** (all < 400 LOC)
- **6 background workers**
- **2 new SDK clients**: Resend email + Stripe
- **1 Dockerfile + docker-compose.yml** (for self-hosted install)
- **.env.example extended** with 12 new env vars

## Risks

1. **External service integration** (Stripe, Resend) — implement
   dev-mode stubs that log instead of send.
2. **Signup abuse** — captcha + per-IP rate limit + email verification
   + audit log on signup attempts.
3. **Backup/restore** on a live DB — use `pg_dump` with `--format=custom`
   so restore is incremental + parallel-safe.
4. **Multi-region HA** is hard — v1 ships read-only replicas + manual
   failover. Auto-failover is v2.
5. **Rate limiting** must be in-memory (sync.Map) AND backed by DB
   for cluster mode (Phase 6 future).

## Verifier

`hermes-verify-tier11-final.py` — checks every route + every plan
limit + every worker + every external client dev-mode.

## Out of Scope

- Marketing site (Tier 12)
- Docs (Tier 12)
- Mobile (Tier 13)
- Auto-failover (future v2)
