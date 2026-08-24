// Tier 7 Phase 3 — Service Management (D10). Tasks CRUD.
//
// Routes:
//
//	GET /api/v1/incidents/:id/tasks          — list tasks for one incident
//	POST /api/v1/incidents/:id/tasks         — create task (incident_id from URL)
//	PUT /api/v1/incidents/:id/tasks/:task_id — partial update (any field)
//
// The list endpoint accepts an optional ?status=todo|in_progress|done
// filter; the create endpoint defaults to status='todo' if the caller
// doesn't specify one. The update endpoint uses pointer fields so the
// handler can distinguish "leave alone" from "set to empty".
package handler

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ListTasks returns the tasks attached to one incident. Optional
// ?status=todo filter.
func ListTasks(pool *db.Pool) gin.HandlerFunc {
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
		status := strings.ToLower(strings.TrimSpace(c.Query("status")))

		args := []any{tenantID, id}
		q := `SELECT id::text, incident_id::text, title,
		             assignee_id::text, status, due_at::text, created_at::text
		      FROM tasks
		      WHERE tenant_id = $1 AND incident_id = $2`
		if allowedTaskStatuses[status] {
			args = append(args, status)
			q += " AND status = $" + itoa(len(args))
		}
		q += " ORDER BY created_at DESC"

		rows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []taskRow{}
		for rows.Next() {
			var t taskRow
			var incidentID *string
			var assignee *string
			var due *string
			if err := rows.Scan(&t.ID, &incidentID, &t.Title,
				&assignee, &t.Status, &due, &t.CreatedAt); err != nil {
				continue
			}
			t.IncidentID = incidentID
			t.AssigneeID = assignee
			t.DueAt = due
			out = append(out, t)
		}
		kernel.RespondOK(c, gin.H{"tasks": out, "total": len(out)})
	}
}

// CreateTask inserts a task attached to one incident. Verifies the
// parent incident belongs to the caller's tenant before inserting.
func CreateTask(pool *db.Pool) gin.HandlerFunc {
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
		var r taskReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		// Tenant ownership check on the parent incident.
		var ownCount int
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT COUNT(*) FROM incidents WHERE id = $1 AND tenant_id = $2`,
			id, tenantID,
		).Scan(&ownCount)
		if err != nil || ownCount != 1 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		status := r.Status
		if status == "" {
			status = "todo"
		}
		var assignee interface{}
		if r.AssigneeID != "" {
			if aid, perr := uuid.Parse(r.AssigneeID); perr == nil {
				assignee = aid
			}
		}
		var due interface{}
		if r.DueAt != "" {
			if t, perr := time.Parse(time.RFC3339, r.DueAt); perr == nil {
				due = t
			}
		}
		var row taskRow
		var incidentID *string
		var aID *string
		var dueAt *string
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO tasks (tenant_id, incident_id, title, assignee_id, status, due_at)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING id::text, incident_id::text, title,
			           assignee_id::text, status, due_at::text, created_at::text`,
			tenantID, id, strings.TrimSpace(r.Title), assignee, status, due,
		).Scan(&row.ID, &incidentID, &row.Title,
			&aID, &row.Status, &dueAt, &row.CreatedAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		row.IncidentID = incidentID
		row.AssigneeID = aID
		row.DueAt = dueAt
		kernel.RespondCreated(c, gin.H{"task": row})
	}
}

// UpdateTask partial-updates a task. Only fields the caller includes
// in the JSON body are touched; the rest are left untouched.
func UpdateTask(pool *db.Pool) gin.HandlerFunc {
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
		taskID, err := uuid.Parse(c.Param("task_id"))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var r taskUpdateReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		// Tenant ownership check on the parent incident (defense in
		// depth; the WHERE on tasks already filters by tenant_id).
		var ownCount int
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT COUNT(*) FROM incidents WHERE id = $1 AND tenant_id = $2`,
			id, tenantID,
		).Scan(&ownCount)
		if err != nil || ownCount != 1 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		// Build a dynamic UPDATE so we only touch the columns the
		// caller actually set. starts at $3 ($1 = task_id, $2 = tenant).
		args := []any{taskID, tenantID}
		sets := []string{}
		if r.Title != nil {
			args = append(args, strings.TrimSpace(*r.Title))
			sets = append(sets, "title = $"+itoa(len(args)))
		}
		if r.AssigneeID != nil {
			if *r.AssigneeID == "" {
				args = append(args, nil)
			} else {
				if aid, perr := uuid.Parse(*r.AssigneeID); perr == nil {
					args = append(args, aid)
				} else {
					kernel.RespondError(c, kernel.ErrBadRequest)
					return
				}
			}
			sets = append(sets, "assignee_id = $"+itoa(len(args)))
		}
		if r.Status != nil {
			if !allowedTaskStatuses[*r.Status] {
				kernel.RespondError(c, kernel.ErrBadRequest)
				return
			}
			args = append(args, *r.Status)
			sets = append(sets, "status = $"+itoa(len(args)))
		}
		if r.DueAt != nil {
			if *r.DueAt == "" {
				args = append(args, nil)
			} else if t, perr := time.Parse(time.RFC3339, *r.DueAt); perr == nil {
				args = append(args, t)
			} else {
				kernel.RespondError(c, kernel.ErrBadRequest)
				return
			}
			sets = append(sets, "due_at = $"+itoa(len(args)))
		}
		if len(sets) == 0 {
			// Nothing to update — fetch + return the row as-is.
			kernel.RespondOK(c, gin.H{"noop": true})
			return
		}
		q := `UPDATE tasks SET ` + strings.Join(sets, ", ") +
			` WHERE id = $1 AND tenant_id = $2
			  RETURNING id::text, incident_id::text, title,
			            assignee_id::text, status, due_at::text, created_at::text`
		var row taskRow
		var incidentID *string
		var aID *string
		var dueAt *string
		err = pool.Pgx().QueryRow(c.Request.Context(), q, args...).Scan(
			&row.ID, &incidentID, &row.Title,
			&aID, &row.Status, &dueAt, &row.CreatedAt,
		)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		row.IncidentID = incidentID
		row.AssigneeID = aID
		row.DueAt = dueAt
		kernel.RespondOK(c, gin.H{"task": row})
	}
}
