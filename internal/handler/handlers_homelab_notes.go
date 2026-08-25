// Tier 10 Phase 3 — Personal Notes + Todos (H3). List + create
// endpoints for the per-user note surface (homelab_notes).
//
//	GET  /api/v1/homelab/notes  — ListHomelabNotes
//	POST /api/v1/homelab/notes  — CreateHomelabNote
//
// PATCH /notes/:id and DELETE /notes/:id live in
// handlers_homelab_notes_extra.go (split for file-size discipline
// — this file stays focused on the list-and-create hot path).
// GET /notes/:id and GET /notes/search also live in the extras
// file.
//
// Storage: homelab_notes (migrations/041_homelab.sql). One row
// per note; per-user (NOT per-tenant) — every WHERE clause filters
// by both tenant_id and user_id so user A can never see or modify
// user B's notes.
package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ------------------------------------------------------------------
// Validation helpers — kept small so the file stays under 400 LOC.
// ------------------------------------------------------------------

// validateNoteTitle enforces a non-empty title and a sane length
// cap. Notes without titles break the dashboard's card layout (the
// card renders title in bold and a placeholder dash if blank). We
// use 200 chars as the cap — matches a typical note title length.
func validateNoteTitle(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("title is required")
	}
	if len(title) > 200 {
		return fmt.Errorf("title must be 200 characters or fewer")
	}
	return nil
}

// validateNoteTags enforces sane tag bounds: max 16 tags, each
// max 32 chars, no empty strings. The text[] column accepts an
// empty array; the handler validates the array shape before
// INSERT so the dashboard never has to deal with degenerate data.
func validateNoteTags(tags []string) error {
	if len(tags) > notesMaxTags {
		return fmt.Errorf("tags must be %d or fewer (got %d)", notesMaxTags, len(tags))
	}
	for i, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			return fmt.Errorf("tags[%d] is empty", i)
		}
		if len(t) > 32 {
			return fmt.Errorf("tags[%d] must be 32 characters or fewer", i)
		}
	}
	return nil
}

// validateNoteBody enforces the body length cap. Empty body is OK
// (the column default is ''). The cap protects the dashboard from
// a user accidentally pasting in a 10 MB document.
func validateNoteBody(body string) error {
	if len(body) > notesMaxBodyChars {
		return fmt.Errorf("body must be %d characters or fewer", notesMaxBodyChars)
	}
	return nil
}

// scanNoteRow scans one row of the standard notes SELECT into a
// noteRow. Pulled into a helper so all four CRUD handlers share
// the same column order and Scan signature.
func scanNoteRow(scan func(...any) error) (noteRow, error) {
	var (
		row       noteRow
		createdAt string
		updatedAt string
	)
	if err := scan(&row.ID, &row.TenantID, &row.UserID, &row.Title,
		&row.Body, &row.Tags, &row.Pinned, &createdAt, &updatedAt); err != nil {
		return noteRow{}, err
	}
	row.CreatedAt = createdAt
	row.UpdatedAt = updatedAt
	if row.Tags == nil {
		row.Tags = []string{}
	}
	return row, nil
}

// notesListOrderBy returns the ORDER BY clause for GET /notes.
// Pinned notes float to the top (pinned DESC), then most-recently
// updated (updated_at DESC). Matches the homelab dashboard UX:
// the user pins the notes they care about and sees them first.
const notesListOrderBy = ` ORDER BY pinned DESC, updated_at DESC`

// parsePositiveInt parses a query-string integer with bounds. An
// empty input returns def; an out-of-range input returns an error
// so the caller can choose to fall back to the default. Kept
// inline here (no shared helper yet) because the existing handler
// package has parsePositiveInt-like logic scattered across files
// — Phase 4+ can extract it once a 3rd caller appears.
func parsePositiveInt(raw string, min, max, def int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def, fmt.Errorf("not an integer: %s", raw)
	}
	if v < min || v > max {
		return def, fmt.Errorf("must be between %d and %d", min, max)
	}
	return v, nil
}

