package main

import (
	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/handler"
)

// mountTeamRoutes registers the Tier 7.11 Team/Collab (D12)
// protected endpoints — Phase 5 of the 005-tier7-phase3 change,
// the FINAL phase of Tier 7.
//
// Extracted to its own file in Phase 5 because routes_protected.go
// was already mounting two extraction-based route groups
// (mountIncidentRoutes, mountNotebookRoutes) — adding a third tier-
// level group follows the same pattern and keeps routes_protected.go
// focused on top-level tier wiring.
//
// Routes (5):
//
//	GET  /api/v1/dashboards/shared               — list dashboards shared with me
//	POST /api/v1/dashboards/share                — share a dashboard with a user
//	GET  /api/v1/annotations/mentions            — list mentions for the caller
//	POST /api/v1/timeline/comments               — add a comment to an incident timeline
//	GET  /api/v1/timeline/:incident_id/comments  — list comments for an incident
//
// Handlers honor tenant_id from the JWT — no cross-tenant data
// ever crosses the wire. Idempotent migrations in
// migrations/038_team.sql set up the 3 backing tables.
func mountTeamRoutes(protected *gin.RouterGroup, pool *db.Pool) {
	// ---- Dashboard shares ----
	protected.GET("/dashboards/shared", handler.ListSharedDashboards(pool))
	protected.POST("/dashboards/share", handler.ShareDashboard(pool))

	// ---- Mentions inbox (annotations) ----
	protected.GET("/annotations/mentions", handler.ListMentions(pool))

	// ---- Timeline comments ----
	protected.POST("/timeline/comments", handler.AddTimelineComment(pool))
	protected.GET("/timeline/:incident_id/comments", handler.ListTimelineComments(pool))
}
