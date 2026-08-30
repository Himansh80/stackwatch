package main

import (
	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/handler"
	"github.com/stackwatch/platform/internal/platform"
)

// mountPlatformRoutes registers the Tier 11 — Platform & Commerce
// endpoints — speckit change 009-tier11-platform-commerce.
//
// Why a 3-group signature (public + protected + admin):
// PL1 / PL3 / PL8 need PUBLIC endpoints (install-script,
// signup/verify, public status), PL2 / PL4 / PL7 need PROTECTED
// (tenant-scoped), PL5 / PL6 / PL8 need ADMIN (super-admin scope).
// The 3-group split keeps `cmd/api-gateway/routes.go` clean and
// routes_platform.go focused on tier wiring while each PL1-PL8
// phase owns a separate `handlers_platform_*.go` file under
// `internal/handler/`.
//
// Phase 1-8 add their routes here in this order:
//
//	Phase 1: Push-Button Deploy (PL1)        ← 3 routes
//	Phase 2: Usage Metering (PL2)            ← 5 routes
//	Phase 3: Self-Service Signup (PL3)       ← 3 routes
//	Phase 4: Tenant Limits (PL4)             ← 4 protected routes
//	Phase 5: Backup/Restore (PL5)            ← 5 routes (added in this commit)
//	Phase 6: Multi-Region / HA (PL6)         ← 3 routes
//	Phase 7: Rate Limiting (PL7)             ← 3 routes (added in this commit)
//	Phase 8: Platform Health (PL8)           ← 5 routes (added in this commit)
//
// All handlers honor tenant_id from the JWT — no cross-tenant data
// ever crosses the wire. Idempotent migration migrations/042_platform.sql
// sets up the backing tables.
//
// PL1 (this commit) lays the foundation for the rest of the tier:
//   - protected POST /platform/deploy/install-token  (CreateInstallToken)
//   - public   GET  /platform/deploy/install-script (GetInstallScript)
//   - protected GET  /platform/deploy/stats         (GetInstallStats)
//
// The 3 PUBLIC PL1/PL3/PL8 routes live on the public engine via
// mountPlatformPublicRoutes (called from routes.go where the
// *gin.Engine is in scope) — the public install-script URL is meant
// to be hit by a freshly-installed Linux box with NO JWT.
//
// Phase 3 (this commit) adds the 3 PUBLIC PL3 routes — all sit on
// the public engine for the same reason (a freshly-typed email
// has no StackWatch credentials yet):
//   - public POST /platform/signup         (CreateSignup)
//   - public POST /platform/signup/verify  (VerifySignup — needs
//     the JWT issuer so it
//     can log the user in
//     immediately)
//   - public POST /platform/signup/resend  (ResendSignup)
func mountPlatformRoutes(public, protected, admin *gin.RouterGroup, pool *db.Pool, rateLimiter *platform.Limiter, ratePlans *platform.PlanCache) {
	// ---- Tier 11.1: Push-Button Deploy (Phase 1 — PL1) ----
	//
	// Per spec §"PL1 — Push-Button Deploy", these 3 endpoints
	// back the Datadog-style one-liner install surface. Tenants
	// mint a one-time-use install token (valid for 1h) via the
	// protected endpoint; the token is then baked into a bash
	// snippet that the PUBLIC endpoint returns to any caller —
	// the box has no JWT yet, so the script itself carries the
	// credential. Marking the token `used` on first GET ensures
	// the token can't be replayed from a different machine.
	//
	// Token storage (deploy_install_tokens) mirrors the SCIM
	// pattern from Tier 9.2: bcrypt-hashed at rest, plaintext
	// returned EXACTLY ONCE on POST, never persisted. Tokens
	// expire in 1h; the partial index on `expires_at WHERE
	// used_at IS NULL` keeps the lookup hot.
	//
	// Routes (2 protected + 1 public):
	//   POST /api/v1/platform/deploy/install-token  — CreateInstallToken
	//   GET  /api/v1/platform/deploy/install-script — GetInstallScript  (PUBLIC)
	//   GET  /api/v1/platform/deploy/stats          — GetInstallStats
	//
	// The PUBLIC GET /install-script is registered on the public
	// engine by mountPlatformPublicRoutes (routes.go) so it sits
	// OUTSIDE the RequireAuth middleware — same pattern Tier 9
	// uses for SCIM 2.0.
	protected.POST("/platform/deploy/install-token", handler.CreateInstallToken(pool))
	protected.GET("/platform/deploy/stats", handler.GetInstallStats(pool))

	// ---- Tier 11.2: Usage Metering (Phase 2 — PL2) ----
	//
	// Per spec §"PL2 — Usage Metering" these 5 endpoints back the
	// metering surface. Every handler honors tenant_id from the
	// JWT (handlers_platform_usage*.go). The cross-tenant
	// /summary endpoint additionally checks claims.Role ==
	// "super_admin" so org admins can't enumerate other tenants.
	//
	// Routes:
	//   POST /api/v1/platform/usage-events  — RecordUsageEvent
	//   GET  /api/v1/platform/usage/current — GetUsageCurrent
	//   GET  /api/v1/platform/usage/history — GetUsageHistory
	//   GET  /api/v1/platform/usage/summary — GetUsageSummary (super_admin)
	//   GET  /api/v1/platform/usage/export  — GetUsageExport
	protected.POST("/platform/usage-events", handler.RecordUsageEvent(pool))
	protected.GET("/platform/usage/current", handler.GetUsageCurrent(pool))
	protected.GET("/platform/usage/history", handler.GetUsageHistory(pool))
	protected.GET("/platform/usage/summary", handler.GetUsageSummary(pool))
	protected.GET("/platform/usage/export", handler.GetUsageExport(pool))

	// ---- Tier 11.4: Tenant Limits (Phase 4 — PL4) ----
	//
	// Per spec §"PL4 — Tenant Limits" these 4 protected
	// endpoints back the per-tenant plan + cap surface. Every
	// handler honors tenant_id from the JWT (handlers_platform_limits*.go).
	// PatchMyLimits additionally checks claims.Role ==
	// "super_admin" because plan changes have billing
	// implications — a regular admin can't upgrade themselves.
	// The 5th PL4 endpoint (GetLimitDefinitions) is PUBLIC
	// and registered on the public engine by
	// mountPlatformPublicRoutes — anyone can see pricing
	// without authenticating, which matches the pricing-page
	// widget pattern.
	//
	// Routes:
	//   GET   /api/v1/platform/limits/me     — GetMyLimits
	//   PATCH /api/v1/platform/limits/me     — PatchMyLimits  (super_admin only)
	//   POST  /api/v1/platform/limits/check  — CheckLimit     (dry-run)
	//   GET   /api/v1/platform/limits/usage  — GetLimitsUsage (KPI strip)
	//
	// Storage (platform_plan_definitions + platform_tenant_limits)
	// extends migrations/042_platform.sql with two new tables;
	// idempotent CREATE TABLE IF NOT EXISTS keeps re-applying
	// safe. The RetentionWorker (internal/platform/retention.go)
	// runs daily to enforce the data_retention_days cap on each
	// tenant — see main.go for its Start() wiring.
	protected.GET("/platform/limits/me", handler.GetMyLimits(pool))
	protected.PATCH("/platform/limits/me", handler.PatchMyLimits(pool))
	protected.POST("/platform/limits/check", handler.CheckLimit(pool))
	protected.GET("/platform/limits/usage", handler.GetLimitsUsage(pool))

	// ---- Tier 11.5: Backup / Restore (Phase 5 — PL5) ----
	//
	// Per spec §"PL5 — Backup/Restore". The 5 endpoints
	// back the per-tenant manual-backup surface. Every
	// handler enforces tenant_id isolation (claims.TenantID)
	// and the cross-tenant restore attack vector is
	// closed by checking row ownership BEFORE decrypting
	// or unpacking anything.
	//
	// Encryption posture (per proposal.md §"Risks" §"Backup
	// encryption"): every backup file is AES-256-GCM
	// sealed with a HKDF-SHA256-derived per-tenant key.
	// The decryption key is NEVER stored — only ciphertext
	// + nonce + tag land on disk. See
	// handlers_platform_backup_crypto.go for the layer;
	// platform/backup_helpers.go for the per-tenant
	// pipeline; platform/backup_scheduler.go for the
	// hourly scheduler worker.
	//
	// Why the router-order matters: static paths
	// (`/backup/create`, `/backup/list`, `/backup/restore`)
	// MUST be registered before `/backup/:id` siblings
	// so Gin's tree-router matches the literal suffix
	// first. Otherwise a stray `GET /backup/create` would
	// hit `:id="create"` and 400 on the UUID parser.
	//
	// Routes (5 protected):
	//   POST /api/v1/platform/backup/create        — CreateBackup
	//   GET  /api/v1/platform/backup/list          — ListBackups
	//   POST /api/v1/platform/backup/restore       — RestoreBackup  (multipart upload)
	//   GET  /api/v1/platform/backup/:id/download  — DownloadBackup (streamed ciphertext)
	//   DELETE /api/v1/platform/backup/:id         — DeleteBackup  (unlink file + row)
	protected.POST("/platform/backup/create", handler.CreateBackup(pool))
	protected.GET("/platform/backup/list", handler.ListBackups(pool))
	protected.POST("/platform/backup/restore", handler.RestoreBackup(pool))
	protected.GET("/platform/backup/:id/download", handler.DownloadBackup(pool))
	protected.DELETE("/platform/backup/:id", handler.DeleteBackup(pool))

	// ---- Tier 11.6: Multi-Region / HA (Phase 6 — PL6) ----
	//
	// Per spec §"PL6 — Multi-Region/HA". The 3 endpoints
	// back the platform-admin region catalog + health
	// surface. The catalog is PLATFORM-WIDE (no tenant
	// scoping) so every super_admin sees the same set of
	// regions. ListRegions + ProbeRegionsHealth require a
	// JWT; CreateRegion additionally gates on
	// claims.Role == "super_admin" because adding
	// infrastructure is an admin-only operation.
	//
	// Probe semantics: every POST /regions triggers a
	// fire-and-forget HTTP probe against the new region's
	// endpoint_url so the row appears in
	// /regions/health within ~1s. The /regions/health
	// endpoint itself probes every active region
	// sequentially (2s timeout each) and writes the
	// result to platform_regions.last_health_*. See
	// handlers_platform_regions.go + the probe helper
	// in handlers_platform_regions_probe.go.
	//
	// Why the router-order matters: `/regions/health`
	// is a STATIC path sibling of the dynamic
	// `/regions/:id` would-be-suffix; registering
	// `/regions/health` BEFORE any future `:id` handler
	// keeps Gin's tree-router matching the literal
	// suffix first. Today Phase 6 has no :id endpoint —
	// the explicit ordering is just future-proofing.
	//
	// Routes (2 protected + 1 super_admin):
	//   GET  /api/v1/platform/regions         — ListRegions
	//   POST /api/v1/platform/regions         — CreateRegion         (super_admin only)
	//   GET  /api/v1/platform/regions/health  — ProbeRegionsHealth
	protected.GET("/platform/regions/health", handler.ProbeRegionsHealth(pool))
	protected.GET("/platform/regions", handler.ListRegions(pool))
	protected.POST("/platform/regions", handler.CreateRegion(pool))

	// ---- Tier 11.7: Rate Limiting (Phase 7 — PL7) ----
	//
	// Per spec §"PL7 — Rate Limiting" these 3 endpoints
	// back the admin VIEW surface for the in-process
	// token-bucket limiter. The middleware that ENFORCES
	// the bucket on every authenticated request lives
	// in internal/middleware/ratelimit.go and is wired
	// in cmd/api-gateway/routes.go (after RequireAuth
	// so claims.TenantID is available, before the
	// handlers so a 429 short-circuits expensive work).
	//
	// /me is PROTECTED (every authenticated user can
	// see their own bucket). /global + /blocked are
	// PROTECTED + super_admin gated — a regular admin
	// can't see other tenants' usage or change the
	// global caps.
	//
	// Storage: NONE. PL7 is in-memory only (the Limiter
	// lives in internal/platform/ratelimit.go). A future
	// Redis-backed adapter is a drop-in replacement for
	// the Limiter interface; the handler shape doesn't
	// change.
	//
	// Routes (1 protected + 2 super_admin):
	//   GET  /api/v1/platform/ratelimit/me      — GetMyRateLimit
	//   PATCH /api/v1/platform/ratelimit/global  — UpdateGlobalLimits  (super_admin)
	//   GET  /api/v1/platform/ratelimit/blocked  — ListBlockedTenants  (super_admin)
	//
	// The mountPlatformRoutes signature carries the
	// *platform.Limiter + *platform.PlanCache through
	// from buildRouter — see main.go for the
	// construction site.
	protected.GET("/platform/ratelimit/me", handler.GetMyRateLimit(rateLimiter, ratePlans))
	protected.PATCH("/platform/ratelimit/global", handler.UpdateGlobalLimits(rateLimiter, ratePlans))
	protected.GET("/platform/ratelimit/blocked", handler.ListBlockedTenants(rateLimiter))

	// ---- Tier 11.8: Platform Health (Phase 8 — PL8) ----
	//
	// Per spec §"PL8 — Platform Health" these 5 endpoints
	// back the operator (super_admin) dashboard. Every
	// handler is PROTECTED + super_admin gated (a regular
	// admin can't see platform-wide health — that would leak
	// other tenants' infrastructure state). The cached
	// surface itself lives in platform_health_snapshots,
	// populated hourly by the CapacityForecastWorker
	// (internal/platform/capacity_forecast.go) wired in
	// main.go alongside UsageMeter / Retention /
	// BackupScheduler.
	//
	// Why the router-order matters: the 5 static paths
	// below MUST be registered BEFORE any future
	// /platform/health/:id sibling so Gin's tree-router
	// matches the literal suffix first. Today Phase 8 has
	// no :id endpoint — the explicit ordering is just
	// future-proofing.
	//
	// Routes (5 protected + super_admin):
	//   GET /api/v1/platform/health/summary            — HealthSummary
	//   GET /api/v1/platform/health/regions            — HealthRegionsSummary
	//   GET /api/v1/platform/health/tenants/top        — HealthTopTenants
	//   GET /api/v1/platform/health/capacity/forecast  — HealthCapacityForecast
	//   GET /api/v1/platform/health/alerts             — HealthAlerts
	protected.GET("/platform/health/summary", handler.HealthSummary(pool))
	protected.GET("/platform/health/regions", handler.HealthRegionsSummary(pool))
	protected.GET("/platform/health/tenants/top", handler.HealthTopTenants(pool))
	protected.GET("/platform/health/capacity/forecast", handler.HealthCapacityForecast(pool))
	protected.GET("/platform/health/alerts", handler.HealthAlerts(pool))
}

