// Tier 10 Phase 1 — Widget Framework + Page shell (Tier 10.1).
//
// HTTP route handlers for the per-user preferences endpoints. These
// 3 of the 6 Phase 1 protected routes back the per-user theme,
// refresh interval, pinned widgets, and default landing page.
//
//	GET    /api/v1/homelab/prefs        — GetHomelabPrefs
//	PUT    /api/v1/homelab/prefs        — PutHomelabPrefs
//	DELETE /api/v1/homelab/prefs/layout — DeleteHomelabPrefsLayout
//
// The 3 layout endpoints (GET/PUT /homelab/layout +
// GET /homelab/layout/defaults) live in handlers_homelab_layout.go.
//
// Storage: one row per (tenant_id, user_id) in homelab_user_prefs
// (migrations/041_homelab.sql). First-run UX: GET /prefs lazy-creates
// the row with platform defaults so the UI never has to defend
// against 404. DELETE /prefs/layout resets ONLY the layout — prefs
// (theme/refresh/pinned/landing) survive a layout reset.
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ------------------------------------------------------------------
// JSON shapes — see migration 041_homelab.sql for the column shape.
// ------------------------------------------------------------------

// homelabPrefsRow is the JSON shape returned by GET /prefs and
// PUT /prefs. `pinned_widgets` is the list of widget-type ids the
// user has pinned to their KPI strip ('services', 'notes', etc.).
// `default_landing` is the route the user lands on after sign-in
// ('homelab' / 'dashboard' / ...).
type homelabPrefsRow struct {
	TenantID       string   `json:"tenant_id"`
	UserID         string   `json:"user_id"`
	Theme          string   `json:"theme"`
	RefreshSeconds int      `json:"refresh_seconds"`
	PinnedWidgets  []string `json:"pinned_widgets"`
	DefaultLanding string   `json:"default_landing"`
	UpdatedAt      string   `json:"updated_at"`
}

// homelabPrefsReq is the JSON body for PUT /prefs. All fields are
// optional — the handler does a partial update (only updates the
// fields the caller sent). Use `binding:"omitempty"` + handler-side
// validation for the enums.
type homelabPrefsReq struct {
	Theme          *string   `json:"theme,omitempty"`
	RefreshSeconds *int      `json:"refresh_seconds,omitempty"`
	PinnedWidgets  *[]string `json:"pinned_widgets,omitempty"`
	DefaultLanding *string   `json:"default_landing,omitempty"`
}

// ------------------------------------------------------------------
// Allowed values — enforce on both sides of the wire. The DB column
// has a TEXT default but no CHECK constraint (the platform wants
// the freedom to add new themes / landing pages without an ALTER
// TABLE) — so handler-side validation is the only line of defense.
// ------------------------------------------------------------------

// allowedThemes is the whitelist for the `theme` enum. Adding a new
// theme (e.g. 'sepia') means appending here AND updating the
// frontend tokens — both happen together in one PR.
var allowedThemes = map[string]bool{
	"light": true,
	"dark":  true,
	"auto":  true,
}

// minRefreshSeconds / maxRefreshSeconds clamp the polling interval
// to 30s..300s. Below 30s is wasteful (DDoS the backend) and above
// 300s the dashboard feels dead. Future tiers can revisit these
// limits; they're central to the user-experience contract.
const (
	minRefreshSeconds = 30
	maxRefreshSeconds = 300
)

// pinnedWidgetAllowlist is the closed set of widget types a user
// can pin to their KPI strip. Adding a new pinned widget means
// appending here (the layout handler has its own allowlist for
// widget types in the layout JSONB — they're separate concerns).
//
// Phase 1 only knows the four KPI widgets that ship in this phase;
// Phase 2+ will append more as their widgets land.
var pinnedWidgetAllowlist = map[string]bool{
	"kpi-services": true,
	"kpi-notes":    true,
	"kpi-todos":    true,
	"kpi-rss":      true,
}

// allowedDefaultLandings is the whitelist for the `default_landing`
// enum. Routes are validated to be one of the known top-level
// paths so we never redirect a user to a 404.
var allowedDefaultLandings = map[string]bool{
	"homelab":      true,
	"dashboard":    true,
	"intelligence": true,
	"enterprise":   true,
	"profile":      true,
}

// ------------------------------------------------------------------
// Internal helper — render a row after INSERT/UPDATE.
// ------------------------------------------------------------------

// homelabPrefsFromRow is a tiny adapter so both Get* and Put* share
// the same response shape. Returns the row as JSON-ready struct.
func homelabPrefsFromRow(
	tenantID, userID, theme string,
	refresh int,
	pinned []string,
	landing, updatedAt string,
) homelabPrefsRow {
	if pinned == nil {
		pinned = []string{}
	}
	return homelabPrefsRow{
		TenantID:       tenantID,
		UserID:         userID,
		Theme:          theme,
		RefreshSeconds: refresh,
		PinnedWidgets:  pinned,
		DefaultLanding: landing,
		UpdatedAt:      updatedAt,
	}
}

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/homelab/prefs
// ------------------------------------------------------------------

