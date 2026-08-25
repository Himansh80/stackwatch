// Tier 11 Phase 4 — Tenant Limits (PL4).
//
// Shared helpers for the 5 limits endpoints. Split out of
// handlers_platform_limits.go to keep the handler file under
// the 400-LOC cap (with the 5 handler funcs + comments the
// handler file alone was ~570 LOC).
//
// What lives here:
//   - effectiveToResp — mechanical struct-to-JSON conversion.
//   - loadRawLimits   — fetch the raw platform_tenant_limits
//                       row so CustomOverrides can be projected
//                       into the JSON response (EffectiveLimits
//                       only carries merged numeric caps).
//   - countServers / countAlerts / countDashboards /
//     countTeamMembers / storageUsageGB — per-resource-family
//                       counts used by GetLimitsUsage + CheckLimit.
//   - usageForAction  — (action → (current, limit)) lookup.
//   - buildCheckMessage / buildUsageWarnings — human-readable
//                       string builders used by CheckLimit +
//                       GetLimitsUsage.
//   - newUsageMetric / isPaidPlan / itoa64 — tiny constructors
//                       + predicates.
//
// All functions take *db.Pool rather than living in the
// platform package because they read TIER-0-7 tables (homelab_,
// alerts, dashboards, users, audit_log) and pulling those
// imports into the platform package would invert the
// dependency direction. Keeping them in the handler package
// matches the pattern set by handlers_platform_usage.go
// (countAlerts etc. live there too — handlers_platform_usage_summary.go).
package handler

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/platform"
)

// effectiveToResp converts the helper-layer struct into the
// handler-layer JSON shape. The conversion is mechanical —
// every numeric field passes through. CustomOverrides is the
// only place where we re-parse the JSONB blob into a map.
func effectiveToResp(tenantID uuid.UUID, eff platform.EffectiveLimits, raw platform.TenantLimits) tenantLimitsResp {
	out := tenantLimitsResp{
		TenantID:             tenantID.String(),
		PlanName:             eff.PlanName,
		PlanDisplayName:      eff.PlanDisplayName,
		MonthlyPriceCents:    eff.MonthlyPriceCents,
		Currency:             eff.Currency,
		MaxServers:           eff.MaxServers,
		MaxAlerts:            eff.MaxAlerts,
		MaxDashboards:        eff.MaxDashboards,
		MaxTeamMembers:       eff.MaxTeamMembers,
		DataRetentionDays:    eff.DataRetentionDays,
		MetricsRetentionDays: eff.MetricsRetentionDays,
		StorageGBLimit:       eff.StorageGBLimit,
		APICallsPerMinute:    eff.APICallsPerMinute,
		Features:             eff.Features,
		Suspended:            eff.Suspended,
		SuspendReason:        eff.SuspendReason,
	}
	if eff.TrialEndsAt != nil {
		s := eff.TrialEndsAt.UTC().Format("2006-01-02T15:04:05Z")
		out.TrialEndsAt = &s
	}
	// raw.CustomOverrides is json.RawMessage. Empty → empty
	// map (so the JSON serializes as {} not null). Non-empty
	// → unmarshal into the int64 map.
	out.CustomOverrides = map[string]int64{}
	if len(raw.CustomOverrides) > 0 {
		var m map[string]int64
		_ = json.Unmarshal(raw.CustomOverrides, &m)
		if m != nil {
			out.CustomOverrides = m
		}
	}
	return out
}

// loadRawLimits is a thin scan of the platform_tenant_limits row
// — used by the handlers to recover CustomOverrides (which the
// EffectiveLimits helper does not surface in its merged form).
// Returns an empty struct on pgx.ErrNoRows so the caller can
// still render a "free" response.
func loadRawLimits(ctx context.Context, tenantID uuid.UUID, pool *db.Pool) (platform.TenantLimits, error) {
	var tl platform.TenantLimits
	err := pool.Pgx().QueryRow(ctx,
		`SELECT id, tenant_id, plan_name, custom_overrides,
		        trial_ends_at, suspended_at, suspend_reason,
		        created_at, updated_at
		   FROM platform_tenant_limits
		  WHERE tenant_id = $1`,
		tenantID,
	).Scan(&tl.ID, &tl.TenantID, &tl.PlanName, &tl.CustomOverrides,
		&tl.TrialEndsAt, &tl.SuspendedAt, &tl.SuspendReason,
		&tl.CreatedAt, &tl.UpdatedAt)
	if err != nil && err != pgx.ErrNoRows {
		return tl, err
	}
	if err == pgx.ErrNoRows {
		// Synthesize an empty row so the merge math doesn't
		// have to special-case "no row".
		tl.PlanName = "free"
		tl.CustomOverrides = []byte("{}")
		tl.TenantID = tenantID
	}
	return tl, nil
}

// countServers returns the current server count for the tenant.
// Errors collapse to 0 (the dashboard renders "0" rather than
// failing the whole response on a transient query hiccup).
func countServers(ctx context.Context, tenantID uuid.UUID, pool *db.Pool) (int, error) {
	var n int
	err := pool.Pgx().QueryRow(ctx,
		`SELECT COUNT(*)::integer FROM homelab_servers WHERE tenant_id = $1`,
		tenantID,
	).Scan(&n)
	return n, err
}

