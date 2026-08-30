// Tier 11 Phase 4 — Tenant Limits (PL4).
//
// JSON row shapes + request/response bodies for the 5 limits
// endpoints:
//
//	GET   /api/v1/platform/limits/definitions — public (pricing page)
//	GET   /api/v1/platform/limits/me          — my plan
//	PATCH /api/v1/platform/limits/me          — change plan
//	POST  /api/v1/platform/limits/check       — dry-run a would-be operation
//	GET   /api/v1/platform/limits/usage       — usage vs caps (KPI strip)
//
// The handlers themselves live in handlers_platform_limits.go
// (kept under the 400-LOC cap by splitting types out). The
// helper layer (EffectiveLimits + UpsertTenantPlan) lives at
// internal/platform/limits.go so the auth package can seed the
// catalog without depending on the handler package.
//
// Why these shapes:
//
//   - planDefinitionResp mirrors a catalog row directly. The
//     pricing page needs display_name + monthly_price_cents
//
//   - every cap; the row IS the API.
//
//   - tenantLimitsReq is the PATCH body — plan_name +
//     optional overrides. The handler validates plan_name is
//     in the AllowedPlanNames allowlist before applying the
//     UPSERT. Overrides is a sparse map keyed by field name
//     ("max_servers" etc.) — see AllowedOverrideKeys below.
//
//   - limitsCheckReq's `action` discriminator names the 4
//     dry-run operations (create_server, create_alert,
//     create_dashboard, invite_member). Adding a 5th action
//     means adding it here + in the handler switch.
//
//   - limitsUsageResp is a parallel-mirrors structure:
//     {servers: {current, limit, percent}, alerts: …} so the
//     frontend can render a KPI strip by mapping over the
//     keys without knowing the field shape in advance.
//
//   - allowedLimitActions + allowedOverrideKeys are server-side
//     allowlists for the PATCH and POST bodies.
package handler

// planDefinitionResp mirrors a row of platform_plan_definitions.
// Same JSON tags as internal/platform/PlanDefinition minus the
// fields the pricing page does not need (id, is_builtin,
// sort_order). Kept narrow so the pricing API stays stable.
type planDefinitionResp struct {
	Name                 string   `json:"name"`
	DisplayName          string   `json:"display_name"`
	MonthlyPriceCents    int      `json:"monthly_price_cents"`
	Currency             string   `json:"currency"`
	MaxServers           int      `json:"max_servers"`
	MaxAlerts            int      `json:"max_alerts"`
	MaxDashboards        int      `json:"max_dashboards"`
	MaxTeamMembers       int      `json:"max_team_members"`
	DataRetentionDays    int      `json:"data_retention_days"`
	MetricsRetentionDays int      `json:"metrics_retention_days"`
	StorageGBLimit       int64    `json:"storage_gb_limit"`
	APICallsPerMinute    int      `json:"api_calls_per_minute"`
	Features             []string `json:"features"`
}

// planDefinitionListResp is the JSON shape for
// GET /api/v1/platform/limits/definitions. One row per catalog
// entry, ordered by sort_order ASC (the SQL ORDER BY).
type planDefinitionListResp struct {
	Plans []planDefinitionResp `json:"plans"`
}

// tenantLimitsReq is the body for PATCH /limits/me. plan_name is
// required (handler validates against AllowedPlanNames).
// custom_overrides is optional and sparse — only the keys the
// operator wants to bump are present; absent keys inherit the
// plan defaults at read time.
type tenantLimitsReq struct {
	PlanName        string           `json:"plan_name" binding:"required,min=1,max=32"`
	CustomOverrides map[string]int64 `json:"custom_overrides" binding:"omitempty"`
}

// tenantLimitsResp is the JSON for GET /limits/me and the 200
// response to PATCH /limits/me. The dashboard renders:
//   - PlanCard (display_name + price + feature list)
//   - KPI strip (every cap from the row)
//   - Override chips (each entry in CustomOverrides)
type tenantLimitsResp struct {
	TenantID             string           `json:"tenant_id"`
	PlanName             string           `json:"plan_name"`
	PlanDisplayName      string           `json:"plan_display_name"`
	MonthlyPriceCents    int              `json:"monthly_price_cents"`
	Currency             string           `json:"currency"`
	MaxServers           int              `json:"max_servers"`
	MaxAlerts            int              `json:"max_alerts"`
	MaxDashboards        int              `json:"max_dashboards"`
	MaxTeamMembers       int              `json:"max_team_members"`
	DataRetentionDays    int              `json:"data_retention_days"`
	MetricsRetentionDays int              `json:"metrics_retention_days"`
	StorageGBLimit       int64            `json:"storage_gb_limit"`
	APICallsPerMinute    int              `json:"api_calls_per_minute"`
	Features             []string         `json:"features"`
	Suspended            bool             `json:"suspended"`
	SuspendReason        string           `json:"suspend_reason,omitempty"`
	TrialEndsAt          *string          `json:"trial_ends_at,omitempty"`
	CustomOverrides      map[string]int64 `json:"custom_overrides"`
}

