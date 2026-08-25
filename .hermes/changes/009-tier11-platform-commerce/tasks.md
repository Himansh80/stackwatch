# Tier 11 Tasks — Platform & Commerce

## Phase 0 — Routes split (foundation)

- [ ] 0.1 Create `cmd/api-gateway/routes_platform.go` (~70 LOC) with stub `mountPlatformRoutes(public, protected, admin *gin.RouterGroup, pool *db.Pool)`
- [ ] 0.2 Modify `cmd/api-gateway/routes.go` to call `mountPlatformRoutes` after `mountHomelabRoutes`. Wire public + protected + admin groups.
- [ ] 0.3 Verify `go build ./cmd/api-gateway` exits 0
- [ ] 0.4 Commit: `feat(tier11): routes split (Phase 0)`

## Phase 1 — Push-Button Deploy (PL1)

- [ ] 1.1 Migration `042_platform.sql`: table `deploy_install_tokens`
- [ ] 1.2 Apply migration to prod
- [ ] 1.3 `internal/handler/handlers_platform_deploy.go` (~250 LOC, 3 protected routes)
- [ ] 1.4 `internal/handler/handlers_platform_deploy_types.go` (~80 LOC)
- [ ] 1.5 `internal/platform/deploy.go` (~150 LOC) — token generator + validator
- [ ] 1.6 Register 3 routes in `mountPlatformRoutes`
- [ ] 1.7 `web/src/components/admin/DeploySection.tsx` (~180 LOC) — curl snippet + copy + OS tabs
- [ ] 1.8 Verify gates + deploy + commit

## Phase 2 — Usage Metering (PL2)

- [ ] 2.1 Migration adds `usage_events` + `usage_daily_rollups`
- [ ] 2.2 Apply migration to prod
- [ ] 2.3 `internal/handler/handlers_platform_usage.go` (~250 LOC, 3 routes)
- [ ] 2.4 `internal/handler/handlers_platform_usage_types.go` (~80 LOC)
- [ ] 2.5 `internal/platform/usage.go` (~150 LOC) — meter helper
- [ ] 2.6 `internal/platform/usage_rollup.go` (~200 LOC) — hourly rollup worker
- [ ] 2.7 `web/src/components/admin/UsageSection.tsx` (~180 LOC) — Datadog charts + per-event-type breakdown
- [ ] 2.8 Register 3 routes
- [ ] 2.9 Verify + deploy + commit

## Phase 3 — Self-Service Signup + Email (PL3)

- [ ] 3.1 `internal/client/resend/client.go` (~150 LOC) — Resend API client
- [ ] 3.2 `internal/client/resend/dev.go` (~50 LOC) — dev-mode logger
- [ ] 3.3 `internal/platform/email.go` (~200 LOC) — email templates (verification + welcome + billing)
- [ ] 3.4 `internal/platform/signup.go` (~200 LOC) — signup flow + verification
- [ ] 3.5 `internal/handler/handlers_platform_signup.go` (~200 LOC, 3 public routes)
- [ ] 3.6 `internal/handler/handlers_platform_signup_types.go` (~80 LOC)
- [ ] 3.7 `internal/platform/ratelimit_signup.go` (~150 LOC) — per-IP signup rate limit
- [ ] 3.8 Register 3 public routes
- [ ] 3.9 `web/src/components/admin/SignupSection.tsx` (~180 LOC) — signup config UI
- [ ] 3.10 Verify + deploy + commit

## Phase 4 — Tenant Limits (PL4)

- [ ] 4.1 Migration adds `plan_limits`
- [ ] 4.2 Apply migration to prod
- [ ] 4.3 `internal/handler/handlers_platform_limits.go` (~200 LOC, 2 admin routes)
- [ ] 4.4 `internal/handler/handlers_platform_limits_types.go` (~80 LOC)
- [ ] 4.5 `internal/platform/limits.go` (~200 LOC) — enforceLimit middleware
- [ ] 4.6 `internal/platform/limits_seed.go` (~100 LOC) — default plan limits (free/pro/business/enterprise)
- [ ] 4.7 Register 2 routes
- [ ] 4.8 `web/src/components/admin/LimitsSection.tsx` (~180 LOC) — plan editor + tier comparison
- [ ] 4.9 Verify + deploy + commit