// countAlerts returns the current alert count.
func countAlerts(ctx context.Context, tenantID uuid.UUID, pool *db.Pool) (int, error) {
	var n int
	err := pool.Pgx().QueryRow(ctx,
		`SELECT COUNT(*)::integer FROM alerts WHERE tenant_id = $1`,
		tenantID,
	).Scan(&n)
	return n, err
}

// countDashboards returns the current dashboard count.
func countDashboards(ctx context.Context, tenantID uuid.UUID, pool *db.Pool) (int, error) {
	var n int
	err := pool.Pgx().QueryRow(ctx,
		`SELECT COUNT(*)::integer FROM dashboards WHERE tenant_id = $1`,
		tenantID,
	).Scan(&n)
	return n, err
}

// countTeamMembers returns the current user count for the tenant.
func countTeamMembers(ctx context.Context, tenantID uuid.UUID, pool *db.Pool) (int, error) {
	var n int
	err := pool.Pgx().QueryRow(ctx,
		`SELECT COUNT(*)::integer FROM users WHERE tenant_id = $1`,
		tenantID,
	).Scan(&n)
	return n, err
}

// storageUsageGB returns the tenant's storage usage in GB. Today
// we approximate via the sum of audit_log row sizes (the most
// reliable tenant-scoped storage signal we have across the
// schema). A future Phase 5 polish will swap this for a real
// pg_database_size call.
func storageUsageGB(ctx context.Context, tenantID uuid.UUID, pool *db.Pool) (int, error) {
	var bytes int64
	err := pool.Pgx().QueryRow(ctx,
		`SELECT COALESCE(SUM(octet_length(payload::text)), 0)::bigint
		   FROM audit_log
		  WHERE tenant_id = $1`,
		tenantID,
	).Scan(&bytes)
	if err != nil {
		return 0, err
	}
	// bytes / (1024^3) → GB. Ceiling so a single small audit
	// row counts as 1 GB rather than 0 — the dashboard
	// expects "you're using 1 GB" not "you're using 0 GB".
	gb := int((bytes + 1073741823) / 1073741824)
	return gb, nil
}

// usageForAction resolves the (current, limit) pair for the
// requested action. Returns limit = 0 → caller treats as
// "no limit configured" (the response shows allowed=true).
func usageForAction(ctx context.Context, action string, tenantID uuid.UUID, eff platform.EffectiveLimits, pool *db.Pool) (int, int64) {
	switch action {
	case "create_server":
		n, _ := countServers(ctx, tenantID, pool)
		return n, int64(eff.MaxServers)
	case "create_alert":
		n, _ := countAlerts(ctx, tenantID, pool)
		return n, int64(eff.MaxAlerts)
	case "create_dashboard":
		n, _ := countDashboards(ctx, tenantID, pool)
		return n, int64(eff.MaxDashboards)
	case "invite_member":
		n, _ := countTeamMembers(ctx, tenantID, pool)
		return n, int64(eff.MaxTeamMembers)
	}
	return 0, 0
}

// buildCheckMessage renders the human-readable "would this
// exceed?" message. Branching on allowed makes the negative
// case concrete ("you're at 3/3") and the positive case
// reassuring ("3 used, 0 remaining in your free tier").
func buildCheckMessage(action string, current, limit, remaining int, allowed bool, plan string) string {
	if allowed {
		if limit <= 0 {
			return "no cap configured for " + action + " on " + plan
		}
		if remaining == 0 {
			return "you have no remaining capacity for " + action + " (at limit)"
		}
		return "you can create " + strconv.Itoa(remaining) + " more " + action + " before hitting your " + plan + " limit"
	}
	return "your " + plan + " plan only allows " + strconv.Itoa(limit) + " " + action +
		" and you currently have " + strconv.Itoa(current)
}

// buildUsageWarnings turns the resp into a flat list of
// human-readable warnings for the billing-banner UI. Only
// status == "warn" or "exceeded" surfaces a string; "ok" cells
// are silent (otherwise the banner would always be full of
// "you're at 30% of your server limit" noise).
func buildUsageWarnings(r limitsUsageResp) []string {
	var w []string
	add := func(name string, m limitsUsageMetric) {
		if m.Status == "ok" {
			return
		}
		verb := "approaching"
		if m.Status == "exceeded" {
			verb = "over"
		}
		w = append(w, "you're "+verb+" your "+name+" limit ("+strconv.Itoa(m.Current)+"/"+itoa64(m.Limit)+")")
	}
	add("server", r.Servers)
	add("alert", r.Alerts)
	add("dashboard", r.Dashboards)
	add("team-member", r.TeamMembers)
	add("storage GB", r.StorageGB)
	return w
}

// newUsageMetric is a 3-line constructor so the GetLimitsUsage
// handler stays readable.
func newUsageMetric(current int, limit int64) limitsUsageMetric {
	pct := usagePercent(current, limit)
	return limitsUsageMetric{
		Current: current,
		Limit:   limit,
		Percent: pct,
		Status:  usageStatus(pct),
	}
}

// isPaidPlan is a tiny predicate for the downgrade-detection
// logic in PatchMyLimits. "free" is the only non-paid plan.
func isPaidPlan(name string) bool {
	switch name {
	case "starter", "pro", "enterprise":
		return true
	default:
		return false
	}
}

// itoa64 is strconv.FormatInt(n, 10) for int64 — kept inline
// so the warning strings don't have to import strconv twice.
func itoa64(n int64) string {
	return strconv.FormatInt(n, 10)
}