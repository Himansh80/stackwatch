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
func buildRouter(ctx context.Context, logger *slog.Logger, pool *db.Pool, issuer *auth.Issuer, installMode, webTerminalURL string, synthRunner *handler.SyntheticsRunner) *gin.Engine {
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
	protected := r.Group("/api/v1", handler.RequireAuth(issuer))
	// Auth self-service lives on the protected group so RequireAuth runs
	// before the handler reads the JWT subject from context.
	protected.GET("/auth/me", authH.Me)
	protected.POST("/auth/logout", authH.Logout)
	protected.PATCH("/auth/profile", authH.UpdateProfile)
	protected.POST("/auth/change-password", authH.ChangePassword)
	mountProtectedRoutes(protected, pool, webTerminalURL, synthRunner)

	// Public CI/CD webhook receivers (no JWT — providers can't carry one).
	// Mounted at /api/v1/cicd/webhook/{github,gitlab} on the public router.
	// The handlers accept provider-native payloads and synthesize a pipeline
	// + deployment record for the tenant identified by repo slug.
	cicdWebhook := r.Group("/api/v1/cicd/webhook")
	cicdWebhook.POST("/github", handler.GitHubCICDWebhook(pool))
	cicdWebhook.POST("/gitlab", handler.GitLabCICDWebhook(pool))

	// Public container templates (no auth — public knowledge endpoint).
	// Mounted at the end so it sits OUTSIDE the protected group; it was
	// historically registered near the protected container block but its
	// auth posture is "no JWT, just data".
	containerHForTemplates := handler.NewContainerHandler(handler.NewAdminHandler(webTerminalURL))
	r.GET("/api/v1/containers/templates", containerHForTemplates.ListTemplates)

	return r
}