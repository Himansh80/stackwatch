package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/handler"
	"github.com/stackwatch/platform/internal/middleware"
	"github.com/stackwatch/platform/internal/platform"
)

// buildRouter constructs the gin engine with all middleware + routes.
//
// Layout:
//   - Global middleware applied to every route
//   - Public (no-auth) routes: /health, setup wizard, auth endpoints
//   - Public webhook receivers (no-auth, called by GitHub/GitLab/CI
//     systems): /api/v1/cicd/webhook/{github,gitlab}
//   - Public unauthenticated ingest (no JWT, tenant via query param):
//     /api/v1/rum/events, /api/v1/containers/templates
//   - Protected "/api/v1" group: every Tier 0-7 endpoint — registered
//     by mountProtectedRoutes() in routes_protected.go so this file
//     stays under the 400-LOC cap while new Tiers add routes freely.
//
// To add a new protected route, edit routes_protected.go.
// To add a new public route, edit this file directly.
func buildRouter(ctx context.Context, logger *slog.Logger, pool *db.Pool, issuer *auth.Issuer, installMode, webTerminalURL string, synthRunner *handler.SyntheticsRunner, rateLimiter *platform.Limiter, ratePlans *platform.PlanCache) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Global middleware (panic-safe + structured + traceable)
	r.Use(middleware.RequestID())
	r.Use(middleware.Recover(logger))
	r.Use(middleware.Logging(logger))
	r.Use(middleware.CORS())
	r.Use(middleware.SecurityHeaders())

	// Health endpoint (no auth)
	r.GET("/health", func(c *gin.Context) {
		if err := pool.Health(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "degraded",
				"db":      "down",
				"mode":    installMode,
				"version": "0.1.0-tier1",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"db":      "ok",
			"mode":    installMode,
			"version": "0.1.0-tier1",
		})
	})

	// Setup wizard endpoints (Tier 0.5) — public, no auth
	setupH := handler.NewSetupHandler(pool, installMode, issuer)
	r.GET("/api/v1/setup/status", setupH.GetSetupStatus)
	r.POST("/api/v1/setup/initialize", setupH.InitializeSetup)
	r.POST("/api/v1/setup/regenerate-cert", setupH.RegenerateCert)

	// Auth endpoints (no auth required for login / signup / token ops)
	authH := handler.NewAuthHandler(pool, issuer, logger)
	r.POST("/api/v1/auth/login", authH.Login)
	r.POST("/api/v1/auth/signup", authH.Signup)
	r.POST("/api/v1/auth/forgot", authH.ForgotPassword)
	r.POST("/api/v1/auth/reset", authH.ResetPassword)
	r.POST("/api/v1/auth/magic-link", authH.MagicLink)
	r.GET("/api/v1/auth/magic-link/consume", authH.ConsumeMagicLink)
	r.POST("/api/v1/auth/accept-invite", authH.AcceptInvite)
	// /auth/refresh is intentionally UNPROTECTED — handler reads token
	// from request body, not the Authorization header. Putting it under
	// the protected group means RequireAuth aborts before Refresh runs.
	r.POST("/api/v1/auth/refresh", authH.Refresh)

	// Public unauthenticated ingest (tenant_id via query param)
	// RUM browser snippet — tier 6 v4 ingest endpoint.
	r.POST("/api/v1/rum/events", handler.IngestRUM(pool))

	// Protected endpoints (require valid JWT). Everything inside this
	// group is registered by mountProtectedRoutes in routes_protected.go.
	//
	// Tier 11 Phase 7 — Rate-limit middleware is applied AFTER
	// RequireAuth (so claims.TenantID is populated for the limiter)
	// and BEFORE every handler (so a 429 short-circuits any
	// expensive work without ever entering the handler). The
	// middleware fails OPEN on routes mounted WITHOUT RequireAuth
	// (e.g. health, signup) — those pass-through unchanged.
	protected := r.Group("/api/v1",
		handler.RequireAuth(issuer),
		middleware.RateLimit(rateLimiter, ratePlans),
	)
	// Auth self-service lives on the protected group so RequireAuth runs
	// before the handler reads the JWT subject from context.
	protected.GET("/auth/me", authH.Me)
	protected.POST("/auth/logout", authH.Logout)
	protected.PATCH("/auth/profile", authH.UpdateProfile)
	protected.POST("/auth/change-password", authH.ChangePassword)
	mountProtectedRoutes(protected, pool, webTerminalURL, synthRunner)

	// Tier 11 — Platform & Commerce (Phase 0 routes split). Three
	// groups because PL1/PL3/PL8 need PUBLIC endpoints (signup,
	// install-script, status page), PL2/PL4/PL7 need PROTECTED
	// (tenant-scoped), and PL5/PL6/PL8 need ADMIN (super-admin
	// scope). The ADMIN group is created here (currently a no-op
	// middleware — Phase 5 will add real role gating) so the
	// 3-group signature is stable across the whole tier and we
	// don't have to touch this file again until Phase 5.
	admin := protected.Group("/admin", handler.RequireRole("admin"))
	mountPlatformRoutes(r.Group("/api/v1"), protected, admin, pool, rateLimiter, ratePlans)
	mountMobileRoutes(r.Group("/api/v1"), protected, admin, pool)

	// Public CI/CD webhook receivers (no JWT — providers can't carry one).
	// Mounted at /api/v1/cicd/webhook/{github,gitlab} on the public router.
	// The handlers accept provider-native payloads and synthesize a pipeline
	// + deployment record for the tenant identified by repo slug.
	cicdWebhook := r.Group("/api/v1/cicd/webhook")
	cicdWebhook.POST("/github", handler.GitHubCICDWebhook(pool))
	cicdWebhook.POST("/gitlab", handler.GitLabCICDWebhook(pool))

	// Public ingest endpoint (no JWT, uses API key or ingest key from header).
	// Agent sends heartbeat + metrics here.
	r.POST("/api/v1/ingest/heartbeat", handler.IngestHeartbeat(pool))

	// Public SSO callback routes (no JWT — IdP-issued code/assertion).
	// Registered on the public engine (not the protected group) so
	// users coming from an IdP can land on /sso/initiate and
	// /sso/callback without already having a StackWatch JWT.
	// These 3 endpoints are the only PUBLIC Tier 9 endpoints — every
	// other enterprise route lives behind RequireAuth.
	mountEnterprisePublicRoutes(r, pool, issuer)

	// Tier 11 PL1 + PL3 — PUBLIC endpoints (no JWT). PL1's
	// /deploy/install-script is hit by a freshly-installed
	// Linux box; PL3's /signup/verify/resend are hit by a
	// freshly-typed email. Both carry their own credential
	// (install token / email token) so they bypass RequireAuth.
	// mountPlatformPublicRoutes now takes *auth.Issuer too
	// because /signup/verify needs to mint a fresh JWT so the
	// user is logged-in immediately after email verification.
	mountPlatformPublicRoutes(r, pool, issuer)

	// Public container templates (no auth — public knowledge endpoint).
	// Mounted at the end so it sits OUTSIDE the protected group; it was
	// historically registered near the protected container block but its
	// auth posture is "no JWT, just data".
	containerHForTemplates := handler.NewContainerHandler(handler.NewAdminHandler(webTerminalURL))
	r.GET("/api/v1/containers/templates", containerHForTemplates.ListTemplates)

	return r
}
