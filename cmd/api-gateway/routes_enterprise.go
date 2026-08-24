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
//	Phase 1: SSO Foundation (8 routes)             ← added in Phase 1
//	Phase 2: SCIM Provisioning (6 routes)
//	Phase 3: Advanced RBAC (6 routes)
//	Phase 4: Audit Log Retention + Export (4 routes)
//	Phase 5: Compliance Reports (5 routes)
//	Phase 6: Enterprise Tenants + Org Settings (3 routes)
//
// All handlers honor tenant_id from the JWT — no cross-tenant data
// ever crosses the wire. Idempotent migrations in
// migrations/040_enterprise.sql set up the backing tables.
//
// The 3 SSO callback routes (`GET /sso/initiate`, `GET /sso/callback`,
// `POST /sso/callback`) are intentionally PUBLIC — they sit on a
// separate router group registered by mountEnterprisePublicRoutes
// from routes.go so users coming from an IdP (with a code/assertion,
// no JWT yet) can land on them. mountEnterprisePublicRoutes lives in
// routes.go because it needs the *auth.Issuer to wire the public
// callbacks.
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

	// (Phases 2-6 leave their mount-call comments as anchors for
	// the next subagent — no actual registration until those phases ship.)
}

// mountEnterprisePublicRoutes registers the PUBLIC SSO callback
// routes. Called from routes.go (where the *gin.Engine and
// *auth.Issuer are in scope) so we don't have to thread those
// through every protected handler signature. Must be called on
// the public engine (NOT the protected group) so the IdP-issued
// code/assertion is the credential — no JWT required.
func mountEnterprisePublicRoutes(r *gin.Engine, pool *db.Pool, issuer *auth.Issuer) {
	iss := handler.NewSSOIssuer(pool, issuer)
	// Public SSO callback routes (no JWT, no RequireAuth).
	r.GET("/api/v1/enterprise/sso/initiate", handler.InitiateSSO(iss))
	r.GET("/api/v1/enterprise/sso/callback", handler.OIDCCallback(iss))
	r.POST("/api/v1/enterprise/sso/callback", handler.SAMLCallback(iss))
}