// mountPlatformPublicRoutes registers the PUBLIC Tier 11 endpoints
// (no JWT, no RequireAuth). Called from routes.go where the
// *gin.Engine and *auth.Issuer are in scope. Lives in routes_platform.go
// because every route in this tier should be discoverable from one place.
//
// Why a signature with *auth.Issuer: Phase 3 /platform/signup/verify
// needs to issue a fresh JWT so the user is logged-in immediately
// after email verification. PL1's only public route doesn't need
// the issuer, but threading it through now means we don't have to
// change the call site again when PL8 lands its public status page
// (which may also want to mint a token for an un-authenticated
// viewer).
//
// PL1 (this commit) + Phase 3 (this commit) register 4 PUBLIC routes:
//
//	GET  /api/v1/platform/deploy/install-script  — GetInstallScript (PL1)
//	POST /api/v1/platform/signup                 — CreateSignup     (PL3)
//	POST /api/v1/platform/signup/verify          — VerifySignup     (PL3)
//	POST /api/v1/platform/signup/resend          — ResendSignup     (PL3)
//
// Future phases add here in order:
//
//	Phase 4 (PL4): GET  /api/v1/platform/limits/definitions (public pricing)
//	Phase 8 (PL8): GET  /public/status (read-only status page)
func mountPlatformPublicRoutes(r *gin.Engine, pool *db.Pool, issuer *auth.Issuer) {
	// Tier 11 PL1 — PUBLIC install-script endpoint.
	// The freshly-installed Linux box hits this URL with no
	// JWT (the token IS the credential). On success the row is
	// marked used_at = now() so replays return 410 Gone.
	r.GET("/api/v1/platform/deploy/install-script", handler.GetInstallScript(pool))

	// Tier 11 PL3 — Self-Service Signup endpoints.
	// All three are PUBLIC because the freshly-typed email has
	// no StackWatch credentials yet. Rate-limit (10/day/IP) is
	// enforced inside CreateSignup (see signupAllowed) — the
	// other two don't create new rows so they don't need a
	// per-IP gate.
	r.POST("/api/v1/platform/signup", handler.CreateSignup(pool))
	r.POST("/api/v1/platform/signup/verify", handler.VerifySignup(pool, issuer))
	r.POST("/api/v1/platform/signup/resend", handler.ResendSignup(pool))

	// Tier 11 PL4 — PUBLIC plan catalog endpoint.
	// Pricing-page widget hits this URL with no JWT (a
	// prospective customer can see the 4 plans before
	// signing up). The handler reads the catalog (idempotent
	// seeded by auth.SeedBuiltinPlans on boot) and returns
	// the 4 rows ordered by sort_order ASC.
	r.GET("/api/v1/platform/limits/definitions", handler.GetLimitDefinitions(pool))
}
