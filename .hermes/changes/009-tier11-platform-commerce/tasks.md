# Tier 11 Tasks — Platform & Commerce

## Phase 0 — Routes split (foundation)

- [ ] 0.1 Create `cmd/api-gateway/routes_platform.go` with stub `mountPlatformRoutes(protected *gin.RouterGroup, pool *db.Pool)`
- [ ] 0.2 Modify `cmd/api-gateway/routes.go` to call `mountPlatformRoutes(protected, pool)` after the existing `mountHomelabRoutes(protected, pool)` call
- [ ] 0.3 Verify `go build ./cmd/api-gateway` exits 0
- [ ] 0.4 Commit: `feat(tier11): routes split (Phase 0)`

## Phase 1 — Push-Button Deploy (PL1)

- [ ] 1.1 Migration `042_platform.sql`: 1 table (`platform_deploy_tokens`) + 1 ALTER on users (`is_platform_admin`)
- [ ] 1.2 Apply migration to prod
- [ ] 1.3 `internal/handler/handlers_platform_deploy.go` (~300 LOC, 4 routes):
  - POST /platform/deploy/install-token
  - GET /platform/deploy/install.sh
  - GET /platform/deploy/status
  - POST /platform/deploy/mode
- [ ] 1.4 `internal/handler/handlers_platform_types.go` (~100 LOC — shared types)
- [ ] 1.5 Register 4 routes in `mountPlatformRoutes`
- [ ] 1.6 Verify + deploy + commit Phase 1

## Phase 2 — Usage Metering (PL2)

- [ ] 2.1 Migration adds `platform_usage_events`, `platform_usage_aggregates`
- [ ] 2.2 Apply migration to prod
- [ ] 2.3 `internal/handler/handlers_platform_usage.go` (~300 LOC, 5 routes):
  - POST /platform/usage/event
  - GET /platform/usage/current
  - GET /platform/usage/history
  - GET /platform/usage/summary (super_admin)
  - GET /platform/usage/export
- [ ] 2.4 `internal/platform/usage_meter.go` (~250 LOC — hourly aggregation worker)
- [ ] 2.5 Register worker to start in main.go
- [ ] 2.6 `web/src/components/platform/MeteringSection.tsx` (~200 LOC)
- [ ] 2.7 Register 5 routes in `mountPlatformRoutes`
- [ ] 2.8 Verify + deploy + commit Phase 2

## Phase 3 — Self-Service Signup (PL3)

- [ ] 3.1 Migration adds `platform_signups`
- [ ] 3.2 Apply migration to prod
- [ ] 3.3 `internal/handler/handlers_platform_signup.go` (~300 LOC, 4 routes):
  - POST /platform/signup (rate-limited 10/day/IP)
  - POST /platform/signup/verify
  - POST /platform/signup/resend
  - GET /platform/signup/check-email
- [ ] 3.4 `scripts/install.sh` (~150 LOC, curl-able script)
- [ ] 3.5 `web/src/components/platform/SignupSection.tsx` (~150 LOC)
- [ ] 3.6 Register 4 routes + deploy + commit Phase 3

## Phase 4 — Tenant Limits (PL4)

- [ ] 4.1 Migration adds `platform_tenant_limits`, `platform_plan_definitions`
- [ ] 4.2 Apply migration to prod
- [ ] 4.3 `internal/handler/handlers_platform_limits.go` (~300 LOC, 5 routes):
  - GET /platform/limits/definitions
  - GET /platform/limits/me
  - PATCH /platform/limits/me (super_admin)
  - POST /platform/limits/check (dry-run)
  - GET /platform/limits/usage (billing warnings)
- [ ] 4.4 `internal/platform/retention.go` (~250 LOC — daily retention worker)
- [ ] 4.5 Register worker to start in main.go
- [ ] 4.6 `web/src/components/platform/LimitsSection.tsx` (~200 LOC)
- [ ] 4.7 Register 5 routes + deploy + commit Phase 4

## Phase 5 — Backup/Restore (PL5)

