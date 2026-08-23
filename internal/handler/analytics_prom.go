// Tier 6 — Prometheus compat (M3) + Loki push (M6).
//
//	POST /api/v1/prom/write  (Prometheus remote_write JSON variant)
//	POST /api/v1/prom/query  (PromQL-lite: metric, agg, label filter)
//	POST /api/v1/loki/push   (Loki-compatible JSON push)
//	GET  /api/v1/loki/query  (count logs per host per minute)
//
// Both use the existing metric_points table. Loki entries use
// metric_name = 'log' with value = 1 and the message stored in the
// message column.
package handler

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/promql"
)

// promWriteSample is one (ts, value) pair in Prometheus remote_write JSON.
type promWriteSample struct {
	// Prometheus uses [ts_ms, "value_str"] but we'll be flexible
	Ts    int64   `json:"ts_ms"`
	Value float64 `json:"value"`
}

// promWriteSeries is one time-series in the write request.
type promWriteSeries struct {
	Labels  map[string]string `json:"labels"`
	Samples []promWriteSample `json:"samples"`
}

// promWriteRequest is the body of POST /prom/write.
type promWriteRequest struct {
	Timeseries []promWriteSeries `json:"timeseries"`
}

// PromWrite accepts Prometheus remote_write (JSON variant).
//
// We require host_id in labels (StackWatch is multi-tenant + per-host).
// tenant_id is taken from JWT, not from request.
func PromWrite(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			c.JSON(401, gin.H{"error": "tenant not in context"})
			return
		}

		var req promWriteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "bad json: " + err.Error()})
			return
		}
		if len(req.Timeseries) == 0 {
			c.JSON(400, gin.H{"error": "no timeseries in request"})
			return
		}

		// Insert all samples in one transaction.
		tx, err := pool.Pgx().Begin(c.Request.Context())
		if err != nil {
			c.JSON(500, gin.H{"error": "tx begin: " + err.Error()})
			return
		}
		defer tx.Rollback(c.Request.Context())

		inserted := 0
		for _, series := range req.Timeseries {
			metricName := series.Labels["metric"]
			if metricName == "" {
				metricName = series.Labels["__name__"] // Prometheus convention
			}
			if metricName == "" {
				continue
			}
			// host_id (optional)
			var serverID *uuid.UUID
			if h := series.Labels["host_id"]; h != "" {
				hid, err := uuid.Parse(h)
				if err == nil {
					serverID = &hid
				}
			}
			// Build labels JSONB from non-special labels
			labelMap := map[string]string{}
			for k, v := range series.Labels {
				if k != "metric" && k != "__name__" && k != "host_id" {
					labelMap[k] = v
				}
			}
			for _, sample := range series.Samples {
				ts := time.UnixMilli(sample.Ts)
				_, err := tx.Exec(c.Request.Context(),
					`INSERT INTO metric_points (tenant_id, server_id, metric_name, value, ts, labels)
					 VALUES ($1, $2, $3, $4, $5, $6)`,
					tenantID, serverID, metricName, sample.Value, ts, mustMarshalJSON(labelMap))
				if err != nil {
					c.JSON(500, gin.H{"error": "insert: " + err.Error()})
					return
				}
				inserted++
			}
		}
		if err := tx.Commit(c.Request.Context()); err != nil {
			c.JSON(500, gin.H{"error": "tx commit: " + err.Error()})
			return
		}
		c.JSON(200, gin.H{
			"status":   "ok",
			"inserted": inserted,
		})
	}
}

// promQueryRequest is the body of POST /prom/query.
//
// Supports PromQL-lite: "metric", "agg(metric)", "metric{label=val}".
// Returns Prometheus-compatible JSON envelope.
type promQueryRequest struct {
	Query  string `json:"query"`
	From   int64  `json:"from"`    // unix ms
	To     int64  `json:"to"`      // unix ms
	Step   int64  `json:"step"`    // bucket size in ms (default 30000)
	HostID string `json:"host_id"` // optional server filter
}