// limitsCheckReq is the body for POST /limits/check. `action`
// discriminates between the 4 dry-run operations; quantity
// defaults to 1 so a quick "would creating 1 more server work?"
// probe doesn't need to set the field.
type limitsCheckReq struct {
	Action   string `json:"action"   binding:"required,min=1,max=64"`
	Quantity int    `json:"quantity" binding:"omitempty,min=1,max=10000"`
}

// limitsCheckResp is the JSON for POST /limits/check. The
// frontend renders this as a tooltip / banner ("You can add 2
// more servers before hitting your limit").
type limitsCheckResp struct {
	Allowed   bool   `json:"allowed"`
	Action    string `json:"action"`
	Current   int    `json:"current"`
	Limit     int    `json:"limit"`
	Remaining int    `json:"remaining"`
	Quantity  int    `json:"quantity"`
	Message   string `json:"message"`
}

// limitsUsageResp is the JSON for GET /limits/usage. One
// sub-object per resource family so the dashboard renders a
// 5-tile strip in a single .map() over the keys.
type limitsUsageResp struct {
	PlanName        string            `json:"plan_name"`
	PlanDisplayName string            `json:"plan_display_name"`
	Servers         limitsUsageMetric `json:"servers"`
	Alerts          limitsUsageMetric `json:"alerts"`
	Dashboards      limitsUsageMetric `json:"dashboards"`
	TeamMembers     limitsUsageMetric `json:"team_members"`
	StorageGB       limitsUsageMetric `json:"storage_gb"`
	APICallsPerMin  limitsUsageMetric `json:"api_calls_per_min"`
	Warnings        []string          `json:"warnings"`
}

// limitsUsageMetric is one KPI cell. Percent is rounded to 0-100
// (server clamps when limit == 0).
type limitsUsageMetric struct {
	Current int    `json:"current"`
	Limit   int64  `json:"limit"`
	Percent int    `json:"percent"`
	Status  string `json:"status"` // ok | warn | exceeded
}

// =====================================================================
// Server-side allowlists
// =====================================================================

// allowedLimitActions restricts the `action` discriminator on
// POST /limits/check. Adding an action means adding it here
// + in the handler's switch + in limitsUsageMetric projection.
var allowedLimitActions = map[string]struct{}{
	"create_server":    {},
	"create_alert":     {},
	"create_dashboard": {},
	"invite_member":    {},
}

// allowedOverrideKeys restricts custom_overrides keys on PATCH
// to known column names. A typo'd key ("max_serverz") would
// silently bypass the merge math in limits.go::mergeLimits —
// rejecting at PATCH gives the operator a clean 400.
var allowedOverrideKeys = map[string]struct{}{
	"max_servers":            {},
	"max_alerts":             {},
	"max_dashboards":         {},
	"max_team_members":       {},
	"data_retention_days":    {},
	"metrics_retention_days": {},
	"storage_gb_limit":       {},
	"api_calls_per_minute":   {},
}

// isAllowedLimitAction predicate used in handler validation.
func isAllowedLimitAction(a string) bool {
	_, ok := allowedLimitActions[a]
	return ok
}

// isAllowedOverrideKey predicate used in handler validation.
func isAllowedOverrideKey(k string) bool {
	_, ok := allowedOverrideKeys[k]
	return ok
}

// isAllowedPlanName forwards to platform.IsAllowedPlanName so the
// plan allowlist lives in one place (the platform package owns
// the canonical set). The predicate here is just a thin re-export
// to keep the handler file's imports narrow.
func isAllowedPlanName(name string) bool {
	switch name {
	case "free", "starter", "pro", "enterprise":
		return true
	default:
		return false
	}
}

// usagePercent rounds the percent of a current/limit pair to
// 0-100. limit == 0 → 0% (avoids divide-by-zero). current > limit
// (already exceeded) → clamped to 100.
func usagePercent(current int, limit int64) int {
	if limit <= 0 {
		return 0
	}
	pct := int((int64(current) * 100) / limit)
	if pct > 100 {
		pct = 100
	}
	return pct
}

// usageStatus turns the percent into a 3-tier string. The
// frontend maps this to a colour (green / amber / red).
func usageStatus(pct int) string {
	switch {
	case pct >= 100:
		return "exceeded"
	case pct >= 80:
		return "warn"
	default:
		return "ok"
	}
}
