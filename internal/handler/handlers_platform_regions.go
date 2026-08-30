// Tier 11 Phase 6 — Multi-Region / HA (PL6).
//
// HTTP route handlers for the 3 region endpoints:
//
//	GET  /api/v1/platform/regions         — ListRegions            (auth-protected)
//	POST /api/v1/platform/regions         — CreateRegion           (super_admin only)
//	GET  /api/v1/platform/regions/health  — ProbeRegionsHealth     (auth-protected)
//
// Shared types live in handlers_platform_regions_types.go (kept
// under the 400-LOC cap by the split — the type file is ~155 LOC
// and this file is ~280 LOC). The probe goroutine writes back to
// platform_regions.last_health_* columns so a subsequent
// ListRegions call surfaces "last probe was 4s ago, status=up".
//
// Why these endpoints:
//
//   ListRegions returns the platform_regions catalog. The
//   dashboard renders one tile per region with the last probe
//   status; the `is_active` filter is applied on the server so
//   deactivated regions drop from the default view.
//
//   CreateRegion adds a new region row. Restricted to
//   super_admin (a regular admin can't add infra). The handler
//   triggers an immediate health probe in a fire-and-forget
//   goroutine so the row appears in /regions/health within ~1s
//   of POST returning.
//
//   ProbeRegionsHealth probes every active region's
//   endpoint_url (2s timeout each, sequential to keep CPU
//   bounded), updates platform_regions.last_health_*, and
//   returns the snapshot inline so the dashboard renders
//   without a second fetch.
//
// Auth & tenancy: ListRegions + ProbeRegionsHealth are
// PROTECTED (RequireAuth). CreateRegion is PROTECTED +
// super_admin gated (a regular admin can't add infra).
//
// Regions are PLATFORM-WIDE — no tenant_id filter. The catalog
// is the same for every super_admin; per-tenant scoping enters
// via the (already-existing) tenants.region_code column.

package handler

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// regionProbeTimeout caps each individual probe. 2s keeps the
// /regions/health response bounded — 10 regions × 2s = 20s in
// the worst case (sequential). A future phase can parallelise
// if more regions get added.
const regionProbeTimeout = 2 * time.Second

// regionsListSelect is the shared SELECT + ORDER BY for the
// region catalog (used by ListRegions). Pulling it into a
// const keeps the WHERE / ORDER BY clauses consistent
// between the active + include_inactive branches.
const regionsListSelect = `
SELECT id, code, display_name, region_kind, endpoint_url,
       is_active, last_health_at, last_health_status,
       last_health_latency_ms, last_health_error,
       created_at, updated_at
  FROM platform_regions
 ORDER BY
   CASE region_kind WHEN 'primary' THEN 0
                    WHEN 'replica' THEN 1
                    ELSE 2 END,
   code`

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/platform/regions
// ------------------------------------------------------------------

// ListRegions returns the platform_regions catalog. The
// default returns ONLY active rows (is_active=true); a
// `?include_inactive=true` query param surfaces the full list
// for the platform_admin "deactivated regions" view.
//
// 200 → {regions: [regionRow, ...]}.
func ListRegions(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := userFromContext(c); !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		includeInactive := c.Query("include_inactive") == "true"

		// List rows. ORDER BY region_kind, code keeps the
		// dashboard deterministic — primary sites first,
		// then replica, then standby; alphabetical within
		// each kind.
		var (
			rows pgx.Rows
			err  error
		)
		if includeInactive {
			rows, err = pool.Pgx().Query(c.Request.Context(),
				regionsListSelect)
		} else {
			rows, err = pool.Pgx().Query(c.Request.Context(),
				regionsListSelect+"\n WHERE is_active = true")
		}
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		out := struct {
			Regions []regionRow `json:"regions"`
		}{}
		for rows.Next() {
			var r regionRow
			var lastAt *time.Time
			var lastStatus *string
			var lastLat *int
			var lastErr *string
			if err := rows.Scan(
				&r.ID, &r.Code, &r.DisplayName, &r.RegionKind, &r.EndpointURL,
				&r.IsActive, &lastAt, &lastStatus, &lastLat, &lastErr,
				&r.CreatedAt, &r.UpdatedAt,
			); err != nil {
				kernel.RespondError(c, err)
				return
			}
			r.LastHealthAt = lastAt
			r.LastHealthStatus = lastStatus
			r.LastHealthLatencyMS = lastLat
			r.LastHealthError = lastErr
			out.Regions = append(out.Regions, r)
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, out)
	}
}

// ------------------------------------------------------------------
// Protected endpoint: POST /api/v1/platform/regions
// ------------------------------------------------------------------

