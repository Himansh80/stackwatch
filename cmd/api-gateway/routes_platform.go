package main

import (
	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/handler"
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
// Phases 1-8 add their routes here in this order:
//
//	Phase 1: Push-Button Deploy (PL1)        ← 3 routes (added in this commit)
//	Phase 2: Usage Metering (PL2)            ← 3 routes
//	Phase 3: Self-Service Signup (PL3)       ← 3 routes
//	Phase 4: Tenant Limits (PL4)             ← 2 routes
//	Phase 5: Backup/Restore (PL5)            ← 3 routes
//	Phase 6: Multi-Region / HA (PL6)         ← 3 routes
//	Phase 7: Rate Limiting (PL7)             ← middleware + 0 routes
//	Phase 8: Platform Health (PL8)           ← 4 routes
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
func mountPlatformRoutes(public, protected, admin *gin.RouterGroup, pool *db.Pool) {
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
}

// mountPlatformPublicRoutes registers the PUBLIC Tier 11 endpoints
// (no JWT, no RequireAuth). Called from routes.go where the
// *gin.Engine is in scope. Lives in routes_platform.go because
// every route in this tier should be discoverable from one place.
//
// PL1 today registers exactly one PUBLIC route:
//
//	GET /api/v1/platform/deploy/install-script  — GetInstallScript
//
// Why a public route: a freshly-installed Linux box has no JWT yet
// — the install token itself IS the credential. Same pattern
// Tier 9 uses for SCIM 2.0 (handlers_scim_public.go registered
// in routes_enterprise.go via mountEnterprisePublicRoutes).
//
// Future phases add here in order:
//
//	Phase 3 (PL3): POST /public/signup + verify-email + resend-verification
//	Phase 8 (PL8): GET  /public/status (read-only status page)
func mountPlatformPublicRoutes(r *gin.Engine, pool *db.Pool) {
	// Tier 11 PL1 — PUBLIC install-script endpoint.
	// The freshly-installed Linux box hits this URL with no
	// JWT (the token IS the credential). On success the row is
	// marked used_at = now() so replays return 410 Gone.
	r.GET("/api/v1/platform/deploy/install-script", handler.GetInstallScript(pool))
}