// PromQuery runs a PromQL-lite query and returns Prometheus-format result.
func PromQuery(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			c.JSON(401, gin.H{"error": "tenant not in context"})
			return
		}

		var req promQueryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "bad json: " + err.Error()})
			return
		}
		if req.Query == "" {
			c.JSON(400, gin.H{"error": "query is required"})
			return
		}
		step := req.Step
		if step <= 0 {
			step = 30000
		}
		if req.From == 0 || req.To == 0 {
			now := time.Now()
			req.From = now.Add(-1 * time.Hour).UnixMilli()
			req.To = now.UnixMilli()
		}

		q, err := promql.Parse(req.Query)
		if err != nil {
			c.JSON(400, gin.H{"error": "parse query: " + err.Error()})
			return
		}

		// Build SQL with time range + tenant filter.
		// We use q.ToSQL for the metric/label part, then add time + tenant WHERE.
		baseSQL, args := q.ToSQL(step)
		from := time.UnixMilli(req.From)
		to := time.UnixMilli(req.To)
		args = append(args, tenantID, from, to)
		tenantIdx := len(args) - 2
		fromIdx := len(args) - 1
		toIdx := len(args)
		extras := fmt.Sprintf(" AND tenant_id = $%d AND ts >= $%d AND ts < $%d", tenantIdx, fromIdx, toIdx)
		if req.HostID != "" {
			hid, err := uuid.Parse(req.HostID)
			if err != nil {
				c.JSON(400, gin.H{"error": "invalid host_id"})
				return
			}
			args = append(args, hid)
			hidIdx := len(args)
			extras += fmt.Sprintf(" AND server_id = $%d", hidIdx)
		}
		// Inject extras BEFORE GROUP BY (if present) else before ORDER BY.
		var fullSQL string
		if strings.Contains(baseSQL, "GROUP BY") {
			fullSQL = strings.Replace(baseSQL, "GROUP BY", extras+" GROUP BY", 1)
		} else {
			fullSQL = strings.Replace(baseSQL, "ORDER BY", extras+" ORDER BY", 1)
		}

		rows, err := pool.Pgx().Query(c.Request.Context(), fullSQL, args...)
		if err != nil {
			c.JSON(500, gin.H{"error": "query: " + err.Error(), "sql": fullSQL})
			return
		}
		defer rows.Close()

		// Prometheus-format result
		type metric struct {
			Metric map[string]string `json:"metric"`
			Values [][2]interface{}  `json:"values"`
		}
		var result []metric
		var single metric
		single.Metric = map[string]string{"__name__": q.Metric}
		for k, v := range q.Labels {
			single.Metric[k] = v
		}
		if q.Agg == "" {
			// Raw samples — return one series per (ts, value)
			single.Values = [][2]interface{}{}
			for rows.Next() {
				var ts time.Time
				var v float64
				if err := rows.Scan(&ts, &v); err == nil {
					single.Values = append(single.Values, [2]interface{}{
						float64(ts.UnixMilli()) / 1000, v,
					})
				}
			}
			result = []metric{single}
		} else {
			// Bucketed aggregation
			single.Values = [][2]interface{}{}
			for rows.Next() {
				var ts time.Time
				var v float64
				if err := rows.Scan(&ts, &v); err == nil {
					single.Values = append(single.Values, [2]interface{}{
						float64(ts.UnixMilli()) / 1000, promql.FormatValue(v),
					})
				}
			}
			result = []metric{single}
		}

		c.JSON(200, gin.H{
			"status": "success",
			"data": gin.H{
				"resultType": "matrix",
				"result":     result,
			},
		})
	}
}

// lokiStream is one stream in Loki push format.
type lokiStream struct {
	Labels  string      `json:"labels"` // raw label string e.g. `{job="syslog"}`
	Entries []lokiEntry `json:"entries"`
}

// lokiEntry is one log line.
type lokiEntry struct {
	Ts   string `json:"ts"` // ISO8601 or nanoseconds
	Line string `json:"line"`
}

// lokiPushRequest is the body of POST /loki/push.
type lokiPushRequest struct {
	Streams []lokiStream `json:"streams"`
}

