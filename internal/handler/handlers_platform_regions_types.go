// Tier 11 Phase 6 — Multi-Region / HA (PL6).
//
// JSON row shapes + request/response bodies for the 3 region
// endpoints:
//
//	GET  /api/v1/platform/regions         — ListRegions   (auth-protected)
//	POST /api/v1/platform/regions         — CreateRegion  (super_admin only)
//	GET  /api/v1/platform/regions/health  — ProbeRegionsHealth (auth-protected)
//
// The handlers themselves live in handlers_platform_regions.go
// (kept under the 400-LOC cap by the split — 3 routes + a
// fire-and-forget probe goroutine fit in ~300 LOC). Storage is
// migrations/042_platform.sql (Phase 6 tail: `platform_regions` +
// `platform_region_replicas`).
//
// Why these shapes:
//
//   - regionRow mirrors the platform_regions row directly minus
//     a few handler-irrelevant columns (last_health_error is
//     surfaced only when a probe failed — non-null signals the
//     dashboard to render the "last failed: …" tooltip).
//
//   - regionReq is the JSON body for POST /regions. Code +
//     display_name + endpoint_url are required; region_kind
//     defaults to 'primary'. The regex check on `code`
//     (^[a-z0-9-]{2,32}$) is server-side — a UI-only check
//     would let a curl caller smuggle in arbitrary strings.
//
//   - regionHealthResp is the JSON for GET /regions/health.
//     One row per region with the current probe result so the
//     dashboard renders the status pill without a follow-up
//     fetch to /regions.
//
//   - allowedRegionKinds restricts region_kind to
//     ('primary' | 'replica' | 'standby'). A future
//     'edge' kind is intentionally left out until the
//     edge-cache layer ships.
//
//   - regionCodeRegex is compiled once at package init (via
//     the `^`/`$` anchors + MustCompile wrapper below) so
//     every POST reuses the same compiled pattern instead of
//     re-compiling per request.

package handler

import (
	"regexp"
	"time"
)

// regionRow mirrors one row of platform_regions. JSON tags
// match the column names so a future endpoint can switch to
// straight json.Marshal over the row without re-mapping.
type regionRow struct {
	ID                  string     `json:"id"`
	Code                string     `json:"code"`
	DisplayName         string     `json:"display_name"`
	RegionKind          string     `json:"region_kind"`
	EndpointURL         string     `json:"endpoint_url"`
	IsActive            bool       `json:"is_active"`
	LastHealthAt        *time.Time `json:"last_health_at,omitempty"`
	LastHealthStatus    *string    `json:"last_health_status,omitempty"`
	LastHealthLatencyMS *int       `json:"last_health_latency_ms,omitempty"`
	LastHealthError     *string    `json:"last_health_error,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// regionReq is the JSON body for POST /api/v1/platform/regions.
// `code` is the URL-safe lowercase handle (`us-east-1`,
// `eu-west-2`); `display_name` is the human-readable label;
// `region_kind` defaults to 'primary' but accepts 'replica' +
// 'standby'; `endpoint_url` is the absolute URL the probe
// path will GET to verify region health.
type regionReq struct {
	Code        string `json:"code"         binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
	RegionKind  string `json:"region_kind"  binding:"omitempty"`
	EndpointURL string `json:"endpoint_url" binding:"required"`
}

// regionHealthRow is the JSON shape for one region in the
// /regions/health response. Decoupled from regionRow because
// the probe response is intentionally lighter — no
// endpoint_url, no created_at, no last_health_latency_ms
// (latency is computed by the probe, not stored).
type regionHealthRow struct {
	Code          string `json:"code"`
	DisplayName   string `json:"display_name"`
	Status        string `json:"status"`          // up | degraded | down | unknown
	LatencyMS     int    `json:"latency_ms"`      // ms from probe; 0 if unknown
	LastCheckedAt string `json:"last_checked_at"` // RFC3339 or ""
}

// regionHealthResp is the JSON envelope for GET
// /regions/health. The dashboard renders one tile per row.
type regionHealthResp struct {
	Regions   []regionHealthRow `json:"regions"`
	CheckedAt time.Time         `json:"checked_at"`
}

// =====================================================================
// Server-side allowlists + validators
// =====================================================================

// allowedRegionKinds restricts region_kind to a fixed set.
// 'primary' = the main instance; 'replica' = a hot standby
// receiving streaming WAL; 'standby' = a DR site receiving
// periodic base backups. A future 'edge' kind (read-only
// cache layer) is intentionally left out until the edge-cache
// layer ships.
var allowedRegionKinds = map[string]struct{}{
	"primary": {},
	"replica": {},
	"standby": {},
}

// regionCodeRegex enforces the lowercase URL-safe handle
// format. 2-32 chars, lowercase letters + digits + dashes
// only. The regex is compiled once at package init via
// mustCompileRegionCodeRegex below.
var regionCodeRegex = regexp.MustCompile(`^[a-z0-9-]{2,32}$`)

// isAllowedRegionKind predicate used in CreateRegion to
// validate the `region_kind` field if the operator wants
// to override the default ('primary').
func isAllowedRegionKind(k string) bool {
	_, ok := allowedRegionKinds[k]
	return ok
}

// isValidRegionCode predicate used in CreateRegion to
// validate the `code` field against the lowercase URL-safe
// regex.
func isValidRegionCode(code string) bool {
	return regionCodeRegex.MatchString(code)
}
