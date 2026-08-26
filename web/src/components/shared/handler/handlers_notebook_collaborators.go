// Tier 7 Phase 3 — Notebook (D11). Collaborator management.
//
// Routes:
//
//	POST   /api/v1/notebooks/:id/collaborators            — add collaborator
//	DELETE /api/v1/notebooks/:id/collaborators/:user_id   — remove collaborator
//
// The collaborator table is the only Notebook (D11) table that
// doesn't carry tenant_id directly — every read and write joins
// through notebooks.tenant_id so cross-tenant collab smuggling is
// structurally impossible. Only the notebook author or an existing
// 'editor' collaborator may add or remove other collaborators;
// viewers cannot. 404 is returned on permission failures so we never
// leak the existence of a foreign row.
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// AddCollaborator inserts a (notebook_id, user_id, role) row. The
// composite PRIMARY KEY on notebook_collaborators turns duplicates
// into an ON CONFLICT no-op (we update the role if it changed).
// Anyone who can edit the notebook can also add collaborators — we
// re-use the same author/editor check.
func AddCollaborator(pool *db.Pool) gin.HandlerFunc {
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
		var r collaboratorReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		role := strings.ToLower(strings.TrimSpace(r.Role))
		if role == "" {
			role = "viewer"
		}
		if !allowedNotebookRoles[role] {
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
		userID, err := uuid.Parse(r.UserID)
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var row collaboratorRow
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO notebook_collaborators (notebook_id, user_id, role)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (notebook_id, user_id) DO UPDATE SET role = EXCLUDED.role
			 RETURNING user_id::text, role, added_at::text, notebook_id::text`,
			id, userID, role,
		).Scan(&row.UserID, &row.Role, &row.AddedAt, &row.NotebookID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondCreated(c, gin.H{"collaborator": row})
	}
}

// RemoveCollaborator deletes a (notebook_id, user_id) row. Only the
// notebook author or an 'editor' collaborator may remove. Returns 404
// if the collaborator didn't exist (treated identically to "wrong
// tenant" so we never leak the existence of a foreign row).
func RemoveCollaborator(pool *db.Pool) gin.HandlerFunc {
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
		userID, err := uuid.Parse(c.Param("user_id"))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		// Tenant ownership check + permission check in one shot.
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
		// The notebook_id in the collaborators table is FK CASCADE, so
		// tenant isolation is implicit — we don't repeat the tenant_id
		// filter on the DELETE.
		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`DELETE FROM notebook_collaborators
			 WHERE notebook_id = $1 AND user_id = $2`,
			id, userID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if tag.RowsAffected() == 0 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		kernel.RespondOK(c, gin.H{"deleted": true, "notebook_id": id.String(), "user_id": userID.String()})
	}
}
