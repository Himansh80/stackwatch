// Tier 11 Phase 4 — Tenant Limits (PL4).
//
// Limits package owns the per-tenant plan + catalog surface for
// Tier 11. The HTTP handlers (handlers_platform_limits.go) read
// + write through these helpers; the background RetentionWorker
// (retention.go in this package) reads per-tenant caps off the
// EffectiveLimits struct below.
//
// Why a separate package boundary (this file, not a sub-helper
// of handlers_platform_limits.go):
//
//	- limits_seed.go (this file) is called from main.go on
//	  api-gateway boot to seed the 4 built-in plans. Doing the
//	  seeding from the handler package would force main.go to
//	  import internal/handler, which already imports the rest of
//	  the app — the dependency direction would invert.
//	- The handlers need the "current plan + overrides" math at
//	  3 different read sites (GetMyLimits, GetLimitsUsage,
//	  CheckLimit). Lifting EffectiveLimits into this package
//	  means each handler is a one-liner around it.
//
// Why plan_name is the only thing stored on the limits row
// (not a foreign key to platform_plan_definitions.id):
//   The catalog can grow (a sales rep can add a "team" plan
//   with bespoke caps), but the per-tenant row only ever needs
//   the catalog's NAME — the catalog's columns are read at the
//   same time we render the dashboard. Storing the FK would
//   require a CASCADE + re-point every time we DELETE a plan
//   from the catalog, which is the opposite of what we want.
package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/stackwatch/platform/internal/db"
)

// =====================================================================
// Catalog row (platform_plan_definitions)
// =====================================================================

