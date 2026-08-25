// Tier 11 Phase 7 — Rate Limiting (PL7).
//
// ratelimit_types.go — JSON shapes + sentinel lists for the
// rate limiter.
//
// The Limiter (token-bucket manager) lives in ratelimit.go.
// This file holds the public-ish types (tokenBucket,
// blockedTenant, tenantUsage) + the per-plan default map +
// the allowed-plan list. Split out so ratelimit.go stays
// under the 400-LOC cap while the types stay grouped for
// any future Redis-backed adapter to import.
//
// Why these shapes:
//
//   - tokenBucket is the in-memory struct held by
//     Limiter.buckets (NOT exported — internal package
//     detail).
//
//   - blockedTenant is the JSON row for
//     GET /platform/ratelimit/blocked (one per tenant
//     currently in its blocked window).
//
//   - tenantUsage is the JSON row for
//     GET /platform/ratelimit/me (a read-only peek at the
//     caller's own bucket — never consumes a token).
//
//   - PlanDefaults is the seed map. Matches the
//     api_calls_per_minute column in
//     auth.builtinPlans so a tenant's plan badge and their
//     bucket cap stay aligned.
//
//   - AllowedPlans is the set of plan names PL7
//     recognises. Mirrors the limits_handler allowlist
//     (defence-in-depth — the handler also checks).

package platform

import (
	"time"

	"github.com/google/uuid"
)

// Default per-plan limits (requests per minute). Kept here
// (not in seed_plans.go) because PL7 is in-memory and
// must start with a non-zero map BEFORE the DB-backed
// plan catalog is even read at boot. Matches
// auth.builtinPlans[*].APICallsPerMinute so a tenant's
// plan badge and their bucket cap stay aligned.
var PlanDefaults = map[string]int{
	"free":       10,
	"starter":    60,
	"pro":        100,
	"enterprise": 1000,
}

// AllowedPlans is the set of plan names PL7 recognises.
// Anything outside this set is treated as 'free' so a
// future plan addition can't accidentally grant a
// free-tier quota. Mirrors the limits_handler allowlist
// (handlers_platform_limits_helpers.go).
var AllowedPlans = []string{"free", "starter", "pro", "enterprise"}

// DefaultPlan is the fallback when a JWT carries no plan
// (a freshly-minted user before tenants.plan is set, or
// a tenant with plan=NULL — the schema column is
// nullable).
const DefaultPlan = "free"

// tokenBucket is the per-tenant in-memory bucket.
//
// Tokens are a float64 so partial refills compose cleanly
// ("free=10/min refills 1 token every 6s" needs float
// precision to sum fractional refills across the minute).
// lastRefill tracks when we last advanced the clock; the
// bucket is FULL on first use (tokens == capacity) so the
// first burst is allowed up to the plan's per-minute cap.
type tokenBucket struct {
	tokens      float64
	capacity    int
	lastRefill  time.Time
	blockedTill time.Time
}

// blockedTenant is the JSON-shape row used by the
// /platform/ratelimit/blocked endpoint. Constructed
// from a live bucket so the dashboard can show the actual
// current count + remaining time + limit per plan.
type blockedTenant struct {
	TenantID            uuid.UUID `json:"tenant_id"`
	Plan                string    `json:"plan"`
	CurrentCount        int       `json:"current_count"`
	LimitPerMinute      int       `json:"limit_per_minute"`
	BlockedUntilSeconds int       `json:"blocked_until_seconds"`
}

// tenantUsage is the JSON-shape row used by the
// /platform/ratelimit/me endpoint (one row per call,
// never persisted).
type tenantUsage struct {
	TenantID            uuid.UUID `json:"tenant_id"`
	Plan                string    `json:"plan"`
	LimitPerMinute      int       `json:"limit_per_minute"`
	CurrentWindowCount  int       `json:"current_window_count"`
	Remaining           int       `json:"remaining"`
	ResetAtSeconds      int       `json:"reset_at_seconds"`
	BlockedUntilSeconds int       `json:"blocked_until_seconds"`
}
