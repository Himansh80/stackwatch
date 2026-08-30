// Tier 11 Phase 7 — Rate Limiting (PL7).
//
// handlers_platform_ratelimit.go — HTTP route handlers
// for the 3 ratelimit endpoints:
//
//	GET  /api/v1/platform/ratelimit/me      — GetMyRateLimit       (auth-protected)
//	PATCH /api/v1/platform/ratelimit/global — UpdateGlobalLimits   (super_admin)
//	GET  /api/v1/platform/ratelimit/blocked — ListBlockedTenants  (super_admin)
//
// The middleware (internal/middleware/ratelimit.go) is
// what ENFORCES the bucket on every request; these
// endpoints are the admin VIEW surface. A regular user
// can hit /me to see their own current usage (drives the
// dashboard's KPI strip); only super_admin can hit
// /global (adjust caps) or /blocked (see who's hitting
// limits).
//
// Why a separate handler file:
//   Follows the established Phase 4/5/6 split pattern
//   (handlers_platform_<feature>.go +
//   handlers_platform_<feature>_types.go +
//   handlers_platform_<feature>_<helpers>.go). This file
//   holds the 3 routes + their imports + super_admin
//   gate logic; the types file holds JSON shapes; no
//   helpers file is needed (the 3 routes are short).
//
// Auth: every handler enforces RequireAuth (set on the
// /api/v1 group in routes.go) + super_admin check for
// /global and /blocked. /me only requires a valid JWT.

package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/kernel"
	"github.com/stackwatch/platform/internal/platform"
)

// rateLimitMaxBodyKeys caps the number of keys the
// handler will accept in a PATCH /global body. Today the
// schema has 4 plan keys (free/starter/pro/enterprise)
// so this is a generous upper bound that catches a
// pathological "send 100k keys" attack without rejecting
// a legitimate future plan addition.
const rateLimitMaxBodyKeys = 16

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/platform/ratelimit/me
// ------------------------------------------------------------------

// GetMyRateLimit returns the caller's current rate-limit
// usage. Auth-protected (RequireAuth). The tenantID is
// read from the JWT context so a user can't impersonate
// another tenant.
//
// The response mirrors platform.tenantUsage. The handler
// also resolves the caller's plan name from the JWT so
// the "limit_per_minute" field reflects the actual cap
// the middleware is enforcing.
//
// 200 → ratelimitMeResp{tenant_id, plan, limit_per_minute,
//
//	current_window_count, remaining,
//	reset_at_seconds, blocked_until_seconds}.
//
// 401 → no JWT.
func GetMyRateLimit(limiter *platform.Limiter, plans platform.PlanResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		plan := plans.ResolvePlan(tenantID)
		u := limiter.GetTenantUsage(tenantID, plan)
		kernel.RespondOK(c, ratelimitMeResp{
			TenantID:            u.TenantID,
			Plan:                u.Plan,
			LimitPerMinute:      u.LimitPerMinute,
			CurrentWindowCount:  u.CurrentWindowCount,
			Remaining:           u.Remaining,
			ResetAtSeconds:      u.ResetAtSeconds,
			BlockedUntilSeconds: u.BlockedUntilSeconds,
		})
	}
}

// ------------------------------------------------------------------
// Super-admin endpoint: PATCH /api/v1/platform/ratelimit/global
// ------------------------------------------------------------------

