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
//	Phase 3: Notes + Todos (H3)                          ← 12 routes
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

	// ---- Tier 10.2: Service Status (Phase 2) ----
	//
	// Per spec §"H2 — Service Status", these 8 protected endpoints
	// back the per-user pinboard + the 60s background health probe.
	// Handlers are split across handlers_homelab_services.go (5
	// routes — CRUD + aggregate) and
	// handlers_homelab_services_probe.go (3 routes — one-shot probe,
	// history, fire-and-forget probe-all).
	//
	// Per-user (NOT per-tenant): every query filters by both
	// tenant_id and user_id so a user can never probe or read
	// another user's pins. ServiceHealthWorker (started in main.go)
	// ticks every 60s and INSERTs results into homelab_service_health.
	//
	// Routes (8 protected):
	//   GET    /api/v1/homelab/services            — ListHomelabServices
	//   POST   /api/v1/homelab/services            — CreateHomelabService
	//   PATCH  /api/v1/homelab/services/:id        — PatchHomelabService
	//   DELETE /api/v1/homelab/services/:id        — DeleteHomelabService
	//   GET    /api/v1/homelab/services/aggregate  — GetHomelabServicesAggregate
	//   POST   /api/v1/homelab/services/:id/probe  — ProbeHomelabService
	//   GET    /api/v1/homelab/services/:id/history — GetHomelabServiceHistory
	//   POST   /api/v1/homelab/services/probe-all  — ProbeAllHomelabServices
	protected.GET("/homelab/services", handler.ListHomelabServices(pool))
	protected.POST("/homelab/services", handler.CreateHomelabService(pool))
	protected.GET("/homelab/services/aggregate", handler.GetHomelabServicesAggregate(pool))
	protected.POST("/homelab/services/probe-all", handler.ProbeAllHomelabServices(pool))
	// /:id/* routes MUST come after the static-path siblings above
	// (Gin matches in order) — otherwise /probe-all would be parsed
	// as :id="probe-all" and the static handler would never fire.
	protected.PATCH("/homelab/services/:id", handler.PatchHomelabService(pool))
	protected.DELETE("/homelab/services/:id", handler.DeleteHomelabService(pool))
	protected.POST("/homelab/services/:id/probe", handler.ProbeHomelabService(pool))
	protected.GET("/homelab/services/:id/history", handler.GetHomelabServiceHistory(pool))

	// ---- Tier 10.3: Notes + Todos (Phase 3) ----
	//
	// Per spec §"H3 — Personal Notes + Todos", these 12 protected
	// endpoints back the per-user note + todo surfaces. Handlers
	// split across:
	//   - handlers_homelab_notes.go      (2 routes — list, create)
	//   - handlers_homelab_notes_extra.go (4 routes — get, search, patch, delete)
	//   - handlers_homelab_todos.go       (2 routes — list, create)
	//   - handlers_homelab_todos_extra.go (4 routes — patch, delete, complete, due-soon)
	//
	// Notes are Markdown bodies with text[] tags + a pinned flag;
	// todos have priority (low|medium|high|urgent) + optional due
	// date + completion timestamp. Every query is filtered by both
	// tenant_id AND user_id so a user can never see another user's
	// notes/todos.
	//
	// Routes (12 protected):
	//   GET    /api/v1/homelab/notes              — ListHomelabNotes
	//   POST   /api/v1/homelab/notes              — CreateHomelabNote
	//   GET    /api/v1/homelab/notes/:id          — GetHomelabNote
	//   GET    /api/v1/homelab/notes/search       — SearchHomelabNotes
	//   PATCH  /api/v1/homelab/notes/:id          — PatchHomelabNote
	//   DELETE /api/v1/homelab/notes/:id          — DeleteHomelabNote
	//   GET    /api/v1/homelab/todos              — ListHomelabTodos
	//   POST   /api/v1/homelab/todos              — CreateHomelabTodo
	//   GET    /api/v1/homelab/todos/due-soon     — ListDueSoonHomelabTodos
	//   PATCH  /api/v1/homelab/todos/:id          — PatchHomelabTodo
	//   DELETE /api/v1/homelab/todos/:id          — DeleteHomelabTodo
	//   POST   /api/v1/homelab/todos/:id/complete — CompleteHomelabTodo
	//
	// Static-path siblings (search, due-soon, complete) MUST be
	// registered BEFORE the /:id patterns below — Gin's radix tree
	// matches in registration order, so /due-soon would otherwise
	// be parsed as :id="due-soon".
	protected.GET("/homelab/notes", handler.ListHomelabNotes(pool))
	protected.POST("/homelab/notes", handler.CreateHomelabNote(pool))
	protected.GET("/homelab/notes/search", handler.SearchHomelabNotes(pool))
	protected.GET("/homelab/notes/:id", handler.GetHomelabNote(pool))
	protected.PATCH("/homelab/notes/:id", handler.PatchHomelabNote(pool))
	protected.DELETE("/homelab/notes/:id", handler.DeleteHomelabNote(pool))

	protected.GET("/homelab/todos", handler.ListHomelabTodos(pool))
	protected.POST("/homelab/todos", handler.CreateHomelabTodo(pool))
	protected.GET("/homelab/todos/due-soon", handler.ListDueSoonHomelabTodos(pool))
	// /:id/* static siblings (complete) before the /:id patterns.
	protected.POST("/homelab/todos/:id/complete", handler.CompleteHomelabTodo(pool))
	protected.PATCH("/homelab/todos/:id", handler.PatchHomelabTodo(pool))
	protected.DELETE("/homelab/todos/:id", handler.DeleteHomelabTodo(pool))

	// ---- Tier 10.4: Calendar (Phase 4) ----
	//
	// Per spec §"H4 — Calendar", these 5 protected endpoints
	// back the per-user iCal subscription surface
	// (homelab_calendars) and the events read endpoint
	// (homelab_events). The CalendarWorker (started in main.go)
	// ticks every 30min and upserts VEVENTs from each enabled
	// feed; POST /calendars fires an immediate sync after create
	// so a freshly-pinned feed shows events within seconds.
	//
	// Routes (5 protected):
	//   GET    /api/v1/homelab/calendars              — ListHomelabCalendars
	//   POST   /api/v1/homelab/calendars              — CreateHomelabCalendar
	//   GET    /api/v1/homelab/calendars/events       — ListHomelabCalendarEvents
	//   POST   /api/v1/homelab/calendars/:id/refresh  — RefreshHomelabCalendar
	//   DELETE /api/v1/homelab/calendars/:id          — DeleteHomelabCalendar
	//
	// Static-path siblings (/events) MUST come BEFORE the /:id
	// patterns below — Gin's radix tree matches in registration
	// order, so /events would otherwise be parsed as :id="events"
	// and the static handler would never fire.
	protected.GET("/homelab/calendars", handler.ListHomelabCalendars(pool))
	protected.POST("/homelab/calendars", handler.CreateHomelabCalendar(pool))
	protected.GET("/homelab/calendars/events", handler.ListHomelabCalendarEvents(pool))
	// /:id/* static siblings (refresh) before the /:id patterns.
	protected.POST("/homelab/calendars/:id/refresh", handler.RefreshHomelabCalendar(pool))
	protected.DELETE("/homelab/calendars/:id", handler.DeleteHomelabCalendar(pool))

	// ---- Tier 10.5: Download Stats (Phase 5) ----
	//
	// Per spec §"H5 — Download Stats", these 4 protected endpoints
	// back the per-user download-client registry
	// (homelab_download_clients) + the polled-state read endpoint
	// (homelab_download_snapshots). The DownloadsWorker (started in
	// main.go) ticks every 60s and INSERTs a snapshot row per
	// enabled client. POST /downloads/clients fires an immediate
	// fire-and-forget poll after create via
	// PollClientAndInsertSnapshot so a freshly-pinned client shows
	// stats within seconds.
	//
	// Routes (4 protected):
	//   GET    /api/v1/homelab/downloads/clients      — ListHomelabDownloadClients
	//   POST   /api/v1/homelab/downloads/clients      — CreateHomelabDownloadClient
	//   GET    /api/v1/homelab/downloads/snapshots   — ListHomelabDownloadSnapshots
	//   DELETE /api/v1/homelab/downloads/clients/:id  — DeleteHomelabDownloadClient
	//
	// Static-path siblings (/snapshots) MUST come BEFORE the /:id
	// patterns below — Gin's radix tree matches in registration
	// order, so /snapshots would otherwise be parsed as :id="snapshots"
	// and the static handler would never fire.
	protected.GET("/homelab/downloads/clients", handler.ListHomelabDownloadClients(pool))
	protected.POST("/homelab/downloads/clients", handler.CreateHomelabDownloadClient(pool))
	protected.GET("/homelab/downloads/snapshots", handler.ListHomelabDownloadSnapshots(pool))
	protected.DELETE("/homelab/downloads/clients/:id", handler.DeleteHomelabDownloadClient(pool))

	// ---- Tier 10.6: Media Server (Phase 6) ----
	//
	// Per spec §"H6 — Media Server", these 4 protected endpoints
	// back the per-user media-server registry
	// (homelab_media_servers) + the polled-state read endpoint
	// (now_playing + recent_additions). The MediaWorker (started
	// in main.go) ticks every 60s and UPSERTs the parsed state.
	// POST /media/servers fires an immediate fire-and-forget poll
	// after create via PollMediaServerAndInsertState so a
	// freshly-pinned server shows now-playing + recent additions
	// within seconds.
	//
	// Routes (4 protected):
	//   GET    /api/v1/homelab/media/servers      — ListHomelabMediaServers
	//   POST   /api/v1/homelab/media/servers      — CreateHomelabMediaServer
	//   GET    /api/v1/homelab/media/state        — GetHomelabMediaState
	//   DELETE /api/v1/homelab/media/servers/:id  — DeleteHomelabMediaServer
	//
	// Static-path siblings (/state) MUST come BEFORE the /:id
	// patterns below — Gin's radix tree matches in registration
	// order, so /state would otherwise be parsed as :id="state"
	// and the static handler would never fire.
	protected.GET("/homelab/media/servers", handler.ListHomelabMediaServers(pool))
	protected.POST("/homelab/media/servers", handler.CreateHomelabMediaServer(pool))
	protected.GET("/homelab/media/state", handler.GetHomelabMediaState(pool))
	protected.DELETE("/homelab/media/servers/:id", handler.DeleteHomelabMediaServer(pool))

	// ---- Tier 10.7: Search (Phase 7) ----
	//
	// Per spec §"H7 — Search", these 3 protected endpoints back
	// the global search surface that crosses every per-user
	// homelab sub-feature. No new tables — search reads the 13
	// existing tables via a parameterized UNION ALL. Handlers
	// live in handlers_homelab_search.go + the SQL-builder
	// helpers in handlers_homelab_search_helpers.go (kept
	// under 400 LOC each via the file split).
	//
	// Routes (3 protected):
	//   GET /api/v1/homelab/search/kinds        — ListHomelabSearchKinds (static catalog)
	//   GET /api/v1/homelab/search/suggestions  — SearchHomelabSuggestions (top-8 autocomplete)
	//   GET /api/v1/homelab/search               — SearchHomelabGlobal (full results w/ snippet+score)
	//
	// Static-path siblings (/kinds, /suggestions) MUST come
	// BEFORE the bare /search path — Gin's radix tree matches
	// in registration order. With three distinct sub-paths
	// (kinds / suggestions / search) there's no actual
	// shadowing risk, but alphabetical order keeps the
	// registration block readable for the next reviewer.
	protected.GET("/homelab/search/kinds", handler.ListHomelabSearchKinds())
	protected.GET("/homelab/search/suggestions", handler.SearchHomelabSuggestions(pool))
	protected.GET("/homelab/search", handler.SearchHomelabGlobal(pool))
}
