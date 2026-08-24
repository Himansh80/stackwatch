// Tier 7 Phase 3 — Notebook (D11). Shared types + access-control maps.
//
// All queries honor tenant_id from the JWT. Role values are enforced
// at the API edge so the column never holds garbage. The two tables
// backing this surface (notebooks, notebook_collaborators) are created
// by the idempotent migration migrations/038_notebook.sql.
//
// Collaborator roles:
//   - 'owner'   → implicit on the author_id (NOT a row in
//                 notebook_collaborators — derived in the API).
//   - 'editor'  → can PUT notebook content.
//   - 'viewer'  → read-only.
//
// The collaborators table does NOT carry tenant_id — every read joins
// through notebooks.tenant_id so cross-tenant collab smuggling is
// structurally impossible.
package handler

import "time"

// Compile-time guard: keep time import even when not used here.
var _ = time.RFC3339

// allowedNotebookRoles — defense in depth at the API edge.
var allowedNotebookRoles = map[string]bool{
	"viewer": true, "editor": true,
}

// allowedNotebookFilters — the ?role=… query values the list endpoint
// honors. 'owner' is filtered server-side via author_id (the creator);
// 'editor' / 'viewer' filter via JOIN to notebook_collaborators.
var allowedNotebookFilters = map[string]bool{
	"owner": true, "editor": true, "viewer": true, "all": true,
}

// notebookRow is the JSON shape returned for a single notebook.
type notebookRow struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	Content      string  `json:"content"`
	AuthorID     *string `json:"author_id,omitempty"`
	LastEditedAt string  `json:"last_edited_at"`
	CreatedAt    string  `json:"created_at"`
}

// notebookReq is the JSON shape for POST /notebooks.
type notebookReq struct {
	Title   string `json:"title"   binding:"required,min=1,max=256"`
	Content string `json:"content" binding:"max=131072"`
}

// notebookUpdateReq is the JSON shape for PUT /notebooks/:id.
// All fields optional — partial updates allowed. last_edited_at is
// always bumped server-side on a successful UPDATE.
type notebookUpdateReq struct {
	Title   *string `json:"title"   binding:"omitempty,min=1,max=256"`
	Content *string `json:"content" binding:"omitempty,max=131072"`
}

// collaboratorRow is the JSON shape for a single collaborator entry.
type collaboratorRow struct {
	UserID    string `json:"user_id"`
	Role      string `json:"role"`
	AddedAt   string `json:"added_at"`
	NotebookID string `json:"notebook_id"`
}

// collaboratorReq is the JSON shape for POST /notebooks/:id/collaborators.
type collaboratorReq struct {
	UserID string `json:"user_id" binding:"required,uuid"`
	Role   string `json:"role"    binding:"omitempty,oneof=viewer editor"`
}
