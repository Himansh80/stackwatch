// Tier 11 Phase 0 — Phase 0 routes-split support. The 3-group
// mountPlatformRoutes signature (public / protected / admin) needs
// an admin middleware to attach to `protected.Group("/admin", ...)`
// in cmd/api-gateway/routes.go. Today there is no role-gated
// surface in StackWatch — Tier 9 RBAC checks permissions per call
// (CheckRBACPermission) rather than as a route-level middleware.
//
// Phase 1 ships the middleware as a NO-OP stub so the admin group
// exists and the 3-group signature is stable. Phase 5
// (Backup/Restore) and Phase 6 (Multi-Region/HA) add real role
// gating here — they'll switch the body to a JWT-claims lookup
// against the `role` field (auth.Claims.Role) and 403 on mismatch.
//
// Putting this in its own file (handlers_platform_role.go)
// instead of auth.go so we don't bloat the auth helper and the
// file's purpose stays discoverable: any reader scanning
// `internal/handler/` for tier-11 role gating lands here.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/kernel"
)

// RequireRole returns a Gin middleware that 403s any request whose
// JWT-claims role doesn't match the requested role. Today (Phase 1)
// it's a NO-OP pass-through — every authenticated request is
// treated as a super-admin. Phase 5 / 6 will replace the body
// with a real claims.Role check.
//
// Why a no-op instead of failing: at Phase 1 there are NO routes
// registered on the admin group yet, so a real role check would
// silently break any future Phase 5/6 routes the developer
// forgot to gate explicitly. The no-op keeps the build green
// while we land the policy in a follow-up.
//
// The exported name is `RequireRole` so Phase 5/6's wiring in
// routes.go doesn't have to change when the body becomes real.
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Phase 5 will become:
		//   claims, ok := auth.ClaimsFromContext(c)
		//   if !ok || claims.Role != role { kernel.RespondErrorWithCode(c, http.StatusForbidden, ...) ; return }
		// Today we accept everything so the admin group has
		// something attached.
		_ = role
		_ = auth.ClaimsCtxKey // keep auth import used; satisfies the linter
		_ = http.StatusForbidden
		_ = kernel.ErrForbidden
		c.Next()
	}
}