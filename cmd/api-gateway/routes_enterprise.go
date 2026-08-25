package main

import (
	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/handler"
)

// mountEnterpriseRoutes registers the Tier 9 Security & Enterprise
// endpoints — speckit change 007-tier9-security-enterprise.
//
// Created in Phase 0 because routes_protected.go was at the 396-LOC
// cap after Tier 8.5 Intelligence Dashboard (006-tier8-intelligence-
// alerting Phase 5). Adding 32 new Tier 9 routes without a split
// would push that file past the 400-LOC limit. Splitting keeps
// routes_protected.go focused on top-level tier wiring while the
// SSO / SCIM / RBAC / Audit / Compliance / Enterprise-tenants CRUD
// lives here.
//
// Phases 1-6 add their routes here in this order:
//
//	Phase 1: SSO Foundation (8 routes)              ← added in Phase 1
//	Phase 2: SCIM Provisioning (4 protected + 5 public)  ← added in Phase 2
//	Phase 3: Advanced RBAC (6 routes)               ← added in Phase 3
//	Phase 4: Audit Log Retention + Export (4 routes) ← added in Phase 4
//	Phase 5: Compliance Reports (7 routes)         ← added in Phase 5
//	Phase 6: Enterprise Tenants + Org Settings (3 routes)
//
// All handlers honor tenant_id from the JWT — no cross-tenant data
// ever crosses the wire. Idempotent migrations in
// migrations/040_enterprise.sql set up the backing tables.
//
// The 3 SSO callback routes (`GET /sso/initiate`, `GET /sso/callback`,
// `POST /sso/callback`) and the 5 PUBLIC SCIM 2.0 routes (`/scim/v2/*`)
// are intentionally PUBLIC — they sit on a separate router group
// registered by mountEnterprisePublicRoutes from routes.go so users
// coming from an IdP (with a code/assertion, no JWT yet) can land on
// them. mountEnterprisePublicRoutes lives in routes.go because it
// needs the *auth.Issuer to wire the public callbacks.
func mountEnterpriseRoutes(protected *gin.RouterGroup, pool *db.Pool) {
	// ---- Tier 9.1: SSO Foundation (Phase 1) ----
	// Per spec §"Story 1 — SSO Foundation", these 6 protected endpoints
	// let tenant admins configure OIDC/SAML providers. Sensitive
	// fields (client_secret, SAML x509 cert) are encrypted at rest via
	// the existing internal/handler.encryptSecret helper (AES-GCM)
	// before INSERT, so the JSONB config column never holds plaintext.
	//
	// Routes (6 protected):
	//   GET    /api/v1/enterprise/sso/providers          — ListSSOProviders
	//   POST   /api/v1/enterprise/sso/providers          — CreateSSOProvider
	//   PATCH  /api/v1/enterprise/sso/providers/:id      — UpdateSSOProvider
	//   DELETE /api/v1/enterprise/sso/providers/:id      — DeleteSSOProvider
	//   POST   /api/v1/enterprise/sso/test               — TestSSOProvider
	//   GET    /api/v1/enterprise/sso/connections        — ListSSOConnections
	//
	// Plus 3 PUBLIC callback routes registered separately in routes.go
	// (mountEnterprisePublicRoutes) because they must NOT live behind
	// RequireAuth — users arrive from the IdP with a code/assertion,
	// not a JWT:
	//   GET    /api/v1/enterprise/sso/initiate           — InitiateSSO
	//   GET    /api/v1/enterprise/sso/callback           — OIDCCallback
	//   POST   /api/v1/enterprise/sso/callback           — SAMLCallback
	protected.GET("/enterprise/sso/providers", handler.ListSSOProviders(pool))
	protected.POST("/enterprise/sso/providers", handler.CreateSSOProvider(pool))
	protected.PATCH("/enterprise/sso/providers/:id", handler.UpdateSSOProvider(pool))
	protected.DELETE("/enterprise/sso/providers/:id", handler.DeleteSSOProvider(pool))
	protected.POST("/enterprise/sso/test", handler.TestSSOProvider(pool))
	protected.GET("/enterprise/sso/connections", handler.ListSSOConnections(pool))

	// ---- Tier 9.2: SCIM Provisioning (Phase 2) ----
	// Per spec §"Story 2 — SCIM Provisioning", these 4 protected
	// endpoints let tenant admins manage SCIM bearer tokens. Tokens
	// are bcrypt-hashed (cost 10) before INSERT and the plaintext is
	// returned to the caller EXACTLY ONCE on POST — we never persist
	// plaintext. Every successful/failed SCIM operation is logged
	// to scim_sync_log via logSCIMSync.
	//
	// Routes (4 protected):
	//   GET    /api/v1/enterprise/scim/tokens            — ListSCIMTokens
	//   POST   /api/v1/enterprise/scim/tokens            — CreateSCIMToken
	//   DELETE /api/v1/enterprise/scim/tokens/:id        — RevokeSCIMToken
	//   GET    /api/v1/enterprise/scim/sync-log          — ListSCIMSyncLog
	//
	// Plus 5 PUBLIC SCIM 2.0 endpoints registered separately in routes.go
	// (mountEnterprisePublicRoutes) because they must NOT live behind
	// RequireAuth — IdPs (Okta, Azure AD, Google Workspace) carry a
	// Bearer token, not a JWT. The scimAuthMiddleware
	// (handlers_scim_protected.go) bcrypt-compares the supplied
	// Bearer against scim_tokens.token_hash:
	//   GET    /scim/v2/Users                            — SCIMListUsers
	//   POST   /scim/v2/Users                            — SCIMCreateUser
	//   PUT    /scim/v2/Users/:id                        — SCIMUpdateUser
	//   DELETE /scim/v2/Users/:id                        — SCIMDisableUser
	//   POST   /scim/v2/Groups                           — SCIMCreateGroup (stub)
	protected.GET("/enterprise/scim/tokens", handler.ListSCIMTokens(pool))
	protected.POST("/enterprise/scim/tokens", handler.CreateSCIMToken(pool))
	protected.DELETE("/enterprise/scim/tokens/:id", handler.RevokeSCIMToken(pool))
	protected.GET("/enterprise/scim/sync-log", handler.ListSCIMSyncLog(pool))

	// ---- Tier 9.3: Advanced RBAC (Phase 3) ----
	// Per spec §"Story 3 — Advanced RBAC", these 6 protected endpoints
	// let tenant admins manage custom roles and assign them to users.
	// Built-in roles (Admin / Operator / Viewer / Billing) are
	// lazy-seeded on first GET /rbac/roles per tenant via
	// seedBuiltinRolesForTenant (handlers_rbac_roles.go) and are
	// READ-ONLY — UpdateRBACRole / DeleteRBACRole return 403 on
	// is_builtin=true rows.
	//
	// The check endpoint caches the union of effective permissions
	// per (tenant_id, user_id) for 60s (effectivePermCacheTTL in
	// handlers_rbac_types.go) so a burst of gated calls doesn't
	// hammer the DB. Cache is invalidated by UpdateRBACRole /
	// DeleteRBACRole / AssignRBACRole / UnassignRBACRole.
	//
	// Routes (6 protected):
	//   GET    /api/v1/enterprise/rbac/roles                  — ListRBACRoles
	//   POST   /api/v1/enterprise/rbac/roles                  — CreateRBACRole
	//   PATCH  /api/v1/enterprise/rbac/roles/:id              — UpdateRBACRole
	//   DELETE /api/v1/enterprise/rbac/roles/:id              — DeleteRBACRole
	//   POST   /api/v1/enterprise/rbac/users/:user_id/roles   — AssignRBACRole
	//   DELETE /api/v1/enterprise/rbac/users/:user_id/roles/:role_id — UnassignRBACRole
	//   GET    /api/v1/enterprise/rbac/check                  — CheckRBACPermission
	protected.GET("/enterprise/rbac/roles", handler.ListRBACRoles(pool))
	protected.POST("/enterprise/rbac/roles", handler.CreateRBACRole(pool))
	protected.PATCH("/enterprise/rbac/roles/:id", handler.UpdateRBACRole(pool))
	protected.DELETE("/enterprise/rbac/roles/:id", handler.DeleteRBACRole(pool))
	protected.POST("/enterprise/rbac/users/:user_id/roles", handler.AssignRBACRole(pool))
	protected.DELETE("/enterprise/rbac/users/:user_id/roles/:role_id", handler.UnassignRBACRole(pool))
	protected.GET("/enterprise/rbac/check", handler.CheckRBACPermission(pool))

	// ---- Tier 9.4: Audit Log Retention + Export (Phase 4) ----
	// Per spec §"Story 4 — Audit Log Retention + Export", these 4
	// protected endpoints let compliance officers archive historical
	// audit_log events (gzip-compressed) and stream filtered exports
	// to CSV or NDJSON for auditors.
	//
	// Background-job design:
	//   POST /audit/archive returns 201 IMMEDIATELY with the row id
	//   after INSERTing a `status='running'` row. The compression
	//   itself happens in a goroutine spawned by
	//   handlers_audit_archive.go::runArchiveJob, rate-limited by
	//   a buffered semaphore (`auditArchiveSem`, size = 4) so a
	//   burst of 10 parallel POSTs only spawns 4 concurrent workers
	//   + 6 queued submits. The job updates the row to
	//   status='completed' (with the gzip blob) or 'failed' when
	//   done. The /download endpoint refuses to serve a non-
	//   completed row so a racing download sees a 409.
	//
	// Streaming-export design:
	//   POST /audit/export streams rows directly from pgx.Rows
	//   to gin.ResponseWriter via csv.Writer (CSV) or
	//   json.Encoder (NDJSON) so 1M-row exports don't load the
	//   entire resultset into memory. Content-Disposition is set
	//   before the first Write so the browser auto-downloads.
	//
	// Routes (4 protected):
	//   GET    /api/v1/enterprise/audit/archives                   — ListAuditArchives
	//   POST   /api/v1/enterprise/audit/archive                    — CreateAuditArchive
	//   GET    /api/v1/enterprise/audit/archives/:id/download      — DownloadAuditArchive
	//   POST   /api/v1/enterprise/audit/export                     — ExportAuditEvents
	protected.GET("/enterprise/audit/archives", handler.ListAuditArchives(pool))
	protected.POST("/enterprise/audit/archive", handler.CreateAuditArchive(pool))
	protected.GET("/enterprise/audit/archives/:id/download", handler.DownloadAuditArchive(pool))
	protected.POST("/enterprise/audit/export", handler.ExportAuditEvents(pool))

	// ---- Tier 9.5: Compliance Reports (Phase 5) ----
	// Per spec §"Story 5 — Compliance Reports", these 7 protected
	// endpoints let compliance officers generate evidence packages
	// for SOC2 / ISO27001 / HIPAA / PCI / GDPR auditors and configure
	// recurring schedules that future Tier 9.x cron sweeps will
	// dispatch.
	//
	// Background-job design (mirrors Phase 4):
	//   POST /compliance/reports returns 201 IMMEDIATELY with the
	//   row id after INSERTing a status='pending' row. The
	//   generator (runReportJob in handlers_compliance_worker.go)
	//   flips the row to 'running', SELECTs several real-DB counts
	//   (audit_log / anomaly_events / active users / SSO
	//   connections) that serve as evidence sections, composes a
	//   plain-text artifact at
	//     /opt/stackwatch/reports/{tenant_id}/{report_id}.txt
	//   and UPDATEs the row to status='completed' (with
	//   artifact_path + completed_at) or 'failed' (with
	//   error_message). Concurrent generators are capped at
	//   complianceReportMaxParallel (4) via a buffered semaphore
	//   (`complianceReportSem`) so a burst of 10 parallel POSTs
	//   only spawns 4 goroutines + 6 queued submits.
	//
	// Schedule design (config-only this phase):
	//   POST /compliance/schedules persists a recurring-job config
	//   row (framework / frequency / recipients / next_run_at).
	//   Phase 5 ships the config surface only — the actual cron
	//   sweep that reads next_run_at, generates the report, and
	//   dispatches to recipients is a future Tier 9.x janitor and
	//   reads the same table unchanged.
	//
	// Routes (7 protected):
	//   GET    /api/v1/enterprise/compliance/reports                 — ListComplianceReports
	//   POST   /api/v1/enterprise/compliance/reports                 — CreateComplianceReport
	//   GET    /api/v1/enterprise/compliance/reports/:id              — GetComplianceReport
	//   GET    /api/v1/enterprise/compliance/reports/:id/download     — DownloadComplianceReport
	//   GET    /api/v1/enterprise/compliance/schedules               — ListComplianceSchedules
	//   POST   /api/v1/enterprise/compliance/schedules               — CreateComplianceSchedule
	//   DELETE /api/v1/enterprise/compliance/schedules/:id           — DeleteComplianceSchedule
	protected.GET("/enterprise/compliance/reports", handler.ListComplianceReports(pool))
	protected.POST("/enterprise/compliance/reports", handler.CreateComplianceReport(pool))
	protected.GET("/enterprise/compliance/reports/:id", handler.GetComplianceReport(pool))
	protected.GET("/enterprise/compliance/reports/:id/download", handler.DownloadComplianceReport(pool))
	protected.GET("/enterprise/compliance/schedules", handler.ListComplianceSchedules(pool))
	protected.POST("/enterprise/compliance/schedules", handler.CreateComplianceSchedule(pool))
	protected.DELETE("/enterprise/compliance/schedules/:id", handler.DeleteComplianceSchedule(pool))

	// (Phase 6 leaves its mount-call comment as an anchor for the
	// next subagent — no actual registration until that phase ships.)
}

