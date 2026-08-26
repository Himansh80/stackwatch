// Tier 7 Phase 3 — Notebook (D11). CRUD + collaborator management.
//
// Routes:
//
//	GET    /api/v1/notebooks                       — list (filter ?role=viewer|editor|owner)
//	POST   /api/v1/notebooks                       — create (author_id from JWT)
//	GET    /api/v1/notebooks/:id                   — detail (with collaborators)
//	PUT    /api/v1/notebooks/:id                   — update title/content
//	POST   /api/v1/notebooks/:id/collaborators     — add collaborator
//	DELETE /api/v1/notebooks/:id/collaborators/:user_id — remove collaborator
//
// Every protected query honors tenant_id from the JWT — no cross-
// tenant data ever crosses the wire. Author_id is stamped from the
// authenticated user on POST, so the "owner" filter is structural
// (no collaborator row needed). last_edited_at is bumped on every PUT.
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ListNotebooks returns the caller's tenant notebooks, newest-edited
// first. Optional filters: ?role=owner|editor|viewer|all. The "owner"
// filter narrows to notebooks the caller authored (author_id = JWT
// user_id). The "editor" / "viewer" filter joins through the
// notebook_collaborators table. "all" (default) returns everything in
// the tenant — fine for tenants with only a few notebooks.
func ListNotebooks(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		claims, ok := auth.ClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		role := strings.ToLower(strings.TrimSpace(c.Query("role")))
		if role == "" {
			role = "all"
		}
		if !allowedNotebookFilters[role] {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		limit := clampLimit(c.Query("limit"), 100, 500)

		var (
			rows interface {
				Next() bool
				Close()
				Err() error
				// Scan signature matched by the per-case scan() below.
				Scan(...any) error
			}
			err error
		)

		switch role {
		case "owner":
			r, qerr := pool.Pgx().Query(c.Request.Context(),
				`SELECT id::text, title, COALESCE(content, ''),
				        author_id::text, last_edited_at::text, created_at::text
				 FROM notebooks
				 WHERE tenant_id = $1 AND author_id = $2
				 ORDER BY last_edited_at DESC LIMIT $3`,
				tenantID, claims.UserID, limit)
			err = qerr
			rows = r
		case "editor", "viewer":
			r, qerr := pool.Pgx().Query(c.Request.Context(),
				`SELECT n.id::text, n.title, COALESCE(n.content, ''),
				        n.author_id::text, n.last_edited_at::text, n.created_at::text
				 FROM notebooks n
				 JOIN notebook_collaborators c ON c.notebook_id = n.id
				 WHERE n.tenant_id = $1 AND c.user_id = $2 AND c.role = $3
				 ORDER BY n.last_edited_at DESC LIMIT $4`,
				tenantID, claims.UserID, role, limit)
			err = qerr
			rows = r
		default: // "all"
			r, qerr := pool.Pgx().Query(c.Request.Context(),
				`SELECT id::text, title, COALESCE(content, ''),
				        author_id::text, last_edited_at::text, created_at::text
				 FROM notebooks
				 WHERE tenant_id = $1
				 ORDER BY last_edited_at DESC LIMIT $2`,
				tenantID, limit)
			err = qerr
			rows = r
		}
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []notebookRow{}
		for rows.Next() {
			var r notebookRow
			var author *string
			if err := rows.Scan(&r.ID, &r.Title, &r.Content,
				&author, &r.LastEditedAt, &r.CreatedAt); err != nil {
				continue
			}
			r.AuthorID = author
			out = append(out, r)
		}
		kernel.RespondOK(c, gin.H{"notebooks": out, "total": len(out)})
	}
}

