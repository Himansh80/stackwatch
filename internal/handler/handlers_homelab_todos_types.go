// Tier 10 Phase 3 — Personal Notes + Todos (H3). Shared types
// for the homelab todos surface.
//
// Storage: homelab_todos (migrations/041_homelab.sql).
//
// Per-user design: every query is filtered by both tenant_id and
// user_id. A user in tenant A cannot see (or modify) another
// user's todos even if they share a tenant — matches US-8 in the
// speckit proposal ("my homelab doesn't change under me when my
// co-founder rearranges theirs").
package handler

import (
	"fmt"
	"time"
)

// ------------------------------------------------------------------
// Todo-row shape — what /homelab/todos returns + accepts on POST/PATCH.
// ------------------------------------------------------------------

// todoRow is the JSON shape returned by GET /todos, POST /todos,
// and PATCH /todos/:id. Mirrors the columns of homelab_todos so
// the frontend can use field names directly without a transformer.
//
// The Priority field uses the allowedTodoPriorities whitelist;
// the handler rejects anything else with 400. The DueDate + CompletedAt
// fields are nullable — a todo without a due date or completion
// timestamp is normal. The frontend renders overdue todos in red.
type todoRow struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	UserID      string    `json:"user_id"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	Priority    string    `json:"priority"`
	DueDate     *string   `json:"due_date,omitempty"`
	CompletedAt *string   `json:"completed_at,omitempty"`
	Tags        []string  `json:"tags"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

// todoReq is the body for POST /todos. All fields are required
// except description (defaults to null), due_date (defaults to
// null), tags (defaults to []), and priority (defaults to medium).
// PATCH uses a separate type so fields can be optional.
type todoReq struct {
	Title       string   `json:"title"       binding:"required"`
	Description *string  `json:"description,omitempty"`
	Priority    string   `json:"priority"`
	DueDate     *string  `json:"due_date,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// todoPatchReq is the body for PATCH /todos/:id. All fields are
// optional — the handler builds a dynamic UPDATE with only the
// fields the caller sent (same pattern as the services/notes
// PATCH handler).
//
// CompletedAt is special: when the caller sends a non-null value
// the todo is marked complete; when the caller sends null the
// todo is reset to active (completed_at = NULL). The handler
// distinguishes "not provided" (don't touch the field) from
// "explicitly null" (reset to active).
type todoPatchReq struct {
	Title       *string   `json:"title,omitempty"`
	Description *string   `json:"description,omitempty"`
	Priority    *string   `json:"priority,omitempty"`
	DueDate     *string   `json:"due_date,omitempty"`
	Tags        *[]string `json:"tags,omitempty"`
	// MarkComplete is a tri-state:
	//   nil       → don't change completed_at
	//   *true     → set completed_at = NOW()
	//   *false    → set completed_at = NULL
	// The frontend doesn't need to send an ISO timestamp — the
	// server timestamps the transition itself.
	MarkComplete *bool `json:"mark_complete,omitempty"`
}

// ------------------------------------------------------------------
// Validation enums — enforced on POST/PATCH to reject bad input
// before any DB call. Mirrors the no-CHECK-constraint design note
// in migrations/041_homelab.sql.
// ------------------------------------------------------------------

// allowedTodoPriorities is the whitelist for the `priority` enum.
// Mirrors the four UI pill colors (low=muted, medium=blue,
// high=amber, urgent=red). Adding a new priority means appending
// here AND teaching the frontend's priority pill color map AND
// the dashboard widget — all in one PR so the wire types and
// backend stay in sync.
var allowedTodoPriorities = map[string]bool{
	"low":    true,
	"medium": true,
	"high":   true,
	"urgent": true,
}

// todosListLimit caps the GET /todos response. The frontend only
// shows the first N in the widget; the rest live behind the
// search endpoint. The (user_id, due_date) index makes the LIMIT
// cheap.
const todosListLimit = 50

// todosDueSoonDays is the window the due-soon endpoint looks at.
// Powers the KPI strip ("todos due soon" card) — todos with
// due_date between now and now+7d.
const todosDueSoonDays = 7

// todosMaxTags caps how many tags a single todo can carry. Same
// as notes (homogenous tagging UX).
const todosMaxTags = 16

// todosMaxTitleChars caps the todo title. Todos have shorter
// titles than notes — 200 is generous.
const todosMaxTitleChars = 200

// time.Time helper — Postgres timestamptz round-trip uses
// time.Time so the driver handles the conversion. Kept here as
// a compile-time guard so the import survives refactors.
var _ time.Time = time.Now()

// ------------------------------------------------------------------
// Date / time helpers — shared across todos + handlers (and any
// future homelab table that accepts a user-supplied timestamp).
// ------------------------------------------------------------------

// parseFlexibleDate accepts a small set of date shapes the UI
// might send: bare ISO date ("2025-12-31"), RFC3339 with timezone
// ("2025-12-31T23:59:00Z"), and RFC3339Nano ("2025-12-31T23:59:00.123Z").
// Returns the parsed time on success, an error otherwise. We do
// NOT support space-separated timestamps ("2025-12-31 23:59:00")
// because the frontend should always emit a T-separated string.
//
// The set is intentionally small — adding more shapes increases
// the surface area for ambiguous user input. If a different shape
// is needed, the caller should normalize it client-side first.
func parseFlexibleDate(raw string) (time.Time, error) {
	shapes := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	var lastErr error
	for _, layout := range shapes {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, nil
		} else {
			lastErr = err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no shapes matched")
	}
	return time.Time{}, lastErr
}

// nullableTime converts a pointer-to-string RFC3339-ish timestamp
// into a value that the pgx driver can pass as a NULL when the
// pointer is nil. Used for INSERT/UPDATE statements that bind a
// nullable timestamptz column directly from a request payload.
//
// The trick: when the pointer is nil we pass nil (which the driver
// translates to SQL NULL); when the pointer is non-nil we pass the
// parsed time.Time. We deliberately do not validate the format
// here — the caller already validated via validateTodoDueDate and
// parseFlexibleDate returns an error if the shape is wrong.
func nullableTime(raw *string) interface{} {
	if raw == nil {
		return nil
	}
	t, err := parseFlexibleDate(*raw)
	if err != nil {
		// The caller validated; if we still fail here the input
		// was malformed at SQL parse time. Return nil so the
		// driver gets NULL rather than a bad string — Postgres
		// will reject the bad string with a clearer error than
		// the generic "cannot encode" driver error.
		return nil
	}
	return t
}