// LokiPush accepts a Loki-compatible push.
//
// Each entry becomes a metric_points row with metric_name='log',
// value=1, message=<line>, labels=<parsed labels>.
func LokiPush(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			c.JSON(401, gin.H{"error": "tenant not in context"})
			return
		}

		var req lokiPushRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "bad json: " + err.Error()})
			return
		}
		if len(req.Streams) == 0 {
			c.JSON(400, gin.H{"error": "no streams"})
			return
		}

		var serverID *uuid.UUID
		if sid := c.Query("host_id"); sid != "" {
			hid, err := uuid.Parse(sid)
			if err == nil {
				serverID = &hid
			}
		}

		tx, err := pool.Pgx().Begin(c.Request.Context())
		if err != nil {
			c.JSON(500, gin.H{"error": "tx begin: " + err.Error()})
			return
		}
		defer tx.Rollback(c.Request.Context())

		inserted := 0
		for _, stream := range req.Streams {
			labels := parseLokiLabels(stream.Labels)
			for _, entry := range stream.Entries {
				ts, err := parseLokiTs(entry.Ts)
				if err != nil {
					ts = time.Now()
				}
				_, err = tx.Exec(c.Request.Context(),
					`INSERT INTO metric_points (tenant_id, server_id, metric_name, value, ts, message, labels)
					 VALUES ($1, $2, 'log', 1, $3, $4, $5)`,
					tenantID, serverID, ts, entry.Line, mustMarshalJSON(labels))
				if err != nil {
					c.JSON(500, gin.H{"error": "insert: " + err.Error()})
					return
				}
				inserted++
			}
		}
		if err := tx.Commit(c.Request.Context()); err != nil {
			c.JSON(500, gin.H{"error": "tx commit: " + err.Error()})
			return
		}
		c.JSON(200, gin.H{
			"status":   "ok",
			"inserted": inserted,
		})
	}
}

// LokiQuery returns a count of log entries per host per minute bucket.
//
// Query params:
//   - since (duration string, default 24h)
//   - host_id (optional)
//   - job (optional, label filter)
func LokiQuery(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			c.JSON(401, gin.H{"error": "tenant not in context"})
			return
		}

		sinceStr := c.DefaultQuery("since", "24h")
		since, err := time.ParseDuration(sinceStr)
		if err != nil {
			since = 24 * time.Hour
		}
		cutoff := time.Now().Add(-since)

		args := []interface{}{tenantID, "log", cutoff}
		where := "tenant_id = $1 AND metric_name = $2 AND ts >= $3"

		if h := c.Query("host_id"); h != "" {
			hid, err := uuid.Parse(h)
			if err != nil {
				c.JSON(400, gin.H{"error": "invalid host_id"})
				return
			}
			args = append(args, hid)
			where += fmt.Sprintf(" AND server_id = $%d", len(args))
		}
		if job := c.Query("job"); job != "" {
			args = append(args, job)
			where += fmt.Sprintf(" AND labels->>'job' = $%d", len(args))
		}

		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT server_id, COUNT(*), MIN(ts), MAX(ts)
			 FROM metric_points
			 WHERE `+where+`
			 GROUP BY server_id
			 ORDER BY COUNT(*) DESC`,
			args...)
		if err != nil {
			c.JSON(500, gin.H{"error": "query: " + err.Error()})
			return
		}
		defer rows.Close()

		type hostCount struct {
			ServerID  *string `json:"server_id"`
			Count     int64   `json:"count"`
			FirstSeen string  `json:"first_seen"`
			LastSeen  string  `json:"last_seen"`
		}
		out := []hostCount{}
		for rows.Next() {
			var hc hostCount
			var sid *uuid.UUID
			var first, last time.Time
			if err := rows.Scan(&sid, &hc.Count, &first, &last); err != nil {
				continue
			}
			if sid != nil {
				s := sid.String()
				hc.ServerID = &s
			}
			hc.FirstSeen = first.UTC().Format(time.RFC3339)
			hc.LastSeen = last.UTC().Format(time.RFC3339)
			out = append(out, hc)
		}
		c.JSON(200, gin.H{
			"hosts": out,
			"total": len(out),
			"since": sinceStr,
		})
	}
}

// parseLokiLabels parses `{job="syslog",host="x"}` into a map.
func parseLokiLabels(s string) map[string]string {
	out := map[string]string{}
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "{")
	s = strings.TrimSuffix(s, "}")
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		if !strings.Contains(pair, "=") {
			continue
		}
		kv := strings.SplitN(pair, "=", 2)
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}
		out[key] = val
	}
	return out
}

// parseLokiTs parses a Loki timestamp — either an ISO string or nanosec string.
func parseLokiTs(s string) (time.Time, error) {
	if s == "" {
		return time.Now(), nil
	}
	// Try as int64 nanoseconds
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(0, n), nil
	}
	// Try as float (Loki sometimes sends "1.7e+09")
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return time.Unix(0, int64(f*1e9)), nil
	}
	// Try as RFC3339Nano
	return time.Parse(time.RFC3339Nano, s)
}

// mustMarshalJSON marshals v or returns "{}" on error.
// Used for JSONB columns where a bad value should fail-soft to empty.
func mustMarshalJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}