- [ ] 5.1 Migration adds `platform_backups`, `platform_backup_jobs`
- [ ] 5.2 Apply migration to prod
- [ ] 5.3 `internal/handler/handlers_platform_backup.go` (~300 LOC, 5 routes):
  - POST /platform/backup/create
  - GET /platform/backup/list
  - GET /platform/backup/download
  - POST /platform/backup/restore
  - GET /platform/backup/schedule
- [ ] 5.4 `internal/handler/handlers_platform_backup_crypto.go` (~150 LOC — AES-256-GCM)
- [ ] 5.5 `internal/handler/handlers_platform_backup_restore.go` (~200 LOC — pg_restore)
- [ ] 5.6 `internal/platform/backup_scheduler.go` (~250 LOC, scheduled backup worker)
- [ ] 5.7 Register worker to start in main.go
- [ ] 5.8 `web/src/components/platform/BackupSection.tsx` (~250 LOC)
- [ ] 5.9 Register 5 routes + deploy + commit Phase 5

## Phase 6 — Multi-Region/HA (PL6)

- [ ] 6.1 Migration adds `platform_regions`, `platform_region_replicas`
- [ ] 6.2 Apply migration to prod
- [ ] 6.3 `internal/handler/handlers_platform_regions.go` (~250 LOC, 3 routes):
  - GET /platform/regions
  - POST /platform/regions (cloud mode only)
  - GET /platform/regions/health
- [ ] 6.4 `web/src/components/platform/RegionsSection.tsx` (~150 LOC)
- [ ] 6.5 Register 3 routes + deploy + commit Phase 6

## Phase 7 — Rate Limiting (PL7)

- [ ] 7.1 `internal/handler/handlers_platform_ratelimit.go` (~250 LOC, 3 routes):
  - GET /platform/ratelimit/me
  - PATCH /platform/ratelimit/global (super_admin)
  - GET /platform/ratelimit/blocked (super_admin)
- [ ] 7.2 `internal/platform/rate_limiter.go` (~300 LOC — token-bucket + tier-based limits)
- [ ] 7.3 `internal/middleware/ratelimit.go` (~100 LOC — Gin middleware)
- [ ] 7.4 Wire middleware into router for /api/v1/* (after auth, before handler)
- [ ] 7.5 `web/src/components/platform/RateLimitSection.tsx` (~150 LOC)
- [ ] 7.6 Register 3 routes + deploy + commit Phase 7

## Phase 8 — Platform Health (PL8) + Finalize (TIER 11 COMPLETE)

- [ ] 8.1 Migration adds `platform_health_snapshots`
- [ ] 2.2 Apply migration to prod
- [ ] 8.3 `internal/handler/handlers_platform_health.go` (~300 LOC, 5 routes):
  - GET /platform/health/summary
  - GET /platform/health/regions
  - GET /platform/health/tenants/top
  - GET /platform/health/capacity/forecast
  - GET /platform/health/alerts
- [ ] 8.4 `internal/platform/capacity_forecast.go` (~250 LOC — daily forecast worker)
- [ ] 8.5 Register worker to start in main.go
- [ ] 8.6 `web/src/components/platform/HealthSection.tsx` (~250 LOC)
- [ ] 8.7 `web/src/pages/PlatformPage.tsx` (~200 LOC, 6-tab unified dashboard)
- [ ] 8.8 AppSidebar: "Platform" link (super_admin only)
- [ ] 8.9 All 34 routes verified live with real auth
- [ ] 8.10 Cross-tier regression check (Tier 0-10 still pass)
- [ ] 8.11 `journal-tier11-final.md` written
- [ ] 8.12 Archive `.hermes/changes/009-tier11-platform-commerce/`
- [ ] 8.13 Final commit: `docs(tier11): TIER 11 COMPLETE marker + archive + journal`

## Out-of-scope (Tier 12+)

- Public marketing site — Tier 12
- Razorpay webhook HMAC validation — Tier 12
- Email service production config (Resend/SMTP) — Tier 12
- Mobile app — Tier 13
