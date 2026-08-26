# Tier 11 Plan — Platform & Commerce

## Phase 0 — Routes split (mandatory before Phase 1)

Why: `routes_homelab.go` is at 345 LOC; adding 45+ more Tier 11 routes
would push it over 400. Extract `routes_platform.go` (mountPlatformRoutes)
following the Tier 9 + Tier 10 Phase 0 pattern.

**Outcome:** `routes_platform.go` placeholder created, `routes_protected.go`
untouched, `routes_homelab.go` unchanged.

**Files:** 1 new, 1 modified (cmd/api-gateway/routes.go to call mountPlatformRoutes).

## Phase 1 — Push-Button Deploy (PL1)

**Files (estimated):**
- `internal/handler/handlers_platform_deploy.go` (~300 LOC, 4 routes)
- `internal/handler/handlers_platform_types.go` (~100 LOC — shared types)
- `migrations/042_platform.sql` (initial 1 table + 1 ALTER on users)
- `cmd/api-gateway/routes_platform.go` (~50 LOC stub + 4 routes)

**Tables:** 1 + `is_platform_admin BOOLEAN` column on users
**Routes:** 4

## Phase 2 — Usage Metering (PL2)

**Files:**
- `internal/handler/handlers_platform_usage.go` (~300 LOC, 5 routes)
- `internal/platform/usage_meter.go` (~250 LOC, hourly aggregation worker)
- `web/src/components/platform/MeteringSection.tsx` (~200 LOC)

**Tables:** +2 (3 total)
**Routes:** +5 (9 total)

## Phase 3 — Self-Service Signup (PL3)

**Files:**
- `internal/handler/handlers_platform_signup.go` (~300 LOC, 4 routes)
- `web/src/components/platform/SignupSection.tsx` (~150 LOC)
- Install script at `scripts/install.sh` (~150 LOC, the curl-able file)

**Tables:** +1 (4 total)
**Routes:** +4 (13 total)

## Phase 4 — Tenant Limits (PL4)

**Files:**
- `internal/handler/handlers_platform_limits.go` (~300 LOC, 5 routes)
- `internal/platform/retention.go` (~250 LOC, daily retention worker)
- `web/src/components/platform/LimitsSection.tsx` (~200 LOC)

**Tables:** +2 (6 total)
**Routes:** +5 (18 total)

## Phase 5 — Backup/Restore (PL5)

**Files:**
- `internal/handler/handlers_platform_backup.go` (~300 LOC, 5 routes)
- `internal/handler/handlers_platform_backup_crypto.go` (~150 LOC — AES-256-GCM)
- `internal/handler/handlers_platform_backup_restore.go` (~200 LOC — pg_restore)
- `internal/platform/backup_scheduler.go` (~250 LOC, scheduled backup worker)
- `web/src/components/platform/BackupSection.tsx` (~250 LOC)

**Tables:** +2 (8 total)
**Routes:** +5 (23 total)

## Phase 6 — Multi-Region/HA (PL6)

**Files:**
- `internal/handler/handlers_platform_regions.go` (~250 LOC, 3 routes)
- `web/src/components/platform/RegionsSection.tsx` (~150 LOC)

**Tables:** +2 (10 total)
**Routes:** +3 (26 total)

## Phase 7 — Rate Limiting (PL7)

**Files:**
- `internal/handler/handlers_platform_ratelimit.go` (~250 LOC, 3 routes)
- `internal/platform/rate_limiter.go` (~300 LOC — token-bucket + tier-based limits)
- `internal/middleware/ratelimit.go` (~100 LOC — Gin middleware)
- `web/src/components/platform/RateLimitSection.tsx` (~150 LOC)

**Tables:** +0 (10 total — in-memory bucket only)
**Routes:** +3 (29 total)

## Phase 8 — Platform Health (PL8) + Finalize (TIER 11 COMPLETE)

**Files:**
- `internal/handler/handlers_platform_health.go` (~300 LOC, 5 routes)
- `internal/platform/capacity_forecast.go` (~250 LOC, daily forecast worker)
- `web/src/components/platform/HealthSection.tsx` (~250 LOC)
- `web/src/pages/PlatformPage.tsx` (~200 LOC, 6-tab unified dashboard)
- AppSidebar: "Platform" link (super_admin only)
- `journal-tier11-final.md`
- Archive `.hermes/changes/009-tier11-platform-commerce/`
- Final commit: `docs(tier11): TIER 11 COMPLETE marker + archive + journal`

**Tables:** +1 (11 total)
**Routes:** +5 (34 total)

## Total (estimated)

- **11 DB tables** (1 new column on users)
- **34 routes** (revised down from 45 — merging similar endpoints)
- **30+ files** (all <400 LOC)
- **6 sections** + 1 unified page
- **5 background workers** (usage_meter, retention, backup_scheduler, rate_limiter, capacity_forecast)
- **No new deps** (stdlib only)

## Risks

1. **Self-service signup opens abuse vector** — mitigation: email
   verification + signup rate limit (10/day per IP) + CAPTCHA (out of scope)
2. **Multi-region replication is complex** — start with active-passive
   only (cloud mode); document self-host replica setup for Tier 12.
3. **Backup encryption** — use AES-256-GCM with tenant-derived key.
4. **Capacity forecast must be cheap** — simple linear regression on raw
   metrics; no ML models.
5. **Rate limiter must not lock out legit users** — soft 429 with
   `Retry-After`, no hard cutoffs.
6. **Platform admin role separation** — `is_platform_admin` column on
   users, distinct from `super_admin` (tenant-scoped) to prevent
   privilege escalation bugs.

## Verifier

`hermes-verify-tier11-final.py` — checks every route + every widget
renders + every worker started without panic + install.sh downloads
+ signup flow + backup round-trip.