// CreateNotebook inserts a new notebook for the caller's tenant.
// author_id is stamped from the JWT user. content defaults to empty
// if the request omits it.
func CreateNotebook(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		claims, ok := auth.ClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r notebookReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var row notebookRow
		var author *string
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO notebooks (tenant_id, title, content, author_id)
			 VALUES ($1, $2, $3, $4)
			 RETURNING id::text, title, COALESCE(content, ''),
			           author_id::text, last_edited_at::text, created_at::text`,
			tenantID, strings.TrimSpace(r.Title), r.Content, claims.UserID,
		).Scan(&row.ID, &row.Title, &row.Content,
			&author, &row.LastEditedAt, &row.CreatedAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		row.AuthorID = author
		kernel.RespondCreated(c, gin.H{"notebook": row})
	}
}

// GetNotebook returns one notebook + its collaborators array.
// 404 if the notebook isn't in the caller's tenant.
func GetNotebook(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var row notebookRow
		var author *string
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT id::text, title, COALESCE(content, ''),
			        author_id::text, last_edited_at::text, created_at::text
			 FROM notebooks
			 WHERE id = $1 AND tenant_id = $2`, id, tenantID,
		).Scan(&row.ID, &row.Title, &row.Content,
			&author, &row.LastEditedAt, &row.CreatedAt)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		row.AuthorID = author

		// Side-load collaborators. We don't filter by tenant_id here —
		// every notebook_id in the result is already proven to belong
		// to this tenant by the SELECT above, so the JOIN can never
		// leak a row from another tenant.
		cols, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT user_id::text, role, added_at::text, notebook_id::text
			 FROM notebook_collaborators
			 WHERE notebook_id = $1
			 ORDER BY added_at ASC`, id)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer cols.Close()
		collabList := []collaboratorRow{}
		for cols.Next() {
			var cc collaboratorRow
			if err := cols.Scan(&cc.UserID, &cc.Role, &cc.AddedAt, &cc.NotebookID); err != nil {
				continue
			}
			collabList = append(collabList, cc)
		}
		kernel.RespondOK(c, gin.H{
			"notebook":      row,
			"collaborators": collabList,
		})
	}
}

// UpdateNotebook updates title and/or content, bumping last_edited_at.
// Only the author OR an 'editor' collaborator can PUT. Owner/editor
// check is structural — we always re-fetch tenant ownership first so
// a non-tenant member can't even read-then-PUT a foreign notebook.
func UpdateNotebook(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		claims, ok := auth.ClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var r notebookUpdateReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		if r.Title == nil && r.Content == nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		// Permission check: author OR collaborator with role='editor'.
		var canEdit bool
		_ = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT EXISTS (
			   SELECT 1 FROM notebooks
			   WHERE id = $1 AND tenant_id = $2 AND author_id = $3
			 ) OR EXISTS (
			   SELECT 1 FROM notebook_collaborators
			   WHERE notebook_id = $1 AND user_id = $2 AND role = 'editor'
			 )`,
			id, tenantID, claims.UserID,
		).Scan(&canEdit)
		if !canEdit {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}

		// Compose UPDATE — only set fields that were provided.
		args := []any{id, tenantID}
		q := "UPDATE notebooks SET last_edited_at = now()"
		if r.Title != nil {
			args = append(args, strings.TrimSpace(*r.Title))
			q += ", title = $" + itoa(len(args))
		}
		if r.Content != nil {
			args = append(args, *r.Content)
			q += ", content = $" + itoa(len(args))
		}
		q += ` WHERE id = $1 AND tenant_id = $2
		       RETURNING id::text, title, COALESCE(content, ''),
		                 author_id::text, last_edited_at::text, created_at::text`

		var row notebookRow
		var author *string
		err = pool.Pgx().QueryRow(c.Request.Context(), q, args...).Scan(
			&row.ID, &row.Title, &row.Content,
			&author, &row.LastEditedAt, &row.CreatedAt)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		row.AuthorID = author
		kernel.RespondOK(c, gin.H{"notebook": row})
	}
}

// AddCollaborator lives in handlers_notebook_collaborators.go
// (split out in Phase 4 to keep handlers_notebook.go under 400 LOC).

// RemoveCollaborator lives in handlers_notebook_collaborators.go
// (split out in Phase 4 to keep handlers_notebook.go under 400 LOC).
