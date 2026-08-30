// Tier 11 Phase 4 — Tenant Limits (PL4).
//
// HTTP route handlers for the 5 limits endpoints:
//
//	GET   /api/v1/platform/limits/definitions — GetLimitDefinitions (PUBLIC)
//	GET   /api/v1/platform/limits/me          — GetMyLimits        (auth-protected)
//	PATCH /api/v1/platform/limits/me          — PatchMyLimits      (auth-protected, super_admin)
//	POST  /api/v1/platform/limits/check       — CheckLimit         (auth-protected)
//	GET   /api/v1/platform/limits/usage       — GetLimitsUsage     (auth-protected)
//
// Shared types live in handlers_platform_limits_types.go (kept
// under the 400-LOC cap by the split). The helper layer lives
// in internal/platform/limits.go (EffectiveLimits +
// UpsertTenantPlan + LoadEffectiveLimits); this file is the
// HTTP-boundary glue between JSON shapes and helper calls.
//
// Auth & tenancy: 4 are PROTECTED (RequireAuth), 1 is PUBLIC
// (the definitions endpoint — anyone can see pricing).
// Tenant isolation: GetMyLimits + PatchMyLimits + CheckLimit +
// GetLimitsUsage all filter by claims.TenantID. PatchMyLimits
// additionally requires Role == "super_admin" because changing
// your plan has billing implications.
//
// Why these endpoints:
//
//	GetLimitDefinitions backs the public pricing-page widget.
//	No JWT required — visitors who aren't logged in should still
//	see what plans exist.
//
//	GetMyLimits returns the calling tenant's effective limits
//	(catalog defaults ∪ custom_overrides). The dashboard
//	renders this as the KPI strip + plan badge.
//
//	PatchMyLimits changes the calling tenant's plan + (optionally)
//	merges a custom_overrides map. Uses UPSERT so the first PATCH
//	from a freshly-onboarded tenant creates the row instead of
//	404-ing. Restricted to super_admin (a regular admin can't
//	upgrade their own plan — billing-driven change).
//
//	CheckLimit is the dry-run probe ("would this action
//	exceed my limit?"). Returns {allowed, current, limit,
//	remaining, message}. The frontend uses this to grey out
//	"Create" buttons before the user fills out a long form.
//
//	GetLimitsUsage is the live KPI strip — same numbers as
//	GetMyLimits but joined with the caller's actual usage
//	counts so each cell can render a progress bar. The "warn"
//	statuses come from the >= 80% threshold; "exceeded" from
//	>= 100%. The Warnings slice is a flat list of human
//	strings ("You're at 90% of your alert limit") for the
//	billing-warnings banner.
package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
	"github.com/stackwatch/platform/internal/platform"
)

// =====================================================================
// Public endpoint: GET /api/v1/platform/limits/definitions
// =====================================================================