// PlanDefinition mirrors one row of platform_plan_definitions.
// Mirrored here (not in handler/) because the helper layer
// computes "effective limits" by joining this row with the
// per-tenant overrides row — so the struct is the carrier of
// both inputs + the merged result.
//
// Field names match the column names; the handler layer projects
// into the JSON shape (planDefinitionRow in handlers_platform_limits_types.go).
type PlanDefinition struct {
	ID                  uuid.UUID
	Name                string
	DisplayName         string
	MonthlyPriceCents   int
	Currency            string
	MaxServers          int
	MaxAlerts           int
	MaxDashboards       int
	MaxTeamMembers      int
	DataRetentionDays   int
	MetricsRetentionDays int
	StorageGBLimit      int64
	APICallsPerMinute   int
	Features            []string
	IsBuiltin           bool
	SortOrder           int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// =====================================================================
// Per-tenant row (platform_tenant_limits)
// =====================================================================

// TenantLimits mirrors one row of platform_tenant_limits.
// CustomOverrides is the raw JSONB blob — we keep it as
// json.RawMessage so the handler layer can either pass it
// through (PATCH round-trip) or interpret it (EffectiveLimits
// merge below).
type TenantLimits struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	PlanName        string
	CustomOverrides json.RawMessage
	TrialEndsAt     *time.Time
	SuspendedAt     *time.Time
	SuspendReason   *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// =====================================================================
// Effective limits (plan ∪ overrides)
// =====================================================================

// EffectiveLimits is the merged view returned to every caller.
// It combines the plan-definition columns with the per-tenant
// override map. JSON tags match the planDefinitionRow JSON
// shape so the handler layer can pass it through unchanged.
type EffectiveLimits struct {
	PlanName             string            `json:"plan_name"`
	PlanDisplayName      string            `json:"plan_display_name"`
	MonthlyPriceCents    int               `json:"monthly_price_cents"`
	Currency             string            `json:"currency"`
	MaxServers           int               `json:"max_servers"`
	MaxAlerts            int               `json:"max_alerts"`
	MaxDashboards        int               `json:"max_dashboards"`
	MaxTeamMembers       int               `json:"max_team_members"`
	DataRetentionDays    int               `json:"data_retention_days"`
	MetricsRetentionDays int               `json:"metrics_retention_days"`
	StorageGBLimit       int64             `json:"storage_gb_limit"`
	APICallsPerMinute    int               `json:"api_calls_per_minute"`
	Features             []string          `json:"features"`
	Suspended            bool              `json:"suspended"`
	SuspendReason        string            `json:"suspend_reason,omitempty"`
	TrialEndsAt          *time.Time        `json:"trial_ends_at,omitempty"`
	Overrides            map[string]int64  `json:"overrides,omitempty"`
}

// =====================================================================
// Default plan (used when a tenant has no platform_tenant_limits row)
// =====================================================================

// DefaultPlanName is the plan assigned on first PATCH when the
// tenant has no row yet. Matches the spec PL4 "every signup lands
// on free" + the existing tenants.plan default in 000_init_schema.sql.
const DefaultPlanName = "free"

// LoadEffectiveLimits fetches the tenant's plan + catalog row +
// applies overrides + returns the merged view. Returns the
// "free" plan defaults when no tenant row exists yet (first PATCH
// will create one).
//
// The function is the single read path used by GetMyLimits,
// GetLimitsUsage, and CheckLimit — keeps the merge math in one
// place instead of duplicated across 3 handlers.
func LoadEffectiveLimits(ctx context.Context, tenantID uuid.UUID, pool *db.Pool) (EffectiveLimits, error) {
	limits, plan, err := loadLimitsAndPlan(ctx, tenantID, pool)
	if err != nil {
		return EffectiveLimits{}, err
	}
	return mergeLimits(limits, plan), nil
}

// loadLimitsAndPlan reads both rows. Uses LEFT JOIN-style logic
// (two queries) instead of a JOIN so we can distinguish "no
// tenant row" (legitimate "free default" case) from "no plan
// row" (a catalog corruption that should error out loud).
func loadLimitsAndPlan(ctx context.Context, tenantID uuid.UUID, pool *db.Pool) (TenantLimits, PlanDefinition, error) {
	var tl TenantLimits
	// First: catalog row for the plan_name. If the tenant has no
	// row yet, this is the DefaultPlanName ("free").
	planName := DefaultPlanName
	err := pool.Pgx().QueryRow(ctx,
		`SELECT plan_name, custom_overrides, trial_ends_at, suspended_at, suspend_reason,
		        created_at, updated_at
		   FROM platform_tenant_limits
		  WHERE tenant_id = $1`,
		tenantID,
	).Scan(&tl.PlanName, &tl.CustomOverrides, &tl.TrialEndsAt, &tl.SuspendedAt, &tl.SuspendReason, &tl.CreatedAt, &tl.UpdatedAt)
	if err == nil {
		planName = tl.PlanName
		tl.TenantID = tenantID
	} else if err != pgx.ErrNoRows {
		return tl, PlanDefinition{}, fmt.Errorf("query platform_tenant_limits: %w", err)
	}
	// Second: catalog row.
	var pd PlanDefinition
	err = pool.Pgx().QueryRow(ctx,
		`SELECT id, name, display_name, monthly_price_cents, currency,
		        max_servers, max_alerts, max_dashboards, max_team_members,
		        data_retention_days, metrics_retention_days, storage_gb_limit,
		        api_calls_per_minute, features, is_builtin, sort_order,
		        created_at, updated_at
		   FROM platform_plan_definitions
		  WHERE name = $1`,
		planName,
	).Scan(&pd.ID, &pd.Name, &pd.DisplayName, &pd.MonthlyPriceCents, &pd.Currency,
		&pd.MaxServers, &pd.MaxAlerts, &pd.MaxDashboards, &pd.MaxTeamMembers,
		&pd.DataRetentionDays, &pd.MetricsRetentionDays, &pd.StorageGBLimit,
		&pd.APICallsPerMinute, &pd.Features, &pd.IsBuiltin, &pd.SortOrder,
		&pd.CreatedAt, &pd.UpdatedAt)
	if err != nil {
		return tl, PlanDefinition{}, fmt.Errorf("query platform_plan_definitions %q: %w", planName, err)
	}
	return tl, pd, nil
}

// mergeLimits is the pure function that combines catalog +
// overrides. Kept separate from loadLimitsAndPlan so a future
// in-memory cache (PL8 health) can reuse it.
func mergeLimits(tl TenantLimits, pd PlanDefinition) EffectiveLimits {
	out := EffectiveLimits{
		PlanName:             pd.Name,
		PlanDisplayName:      pd.DisplayName,
		MonthlyPriceCents:    pd.MonthlyPriceCents,
		Currency:             pd.Currency,
		MaxServers:           pd.MaxServers,
		MaxAlerts:            pd.MaxAlerts,
		MaxDashboards:        pd.MaxDashboards,
		MaxTeamMembers:       pd.MaxTeamMembers,
		DataRetentionDays:    pd.DataRetentionDays,
		MetricsRetentionDays: pd.MetricsRetentionDays,
		StorageGBLimit:       pd.StorageGBLimit,
		APICallsPerMinute:    pd.APICallsPerMinute,
		Features:             pd.Features,
	}
	// Suspended state lives on the tenant row, not the catalog.
	if tl.SuspendedAt != nil {
		out.Suspended = true
		if tl.SuspendReason != nil {
			out.SuspendReason = *tl.SuspendReason
		}
	}
	if tl.TrialEndsAt != nil {
		t := tl.TrialEndsAt.UTC()
		out.TrialEndsAt = &t
	}
	// Apply overrides. Empty CustomOverrides → no-op. Only
	// recognized field names are honored so a typo'd key
	// silently no-ops (the handler validates names at PATCH
	// time too).
	if len(tl.CustomOverrides) > 0 {
		var overrides map[string]int64
		if err := json.Unmarshal(tl.CustomOverrides, &overrides); err == nil {
			if v, ok := overrides["max_servers"]; ok {
				out.MaxServers = int(v)
			}
			if v, ok := overrides["max_alerts"]; ok {
				out.MaxAlerts = int(v)
			}
			if v, ok := overrides["max_dashboards"]; ok {
				out.MaxDashboards = int(v)
			}
			if v, ok := overrides["max_team_members"]; ok {
				out.MaxTeamMembers = int(v)
			}
			if v, ok := overrides["data_retention_days"]; ok {
				out.DataRetentionDays = int(v)
			}
			if v, ok := overrides["metrics_retention_days"]; ok {
				out.MetricsRetentionDays = int(v)
			}
			if v, ok := overrides["storage_gb_limit"]; ok {
				out.StorageGBLimit = v
			}
			if v, ok := overrides["api_calls_per_minute"]; ok {
				out.APICallsPerMinute = int(v)
			}
			out.Overrides = overrides
		}
	}
	return out
}

// =====================================================================
// Upsert (handler-side helper)
// =====================================================================

// UpsertTenantPlan assigns the tenant a plan (creating the
// platform_tenant_limits row on first call). customOverrides is
// MERGED into the existing map — never replaced wholesale.
//
// Merge semantics (jsonb || operator):
//   - If no row exists yet, the overrides object becomes the row.
//   - If a row exists, the new keys OVERWRITE the matching old
//     keys but other keys are preserved. This matches the
//     "PATCH partial-update" semantics the spec calls for: a
//     sales rep bumping max_servers from 50 → 100 does NOT lose
//     their custom data_retention_days override.
func UpsertTenantPlan(ctx context.Context, tenantID uuid.UUID, planName string, customOverrides map[string]int64, pool *db.Pool) error {
	overridesJSON, err := json.Marshal(customOverrides)
	if err != nil {
		return fmt.Errorf("marshal overrides: %w", err)
	}
	_, err = pool.Pgx().Exec(ctx,
		`INSERT INTO platform_tenant_limits (tenant_id, plan_name, custom_overrides)
		 VALUES ($1, $2, $3::jsonb)
		 ON CONFLICT (tenant_id) DO UPDATE
		    SET plan_name = EXCLUDED.plan_name,
		        custom_overrides = platform_tenant_limits.custom_overrides || EXCLUDED.custom_overrides,
		        updated_at = now()`,
		tenantID, planName, string(overridesJSON))
	if err != nil {
		return fmt.Errorf("upsert platform_tenant_limits: %w", err)
	}
	return nil
}

// =====================================================================
// Allowed plan names (server allowlist)
// =====================================================================

// AllowedPlanNames is the membership list for `plan_name` on
// PATCH /limits/me. Same pattern as handlers_platform_usage_types.go
// (allowedEventKinds) — a fixed map so a non-customer can't
// invent arbitrary plan values and bypass billing.
var AllowedPlanNames = map[string]struct{}{
	"free":       {},
	"starter":    {},
	"pro":        {},
	"enterprise": {},
}

// IsAllowedPlanName predicate the handler layer uses at PATCH
// validation time.
func IsAllowedPlanName(name string) bool {
	_, ok := AllowedPlanNames[name]
	return ok
}