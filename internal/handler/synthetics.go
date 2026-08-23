// Tier 6 v2 — Synthetics CRUD endpoints (M7).
//
//	POST   /api/v1/synthetics                — create
//	GET    /api/v1/synthetics                — list
//	GET    /api/v1/synthetics/:id            — get
//	PATCH  /api/v1/synthetics/:id            — update
//	DELETE /api/v1/synthetics/:id            — delete
//	POST   /api/v1/synthetics/:id/run        — run now, return latest result
//	GET    /api/v1/synthetics/:id/results    — result history
package handler

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/synthetics"
)

// createSynthRequest is the body for POST /synthetics.
type createSynthRequest struct {
	Name        string `json:"name" binding:"required"`
	Kind        string `json:"kind" binding:"required,oneof=http tcp icmp"`
	Target      string `json:"target" binding:"required"`
	IntervalSec int    `json:"interval_sec"` // default 60
	TimeoutMs   int    `json:"timeout_ms"`   // default 5000
}

// patchSynthRequest is the body for PATCH /synthetics/:id.
type patchSynthRequest struct {
	Name        *string `json:"name"`
	IntervalSec *int    `json:"interval_sec"`
	TimeoutMs   *int    `json:"timeout_ms"`
	Enabled     *bool   `json:"enabled"`
}

// ListSynthetics returns all synthetics checks for the caller's tenant.
func ListSynthetics(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			c.JSON(401, gin.H{"error": "tenant not in context"})
			return
		}

		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id, name, kind, target, interval_sec, timeout_ms,
			        enabled, last_run_at, last_status, last_ms, created_at
			 FROM synthetics_checks
			 WHERE tenant_id = $1
			 ORDER BY created_at DESC`, tenantID)
		if err != nil {
			c.JSON(500, gin.H{"error": "query: " + err.Error()})
			return
		}
		defer rows.Close()

		type check struct {
			ID          string     `json:"id"`
			Name        string     `json:"name"`
			Kind        string     `json:"kind"`
			Target      string     `json:"target"`
			IntervalSec int        `json:"interval_sec"`
			TimeoutMs   int        `json:"timeout_ms"`
			Enabled     bool       `json:"enabled"`
			LastRunAt   *time.Time `json:"last_run_at,omitempty"`
			LastStatus  *string    `json:"last_status,omitempty"`
			LastMs      *int       `json:"last_ms,omitempty"`
			CreatedAt   time.Time  `json:"created_at"`
		}

		out := []check{}
		for rows.Next() {
			var r check
			var enabled bool
			var lastRun *time.Time
			var lastStatus *string
			var lastMs *int
			if err := rows.Scan(&r.ID, &r.Name, &r.Kind, &r.Target,
				&r.IntervalSec, &r.TimeoutMs, &enabled,
				&lastRun, &lastStatus, &lastMs, &r.CreatedAt); err != nil {
				continue
			}
			r.Enabled = enabled
			r.LastRunAt = lastRun
			r.LastStatus = lastStatus
			r.LastMs = lastMs
			out = append(out, r)
		}
		c.JSON(200, gin.H{"checks": out, "total": len(out)})
	}
}

// CreateSynthetics creates a new check.
func CreateSynthetics(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			c.JSON(401, gin.H{"error": "tenant not in context"})
			return
		}

		var req createSynthRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "bad json: " + err.Error()})
			return
		}
		if req.IntervalSec <= 0 {
			req.IntervalSec = 60
		}
		if req.TimeoutMs <= 0 {
			req.TimeoutMs = 5000
		}
		if req.IntervalSec < 10 {
			c.JSON(400, gin.H{"error": "interval_sec must be >= 10 (rate-limit protection)"})
			return
		}
		if req.TimeoutMs > 60000 {
			c.JSON(400, gin.H{"error": "timeout_ms must be <= 60000"})
			return
		}

		var id uuid.UUID
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO synthetics_checks (tenant_id, name, kind, target, interval_sec, timeout_ms)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING id`,
			tenantID, req.Name, req.Kind, req.Target, req.IntervalSec, req.TimeoutMs).Scan(&id)
		if err != nil {
			c.JSON(500, gin.H{"error": "insert: " + err.Error()})
			return
		}
		c.JSON(201, gin.H{
			"id":           id.String(),
			"name":         req.Name,
			"kind":         req.Kind,
			"target":       req.Target,
			"interval_sec": req.IntervalSec,
			"timeout_ms":   req.TimeoutMs,
			"enabled":      true,
			"status":       "created",
		})
	}
}