// mountEnterprisePublicRoutes registers the PUBLIC SSO callback
// routes AND the PUBLIC SCIM 2.0 routes. Called from routes.go
// (where the *gin.Engine and *auth.Issuer are in scope) so we don't
// have to thread those through every protected handler signature.
// Must be called on the public engine (NOT the protected group) so
// the IdP-issued code/assertion / Bearer token is the credential —
// no JWT required.
func mountEnterprisePublicRoutes(r *gin.Engine, pool *db.Pool, issuer *auth.Issuer) {
	iss := handler.NewSSOIssuer(pool, issuer)

	// --- Phase 1: PUBLIC SSO callbacks (no JWT, no RequireAuth) ---
	r.GET("/api/v1/enterprise/sso/initiate", handler.InitiateSSO(iss))
	r.GET("/api/v1/enterprise/sso/callback", handler.OIDCCallback(iss))
	r.POST("/api/v1/enterprise/sso/callback", handler.SAMLCallback(iss))

	// --- Phase 2: PUBLIC SCIM 2.0 endpoints (Bearer token via
	// scimAuthMiddleware). The middleware bcrypt-compares the
	// Authorization: Bearer *** against scim_tokens.token_hash
	// and stashes the resulting scimClaims in the gin context so
	// each SCIM handler can honor tenant_id + scopes. ---
	scimGroup := r.Group("/scim/v2", handler.SCIMAuth(pool))
	scimGroup.GET("/Users", handler.SCIMListUsers(pool))
	scimGroup.POST("/Users", handler.SCIMCreateUser(pool))
	scimGroup.PUT("/Users/:id", handler.SCIMUpdateUser(pool))
	scimGroup.DELETE("/Users/:id", handler.SCIMDisableUser(pool))
	scimGroup.POST("/Groups", handler.SCIMCreateGroup(pool))
}
