// Tier 11 Phase 7 — Rate Limiting (PL7).
//
// handlers_platform_ratelimit_types.go — JSON request /
// response shapes + allowlist for the 3 ratelimit
// endpoints:
//
//	GET  /api/v1/platform/ratelimit/me      — GetMyRateLimit   (auth-protected)
//	PATCH /api/v1/platform/ratelimit/global — UpdateGlobalLimits (super_admin)
//	GET  /api/v1/platform/ratelimit/blocked — ListBlockedTenants  (super_admin)
//
// The handlers themselves live in
// handlers_platform_ratelimit.go. Splitting the types
// out keeps that file under the 400-LOC cap while
// making the request/response surface easy to scan.
//
// Why these shapes:
//
//   - ratelimitMeResp mirrors platform.tenantUsage (the
//     snapshot helper's return) field-for-field so a
//     future internal refactor can switch to a direct
//     json.Marshal without re-mapping.
//
//   - ratelimitGlobalReq is the body for PATCH /global.
//     Every key is optional — a curl caller can update
//     one plan's limit at a time. The handler validates
//     the keys against allowedRateLimitPlans (server-side
//     allowlist) and the values against
//     [1..100000] (defensive bounds).
//
//   - ratelimitGlobalResp echoes the new limits map plus
//     a `reset` flag so the dashboard "Reset to defaults"
//     button gets a clean response.
//
//   - ratelimitBlockedResp is a flat list (no envelope
//     wrapper) because the response IS the list — the
//     dashboard renders a single table of blocked tenants.
//
//   - allowedRateLimitPlans is the server-side allowlist
//     (mirrors platform.AllowedPlans). Defence-in-depth —
//     platform.Limiter.SetGlobalLimits ALSO checks, but a
//     400 at the handler layer is more honest than
//     silently dropping a stray key.

package handler

import "github.com/google/uuid"

// ratelimitMeResp is the JSON shape for GET
// /api/v1/platform/ratelimit/me. Mirrors
// platform.tenantUsage so a future refactor can drop
// the copy.
type ratelimitMeResp struct {
	TenantID            uuid.UUID `json:"tenant_id"`
	Plan                string    `json:"plan"`
	LimitPerMinute      int       `json:"limit_per_minute"`
	CurrentWindowCount  int       `json:"current_window_count"`
	Remaining           int       `json:"remaining"`
	ResetAtSeconds      int       `json:"reset_at_seconds"`
	BlockedUntilSeconds int       `json:"blocked_until_seconds"`
}

// ratelimitGlobalReq is the JSON body for
// PATCH /api/v1/platform/ratelimit/global.
//
// All fields optional — a PATCH with only `free` set
// updates the free cap and leaves the others alone. To
// reset ALL plans to defaults the caller sends an empty
// body (or a body where every key matches the default).
type ratelimitGlobalReq struct {
	Free       *int `json:"free,omitempty"`
	Starter    *int `json:"starter,omitempty"`
	Pro        *int `json:"pro,omitempty"`
	Enterprise *int `json:"enterprise,omitempty"`
}

// ratelimitGlobalResp echoes the new limits map so the
// dashboard can render "saved: free=5, starter=60..."
// without a follow-up GET.
type ratelimitGlobalResp struct {
	Limits map[string]int `json:"limits"`
	Reset  bool           `json:"reset"` // true when ResetToDefaults was called
}

// ratelimitBlockedResp is a flat list. The dashboard
// renders one row per blocked tenant; an empty list
// serialises as [] (not null) thanks to the make() in
// the handler.
type ratelimitBlockedResp struct {
	Blocked []ratelimitBlockedRow `json:"blocked"`
}

// ratelimitBlockedRow is one tenant currently in its
// blocked window. Mirrors platform.blockedTenant so a
// future refactor can switch to a direct json.Marshal.
type ratelimitBlockedRow struct {
	TenantID            uuid.UUID `json:"tenant_id"`
	Plan                string    `json:"plan"`
	CurrentCount        int       `json:"current_count"`
	LimitPerMinute      int       `json:"limit_per_minute"`
	BlockedUntilSeconds int       `json:"blocked_until_seconds"`
}

// allowedRateLimitPlans is the server-side allowlist for
// PATCH /ratelimit/global keys. Mirrors
// platform.AllowedPlans (defence-in-depth — the limiter
// also checks).
var allowedRateLimitPlans = []string{"free", "starter", "pro", "enterprise"}

// rateLimitMin + rateLimitMax bound the per-plan cap.
// Anything outside the range gets a 400 from the handler.
const (
	rateLimitMin = 1
	rateLimitMax = 100000
)