// GetLimitDefinitions returns every plan in the catalog, ordered
// for the pricing page (sort_order ASC). No JWT, no RequireAuth —
// this is the "what plans exist?" endpoint the marketing site hits.
//
// 200 → {plans: [{name, display_name, monthly_price_cents, …}, …]}
func GetLimitDefinitions(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT name, display_name, monthly_price_cents, currency,
			        max_servers, max_alerts, max_dashboards, max_team_members,
			        data_retention_days, metrics_retention_days, storage_gb_limit,
			        api_calls_per_minute, features
			   FROM platform_plan_definitions
			  ORDER BY sort_order ASC, name ASC`)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		var plans []planDefinitionResp
		for rows.Next() {
			var p planDefinitionResp
			if err := rows.Scan(&p.Name, &p.DisplayName, &p.MonthlyPriceCents, &p.Currency,
				&p.MaxServers, &p.MaxAlerts, &p.MaxDashboards, &p.MaxTeamMembers,
				&p.DataRetentionDays, &p.MetricsRetentionDays, &p.StorageGBLimit,
				&p.APICallsPerMinute, &p.Features); err != nil {
				continue
			}
			plans = append(plans, p)
		}
		kernel.RespondOK(c, planDefinitionListResp{Plans: plans})
	}
}

// =====================================================================
// Protected endpoint: GET /api/v1/platform/limits/me
// =====================================================================

// GetMyLimits returns the caller's tenant_id + effective limits
// + custom_overrides + suspend state. The dashboard renders
// this as the "My Plan" panel.
//
// Auth: RequireAuth (set up by routes_platform.go on the
// protected group). Tenant isolation: always read by
// claims.TenantID — no cross-tenant leakage.
//
// 200 → tenantLimitsResp{TenantID, PlanName, …, CustomOverrides, Suspended}
func GetMyLimits(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		eff, err := platform.LoadEffectiveLimits(c.Request.Context(), tenantID, pool)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		// Also fetch the raw row so the response includes the
		// custom_overrides map (EffectiveLimits only carries
		// the merged numeric caps; the dashboard wants to
		// show "you've overridden max_servers to 50" too).
		raw, _ := loadRawLimits(c.Request.Context(), tenantID, pool)
		kernel.RespondOK(c, effectiveToResp(tenantID, eff, raw))
	}
}

// =====================================================================
// Protected endpoint: PATCH /api/v1/platform/limits/me
// =====================================================================

// PatchMyLimits changes the caller's plan (and optionally merges
// a custom_overrides map). Restricted to super_admin via the
// dashboard role chip — a regular admin can't upgrade themselves
// and rack up the bill. (Auth gating happens at the handler level
// because the role is on the JWT, not the URL.)
//
// Body: {plan_name, custom_overrides?}
// 200 → tenantLimitsResp (the new effective view)
// 400 → unknown plan_name, unknown override key, negative value
// 403 → caller is not super_admin
//
// First-PATCH-on-fresh-tenant behavior: UPSERT creates the row
// when missing. We do NOT 404 if no row exists yet — every
// tenant has implicit "free" until they PATCH.
//
// Free-plan-downgrade warning: if the new plan is "free" and the
// previous plan was paid, the response carries a `warning` field
// (in the headers via X-Tier-Warning header — the body shape is
// unchanged). Today this is informational; Phase 8 PL8 will use
// the same signal to emit a customer-visible email.
func PatchMyLimits(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		// Role check — restricted to super_admin.
		user, ok := userFromContext(c)
		if !ok || !user.IsSuperAdmin() {
			kernel.RespondError(c, kernel.ErrForbidden)
			return
		}
		var req tenantLimitsReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		req.PlanName = strings.TrimSpace(req.PlanName)
		if !isAllowedPlanName(req.PlanName) {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"plan_name is not in the allowlist (free|starter|pro|enterprise)")
			return
		}
		// Validate override keys + values.
		for k, v := range req.CustomOverrides {
			if !isAllowedOverrideKey(k) {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
					"unknown override key: "+k)
				return
			}
			if v < 0 {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
					"override values must be non-negative: "+k)
				return
			}
		}
		// Detect the free-downgrade case BEFORE we apply the
		// UPSERT so the operator's caller sees the warning.
		wasOnPaid := false
		if prev, err := platform.LoadEffectiveLimits(c.Request.Context(), tenantID, pool); err == nil {
			wasOnPaid = isPaidPlan(prev.PlanName)
		}
		if err := platform.UpsertTenantPlan(c.Request.Context(), tenantID, req.PlanName, req.CustomOverrides, pool); err != nil {
			kernel.RespondError(c, err)
			return
		}
		eff, err := platform.LoadEffectiveLimits(c.Request.Context(), tenantID, pool)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		raw, _ := loadRawLimits(c.Request.Context(), tenantID, pool)
		resp := effectiveToResp(tenantID, eff, raw)
		if wasOnPaid && req.PlanName == "free" {
			// Inform the caller. The retention worker will
			// purge data > 7d within 24h.
			c.Header("X-Tier-Warning", "downgrade-to-free-retention-7d")
		}
		kernel.RespondOK(c, resp)
	}
}

// =====================================================================
// Protected endpoint: POST /api/v1/platform/limits/check
// =====================================================================

// CheckLimit is a dry-run probe: would `action` of quantity `n`
// exceed the caller's effective limit? Returns {allowed,
// current, limit, remaining, message} so the frontend can
// preflight a long form before submit.
//
// Auth: RequireAuth. Tenant isolation: by claims.TenantID.
// Action allowlist: server.allowlist per allowedLimitActions
// (PUSH-style validation rejects typos like "create-servr").
//
// Body: {action, quantity}
// 200 → limitsCheckResp{Allowed, action, current, limit, remaining, quantity, message}
// 400 → unknown action, non-positive quantity
func CheckLimit(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req limitsCheckReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		req.Action = strings.TrimSpace(req.Action)
		if !isAllowedLimitAction(req.Action) {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"action is not in the allowlist (create_server|create_alert|create_dashboard|invite_member)")
			return
		}
		if req.Quantity <= 0 {
			req.Quantity = 1
		}
		eff, err := platform.LoadEffectiveLimits(c.Request.Context(), tenantID, pool)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if eff.Suspended {
			kernel.RespondOK(c, limitsCheckResp{
				Allowed:  false,
				Action:   req.Action,
				Quantity: req.Quantity,
				Message:  "tenant is suspended — no new resources permitted",
			})
			return
		}
		current, limit := usageForAction(c.Request.Context(), req.Action, tenantID, eff, pool)
		remaining := int(limit) - current
		if remaining < 0 {
			remaining = 0
		}
		wouldExceed := current+req.Quantity > int(limit)
		allowed := !wouldExceed
		msg := buildCheckMessage(req.Action, current, int(limit), remaining, allowed, eff.PlanName)
		kernel.RespondOK(c, limitsCheckResp{
			Allowed:   allowed,
			Action:    req.Action,
			Current:   current,
			Limit:     int(limit),
			Remaining: remaining,
			Quantity:  req.Quantity,
			Message:   msg,
		})
	}
}

// =====================================================================
// Protected endpoint: GET /api/v1/platform/limits/usage
// =====================================================================

// GetLimitsUsage returns the live KPI strip — current usage
// joined with effective caps, per resource family. The "warn"
// / "exceeded" status thresholds drive the progress-bar color.
//
// Auth: RequireAuth. Tenant isolation: by claims.TenantID.
//
// 200 → limitsUsageResp{Servers, Alerts, Dashboards, TeamMembers,
//
//	StorageGB, APICallsPerMin, Warnings, PlanName}
func GetLimitsUsage(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		eff, err := platform.LoadEffectiveLimits(c.Request.Context(), tenantID, pool)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		// Pull live counts for each resource family.
		servers, _ := countServers(c.Request.Context(), tenantID, pool)
		alerts, _ := countAlerts(c.Request.Context(), tenantID, pool)
		dashboards, _ := countDashboards(c.Request.Context(), tenantID, pool)
		members, _ := countTeamMembers(c.Request.Context(), tenantID, pool)
		storageGB, _ := storageUsageGB(c.Request.Context(), tenantID, pool)

		resp := limitsUsageResp{
			PlanName:        eff.PlanName,
			PlanDisplayName: eff.PlanDisplayName,
			Servers:         newUsageMetric(servers, int64(eff.MaxServers)),
			Alerts:          newUsageMetric(alerts, int64(eff.MaxAlerts)),
			Dashboards:      newUsageMetric(dashboards, int64(eff.MaxDashboards)),
			TeamMembers:     newUsageMetric(members, int64(eff.MaxTeamMembers)),
			StorageGB:       newUsageMetric(storageGB, eff.StorageGBLimit),
			APICallsPerMin:  newUsageMetric(0, int64(eff.APICallsPerMinute)), // per-minute counters live in PL7
		}
		// Build warning strings for any metric at >= 80%.
		resp.Warnings = buildUsageWarnings(resp)
		kernel.RespondOK(c, resp)
	}
}

// =====================================================================
// Shared helpers
// =====================================================================
//
// Helper functions (effectiveToResp, loadRawLimits, count*,
// storageUsageGB, usageForAction, buildCheckMessage,
// buildUsageWarnings, newUsageMetric, isPaidPlan, itoa64)
// live in handlers_platform_limits_helpers.go to keep this
// file under the 400-LOC cap. The handler funcs above are the
// only HTTP-boundary code in this file; everything else is a
// data-shape projection or a SQL count that any of the 5
// endpoints might call.