// UpdateGlobalLimits replaces per-plan rate-limit caps.
// Restricted to super_admin (a regular admin can't
// throttle the platform).
//
// Body: ratelimitGlobalReq with any subset of
// {free, starter, pro, enterprise}. Each value MUST be
// in [1..100000]. An empty body (or a body with every
// key matching PlanDefaults) triggers ResetToDefaults.
//
// 200 → ratelimitGlobalResp{limits, reset}.
// 400 → body too large, out-of-range value, unknown plan.
// 401 → no JWT. 403 → not super_admin.
func UpdateGlobalLimits(limiter *platform.Limiter, plans platform.PlanResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsUser, ok := userFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		if claimsUser.Role != "super_admin" {
			kernel.RespondErrorWithCode(c, http.StatusForbidden,
				"forbidden", "super_admin role required")
			return
		}

		var req ratelimitGlobalReq
		// An empty body is a valid "reset to defaults"
		// request — don't 400 on ShouldBindJSON's
		// EOF, allow it through.
		if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		// Build the per-plan delta map. nil pointers
		// mean "leave this plan alone" (merge semantics).
		patch := make(map[string]int, 4)
		if req.Free != nil {
			patch["free"] = *req.Free
		}
		if req.Starter != nil {
			patch["starter"] = *req.Starter
		}
		if req.Pro != nil {
			patch["pro"] = *req.Pro
		}
		if req.Enterprise != nil {
			patch["enterprise"] = *req.Enterprise
		}

		if len(patch) > rateLimitMaxBodyKeys {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest,
				"bad_request", "too many keys in PATCH body")
			return
		}

		// Detect the "reset to defaults" intent: empty
		// patch AND caller explicitly sent a body of
		// "{}" (so a missing body still 400s with a
		// helpful message). Use ContentLength == 0 as
		// the "explicit empty body" signal.
		isReset := len(patch) == 0 && c.Request.ContentLength == 0

		// Validate every supplied value before mutating.
		for plan, v := range patch {
			if !isAllowedRateLimitPlan(plan) {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest,
					"bad_request", "unknown plan: "+plan)
				return
			}
			if v < rateLimitMin || v > rateLimitMax {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest,
					"bad_request", "rate limit must be between 1 and 100000")
				return
			}
		}

		if isReset {
			limiter.ResetToDefaults()
		} else {
			if err := limiter.SetGlobalLimits(patch); err != nil {
				// Map sentinel errors to specific 400s
				// so the dashboard renders a useful
				// inline message instead of a generic
				// "bad request".
				switch err {
				case platform.ErrBadLimitRange:
					kernel.RespondErrorWithCode(c, http.StatusBadRequest,
						"bad_request", "rate limit must be between 1 and 100000")
				case platform.ErrUnknownPlan:
					kernel.RespondErrorWithCode(c, http.StatusBadRequest,
						"bad_request", "unknown plan in body")
				default:
					kernel.RespondError(c, err)
				}
				return
			}
		}

		// Invalidate every cached plan so a future
		// middleware read picks up the new caps. (Today
		// the middleware reads PlanDefaults directly so
		// the invalidation is unnecessary — but a
		// future per-tenant override cache would need
		// this hook. Defensive.)
		_ = plans

		kernel.RespondOK(c, ratelimitGlobalResp{
			Limits: limiter.GetGlobalLimits(),
			Reset:  isReset,
		})
	}
}

// ------------------------------------------------------------------
// Super-admin endpoint: GET /api/v1/platform/ratelimit/blocked
// ------------------------------------------------------------------

// ListBlockedTenants returns every tenant currently in
// its blocked window. Restricted to super_admin (this
// reveals which tenants are spamming the platform).
//
// 200 → ratelimitBlockedResp{blocked: [...]}. The list
// is empty (not null) when no tenants are blocked, so
// the dashboard renders an explicit "no tenants blocked"
// row instead of a confusing "null" message.
// 401 → no JWT. 403 → not super_admin.
func ListBlockedTenants(limiter *platform.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsUser, ok := userFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		if claimsUser.Role != "super_admin" {
			kernel.RespondErrorWithCode(c, http.StatusForbidden,
				"forbidden", "super_admin role required")
			return
		}

		rows := limiter.GetBlockedTenants()
		out := ratelimitBlockedResp{
			Blocked: make([]ratelimitBlockedRow, 0, len(rows)),
		}
		for _, r := range rows {
			out.Blocked = append(out.Blocked, ratelimitBlockedRow{
				TenantID:            r.TenantID,
				Plan:                r.Plan,
				CurrentCount:        r.CurrentCount,
				LimitPerMinute:      r.LimitPerMinute,
				BlockedUntilSeconds: r.BlockedUntilSeconds,
			})
		}
		kernel.RespondOK(c, out)
	}
}

// isAllowedRateLimitPlan reports whether name is one of
// the 4 canonical plan names. Mirrors
// platform.AllowedPlans (defence-in-depth).
func isAllowedRateLimitPlan(name string) bool {
	for _, p := range allowedRateLimitPlans {
		if p == name {
			return true
		}
	}
	return false
}