// ------------------------------------------------------------------
// GET /api/v1/homelab/notes
// ------------------------------------------------------------------

// ListHomelabNotes returns all notes for the caller, newest first.
// Honors tenant_id + user_id from the JWT — never returns another
// user's notes even if a row id leaks through.
//
// Filters:
//
//	?pinned=true    only pinned notes
//	?tag=X          only notes tagged with X (uses GIN index)
//	?limit=N        cap rows (default 50, max 200)
//
// The default limit is the homelab-notes widget size (50); callers
// can pass a higher limit if they're building a different surface
// (e.g. a search results page that wants more rows).
func ListHomelabNotes(pool *db.Pool) gin.HandlerFunc {
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

		// Optional filters. pinned is a tri-state: missing = no
		// filter; "true" = pinned only; "false" = unpinned only.
		pinnedFilter := strings.TrimSpace(c.Query("pinned"))
		tagFilter := strings.TrimSpace(c.Query("tag"))

		limit := notesListLimit
		if l, perr := parsePositiveInt(c.Query("limit"), 1, 200, notesListLimit); perr == nil {
			limit = l
		}

		// Build the WHERE clause dynamically based on which
		// filters were supplied. The base filter on tenant_id +
		// user_id is non-negotiable.
		conds := []string{"tenant_id = $1", "user_id = $2"}
		args := []interface{}{tenantID, userID}
		if pinnedFilter == "true" {
			conds = append(conds, "pinned = true")
		} else if pinnedFilter == "false" {
			conds = append(conds, "pinned = false")
		}
		if tagFilter != "" {
			args = append(args, tagFilter)
			conds = append(conds, fmt.Sprintf("tags @> ARRAY[$%d]::text[]", len(args)))
		}
		args = append(args, limit)
		q := `SELECT id::text, tenant_id::text, user_id::text, title, body,
		             tags, pinned, created_at::text, updated_at::text
		        FROM homelab_notes
		       WHERE ` + joinStrings(conds, " AND ") +
			notesListOrderBy +
			fmt.Sprintf(" LIMIT $%d", len(args))

		rows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		out := []noteRow{}
		for rows.Next() {
			r, serr := scanNoteRow(rows.Scan)
			if serr != nil {
				kernel.RespondError(c, serr)
				return
			}
			out = append(out, r)
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, gin.H{
			"notes": out,
			"count": len(out),
		})
	}
}

// ------------------------------------------------------------------
// POST /api/v1/homelab/notes
// ------------------------------------------------------------------

// CreateHomelabNote inserts a new note for the caller. Validates:
//
//   - title is non-empty (after trim) and ≤ 200 chars
//   - body is ≤ notesMaxBodyChars
//   - tags is ≤ notesMaxTags entries, each non-empty and ≤ 32 chars
//
// Returns 201 with the created row so the UI can render the new
// card without an extra GET.
func CreateHomelabNote(pool *db.Pool) gin.HandlerFunc {
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

		var req noteReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		req.Title = strings.TrimSpace(req.Title)
		if err := validateNoteTitle(req.Title); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		if err := validateNoteBody(req.Body); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		if err := validateNoteTags(req.Tags); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		if req.Tags == nil {
			req.Tags = []string{}
		}

		var (
			id, createdAt, updatedAt string
		)
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO homelab_notes
			        (tenant_id, user_id, title, body, tags, pinned)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING id::text, created_at::text, updated_at::text`,
			tenantID, userID, req.Title, req.Body, req.Tags, req.Pinned,
		).Scan(&id, &createdAt, &updatedAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		kernel.RespondCreated(c, noteRow{
			ID:        id,
			TenantID:  tenantID.String(),
			UserID:    userID.String(),
			Title:     req.Title,
			Body:      req.Body,
			Tags:      req.Tags,
			Pinned:    req.Pinned,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}
}
