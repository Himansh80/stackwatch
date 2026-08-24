package main

import (
	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/handler"
)

// mountIncidentRoutes registers the Tier 7.9 Service Management (D10)
// protected endpoints — Phase 3 of the 005-tier7-phase3 change.
//
// Extracted to its own file in Phase 3 because routes_protected.go
// was at the 400-LOC cap after Phase 2 (DB Monitoring). Splitting
// keeps routes_protected.go focused on top-level tier wiring while
// the incident / task / war-room / postmortem CRUD lives here.
//
// Routes (10):
//
//	GET    /api/v1/incidents                  — list (filter ?status=X&severity=Y)
//	POST   /api/v1/incidents                  — create incident
//	GET    /api/v1/incidents/:id              — incident detail
//	POST   /api/v1/incidents/:id/acknowledge  — acknowledge (commander_id from JWT)
//	POST   /api/v1/incidents/:id/resolve      — resolve (resolved_at = now())
//	POST   /api/v1/incidents/:id/war-room     — create war room
//	POST   /api/v1/incidents/:id/postmortem   — create postmortem
//	GET    /api/v1/incidents/:id/tasks        — list tasks
//	POST   /api/v1/incidents/:id/tasks        — create task
//	PUT    /api/v1/incidents/:id/tasks/:task_id — update task
//
// Handlers honor tenant_id from the JWT — no cross-tenant data
// ever crosses the wire. Idempotent migrations in
// migrations/037_servicemgmt.sql set up the 4 backing tables.
func mountIncidentRoutes(protected *gin.RouterGroup, pool *db.Pool) {
	// ---- Incident CRUD ----
	protected.GET("/incidents", handler.ListIncidents(pool))
	protected.POST("/incidents", handler.CreateIncident(pool))
	protected.GET("/incidents/:id", handler.GetIncident(pool))
	protected.POST("/incidents/:id/acknowledge", handler.AcknowledgeIncident(pool))
	protected.POST("/incidents/:id/resolve", handler.ResolveIncident(pool))

	// ---- War rooms + postmortems ----
	protected.POST("/incidents/:id/war-room", handler.CreateWarRoom(pool))
	protected.POST("/incidents/:id/postmortem", handler.CreatePostmortem(pool))

	// ---- Tasks (3 routes) ----
	protected.GET("/incidents/:id/tasks", handler.ListTasks(pool))
	protected.POST("/incidents/:id/tasks", handler.CreateTask(pool))
	protected.PUT("/incidents/:id/tasks/:task_id", handler.UpdateTask(pool))
}
