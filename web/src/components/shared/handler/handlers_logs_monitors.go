// Tier 7 — Log Management Full (D3) — Monitor endpoints.
//
//	POST   /api/v1/logs/monitors       — create a log monitor
//	GET    /api/v1/logs/monitors       — list monitors for the caller's tenant
//	PUT    /api/v1/logs/monitors/:id   — update monitor fields (toggle enabled,
//	                                     change threshold, etc.)
//	DELETE /api/v1/logs/monitors/:id   — delete a monitor
//
// Monitors are alert rules: "if the count of log rows matching <query>
// exceeds <threshold_count> within <threshold_window_seconds>, fire an
// alert at <severity>". The evaluator itself is out of scope for
// Phase 2 (the surface is the storage + the UI; the eval worker
// lands in a future change).
package handler

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// logMonitorReq is the JSON shape for creating a monitor.
//
// All four core fields are required. threshold_count defaults to 1,
// threshold_window_seconds defaults to 300 (5m), severity defaults to
// "warn". enabled defaults to true.
type logMonitorReq struct {
	Name                   string `json:"name" binding:"required,min=1,max=128"`
	Query                  string `json:"query" binding:"required,min=1,max=1024"`
	ThresholdCount         int    `json:"threshold_count" binding:"min=1,max=100000"`
	ThresholdWindowSeconds int    `json:"threshold_window_seconds" binding:"min=10,max=86400"`
	Severity               string `json:"severity" binding:"max=32"`
	Enabled                *bool  `json:"enabled"`
}

// logMonitorUpdateReq is the JSON shape for updating a monitor.
// All fields optional (PATCH semantics).
type logMonitorUpdateReq struct {
	Name                   *string `json:"name" binding:"omitempty,min=1,max=128"`
	Query                  *string `json:"query" binding:"omitempty,min=1,max=1024"`
	ThresholdCount         *int    `json:"threshold_count" binding:"omitempty,min=1,max=100000"`
	ThresholdWindowSeconds *int    `json:"threshold_window_seconds" binding:"omitempty,min=10,max=86400"`
	Severity               *string `json:"severity" binding:"omitempty,max=32"`
	Enabled                *bool   `json:"enabled"`
}

// logMonitorRow is the canonical log monitor shape returned to clients.
type logMonitorRow struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Query                  string `json:"query"`
	ThresholdCount         int    `json:"threshold_count"`
	ThresholdWindowSeconds int    `json:"threshold_window_seconds"`
	Severity               string `json:"severity"`
	Enabled                bool   `json:"enabled"`
	CreatedAt              string `json:"created_at"`
}

// CreateLogMonitor inserts a new monitor for the caller's tenant.
func CreateLogMonitor(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r logMonitorReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		r.Name = strings.TrimSpace(r.Name)
		r.Query = strings.TrimSpace(r.Query)
		r.Severity = strings.TrimSpace(r.Severity)
		if r.Severity == "" {
			r.Severity = "warn"
		}
		if r.ThresholdCount == 0 {
			r.ThresholdCount = 1
		}
		if r.ThresholdWindowSeconds == 0 {
			r.ThresholdWindowSeconds = 300
		}
		enabled := true
		if r.Enabled != nil {
			enabled = *r.Enabled
		}
		var id uuid.UUID
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO log_monitors
			   (tenant_id, name, query, threshold_count, threshold_window_seconds, severity, enabled)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 RETURNING id`,
			tenantID, r.Name, r.Query, r.ThresholdCount, r.ThresholdWindowSeconds, r.Severity, enabled,
		).Scan(&id)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondCreated(c, logMonitorRow{
			ID:                     id.String(),
			Name:                   r.Name,
			Query:                  r.Query,
			ThresholdCount:         r.ThresholdCount,
			ThresholdWindowSeconds: r.ThresholdWindowSeconds,
			Severity:               r.Severity,
			Enabled:                enabled,
			CreatedAt:              time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// ListLogMonitors returns monitors for the caller's tenant, newest first.
func ListLogMonitors(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, name, query, threshold_count, threshold_window_seconds,
			        severity, enabled, created_at::text
			 FROM log_monitors
			 WHERE tenant_id = $1
			 ORDER BY created_at DESC`,
			tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []logMonitorRow{}
		for rows.Next() {
			var m logMonitorRow
			var enabled bool
			if err := rows.Scan(&m.ID, &m.Name, &m.Query, &m.ThresholdCount,
				&m.ThresholdWindowSeconds, &m.Severity, &enabled, &m.CreatedAt); err != nil {
				continue
			}
			m.Enabled = enabled
			out = append(out, m)
		}
		kernel.RespondOK(c, gin.H{"monitors": out, "total": len(out)})
	}
}

// UpdateLogMonitor applies a partial update to a monitor. Only the
// fields present in the JSON body are touched; everything else is
// left unchanged. Returns 404 if the monitor doesn't exist or
// belongs to a different tenant.
func UpdateLogMonitor(pool *db.Pool) gin.HandlerFunc {
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
		var r logMonitorUpdateReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		// Build the SET clause dynamically — only touch fields present
		// in the body. Each call appends a placeholder + value, and we
		// track the next placeholder index.
		sets := []string{}
		args := []any{}
		idx := 1
		if r.Name != nil {
			args = append(args, strings.TrimSpace(*r.Name))
			sets = append(sets, "name = $"+itoa(idx))
			idx++
		}
		if r.Query != nil {
			args = append(args, strings.TrimSpace(*r.Query))
			sets = append(sets, "query = $"+itoa(idx))
			idx++
		}
		if r.ThresholdCount != nil {
			args = append(args, *r.ThresholdCount)
			sets = append(sets, "threshold_count = $"+itoa(idx))
			idx++
		}
		if r.ThresholdWindowSeconds != nil {
			args = append(args, *r.ThresholdWindowSeconds)
			sets = append(sets, "threshold_window_seconds = $"+itoa(idx))
			idx++
		}
		if r.Severity != nil {
			args = append(args, strings.TrimSpace(*r.Severity))
			sets = append(sets, "severity = $"+itoa(idx))
			idx++
		}
		if r.Enabled != nil {
			args = append(args, *r.Enabled)
			sets = append(sets, "enabled = $"+itoa(idx))
			idx++
		}
		if len(sets) == 0 {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		// Append tenant + id for the WHERE clause.
		args = append(args, tenantID, id)
		q := "UPDATE log_monitors SET " + strings.Join(sets, ", ") +
			" WHERE tenant_id = $" + itoa(idx) + " AND id = $" + itoa(idx+1)
		tag, err := pool.Pgx().Exec(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if tag.RowsAffected() == 0 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		// Return the updated row so the client can refresh state.
		var m logMonitorRow
		var enabled bool
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT id::text, name, query, threshold_count, threshold_window_seconds,
			        severity, enabled, created_at::text
			 FROM log_monitors
			 WHERE tenant_id = $1 AND id = $2`,
			tenantID, id,
		).Scan(&m.ID, &m.Name, &m.Query, &m.ThresholdCount, &m.ThresholdWindowSeconds,
			&m.Severity, &enabled, &m.CreatedAt)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		m.Enabled = enabled
		kernel.RespondOK(c, m)
	}
}

// DeleteLogMonitor removes a monitor. 404 if missing or cross-tenant.
func DeleteLogMonitor(pool *db.Pool) gin.HandlerFunc {
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
		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`DELETE FROM log_monitors WHERE tenant_id = $1 AND id = $2`,
			tenantID, id)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if tag.RowsAffected() == 0 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		kernel.RespondOK(c, gin.H{"deleted": id.String()})
	}
}
