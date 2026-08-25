// Tier 10 Phase 1 — Widget Framework + Page shell (Tier 10.1).
//
// HTTP route handlers for the per-user grid-layout endpoints. These
// 3 of the 6 Phase 1 protected routes back the HomelabGrid React
// component (drag-resize-reorder widget grid on the HomelabPage).
//
//	GET    /api/v1/homelab/layout          — GetHomelabLayout
//	PUT    /api/v1/homelab/layout          — PutHomelabLayout
//	GET    /api/v1/homelab/layout/defaults — GetHomelabLayoutDefaults
//
// The 3 other Phase 1 endpoints (GET/PUT /homelab/prefs +
// DELETE /homelab/prefs/layout) live in handlers_homelab_prefs.go.
//
// Storage: one row per (tenant_id, user_id) in homelab_user_layouts
// (migrations/041_homelab.sql). The `layout` column is a JSONB array
// of widget descriptors: [{i, x, y, w, h, type, config}, ...].
//
// First-run UX: GET /layout returns the DEFAULT layout (returned by
// /layout/defaults) when the user has never saved one, instead of
// 404. This keeps the HomelabGrid loading state predictable.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ------------------------------------------------------------------
// JSON shapes — see migration 041_homelab.sql for the column shape.
// ------------------------------------------------------------------

