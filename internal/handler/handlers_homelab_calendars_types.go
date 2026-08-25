// Tier 10 Phase 4 — Calendar (H4).
//
// Shared types for the homelab calendar + events surface. Lives
// in its own file so handlers_homelab_calendars.go (5 routes) can
// import these types without growing past the 400-LOC cap.
//
// Storage: homelab_calendars (migrations/041_homelab.sql — Phase 4
// extension) + homelab_events (one row per cached VEVENT).
//
// Per-user design: every query is filtered by both tenant_id and
// user_id. A user in tenant A cannot see (or modify) another
// user's calendars even if they share a tenant — matches US-8 in
// the speckit proposal ("my homelab doesn't change under me when
// my co-founder rearranges theirs").
package handler

import "time"

// ------------------------------------------------------------------
// Calendar-row shape — what /homelab/calendars returns + accepts.
// ------------------------------------------------------------------

// calendarRow is the JSON shape returned by GET /calendars and
// POST /calendars. Mirrors the columns of homelab_calendars plus
// an event_count that the handler computes with a sub-query so
// the frontend can show "Work — 12 events" without a follow-up
// GET. last_synced_at + last_sync_status are RFC3339-ish strings
// so the frontend can render relative timestamps directly.
type calendarRow struct {
	ID             string `json:"id"`
	TenantID       string `json:"tenant_id"`
	UserID         string `json:"user_id"`
	Name           string `json:"name"`
	ICalURL        string `json:"ical_url"`
	Color          string `json:"color"`
	Enabled        bool   `json:"enabled"`
	LastSyncedAt   string `json:"last_synced_at,omitempty"`
	LastSyncStatus string `json:"last_sync_status,omitempty"`
	LastSyncError  string `json:"last_sync_error,omitempty"`
	EventCount     int    `json:"event_count"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// calendarReq is the body for POST /calendars. name + ical_url are
// required; color is optional (defaults to "#3b82f6"). enabled
// defaults to true so a fresh subscription syncs on first worker
// tick.
type calendarReq struct {
	Name    string `json:"name"    binding:"required"`
	ICalURL string `json:"ical_url" binding:"required"`
	Color   string `json:"color"`
	Enabled *bool  `json:"enabled"`
}

// ------------------------------------------------------------------
// Event-row shape — what /homelab/calendars/events returns.
// ------------------------------------------------------------------

// eventRow is the JSON shape returned by GET /calendars/events.
// calendar_id is the homelab_calendars row the event came from —
// the frontend uses it (with the calendar's color) to color the
// chip in the week view. starts_at + ends_at are RFC3339 strings
// so the frontend can do relative-time formatting directly.
type eventRow struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenant_id"`
	UserID      string `json:"user_id"`
	CalendarID  string `json:"calendar_id"`
	UID         string `json:"uid"`
	Summary     string `json:"summary"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location,omitempty"`
	StartsAt    string `json:"starts_at"`
	EndsAt      string `json:"ends_at,omitempty"`
	AllDay      bool   `json:"all_day"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// eventsListReq is the parsed query string for GET /calendars/events.
// from + to are RFC3339; the handler defaults to "now" and
// "now + 30 days" when they're missing. calendar_id is optional
// (omit = all the caller's enabled calendars). limit caps the
// response size (default 200, max 1000) so a power user with 5
// calendars × thousands of events can't blow the dashboard.
type eventsListReq struct {
	From       string `form:"from"`
	To         string `form:"to"`
	CalendarID string `form:"calendar_id"`
	Limit      int    `form:"limit"`
}

// ------------------------------------------------------------------
// Validation enums — enforced on POST to reject bad input before
// any DB call. Mirrors the no-CHECK-constraint design note in
// migrations/041_homelab.sql.
// ------------------------------------------------------------------

// allowedCalendarColors is the palette the frontend's color
// picker offers. Locked to a small set so the chip background
// always has enough contrast against the dashboard surface —
// free-form hex would invite low-contrast colors that the
// accessibility scanner would flag. The frontend renders the
// picker's chip with inline `style={{background: color}}` so the
// backend's choice is just a hint for the next render.
//
// Adding a new color here is safe — existing rows that use a
// removed color simply don't match the new picker; their CSS hex
// still renders fine in the chip. Removing a color would orphan
// any row pinned to it (their chip falls back to surface-3).
var allowedCalendarColors = map[string]struct{}{
	"#3b82f6": {}, // blue (default)
	"#22c55e": {}, // green
	"#ef4444": {}, // red
	"#f59e0b": {}, // amber
	"#a855f7": {}, // purple
	"#06b6d4": {}, // cyan
	"#6b7280": {}, // gray
}

// defaultCalendarColor is the row's default when the user
// doesn't pick one in the POST body. Matches the table DEFAULT.
const defaultCalendarColor = "#3b82f6"

// eventsListLimit is the default cap on GET /calendars/events.
// The week view shows up to 7 days × ~30 events/day; 200 is a
// generous ceiling that still keeps the response <100 KB.
const eventsListLimit = 200

// eventsListLimitMax caps the user-supplied ?limit= so a
// malicious caller can't request millions of rows. 1000 covers
// the legitimate "show me everything for the next 90 days" case.
const eventsListLimitMax = 1000

// eventsDefaultWindowDays is the default "to - from" when the
// caller doesn't supply ?from/?to. Matches the spec §"H4 —
// Calendar" ("upcoming events next 30d").
const eventsDefaultWindowDays = 30

// timeParseLayout is the layout used to parse the frontend's
// datetime-local inputs (YYYY-MM-DDTHH:MM). Used by the
// upcoming-events handler when the caller doesn't pass RFC3339.
const timeParseLayout = "2006-01-02T15:04"

// _ keeps time referenced so the imports list doesn't get
// pruned by goimports in case future revisions want to use the
// time package directly (e.g. for RFC3339 helpers). Cheap
// insurance vs. a future import churn.
var _ = time.RFC3339
