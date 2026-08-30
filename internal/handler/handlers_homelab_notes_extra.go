// Tier 10 Phase 3 — Personal Notes + Todos (H3). The remaining
// 4 note handlers (PATCH, DELETE, GET by id, GET search) split
// out of handlers_homelab_notes.go to keep both files under 400
// LOC.
//
//	PATCH /api/v1/homelab/notes/:id       — PatchHomelabNote
//	DELETE /api/v1/homelab/notes/:id       — DeleteHomelabNote
//	GET   /api/v1/homelab/notes/:id        — GetHomelabNote
//	GET   /api/v1/homelab/notes/search     — SearchHomelabNotes
//
// List + Create live in handlers_homelab_notes.go (the hot path).
// All four honor tenant_id + user_id from the JWT — never touches
// another user's notes.
package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ------------------------------------------------------------------
// GET /api/v1/homelab/notes/:id
// ------------------------------------------------------------------

// GetHomelabNote returns one note by id. 404 when the row does
// not belong to the caller (or doesn't exist). Used by the
// frontend detail view when the user clicks a note card —
// optional, since the list endpoint already includes enough to
// render the card without a follow-up GET.
func GetHomelabNote(pool *db.Pool) gin.HandlerFunc {
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
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "id must be a uuid")
			return
		}

		var (
			rowID, tid, uid, title, body, createdAt, updatedAt string
			tags                                               []string
			pinned                                             bool
		)
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT id::text, tenant_id::text, user_id::text, title, body,
			        tags, pinned, created_at::text, updated_at::text
			   FROM homelab_notes
			  WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
			id, tenantID, userID,
		).Scan(&rowID, &tid, &uid, &title, &body, &tags, &pinned, &createdAt, &updatedAt)
		if err != nil {
			// Treat any error as 404 — the helper detail message
			// would otherwise leak the row's existence.
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		if tags == nil {
			tags = []string{}
		}
		kernel.RespondOK(c, noteRow{
			ID:        rowID,
			TenantID:  tid,
			UserID:    uid,
			Title:     title,
			Body:      body,
			Tags:      tags,
			Pinned:    pinned,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}
}

// ------------------------------------------------------------------
// GET /api/v1/homelab/notes/search
// ------------------------------------------------------------------

// SearchHomelabNotes returns the caller's notes whose title, body,
// or tags match the search query. The match is a case-insensitive
// ILIKE on title + body, plus an exact-match on any tag —
// Postgres's text[] @> operator is exact but the ILIKE catches
// partial tag matches inside title/body text.
//
// Query params:
//
//	q     — required, search string (after trim, max 200 chars)
//	limit — optional, default 50, max 200
//
// We DO NOT use full-text search (tsvector) here because the note
// body is small (≤ 64 KiB) and the user has at most a few hundred
// notes — ILIKE on a (user_id)-indexed table is fast enough and
// avoids the migration overhead of a tsvector column. Phase 4+
// can add FTS later if a power user complains about recall.
func SearchHomelabNotes(pool *db.Pool) gin.HandlerFunc {
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

		var req noteSearchReq
		if err := c.ShouldBindQuery(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		req.Q = strings.TrimSpace(req.Q)
		if req.Q == "" {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "q is required")
			return
		}
		if len(req.Q) > 200 {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "q must be 200 characters or fewer")
			return
		}
		limit := notesListLimit
		if req.Limit > 0 {
			if req.Limit > 200 {
				req.Limit = 200
			}
			limit = req.Limit
		}

		// ILIKE pattern: escape % and _ so a search for "50% off"
		// doesn't turn into "starts with 50". Escape char is \.
		escaped := strings.ReplaceAll(req.Q, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, "%", `\%`)
		escaped = strings.ReplaceAll(escaped, "_", `\_`)
		pattern := "%" + escaped + "%"

		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, tenant_id::text, user_id::text, title, body,
			        tags, pinned, created_at::text, updated_at::text
			   FROM homelab_notes
			  WHERE tenant_id = $1 AND user_id = $2
			    AND (title ILIKE $3 ESCAPE '\' OR body ILIKE $3 ESCAPE '\'
			         OR $4 = ANY(tags))
			  ORDER BY pinned DESC, updated_at DESC
			  LIMIT $5`,
			tenantID, userID, pattern, req.Q, limit,
		)
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
			"query": req.Q,
			"limit": limit,
		})
	}
}

// ------------------------------------------------------------------
// PATCH /api/v1/homelab/notes/:id
// ------------------------------------------------------------------

// PatchHomelabNote updates a note. All fields optional — the
// handler builds a dynamic UPDATE with only the supplied fields,
// matching the services PATCH pattern.
//
// 404 when the row does not belong to the caller (or doesn't exist).
// The frontend sees 404 and silently removes the note from the
// list (the row was deleted by another device, or the user lost
// access).
func PatchHomelabNote(pool *db.Pool) gin.HandlerFunc {
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
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "id must be a uuid")
			return
		}

		var req notePatchReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		sets := []string{}
		args := []interface{}{}
		if req.Title != nil {
			t := strings.TrimSpace(*req.Title)
			if err := validateNoteTitle(t); err != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
				return
			}
			args = append(args, t)
			sets = append(sets, fmt.Sprintf("title = $%d", len(args)))
		}
		if req.Body != nil {
			if err := validateNoteBody(*req.Body); err != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
				return
			}
			args = append(args, *req.Body)
			sets = append(sets, fmt.Sprintf("body = $%d", len(args)))
		}
		if req.Tags != nil {
			if err := validateNoteTags(*req.Tags); err != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
				return
			}
			tags := *req.Tags
			if tags == nil {
				tags = []string{}
			}
			args = append(args, tags)
			sets = append(sets, fmt.Sprintf("tags = $%d", len(args)))
		}
		if req.Pinned != nil {
			args = append(args, *req.Pinned)
			sets = append(sets, fmt.Sprintf("pinned = $%d", len(args)))
		}
		if len(sets) == 0 {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "no fields to update")
			return
		}
		sets = append(sets, "updated_at = NOW()")
		args = append(args, id, tenantID, userID)
		q := "UPDATE homelab_notes SET " + joinStrings(sets, ", ") +
			" WHERE id = $" + fmt.Sprint(len(args)-2) +
			" AND tenant_id = $" + fmt.Sprint(len(args)-1) +
			" AND user_id = $" + fmt.Sprint(len(args))

		tag, err := pool.Pgx().Exec(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if tag.RowsAffected() == 0 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}

		// Return the refreshed row so the UI doesn't have to
		// refetch.
		var (
			rowID, tid, uid, title, body, createdAt, updatedAt string
			tags                                               []string
			pinned                                             bool
		)
		if err := pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT id::text, tenant_id::text, user_id::text, title, body,
			        tags, pinned, created_at::text, updated_at::text
			   FROM homelab_notes
			  WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
			id, tenantID, userID,
		).Scan(&rowID, &tid, &uid, &title, &body, &tags, &pinned, &createdAt, &updatedAt); err != nil {
			kernel.RespondError(c, err)
			return
		}
		if tags == nil {
			tags = []string{}
		}
		kernel.RespondOK(c, noteRow{
			ID:        rowID,
			TenantID:  tid,
			UserID:    uid,
			Title:     title,
			Body:      body,
			Tags:      tags,
			Pinned:    pinned,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}
}

// ------------------------------------------------------------------
// DELETE /api/v1/homelab/notes/:id
// ------------------------------------------------------------------

// DeleteHomelabNote removes a note for the caller. Idempotent:
// 200 with `{deleted: 0}` when the row doesn't exist (or isn't the
// caller's) — the UI treats "it doesn't exist" and "I deleted it"
// as the same outcome.
//
// We do NOT cascade to other tables — notes are isolated by design.
func DeleteHomelabNote(pool *db.Pool) gin.HandlerFunc {
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
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "id must be a uuid")
			return
		}

		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`DELETE FROM homelab_notes
			  WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
			id, tenantID, userID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		kernel.RespondOK(c, gin.H{
			"deleted":       tag.RowsAffected(),
			"note_id":       id.String(),
			"idempotent_ok": true,
		})
	}
}
