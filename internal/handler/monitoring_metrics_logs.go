package handler

// monitoring_metrics_logs.go — /api/v1/metrics/query + /api/v1/logs/query endpoints.

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

func IngestHeartbeat(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req IngestHeartbeatRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}

		var tenantID uuid.UUID
		serverID := req.ServerID

		// Path A: explicit server_id provided
		if serverID != uuid.Nil {
			err := pool.Pgx().QueryRow(c.Request.Context(),
				"SELECT tenant_id FROM servers WHERE id = $1 AND deleted_at IS NULL", serverID).Scan(&tenantID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					kernel.RespondError(c, kernel.ErrNotFound)
					return
				}
				kernel.RespondErrorWithCode(c, http.StatusInternalServerError, "internal", err.Error())
				return
			}
		} else if req.Hostname != "" {
			// Path B: auto-register by hostname (idempotent on (hostname) — first tenant wins for now)
			// For self-hosted, there is usually one tenant. For cloud, agent must provide server_id.
			err := pool.Pgx().QueryRow(c.Request.Context(),
				"SELECT id, tenant_id FROM servers WHERE hostname = $1 AND deleted_at IS NULL ORDER BY created_at ASC LIMIT 1",
				req.Hostname).Scan(&serverID, &tenantID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					// Auto-register under the first tenant (self-hosted default)
					terr := pool.Pgx().QueryRow(c.Request.Context(),
						"SELECT id FROM tenants ORDER BY created_at ASC LIMIT 1").Scan(&tenantID)
					if terr != nil {
						kernel.RespondErrorWithCode(c, http.StatusInternalServerError, "internal", terr.Error())
						return
					}
					serverID = uuid.New()
					_, ierr := pool.Pgx().Exec(c.Request.Context(), `
						INSERT INTO servers (id, tenant_id, name, hostname, status)
						VALUES ($1, $2, $3, $3, 'unknown')
					`, serverID, tenantID, req.Hostname)
					if ierr != nil {
						kernel.RespondErrorWithCode(c, http.StatusInternalServerError, "internal", ierr.Error())
						return
					}
				} else {
					kernel.RespondErrorWithCode(c, http.StatusInternalServerError, "internal", err.Error())
					return
				}
			}
		} else {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "server_id or hostname required")
			return
		}

		// Update server last_seen and status
		_, err := pool.Pgx().Exec(c.Request.Context(), `
			UPDATE servers 
			SET last_seen_at = NOW(), status = 'up', updated_at = NOW()
			WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL
		`, tenantID, serverID)
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusInternalServerError, "internal", err.Error())
			return
		}

		// Insert metrics if provided
		if len(req.Metrics) > 0 {
			tx, err := pool.Pgx().Begin(c.Request.Context())
			if err != nil {
				kernel.RespondErrorWithCode(c, http.StatusInternalServerError, "internal", err.Error())
				return
			}
			defer tx.Rollback(c.Request.Context())

			for name, value := range req.Metrics {
				_, err := tx.Exec(c.Request.Context(), `
					INSERT INTO metric_points (tenant_id, server_id, metric_name, value, labels)
					VALUES ($1, $2, $3, $4, '{}')
				`, tenantID, serverID, name, value)
				if err != nil {
					kernel.RespondErrorWithCode(c, http.StatusInternalServerError, "internal", err.Error())
					return
				}
			}

			if err := tx.Commit(c.Request.Context()); err != nil {
				kernel.RespondErrorWithCode(c, http.StatusInternalServerError, "internal", err.Error())
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"ok": true, "server_id": serverID, "tenant_id": tenantID})
	}
}

// QueryMetrics queries time-series metrics
func QueryMetrics(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := getTenantID(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		var req MetricQueryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}

		// Verify all servers belong to tenant
		if len(req.ServerIDs) > 0 {
			placeholders := make([]string, len(req.ServerIDs))
			args := []interface{}{tenantID}
			for i, id := range req.ServerIDs {
				placeholders[i] = "$" + strconv.Itoa(i+2)
				args = append(args, id)
			}
			query := "SELECT COUNT(*) FROM servers WHERE tenant_id = $1 AND id IN (" + join(placeholders, ",") + ") AND deleted_at IS NULL"
			var count int
			if err := pool.Pgx().QueryRow(c.Request.Context(), query, args...).Scan(&count); err != nil || count != len(req.ServerIDs) {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "some servers not found or not accessible")
				return
			}
		}

		// Build query
		start := req.Start
		if start == nil {
			defaultStart := time.Now().Add(-1 * time.Hour)
			start = &defaultStart
		}
		end := req.End
		if end == nil {
			now := time.Now()
			end = &now
		}

		args := []interface{}{tenantID}
		argIdx := 2

		// Handle empty server_ids - no filter
		serverFilter := ""
		if len(req.ServerIDs) > 0 {
			placeholders := make([]string, len(req.ServerIDs))
			for i, id := range req.ServerIDs {
				placeholders[i] = "$" + strconv.Itoa(argIdx)
				args = append(args, id)
				argIdx++
			}
			serverFilter = "AND server_id IN (" + join(placeholders, ",") + ")"
		}

		metricPlaceholders := make([]string, len(req.MetricNames))
		for i, name := range req.MetricNames {
			metricPlaceholders[i] = "$" + strconv.Itoa(argIdx)
			args = append(args, name)
			argIdx++
		}

		args = append(args, *start, *end)

		query := `
			SELECT server_id, metric_name, value, labels, ts
			FROM metric_points
			WHERE tenant_id = $1
			` + serverFilter + `
			AND metric_name IN (` + join(metricPlaceholders, ",") + `)
			AND ts >= $` + strconv.Itoa(argIdx) + `
			AND ts <= $` + strconv.Itoa(argIdx+1) + `
			ORDER BY ts DESC
		`

		if req.Limit > 0 {
			query += " LIMIT $" + strconv.Itoa(argIdx+2)
			args = append(args, req.Limit)
		} else {
			query += " LIMIT 10000"
		}

		rows, err := pool.Pgx().Query(c.Request.Context(), query, args...)
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		defer rows.Close()

		type Point struct {
			ServerID   uuid.UUID       `json:"server_id"`
			MetricName string          `json:"metric_name"`
			Value      float64         `json:"value"`
			Labels     json.RawMessage `json:"labels"`
			Timestamp  time.Time       `json:"timestamp"`
		}

		var points []Point
		for rows.Next() {
			var p Point
			var labels []byte
			if err := rows.Scan(&p.ServerID, &p.MetricName, &p.Value, &labels, &p.Timestamp); err != nil {
				kernel.RespondError(c, kernel.ErrInternal)
				return
			}
			p.Labels = labels
			points = append(points, p)
		}

		c.JSON(http.StatusOK, gin.H{
			"points": points,
			"total":  len(points),
		})
	}
}

// QueryLogs queries log entries (placeholder - uses metric_points for now)
func QueryLogs(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, ok := getTenantID(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		var req LogQueryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}

		// For now, return empty results - real log storage would be in a separate table or Loki
		c.JSON(http.StatusOK, gin.H{
			"logs":  []interface{}{},
			"total": 0,
		})
	}
}
