// Tier 11 Phase 4 — Tenant Limits (PL4).
//
// seed_plans.go — idempotent seeding of the 4 built-in plan
// definitions on api-gateway boot.
//
// Why in the auth package (not internal/platform):
//   The auth package is the lowest-level dependency in the
//   codebase (everything imports auth; auth imports nothing
//   except stdlib). Placing the seeder here means main.go can
//   call seedPlans BEFORE the handlers package is even loaded,
//   so the "first request after boot" sees a populated catalog.
//
//   Phase 8 will introduce a `SeedPlans` admin endpoint that
//   re-runs this same idempotent INSERT pattern — same function
//   reused, just invoked from the handler layer instead of boot.
//
// Why ON CONFLICT (name) DO NOTHING (not DO UPDATE):
//   DO NOTHING means re-running on a populated DB is a no-op.
//   Crucially, it means an operator who manually edited the
//   catalog (e.g. bumped pro's price during a Black Friday
//   promotion) is NOT silently overwritten on the next gateway
//   restart. DO UPDATE would silently revert manual edits.
//
// Why 4 plans and not 3:
//   The spec's PL4 section calls out free / starter / pro /
//   enterprise. The enterprise row has max_servers = 999999 so
//   the handler math (current/limit) shows "0%" forever —
//   equivalent to "unlimited" for UI purposes without the
//   schema needing an `unlimited` sentinel value.
//
// Why monthly_price_cents (not decimal USD):
//   Integer cents avoid float-rounding drift in billing math.
//   800 cents = $8.00. The frontend formats as USD with two
//   decimal places.
package auth

import (
	"context"
	"log/slog"

	"github.com/stackwatch/platform/internal/db"
)

// seedPlanSpec is one row in the seed catalog. The seed function
// iterates over []seedPlanSpec so adding a plan is a one-line
// change here (not a duplicate INSERT block).
type seedPlanSpec struct {
	Name                string
	DisplayName         string
	MonthlyPriceCents   int
	MaxServers          int
	MaxAlerts           int
	MaxDashboards       int
	MaxTeamMembers      int
	DataRetentionDays   int
	MetricsRetentionDays int
	StorageGBLimit      int64
	APICallsPerMinute   int
	Features            []string
	SortOrder           int
}

// builtinPlans is the 4-row seed list. Source of truth for the
// Phase 4 catalog. Order matters for SortOrder: the pricing
// page renders ASC by sort_order.
var builtinPlans = []seedPlanSpec{
	{
		Name: "free", DisplayName: "Free",
		MonthlyPriceCents: 0,
		MaxServers:        3, MaxAlerts: 10, MaxDashboards: 1, MaxTeamMembers: 1,
		DataRetentionDays:    7,
		MetricsRetentionDays: 14,
		StorageGBLimit:       1,
		APICallsPerMinute:    10,
		Features:             []string{"basic_monitoring", "email_alerts"},
		SortOrder:            10,
	},
	{
		Name: "starter", DisplayName: "Starter",
		MonthlyPriceCents: 800,
		MaxServers:        10, MaxAlerts: 50, MaxDashboards: 5, MaxTeamMembers: 3,
		DataRetentionDays:    30,
		MetricsRetentionDays: 30,
		StorageGBLimit:       10,
		APICallsPerMinute:    60,
		Features:             []string{"basic_monitoring", "email_alerts", "slack_webhooks", "synthetics_5"},
		SortOrder:            20,
	},
	{
		Name: "pro", DisplayName: "Pro",
		MonthlyPriceCents: 2400,
		MaxServers:        50, MaxAlerts: 500, MaxDashboards: 25, MaxTeamMembers: 15,
		DataRetentionDays:    90,
		MetricsRetentionDays: 90,
		StorageGBLimit:       100,
		APICallsPerMinute:    100,
		Features:             []string{"basic_monitoring", "email_alerts", "slack_webhooks", "synthetics_unlimited", "sso_saml", "audit_log"},
		SortOrder:            30,
	},
	{
		Name: "enterprise", DisplayName: "Enterprise",
		MonthlyPriceCents: 0, // contact sales — custom contract
		MaxServers:        999999, MaxAlerts: 999999, MaxDashboards: 999999, MaxTeamMembers: 999999,
		DataRetentionDays:    365,
		MetricsRetentionDays: 365,
		StorageGBLimit:       99999,
		APICallsPerMinute:    1000,
		Features: []string{
			"basic_monitoring", "email_alerts", "slack_webhooks",
			"synthetics_unlimited", "sso_saml", "audit_log",
			"multi_region", "dedicated_support", "custom_sla",
		},
		SortOrder: 40,
	},
}

// SeedBuiltinPlans is called once on api-gateway boot from
// main.go. Idempotent — re-running on a populated DB is a no-op.
//
// Errors are logged but do NOT cause the gateway to exit; the
// platform layer is not boot-critical (the dashboard will render
// "no plans" until the operator fixes the DB).
func SeedBuiltinPlans(ctx context.Context, pool *db.Pool, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	for _, p := range builtinPlans {
		_, err := pool.Pgx().Exec(ctx,
			`INSERT INTO platform_plan_definitions
			        (name, display_name, monthly_price_cents, currency,
			         max_servers, max_alerts, max_dashboards, max_team_members,
			         data_retention_days, metrics_retention_days, storage_gb_limit,
			         api_calls_per_minute, features, is_builtin, sort_order)
			 VALUES ($1, $2, $3, 'USD', $4, $5, $6, $7, $8, $9, $10, $11, $12, true, $13)
			 ON CONFLICT (name) DO NOTHING`,
			p.Name, p.DisplayName, p.MonthlyPriceCents,
			p.MaxServers, p.MaxAlerts, p.MaxDashboards, p.MaxTeamMembers,
			p.DataRetentionDays, p.MetricsRetentionDays, p.StorageGBLimit,
			p.APICallsPerMinute, p.Features, p.SortOrder,
		)
		if err != nil {
			logger.Error("seed builtin plan failed", "name", p.Name, "err", err)
		}
	}
	logger.Info("seeded builtin plan definitions", "count", len(builtinPlans))
}