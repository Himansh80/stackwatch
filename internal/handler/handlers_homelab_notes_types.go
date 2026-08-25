// Tier 10 Phase 3 — Personal Notes + Todos (H3).
//
// Shared types for the homelab notes surface. Lives in its own
// file so handlers_homelab_notes.go (4 routes) and
// handlers_homelab_notes_extra.go (2 routes) can both import
// the types without either growing past the 400-LOC cap.
//
// Storage: homelab_notes (migrations/041_homelab.sql).
//
// Per-user design: every query is filtered by both tenant_id and
// user_id. A user in tenant A cannot see (or modify) another
// user's notes even if they share a tenant — matches US-8 in the
// speckit proposal ("my homelab doesn't change under me when my
// co-founder rearranges theirs").
package handler

// ------------------------------------------------------------------
// Note-row shape — what /homelab/notes returns + accepts on POST/PATCH.
// ------------------------------------------------------------------

// noteRow is the JSON shape returned by GET /notes, POST /notes,
// and PATCH /notes/:id. Mirrors the columns of homelab_notes so
// the frontend can use field names directly without a transformer.
//
// The Tags array uses []string (not pgtype.Array) so the JSON
// response is the user's actual tags, not a struct-wrapped array.
// created_at + updated_at are returned as RFC3339-ish strings so
// the frontend can do relative-time formatting without re-parsing.
type noteRow struct {
	ID        string   `json:"id"`
	TenantID  string   `json:"tenant_id"`
	UserID    string   `json:"user_id"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Tags      []string `json:"tags"`
	Pinned    bool     `json:"pinned"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

// noteReq is the body for POST /notes. All fields are required
// except tags (defaults to []) and pinned (defaults to false).
// PATCH uses a separate type so fields can be optional.
type noteReq struct {
	Title  string   `json:"title"  binding:"required"`
	Body   string   `json:"body"`
	Tags   []string `json:"tags,omitempty"`
	Pinned bool     `json:"pinned"`
}

// notePatchReq is the body for PATCH /notes/:id. All fields are
// optional — the handler builds a dynamic UPDATE with only the
// fields the caller sent (same pattern as the services PATCH
// handler).
type notePatchReq struct {
	Title  *string   `json:"title,omitempty"`
	Body   *string   `json:"body,omitempty"`
	Tags   *[]string `json:"tags,omitempty"`
	Pinned *bool     `json:"pinned,omitempty"`
}

// noteSearchReq is the (parsed) query string for GET /notes/search.
// The /search endpoint takes ?q=... (and optional ?limit=N) and
// runs a LIKE query across title + body plus an exact match on
// tags. Encoded as a struct so the handler can read it with
// `c.ShouldBindQuery(&req)` for type safety.
type noteSearchReq struct {
	Q     string `form:"q"     binding:"required"`
	Limit int    `form:"limit"`
}

// ------------------------------------------------------------------
// Validation enums — enforced on POST/PATCH to reject bad input
// before any DB call. Mirrors the no-CHECK-constraint design note
// in migrations/041_homelab.sql.
// ------------------------------------------------------------------

// notesListLimit caps the GET /notes response. The frontend only
// shows the first N in the widget; the rest live behind the search
// endpoint. The (user_id) index makes the LIMIT cheap.
const notesListLimit = 50

// notesMaxBodyChars caps the body length on POST/PATCH. 64 KiB
// matches Markdown note conventions (most homelab notes are <5 KB;
// this is a generous sanity limit, not a UX constraint).
const notesMaxBodyChars = 65536

// notesMaxTags caps how many tags a single note can carry. 16 is
// generous — anything more is almost certainly a mistake.
const notesMaxTags = 16