// homelabLayoutWidget is the JSON shape of one widget descriptor in
// the `layout` JSONB array. Matches react-grid-layout's lgLayout
// entry shape plus our `type` + `config` extensions.
//
// Fields:
//
//	i      — stable React key (also the widget id when type=='custom')
//	x, y   — grid column / row (12-col grid; lg breakpoint)
//	w, h   — grid width / height (col units / row units)
//	type   — widget type id ('kpi-services', 'notes', 'todos', ...)
//	config — widget-specific overrides (e.g. { "color": "cyan" })
//
// The handler is permissive about extra fields — react-grid-layout
// may add `minW`, `maxH`, `static`, etc. in future and we don't
// want to break the round-trip.
type homelabLayoutWidget struct {
	I      string                 `json:"i"`
	X      int                    `json:"x"`
	Y      int                    `json:"y"`
	W      int                    `json:"w"`
	H      int                    `json:"h"`
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// homelabLayoutReq is the JSON body for PUT /api/v1/homelab/layout.
// The `layout` array replaces the stored JSONB blob wholesale — we
// do not merge per-widget. The frontend HomelabGrid owns the merge
// logic and PUTs the full new layout on every change.
type homelabLayoutReq struct {
	Layout []homelabLayoutWidget `json:"layout" binding:"required"`
}

// homelabLayoutResponse is the JSON shape returned by GET /layout
// and PUT /layout. `is_default` is true when the response is the
// first-run default (no row in homelab_user_layouts yet). The UI
// can show a one-time hint based on this flag.
type homelabLayoutResponse struct {
	Layout    []homelabLayoutWidget `json:"layout"`
	UpdatedAt *string               `json:"updated_at"`
	IsDefault bool                  `json:"is_default"`
}

// ------------------------------------------------------------------
// Default layout — returned by GET /layout/defaults and by GET /layout
// when no row exists yet. Hard-coded so first-run UX never blocks on
// a DB call.
// ------------------------------------------------------------------

// defaultHomelabLayout is the canonical first-run grid.
//
// Layout (12-col grid, row units of ~80px):
//   Row 0: [KPI strip — 12 col wide, 1 row tall]        (full-width KPI)
//   Row 1: [Services — 6 col, 4 rows] [Notes — 6 col, 4 rows]
//   Row 2: [Todos — 12 col wide, 2 rows]                (full-width todos)
//
// Future Phases add more widget types — this default will expand.
// The widget `type` strings here MUST match the constants in the
// frontend (web/src/components/homelab/...) so the UI knows which
// component to render for each entry.
var defaultHomelabLayout = []homelabLayoutWidget{
	{
		I: "kpi-services", X: 0, Y: 0, W: 3, H: 1,
		Type: "kpi-services", Config: map[string]interface{}{"color": "cyan"},
	},
	{
		I: "kpi-notes", X: 3, Y: 0, W: 3, H: 1,
		Type: "kpi-notes", Config: map[string]interface{}{"color": "indigo"},
	},
	{
		I: "kpi-todos", X: 6, Y: 0, W: 3, H: 1,
		Type: "kpi-todos", Config: map[string]interface{}{"color": "amber"},
	},
	{
		I: "kpi-rss", X: 9, Y: 0, W: 3, H: 1,
		Type: "kpi-rss", Config: map[string]interface{}{"color": "violet"},
	},
	{
		I: "services", X: 0, Y: 1, W: 6, H: 4,
		Type: "services", Config: map[string]interface{}{},
	},
	{
		I: "notes", X: 6, Y: 1, W: 6, H: 4,
		Type: "notes", Config: map[string]interface{}{},
	},
	{
		I: "todos", X: 0, Y: 5, W: 12, H: 2,
		Type: "todos", Config: map[string]interface{}{},
	},
}

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/homelab/layout
// ------------------------------------------------------------------

// GetHomelabLayout returns the caller's layout. If no row exists
// yet (first run for this user), returns the default layout with
// `is_default=true` so the UI knows it can save on the first edit
// without surprising the user.
//
// Honors tenant_id + user_id from the JWT — never returns another
// user's layout even if a row id leaks through.
func GetHomelabLayout(pool *db.Pool) gin.HandlerFunc {
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

		var (
			rawLayout  []byte
			updatedAt  *string
		)
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT layout::text, updated_at::text
			   FROM homelab_user_layouts
			  WHERE tenant_id = $1 AND user_id = $2`,
			tenantID, userID,
		).Scan(&rawLayout, &updatedAt)

		if errors.Is(err, pgx.ErrNoRows) {
			// First run — return the default layout with is_default=true.
			kernel.RespondOK(c, homelabLayoutResponse{
				Layout:    defaultHomelabLayout,
				UpdatedAt: nil,
				IsDefault: true,
			})
			return
		}
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		// Parse the stored JSONB back into our typed shape. The
		// column is jsonb so the round-trip always yields valid JSON
		// (the PUT handler validates before INSERT) — but a hand-crafted
		// UPDATE could still poison the column. Treat parse errors
		// as 500 (the DB is in a bad state) so the operator notices.
		var widgets []homelabLayoutWidget
		if jerr := json.Unmarshal(rawLayout, &widgets); jerr != nil {
			kernel.RespondErrorWithCode(c, http.StatusInternalServerError, "layout_corrupt",
				"stored layout JSON is invalid; reset via DELETE /prefs/layout")
			return
		}
		if widgets == nil {
			widgets = []homelabLayoutWidget{}
		}

		kernel.RespondOK(c, homelabLayoutResponse{
			Layout:    widgets,
			UpdatedAt: updatedAt,
			IsDefault: false,
		})
	}
}

// ------------------------------------------------------------------
// Protected endpoint: PUT /api/v1/homelab/layout
// ------------------------------------------------------------------

// PutHomelabLayout upserts the caller's layout. Validates:
//   - body is valid JSON with a non-nil `layout` array (handler-side)
//   - each widget has a non-empty `type` and an `i` (frontend key)
//   - x/y/w/h are non-negative integers (handler-side clamp)
//
// UPSERT on (tenant_id, user_id) — INSERT if no row, UPDATE otherwise.
// Returns 200 with the stored row's updated_at.
func PutHomelabLayout(pool *db.Pool) gin.HandlerFunc {
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

		var req homelabLayoutReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		// gin `binding:"required"` rejects null but accepts an empty
		// array, which is legal — an empty layout = "no widgets on
		// the grid", the user has hidden everything.
		if req.Layout == nil {
			req.Layout = []homelabLayoutWidget{}
		}

		// Validate every widget — fail fast with a precise error so
		// the frontend doesn't have to debug a 500. We clamp x/y/w/h
		// to non-negative ints (no upper bound — frontend owns the
		// grid math and we trust it).
		for i := range req.Layout {
			w := &req.Layout[i]
			if w.I == "" {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
					fmt.Sprintf("layout[%d].i is required", i))
				return
			}
			if w.Type == "" {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
					fmt.Sprintf("layout[%d].type is required", i))
				return
			}
			if w.X < 0 {
				w.X = 0
			}
			if w.Y < 0 {
				w.Y = 0
			}
			if w.W < 1 {
				w.W = 1
			}
			if w.H < 1 {
				w.H = 1
			}
		}

		// Marshal back to a json.RawMessage so the column stores
		// a clean JSONB blob. The round-trip is JSON -> []widget ->
		// JSON; this drops any unknown fields but keeps the canonical
		// shape we know about.
		raw, jerr := json.Marshal(req.Layout)
		if jerr != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		var updatedAt string
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO homelab_user_layouts (tenant_id, user_id, layout, updated_at)
			 VALUES ($1, $2, $3::jsonb, NOW())
			 ON CONFLICT (tenant_id, user_id)
			 DO UPDATE SET layout = EXCLUDED.layout,
			               updated_at = NOW()
			 RETURNING updated_at::text`,
			tenantID, userID, raw,
		).Scan(&updatedAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		kernel.RespondOK(c, homelabLayoutResponse{
			Layout:    req.Layout,
			UpdatedAt: &updatedAt,
			IsDefault: false,
		})
	}
}

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/homelab/layout/defaults
// ------------------------------------------------------------------

// GetHomelabLayoutDefaults returns the hard-coded default layout
// (same one first-run users see on GET /layout). The UI uses this
// endpoint to render the "Reset layout to default" preview before
// committing the reset via DELETE /homelab/prefs/layout.
//
// Hard-coded here rather than fetched from a config table because:
//   1. It is a stable, versioned piece of UI affordance — changes
//      ship with the binary.
//   2. It avoids an extra DB call on every "reset to defaults" click.
func GetHomelabLayoutDefaults(_ *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		kernel.RespondOK(c, gin.H{
			"layout": defaultHomelabLayout,
		})
	}
}

// Compile-time guards — these references ensure the unused-import
// check doesn't drop the uuid import even if all uuid.Parse calls
// are removed in a future refactor (the file genuinely needs uuid
// for the user_id column type, even if no current handler calls
// uuid.Parse on a request param).
var _ = uuid.Nil