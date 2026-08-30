// Tier 11 Phase 1 — Push-Button Deploy (PL1).
// JSON row shapes + request body for the 3 deploy endpoints.
//
// The handlers themselves live in handlers_platform_deploy.go
// (CreateInstallToken / GetInstallScript / GetInstallStats). The
// platform helper (GenerateInstallToken / LookupInstallToken /
// MarkInstallTokenUsed) lives in internal/platform/deploy.go.
//
// Splitting types out keeps handlers_platform_deploy.go under the
// 400-LOC cap while still letting the handler file focus on the
// HTTP-boundary logic (claim extraction, validation, error
// mapping).
package handler

import "time"

// installTokenLifetime is the validity window of a freshly-
// minted install token. Matches the spec PL1 §"Tokens expire in
// 1 hour" + the spec plan.md §"Phase 1" line that says "expires
// in 1 hour". Stored as a const so a future config knob (env
// var, per-tenant override) can be wired without touching every
// call site.
const installTokenLifetime = 1 * time.Hour

// allowedLabelChars is the regex applied to the `label` field on
// POST /install-token. The label is operator-supplied (it shows up
// in the UI token table as a friendly name like "edge-node-01")
// and we don't want a hand-crafted value to smuggle in shell
// metacharacters or path separators that might leak into a future
// admin dashboard render. The regex keeps it boring: letters,
// digits, space, dash, underscore.
//
//	^[A-Za-z0-9 _-]{0,64}$
const allowedLabelChars = `^[A-Za-z0-9 _-]{0,64}$`

// installTokenRow is the JSON shape returned by:
//   - POST /api/v1/platform/deploy/install-token (the create
//     response, which additionally carries `token` plaintext
//     exactly once)
//   - GET /api/v1/platform/deploy/stats (one row per active
//     token when joined with stats later)
//
// and the row shape stored in deploy_install_tokens (the SQL
// SELECT in installTokenLookup mirrors these field names — the
// `label` column is nullable so it's a pointer).
type installTokenRow struct {
	ID        string     `json:"id"`
	TenantID  string     `json:"tenant_id"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	UsedByIP  *string    `json:"used_by_ip,omitempty"`
	Label     *string    `json:"label,omitempty"`
	// Token is ONLY populated on the create response. Never on
	// list / lookup — it's the one-time plaintext.
	Token string `json:"token,omitempty"`
}

// installTokenReq is the JSON body for POST /install-token.
// `label` is optional (max 64 chars, regex-validated); an empty
// string is fine and produces a row with `label = NULL`.
type installTokenReq struct {
	Label string `json:"label" binding:"omitempty,max=64"`
}

// installStats is the JSON shape returned by GET
// /api/v1/platform/deploy/stats. Aggregated counts across the
// caller's tenant so the DeploySection KPI strip can render
// "Tokens active / Installs 24h / 7d / 30d" without needing to
// cross-query.
type installStats struct {
	Installs24h  int `json:"installs_24h"`
	Installs7d   int `json:"installs_7d"`
	Installs30d  int `json:"installs_30d"`
	TokensActive int `json:"tokens_active"`
	TokensTotal  int `json:"tokens_total"`
}
