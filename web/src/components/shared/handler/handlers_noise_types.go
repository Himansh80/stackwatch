// Tier 8 Phase 4 — Alert Deduplication + Noise Reduction (Tier 8.4).
// Shared types + access-control maps for the 6 noise / snooze endpoints.
//
// Six endpoints back the noise-reduction surface:
//
//	GET    /api/v1/noise/rules       — list noise rules (filter ?enabled)
//	POST   /api/v1/noise/rules       — create a noise rule
//	DELETE /api/v1/noise/rules/:id   — delete a noise rule
//	POST   /api/v1/noise/test        — preview suppression (what would the rule catch?)
//	POST   /api/v1/noise/snooze      — snooze an alert (inserts into snooze_log)
//	GET    /api/v1/noise/history     — snooze history (filter ?alert_id&limit)
//
// Every protected query honors tenant_id from the JWT — no cross-tenant
// data ever crosses the wire. `user_id` for snooze inserts is stamped
// from the JWT (never from the body) so a user can't snooze on behalf
// of someone else.
//
// Tables backing this surface are created by the idempotent migration
// migrations/039_noise.sql.
package handler

// Compile-time guard so unused types referenced only via JSON shape
// don't accidentally get pruned.
var _ = struct{}{}

// ------------------------------------------------------------------
// JSON row types — mirror the column shape and map directly to the
// `out` slice returned by each List* handler.
// ------------------------------------------------------------------

// noiseRuleRow is the JSON shape for a single alert noise rule
// returned by GET /noise/rules and POST /noise/rules.
type noiseRuleRow struct {
	ID                       string   `json:"id"`
	TenantID                 string   `json:"tenant_id"`
	Name                     string   `json:"name"`
	FingerprintPattern       string   `json:"fingerprint_pattern"`
	SuppressionWindowSeconds int      `json:"suppression_window_seconds"`
	Channels                 []string `json:"channels"`
	Enabled                  bool     `json:"enabled"`
	CreatedAt                string   `json:"created_at"`
}

// snoozeLogRow is the JSON shape for a single snooze entry returned
// by GET /noise/history. `expires_at` is RFC3339; `active` is a
// convenience flag computed at scan time (expires_at > now()).
type snoozeLogRow struct {
	ID              string  `json:"id"`
	TenantID        string  `json:"tenant_id"`
	AlertID         string  `json:"alert_id"`
	UserID          string  `json:"user_id"`
	DurationSeconds int     `json:"duration_seconds"`
	ExpiresAt       string  `json:"expires_at"`
	Reason          *string `json:"reason,omitempty"`
	Active          bool    `json:"active"`
	CreatedAt       string  `json:"created_at"`
}

// ------------------------------------------------------------------
// Request types — body shape for POST endpoints.
// ------------------------------------------------------------------

// noiseRuleRequest is the JSON body for POST /noise/rules.
// `name` and `fingerprint_pattern` are required. `channels` is
// optional (default '{}'). `enabled` defaults to true at the DB level.
// `suppression_window_seconds` is clamped server-side to 60..604800
// (1 minute .. 7 days).
type noiseRuleRequest struct {
	Name                     string   `json:"name"                     binding:"required,min=1,max=64"`
	FingerprintPattern       string   `json:"fingerprint_pattern"      binding:"required,min=1,max=512"`
	SuppressionWindowSeconds int      `json:"suppression_window_seconds" binding:"omitempty,min=60,max=604800"`
	Channels                 []string `json:"channels"                 binding:"omitempty,max=16,dive,oneof=email slack pagerduty webhook"`
	Enabled                  *bool    `json:"enabled"                  binding:"omitempty"`
}

// noiseTestRequest is the JSON body for POST /noise/test.
// `fingerprint_pattern` and `window_seconds` are required.
// `window_seconds` is clamped 60..604800.
type noiseTestRequest struct {
	FingerprintPattern string `json:"fingerprint_pattern" binding:"required,min=1,max=512"`
	WindowSeconds      int    `json:"window_seconds"      binding:"required,min=60,max=604800"`
}

// snoozeRequest is the JSON body for POST /noise/snooze.
// `alert_id` and `duration_seconds` are required; `reason` is optional.
// user_id is intentionally NOT in the body — we stamp it from the JWT.
type snoozeRequest struct {
	AlertID         string `json:"alert_id"         binding:"required,uuid"`
	DurationSeconds int    `json:"duration_seconds" binding:"required,min=60,max=604800"`
	Reason          string `json:"reason"           binding:"max=2048"`
}

// ------------------------------------------------------------------
// Access-control maps — whitelist valid enum values.
// ------------------------------------------------------------------

// allowedChannels is the whitelist for the `channels` enum on noise
// rules. The same set is enforced at the binding level (via `oneof`)
// AND at the SQL level (via normalizeChannels) so a hand-crafted
// request can't smuggle arbitrary strings.
var allowedChannels = map[string]bool{
	"email":     true,
	"slack":     true,
	"pagerduty": true,
	"webhook":   true,
}