## Phase 5 — Backup/Restore (PL5)

- [ ] 5.1 Migration adds `platform_backups`
- [ ] 5.2 Apply migration to prod
- [ ] 5.3 `internal/handler/handlers_platform_backup.go` (~250 LOC, 3 admin routes)
- [ ] 5.4 `internal/handler/handlers_platform_backup_types.go` (~80 LOC)
- [ ] 5.5 `internal/platform/backup.go` (~250 LOC) — pg_dump wrapper + restore
- [ ] 5.6 `internal/platform/backup_worker.go` (~200 LOC) — daily scheduler
- [ ] 5.7 Register 3 routes
- [ ] 5.8 `web/src/components/admin/BackupSection.tsx` (~200 LOC)
- [ ] 5.9 Verify + deploy + commit

## Phase 6 — Multi-Region / HA (PL6)

- [ ] 6.1 Migration adds `platform_regions` + `platform_replicas`
- [ ] 6.2 Apply migration to prod
- [ ] 6.3 `internal/handler/handlers_platform_regions.go` (~200 LOC, 3 admin routes)
- [ ] 6.4 `internal/handler/handlers_platform_regions_types.go` (~80 LOC)
- [ ] 6.5 `internal/platform/regions.go` (~200 LOC) — region config + replicator worker
- [ ] 6.6 Register 3 routes
- [ ] 6.7 `web/src/components/admin/RegionsSection.tsx` (~180 LOC)
- [ ] 6.8 Verify + deploy + commit

## Phase 7 — Rate Limiting (PL7)

- [ ] 7.1 REFACTOR existing `internal/platform/ratelimit_signup.go` → `internal/platform/ratelimit.go` (general)
- [ ] 7.2 `internal/platform/ratelimit_middleware.go` (~250 LOC) — Gin middleware for all routes
- [ ] 7.3 `internal/platform/ratelimit_cleanup.go` (~150 LOC) — hourly cleanup worker
- [ ] 7.4 Wire middleware into `mountPlatformRoutes` (skip auth-required routes from per-IP limit, only per-tenant)
- [ ] 7.5 `web/src/components/admin/RateLimitSection.tsx` (~200 LOC) — current usage + config
- [ ] 7.6 Verify + deploy + commit

## Phase 8 — Platform Health (PL8)

- [ ] 8.1 Migration adds `platform_health_samples`
- [ ] 8.2 Apply migration to prod
- [ ] 8.3 `internal/handler/handlers_platform_health.go` (~250 LOC, 4 routes)
- [ ] 8.4 `internal/handler/handlers_platform_health_types.go` (~80 LOC)
- [ ] 8.5 `internal/platform/health.go` (~200 LOC) — sampling worker
- [ ] 8.6 `internal/platform/health_collectors.go` (~250 LOC) — per-component collectors
- [ ] 8.7 Register 4 routes (3 admin + 1 public)
- [ ] 8.8 `web/src/components/admin/HealthSection.tsx` (~200 LOC) — operator dashboard
- [ ] 8.9 `web/src/components/PublicStatusPage.tsx` (~150 LOC) — minimal status page
- [ ] 8.10 Verify + deploy + commit

## Phase 9 — Settings + Admin Page + Finalize (TIER 11 COMPLETE)

- [ ] 9.1 Extend `web/src/pages/AdminPage.tsx` with 8 tabs
- [ ] 9.2 `web/src/pages/SettingsPage.tsx` (NEW, billing + limits UI)
- [ ] 9.3 `web/src/components/admin/SettingsSection.tsx` (NEW, billing sub-tab)
- [ ] 9.4 `web/src/App.tsx` — add `/admin` + `/settings` routes
- [ ] 9.5 `web/src/components/AppSidebar.tsx` — add Settings + Admin links
- [ ] 9.6 Verify all 25+ Tier 11 routes live + cross-tier regression-free
- [ ] 9.7 `journal-tier11-final.md` written
- [ ] 9.8 Archive `.hermes/changes/009-tier11-platform-commerce/`
- [ ] 9.9 Final commit: `docs(tier11): TIER 11 COMPLETE marker + archive + journal`

## Out-of-scope (Tier 12+)

- Marketing site (Tier 12.4)
- Docs (Tier 12.1-12.3)
- Mobile app (Tier 13)
- Auto-failover (v2)
