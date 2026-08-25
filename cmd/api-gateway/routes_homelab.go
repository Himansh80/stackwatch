package main

import (
	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/handler"
)

// mountHomelabRoutes registers the Tier 10 — Homelab Dashboard endpoints.
//
// Created in Phase 0 because routes_protected.go is at the 399-LOC cap
// after Tier 9 (Security & Enterprise). Adding 50 more Tier 10 routes
// across Phases 1-9 would push that file past the 400-LOC limit. The
// routes_homelab.go split keeps routes_protected.go focused on
// top-level tier wiring while the homelab layout/prefs/services/
// notes/todos/calendar/downloads/media/search/rss/scheduler surfaces
// live here.
//
// Phases 1-9 add their routes here in this order:
//
//	Phase 1: Widget Framework + Page shell (H1)         ← 6 routes
//	Phase 2: Service Status (H2)                         ← 8 routes
//	Phase 3: Notes + Todos (H3)                          ← 10 routes
//	Phase 4: Calendar (H4)                               ← 5 routes
//	Phase 5: Download Stats (H5)                         ← 4 routes
//	Phase 6: Media Server (H6)                           ← 4 routes
//	Phase 7: Search (H7)                                 ← 3 routes
//	Phase 8: RSS / Activity Feed (H9)                    ← 5 routes
//	Phase 9: Task Scheduler (H10)                        ← 5 routes
//
// All handlers honor tenant_id + user_id from the JWT — every query
// filters by both, so layouts and preferences are scoped PER USER
// (not per tenant). Idempotent migration migrations/041_homelab.sql
// sets up the backing tables.
//
// The migration is a SINGLE file (041_homelab.sql) that holds all
// 17 tables across the 9 sub-features — matches the Tier 9 pattern
// of a single migration per change.
func mountHomelabRoutes(protected *gin.RouterGroup, pool *db.Pool) {
	// ---- Tier 10.1: Widget Framework + Page shell (Phase 1) ----
	//
	// Per spec §"H1 — Widget Framework", these 6 protected endpoints
	// back the per-user layout + preferences surfaces. Layouts are
	// a JSONB array of widget descriptors (i, x, y, w, h, type,
	// config) loaded by HomelabGrid on mount and saved via PUT with
	// a 500ms debounce. Preferences carry the per-user theme,
	// refresh interval, pinned widgets, and default landing page.
	// Both tables are UNIQUE on (tenant_id, user_id) — one row per
	// user. Auto-create semantics: GET /prefs lazy-creates the row
	// with the defaults; DELETE /prefs/layout reverts the layout to
	// defaults but leaves prefs intact (theme/refresh persist).
	//
	// Routes (6 protected):
	//   GET    /api/v1/homelab/layout          — GetHomelabLayout
	//   PUT    /api/v1/homelab/layout          — PutHomelabLayout
	//   GET    /api/v1/homelab/layout/defaults — GetHomelabLayoutDefaults
	//   GET    /api/v1/homelab/prefs           — GetHomelabPrefs
	//   PUT    /api/v1/homelab/prefs           — PutHomelabPrefs
	//   DELETE /api/v1/homelab/prefs/layout    — DeleteHomelabPrefsLayout
	protected.GET("/homelab/layout", handler.GetHomelabLayout(pool))
	protected.PUT("/homelab/layout", handler.PutHomelabLayout(pool))
	protected.GET("/homelab/layout/defaults", handler.GetHomelabLayoutDefaults(pool))
	protected.GET("/homelab/prefs", handler.GetHomelabPrefs(pool))
	protected.PUT("/homelab/prefs", handler.PutHomelabPrefs(pool))
	protected.DELETE("/homelab/prefs/layout", handler.DeleteHomelabPrefsLayout(pool))
}