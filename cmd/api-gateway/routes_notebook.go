package main

import (
	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/handler"
)

// mountNotebookRoutes registers the Tier 7.10 Notebook (D11)
// protected endpoints — Phase 4 of the 005-tier7-phase3 change.
//
// Extracted to its own file in Phase 4 because routes_protected.go
// was at the 400-LOC cap after Phase 3 (Service Management).
// Splitting keeps routes_protected.go focused on top-level tier
// wiring while the notebook + collaborator CRUD lives here.
//
// Routes (6):
//
//	GET    /api/v1/notebooks                        — list (filter ?role=viewer|editor|owner|all)
//	POST   /api/v1/notebooks                        — create notebook (author_id from JWT)
//	GET    /api/v1/notebooks/:id                    — detail (with collaborators array)
//	PUT    /api/v1/notebooks/:id                    — update title/content (bumps last_edited_at)
//	POST   /api/v1/notebooks/:id/collaborators      — add collaborator (body: {user_id, role})
//	DELETE /api/v1/notebooks/:id/collaborators/:user_id — remove collaborator
//
// Handlers honor tenant_id from the JWT — no cross-tenant data
// ever crosses the wire. Idempotent migrations in
// migrations/038_notebook.sql set up the 2 backing tables.
func mountNotebookRoutes(protected *gin.RouterGroup, pool *db.Pool) {
	// ---- Notebook CRUD ----
	protected.GET("/notebooks", handler.ListNotebooks(pool))
	protected.POST("/notebooks", handler.CreateNotebook(pool))
	protected.GET("/notebooks/:id", handler.GetNotebook(pool))
	protected.PUT("/notebooks/:id", handler.UpdateNotebook(pool))

	// ---- Collaborator management ----
	protected.POST("/notebooks/:id/collaborators", handler.AddCollaborator(pool))
	protected.DELETE("/notebooks/:id/collaborators/:user_id", handler.RemoveCollaborator(pool))
}
