// Tier 6 — ML anomaly detection endpoints (M2).
//
//	GET  /api/v1/anomaly/list?host_id=X&metric=cpu_pct&since=1h
//	POST /api/v1/anomaly/detect  body: {values: [...], new_value: N, threshold: 3}
//
// list reads pre-computed anomaly_events for the given host/metric.
// detect is a stateless helper: feed it your last N values and a new
// value, get back the z-score + whether it's an anomaly.
package handler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/ml"
)

// detectRequest is the body for POST /anomaly/detect.
type detectRequest struct {
	Values    []float64 `json:"values"`
	NewValue  float64   `json:"new_value"`
	Threshold float64   `json:"threshold,omitempty"` // default 3.0
}

// DetectAnomaly is a stateless helper that returns z-score + anomaly
// flag for a single new value, given a window of recent values.
func DetectAnomaly(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req detectRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "bad json: " + err.Error()})
			return
		}
		if len(req.Values) == 0 {
			c.JSON(400, gin.H{"error": "values must not be empty"})
			return
		}
		threshold := req.Threshold
		if threshold <= 0 {
			threshold = 3.0
		}
		z, isAnomaly := ml.IsAnomalyValue(req.Values, req.NewValue, threshold)
		resp := gin.H{
			"zscore":    z,
			"anomaly":   isAnomaly,
			"threshold": threshold,
			"count":     len(req.Values),
		}
		if isAnomaly {
			resp["severity"] = severityFromZscore(z)
		}
		c.JSON(200, resp)
	}
}

// severityFromZscore maps |z| to a severity label.
func severityFromZscore(z float64) string {
	if z < 0 {
		z = -z
	}
	switch {
	case z >= 5:
		return "critical"
	case z >= 3:
		return "warning"
	default:
		return "info"
	}
}

// ListAnomalies returns pre-computed anomaly events from the
// anomaly_events table for the given host + metric.
//
// Query params:
//   - host_id (optional, filter by server)
//   - metric (optional, filter by metric name)
//   - since (duration string like "1h", "30m", default 24h)
//   - limit (default 100)
func ListAnomalies(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		hostIDStr := c.Query("host_id")
		metric := c.Query("metric")
		sinceStr := c.DefaultQuery("since", "24h")
		limitStr := c.DefaultQuery("limit", "100")
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 || limit > 1000 {
			limit = 100
		}

		// Parse since — accept "1h", "30m", "1d", or raw seconds
		since, err := time.ParseDuration(sinceStr)
		if err != nil {
			since = 24 * time.Hour
		}
		cutoff := time.Now().Add(-since)

		args := []interface{}{cutoff}
		where := "ts >= $1"

		if hostIDStr != "" {
			hid, err := uuid.Parse(hostIDStr)
			if err != nil {
				c.JSON(400, gin.H{"error": "invalid host_id"})
				return
			}
			args = append(args, hid)
			where += fmt.Sprintf(" AND server_id = $%d", len(args))
		}
		if metric != "" {
			args = append(args, metric)
			where += fmt.Sprintf(" AND metric_name = $%d", len(args))
		}
		args = append(args, limit)
		limitIdx := len(args)

		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id, server_id, metric_name, value, zscore, severity, ts
			 FROM anomaly_events
			 WHERE `+where+`
			 ORDER BY ts DESC
			 LIMIT $`+strconv.Itoa(limitIdx), args...)
		if err != nil {
			c.JSON(500, gin.H{"error": "db query: " + err.Error()})
			return
		}
		defer rows.Close()

		type anomalyEvent struct {
			ID         string    `json:"id"`
			ServerID   *string   `json:"server_id"`
			MetricName string    `json:"metric_name"`
			Value      float64   `json:"value"`
			Zscore     float64   `json:"zscore"`
			Severity   string    `json:"severity"`
			TS         time.Time `json:"ts"`
		}

		out := []anomalyEvent{}
		for rows.Next() {
			var e anomalyEvent
			var sid *uuid.UUID
			if err := rows.Scan(&e.ID, &sid, &e.MetricName, &e.Value, &e.Zscore, &e.Severity, &e.TS); err != nil {
				continue
			}
			if sid != nil {
				s := sid.String()
				e.ServerID = &s
			}
			out = append(out, e)
		}

		c.JSON(200, gin.H{
			"anomalies": out,
			"total":     len(out),
			"since":     sinceStr,
			"host_id":   hostIDStr,
			"metric":    metric,
		})
	}
}

// RecordAnomaly is a helper (called by other handlers when they detect
// an anomaly during normal ingestion). Persists to anomaly_events.
func RecordAnomaly(pool *db.Pool, tenantID uuid.UUID, serverID *uuid.UUID,
	metricName string, value, zscore float64, severity string) error {
	_, err := pool.Pgx().Exec(context.Background(),
		`INSERT INTO anomaly_events (tenant_id, server_id, metric_name, value, zscore, severity)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		tenantID, serverID, metricName, value, zscore, severity)
	return err
}