// GetSynthetics returns one check.
func GetSynthetics(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}
		tenantID, _ := tenantIDFromContext(c)

		var name, kind, target string
		var intervalSec, timeoutMs int
		var enabled bool
		var lastRun *time.Time
		var lastStatus *string
		var lastMs *int
		var createdAt time.Time
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT name, kind, target, interval_sec, timeout_ms, enabled,
			        last_run_at, last_status, last_ms, created_at
			 FROM synthetics_checks
			 WHERE id = $1 AND tenant_id = $2`, id, tenantID).
			Scan(&name, &kind, &target, &intervalSec, &timeoutMs, &enabled,
				&lastRun, &lastStatus, &lastMs, &createdAt)
		if err != nil {
			c.JSON(404, gin.H{"error": "not found: " + err.Error()})
			return
		}
		c.JSON(200, gin.H{
			"id":           id.String(),
			"name":         name,
			"kind":         kind,
			"target":       target,
			"interval_sec": intervalSec,
			"timeout_ms":   timeoutMs,
			"enabled":      enabled,
			"last_run_at":  lastRun,
			"last_status":  lastStatus,
			"last_ms":      lastMs,
			"created_at":   createdAt,
		})
	}
}

// PatchSynthetics updates a check.
func PatchSynthetics(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}
		tenantID, _ := tenantIDFromContext(c)

		var req patchSynthRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "bad json: " + err.Error()})
			return
		}

		// Build dynamic SET
		sets := []string{}
		args := []interface{}{}
		if req.Name != nil {
			args = append(args, *req.Name)
			sets = append(sets, "name = $"+strconv.Itoa(len(args)))
		}
		if req.IntervalSec != nil {
			if *req.IntervalSec < 10 {
				c.JSON(400, gin.H{"error": "interval_sec must be >= 10"})
				return
			}
			args = append(args, *req.IntervalSec)
			sets = append(sets, "interval_sec = $"+strconv.Itoa(len(args)))
		}
		if req.TimeoutMs != nil {
			if *req.TimeoutMs > 60000 {
				c.JSON(400, gin.H{"error": "timeout_ms must be <= 60000"})
				return
			}
			args = append(args, *req.TimeoutMs)
			sets = append(sets, "timeout_ms = $"+strconv.Itoa(len(args)))
		}
		if req.Enabled != nil {
			args = append(args, *req.Enabled)
			sets = append(sets, "enabled = $"+strconv.Itoa(len(args)))
		}
		if len(sets) == 0 {
			c.JSON(400, gin.H{"error": "no fields to update"})
			return
		}
		args = append(args, id, tenantID)
		q := "UPDATE synthetics_checks SET " + strings.Join(sets, ", ") +
			" WHERE id = $" + strconv.Itoa(len(args)-1) +
			" AND tenant_id = $" + strconv.Itoa(len(args))

		_, err = pool.Pgx().Exec(c.Request.Context(), q, args...)
		if err != nil {
			c.JSON(500, gin.H{"error": "update: " + err.Error()})
			return
		}
		c.JSON(200, gin.H{"id": id.String(), "status": "updated"})
	}
}

// DeleteSynthetics deletes a check (cascades to results).
func DeleteSynthetics(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}
		tenantID, _ := tenantIDFromContext(c)

		_, err = pool.Pgx().Exec(c.Request.Context(),
			`DELETE FROM synthetics_checks WHERE id = $1 AND tenant_id = $2`,
			id, tenantID)
		if err != nil {
			c.JSON(500, gin.H{"error": "delete: " + err.Error()})
			return
		}
		c.JSON(200, gin.H{"id": id.String(), "status": "deleted"})
	}
}

// RunSyntheticsNow runs a check synchronously and returns the result.
func RunSyntheticsNow(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}
		tenantID, _ := tenantIDFromContext(c)

		var name, kind, target string
		var timeoutMs int
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT name, kind, target, timeout_ms FROM synthetics_checks
			 WHERE id = $1 AND tenant_id = $2`, id, tenantID).
			Scan(&name, &kind, &target, &timeoutMs)
		if err != nil {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}

		check := synthetics.Check{
			ID:        id.String(),
			TenantID:  tenantID.String(),
			Name:      name,
			Kind:      synthetics.Kind(kind),
			Target:    target,
			TimeoutMs: timeoutMs,
			Enabled:   true,
		}
		result := synthetics.Run(context.Background(), check)

		// Persist
		_, _ = pool.Pgx().Exec(c.Request.Context(),
			`INSERT INTO synthetics_results (check_id, tenant_id, status, response_ms, status_code, error)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			id, tenantID, result.Status, result.ResponseMs, result.StatusCode, result.Error)
		_, _ = pool.Pgx().Exec(c.Request.Context(),
			`UPDATE synthetics_checks SET last_run_at = NOW(), last_status = $1, last_ms = $2
			 WHERE id = $3`, result.Status, result.ResponseMs, id)

		c.JSON(200, gin.H{
			"id":          id.String(),
			"name":        name,
			"status":      result.Status,
			"response_ms": result.ResponseMs,
			"status_code": result.StatusCode,
			"error":       result.Error,
		})
	}
}

// GetSyntheticsResults returns recent results for one check.
func GetSyntheticsResults(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}
		tenantID, _ := tenantIDFromContext(c)
		sinceStr := c.DefaultQuery("since", "24h")
		limitStr := c.DefaultQuery("limit", "100")
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 || limit > 1000 {
			limit = 100
		}
		since, err := time.ParseDuration(sinceStr)
		if err != nil {
			since = 24 * time.Hour
		}
		cutoff := time.Now().Add(-since)

		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT status, response_ms, status_code, error, ts
			 FROM synthetics_results
			 WHERE check_id = $1 AND tenant_id = $2 AND ts >= $3
			 ORDER BY ts DESC LIMIT $4`,
			id, tenantID, cutoff, limit)
		if err != nil {
			c.JSON(500, gin.H{"error": "query: " + err.Error()})
			return
		}
		defer rows.Close()

		type resultRow struct {
			Status     string    `json:"status"`
			ResponseMs int       `json:"response_ms"`
			StatusCode int       `json:"status_code"`
			Error      string    `json:"error"`
			TS         time.Time `json:"ts"`
		}

		out := []resultRow{}
		for rows.Next() {
			var r resultRow
			if err := rows.Scan(&r.Status, &r.ResponseMs, &r.StatusCode, &r.Error, &r.TS); err != nil {
				continue
			}
			out = append(out, r)
		}
		c.JSON(200, gin.H{
			"results": out,
			"total":   len(out),
			"since":   sinceStr,
		})
	}
}