// CreateRegion adds a new region row. Restricted to
// super_admin. The handler validates:
//   - `code`         matches ^[a-z0-9-]{2,32}$ (URL-safe)
//   - `region_kind`  in {primary, replica, standby}
//   - `endpoint_url` is a valid absolute http/https URL
//
// On success it triggers an immediate health probe (fire-
// and-forget) so /regions/health surfaces the new row
// within ~1s.
//
// 201 → regionRow{...the new row...}.
// 400 → bad body / invalid code / invalid region_kind / bad URL.
// 401 → no JWT. 403 → not super_admin.
func CreateRegion(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsUser, ok := userFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		if claimsUser.Role != "super_admin" {
			kernel.RespondErrorWithCode(c, http.StatusForbidden,
				"forbidden", "super_admin role required")
			return
		}

		var req regionReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		// Trim + lowercase the code so a UI that
		// accidentally capitalises "US-East-1" gets a
		// clean "us-east-1" stored.
		req.Code = strings.ToLower(strings.TrimSpace(req.Code))
		req.DisplayName = strings.TrimSpace(req.DisplayName)
		req.EndpointURL = strings.TrimSpace(req.EndpointURL)
		req.RegionKind = strings.TrimSpace(req.RegionKind)

		if req.Code == "" || req.DisplayName == "" || req.EndpointURL == "" {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest,
				"bad_request", "code, display_name, endpoint_url are required")
			return
		}
		if !isValidRegionCode(req.Code) {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest,
				"bad_request", "code must match ^[a-z0-9-]{2,32}$")
			return
		}
		if req.RegionKind == "" {
			req.RegionKind = "primary"
		}
		if !isAllowedRegionKind(req.RegionKind) {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest,
				"bad_request", "region_kind must be primary|replica|standby")
			return
		}
		parsed, err := url.Parse(req.EndpointURL)
		if err != nil ||
			(parsed.Scheme != "http" && parsed.Scheme != "https") ||
			parsed.Host == "" {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest,
				"bad_request", "endpoint_url must be an absolute http/https URL")
			return
		}

		// INSERT the new row. Use ON CONFLICT (code) DO
		// NOTHING + RETURNING so a duplicate code is a
		// clean 409 instead of a generic 500.
		var (
			id          uuid.UUID
			displayName string
			regionKind  string
			endpointURL string
			isActive    bool
			lastAt      *time.Time
			lastStatus  *string
			lastLat     *int
			lastErrStr  *string
			createdAt   time.Time
			updatedAt   time.Time
		)
		row := pool.Pgx().QueryRow(c.Request.Context(), `
			INSERT INTO platform_regions
				(code, display_name, region_kind, endpoint_url)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (code) DO NOTHING
			RETURNING id, code, display_name, region_kind, endpoint_url,
			          is_active, last_health_at, last_health_status,
			          last_health_latency_ms, last_health_error,
			          created_at, updated_at
		`, req.Code, req.DisplayName, req.RegionKind, req.EndpointURL)
		if err := row.Scan(
			&id, &req.Code, &displayName, &regionKind, &endpointURL,
			&isActive, &lastAt, &lastStatus, &lastLat, &lastErrStr,
			&createdAt, &updatedAt,
		); err != nil {
			// ON CONFLICT DO NOTHING + no row returned =
			// duplicate code. Return 409.
			kernel.RespondErrorWithCode(c, http.StatusConflict,
				"conflict", "region code already exists")
			return
		}
		newRow := regionRow{
			ID:                  id.String(),
			Code:                req.Code,
			DisplayName:         displayName,
			RegionKind:          regionKind,
			EndpointURL:         endpointURL,
			IsActive:            isActive,
			LastHealthAt:        lastAt,
			LastHealthStatus:    lastStatus,
			LastHealthLatencyMS: lastLat,
			LastHealthError:     lastErrStr,
			CreatedAt:           createdAt,
			UpdatedAt:           updatedAt,
		}

		// Fire-and-forget probe so the new region
		// appears in /regions/health within ~1s. We
		// snapshot the URL + region_id into a copy so
		// the goroutine doesn't depend on the request
		// context (which is cancelled on response).
		go probeOneRegion(pool, id, endpointURL)

		kernel.RespondStatus(c, http.StatusCreated, newRow)
	}
}

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/platform/regions/health
// ------------------------------------------------------------------

// ProbeRegionsHealth probes every active region's endpoint_url,
// updates platform_regions.last_health_*, and returns the
// snapshot inline. Sequential probing keeps the response
// bounded — 2s timeout per region, N regions = 2N seconds max.
// Phase 8 PL8 may move the probe into a scheduler; today
// /regions/health is the entry point.
//
// 200 → regionHealthResp{regions: [...], checked_at}.
func ProbeRegionsHealth(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := userFromContext(c); !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		rows, err := pool.Pgx().Query(c.Request.Context(), `
			SELECT id, code, display_name, endpoint_url
			  FROM platform_regions
			 WHERE is_active = true
			 ORDER BY
			   CASE region_kind WHEN 'primary' THEN 0
			                    WHEN 'replica' THEN 1
			                    ELSE 2 END,
			   code
		`)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		type probeTarget struct {
			id          uuid.UUID
			code        string
			displayName string
			endpoint    string
		}
		var targets []probeTarget
		for rows.Next() {
			var t probeTarget
			if err := rows.Scan(&t.id, &t.code, &t.displayName, &t.endpoint); err != nil {
				kernel.RespondError(c, err)
				return
			}
			targets = append(targets, t)
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}

		now := time.Now().UTC()
		out := regionHealthResp{CheckedAt: now}
		for _, t := range targets {
			status, latency, errStr := probeOneRegion(pool, t.id, t.endpoint)
			row := regionHealthRow{
				Code:          t.code,
				DisplayName:   t.displayName,
				Status:        status,
				LatencyMS:     latency,
				LastCheckedAt: now.Format(time.RFC3339),
			}
			// errStr is intentionally NOT surfaced in
			// the inline response — the persisted
			// last_health_error column is the source of
			// truth (and is fetched by ListRegions).
			_ = errStr
			out.Regions = append(out.Regions, row)
		}
		kernel.RespondOK(c, out)
	}
}

// probeOneRegion + writeProbeResult live in
// handlers_platform_regions_probe.go — kept out of this
// file so it stays under the 400-LOC cap.
