// Tier 7 Phase 3 — Service Management (D10). Incident CRUD.
//
// Routes:
//
//	GET    /api/v1/incidents                  — list (filter ?status=X&severity=Y)
//	POST   /api/v1/incidents                  — create incident
//	GET    /api/v1/incidents/:id              — incident detail
//	POST   /api/v1/incidents/:id/acknowledge  — acknowledge (commander_id from JWT)
//	POST   /api/v1/incidents/:id/resolve      — resolve (resolved_at = now())
//
// Every protected query honors tenant_id from the JWT — no cross-
// tenant data ever crosses the wire. Severity and status enums are
// enforced at the API edge so the column never holds garbage.
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ListIncidents returns the caller's tenant incidents, newest first.
// Optional filters: ?status=open&severity=sev1&limit=N.
func ListIncidents(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		status := strings.ToLower(strings.TrimSpace(c.Query("status")))
		severity := strings.ToLower(strings.TrimSpace(c.Query("severity")))
		limit := clampLimit(c.Query("limit"), 100, 500)

		args := []any{tenantID}
		q := `SELECT id::text, title, COALESCE(description, ''),
		             severity, status, commander_id::text,
		             started_at::text, resolved_at::text, postmortem_id::text
		      FROM incidents
		      WHERE tenant_id = $1`
		if allowedIncidentStatuses[status] {
			args = append(args, status)
			q += " AND status = $" + itoa(len(args))
		}
		if allowedIncidentSeverities[severity] {
			args = append(args, severity)
			q += " AND severity = $" + itoa(len(args))
		}
		args = append(args, limit)
		q += " ORDER BY started_at DESC LIMIT $" + itoa(len(args))

		rows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []incidentRow{}
		for rows.Next() {
			var r incidentRow
			var commander *string
			var resolved *string
			var postmortem *string
			if err := rows.Scan(&r.ID, &r.Title, &r.Description,
				&r.Severity, &r.Status, &commander,
				&r.StartedAt, &resolved, &postmortem); err != nil {
				continue
			}
			r.CommanderID = commander
			r.ResolvedAt = resolved
			r.PostmortemID = postmortem
			out = append(out, r)
		}
		kernel.RespondOK(c, gin.H{"incidents": out, "total": len(out)})
	}
}

// CreateIncident inserts a new incident for the caller's tenant.
// Defaults to status='open', commander_id=NULL, resolved_at=NULL.
func CreateIncident(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r incidentReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var row incidentRow
		var commander *string
		var resolved *string
		var postmortem *string
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO incidents (tenant_id, title, description, severity)
			 VALUES ($1, $2, $3, $4)
			 RETURNING id::text, title, COALESCE(description, ''),
			           severity, status, commander_id::text,
			           started_at::text, resolved_at::text, postmortem_id::text`,
			tenantID, strings.TrimSpace(r.Title), r.Description, r.Severity,
		).Scan(&row.ID, &row.Title, &row.Description,
			&row.Severity, &row.Status, &commander,
			&row.StartedAt, &resolved, &postmortem)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		row.CommanderID = commander
		row.ResolvedAt = resolved
		row.PostmortemID = postmortem
		kernel.RespondCreated(c, gin.H{"incident": row})
	}
}

// GetIncident returns a single incident + count of war rooms and tasks.
// War rooms + postmortems are looked up lazily by the UI; we only surface
// the counts here so the detail header can render badges.
func GetIncident(pool *db.Pool) gin.HandlerFunc {
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
		var row incidentRow
		var commander *string
		var resolved *string
		var postmortem *string
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT id::text, title, COALESCE(description, ''),
			        severity, status, commander_id::text,
			        started_at::text, resolved_at::text, postmortem_id::text
			 FROM incidents
			 WHERE id = $1 AND tenant_id = $2`, id, tenantID,
		).Scan(&row.ID, &row.Title, &row.Description,
			&row.Severity, &row.Status, &commander,
			&row.StartedAt, &resolved, &postmortem)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		row.CommanderID = commander
		row.ResolvedAt = resolved
		row.PostmortemID = postmortem

		var warRoomCount int
		_ = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT COUNT(*) FROM war_rooms WHERE incident_id = $1`, id,
		).Scan(&warRoomCount)
		var taskCount int
		_ = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT COUNT(*) FROM tasks WHERE incident_id = $1`, id,
		).Scan(&taskCount)

		kernel.RespondOK(c, gin.H{
			"incident":       row,
			"war_room_count": warRoomCount,
			"task_count":     taskCount,
		})
	}
}

// AcknowledgeIncident sets status='acknowledged' and stamps commander_id
// from the JWT user. Idempotent: a second call on an already-acknowledged
// incident just refreshes the commander_id.
func AcknowledgeIncident(pool *db.Pool) gin.HandlerFunc {
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
		var row incidentRow
		var commander *string
		var resolved *string
		var postmortem *string
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`UPDATE incidents
			 SET status = 'acknowledged', commander_id = $3
			 WHERE id = $1 AND tenant_id = $2
			 RETURNING id::text, title, COALESCE(description, ''),
			           severity, status, commander_id::text,
			           started_at::text, resolved_at::text, postmortem_id::text`,
			id, tenantID, claims.UserID,
		).Scan(&row.ID, &row.Title, &row.Description,
			&row.Severity, &row.Status, &commander,
			&row.StartedAt, &resolved, &postmortem)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		row.CommanderID = commander
		row.ResolvedAt = resolved
		row.PostmortemID = postmortem
		kernel.RespondOK(c, gin.H{"incident": row})
	}
}

// ResolveIncident sets status='resolved' and stamps resolved_at=now().
// Returns 404 if the incident isn't found in the caller's tenant.
func ResolveIncident(pool *db.Pool) gin.HandlerFunc {
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
		var row incidentRow
		var commander *string
		var resolved *string
		var postmortem *string
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`UPDATE incidents
			 SET status = 'resolved', resolved_at = now()
			 WHERE id = $1 AND tenant_id = $2
			 RETURNING id::text, title, COALESCE(description, ''),
			           severity, status, commander_id::text,
			           started_at::text, resolved_at::text, postmortem_id::text`,
			id, tenantID,
		).Scan(&row.ID, &row.Title, &row.Description,
			&row.Severity, &row.Status, &commander,
			&row.StartedAt, &resolved, &postmortem)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		row.CommanderID = commander
		row.ResolvedAt = resolved
		row.PostmortemID = postmortem
		kernel.RespondOK(c, gin.H{"incident": row})
	}
}