// GetHomelabPrefs returns the caller's preferences. If no row
// exists yet (first run for this user), the handler UPSERTs the
// default row and returns it. Auto-create semantics mean the
// UI never has to call PUT before the first GET — saves a round-trip
// on every fresh sign-in.
//
// Honors tenant_id + user_id from the JWT — never returns another
// user's prefs even if a row id leaks through.
func GetHomelabPrefs(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		userID, ok := auth.UserIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		// UPSERT the default row, then RETURN it. Single round-trip
		// covers both first-run (INSERT) and steady-state (no-op UPDATE
		// that returns the existing row).
		var (
			theme      string
			refresh    int
			pinned     []string
			landing    string
			updatedAt  string
		)
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO homelab_user_prefs (tenant_id, user_id)
			 VALUES ($1, $2)
			 ON CONFLICT (tenant_id, user_id) DO UPDATE
			   SET tenant_id = EXCLUDED.tenant_id   -- no-op so RETURNING fires
			 RETURNING theme, refresh_seconds, pinned_widgets,
			           default_landing, updated_at::text`,
			tenantID, userID,
		).Scan(&theme, &refresh, &pinned, &landing, &updatedAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		kernel.RespondOK(c, homelabPrefsFromRow(
			tenantID.String(), userID.String(),
			theme, refresh, pinned, landing, updatedAt,
		))
	}
}

// ------------------------------------------------------------------
// Protected endpoint: PUT /api/v1/homelab/prefs
// ------------------------------------------------------------------

// PutHomelabPrefs updates the caller's preferences. Partial-update
// semantics: only the fields present in the body are written; the
// others keep their existing values. This means the UI can PUT just
// `{theme: "dark"}` and not have to round-trip the other fields.
//
// Validation rules (enforced BEFORE the SQL write):
//   - theme, if present, must be in allowedThemes
//   - refresh_seconds, if present, must be in [minRefreshSeconds, maxRefreshSeconds]
//   - pinned_widgets, if present, every entry must be in pinnedWidgetAllowlist
//   - default_landing, if present, must be in allowedDefaultLandings
//
// If the request body is empty (`{}`), the handler is a no-op —
// returns 200 with the current row.
func PutHomelabPrefs(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		userID, ok := auth.UserIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		var req homelabPrefsReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		// Validate enums + ranges BEFORE the SQL write so we never
		// touch the DB on a bad request.
		if req.Theme != nil && !allowedThemes[*req.Theme] {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"theme must be one of: light, dark, auto")
			return
		}
		if req.RefreshSeconds != nil {
			r := *req.RefreshSeconds
			if r < minRefreshSeconds || r > maxRefreshSeconds {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
					"refresh_seconds must be in [30, 300]")
				return
			}
		}
		if req.DefaultLanding != nil && !allowedDefaultLandings[*req.DefaultLanding] {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"default_landing must be a known route (homelab/dashboard/intelligence/enterprise/profile)")
			return
		}
		if req.PinnedWidgets != nil {
			for _, w := range *req.PinnedWidgets {
				if !pinnedWidgetAllowlist[w] {
					kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
						"pinned_widgets contains unknown widget type: "+w)
					return
				}
			}
		}

		// Build a dynamic UPDATE. We use COALESCE so unmentioned
		// fields keep their existing values. The DO NOTHING branch
		// in ON CONFLICT (NOTHING) is intentional — we want the
		// RETURNING to surface the (potentially empty) row even
		// when the user PUTs an empty body, so a fresh-user PUT
		// creates a row with all defaults if none exists yet.
		var (
			theme      string
			refresh    int
			pinned     []string
			landing    string
			updatedAt  string
		)
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO homelab_user_prefs (tenant_id, user_id)
			 VALUES ($1, $2)
			 ON CONFLICT (tenant_id, user_id) DO UPDATE
			   SET theme           = COALESCE($3, homelab_user_prefs.theme),
			       refresh_seconds = COALESCE($4, homelab_user_prefs.refresh_seconds),
			       pinned_widgets  = COALESCE($5, homelab_user_prefs.pinned_widgets),
			       default_landing = COALESCE($6, homelab_user_prefs.default_landing),
			       updated_at      = NOW()
			 RETURNING theme, refresh_seconds, pinned_widgets,
			           default_landing, updated_at::text`,
			tenantID, userID,
			req.Theme,
			req.RefreshSeconds,
			req.PinnedWidgets,
			req.DefaultLanding,
		).Scan(&theme, &refresh, &pinned, &landing, &updatedAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		kernel.RespondOK(c, homelabPrefsFromRow(
			tenantID.String(), userID.String(),
			theme, refresh, pinned, landing, updatedAt,
		))
	}
}

// ------------------------------------------------------------------
// Protected endpoint: DELETE /api/v1/homelab/prefs/layout
// ------------------------------------------------------------------

// DeleteHomelabPrefsLayout resets the caller's layout to defaults.
// Implemented as DELETE on homelab_user_layouts (no row → next GET
// returns the default). Theme / refresh / pinned / default_landing
// in homelab_user_prefs are UNTOUCHED — only the layout vanishes.
//
// Why two tables: layout (drag-resize-reorder widget positions) is
// separate from prefs (theme, refresh, pinned KPI list, default
// landing) so a "reset layout" never accidentally resets the
// user's theme or refresh cadence. Per US-8 in the speckit
// proposal, layouts and prefs are independent knobs.
//
// Returns 200 with `{reset: true, message: "layout reset"}` even
// if no row existed to delete (idempotent DELETE).
func DeleteHomelabPrefsLayout(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		userID, ok := auth.UserIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`DELETE FROM homelab_user_layouts
			  WHERE tenant_id = $1 AND user_id = $2`,
			tenantID, userID,
		)
		if err != nil {
			// pgx.ErrNoRows can't happen on DELETE — only Exec-level
			// errors reach here. Treat any unexpected error as a 500.
			if !errors.Is(err, pgx.ErrNoRows) {
				kernel.RespondError(c, err)
				return
			}
		}

		// 200 even if RowsAffected is 0 — the next GET /layout will
		// return the default layout, which IS the post-reset state.
		kernel.RespondOK(c, gin.H{
			"reset":         true,
			"message":       "layout reset to defaults",
			"rows_deleted":  tag.RowsAffected(),
			"next_get_url":  "/api/v1/homelab/layout",
		})
	}
}