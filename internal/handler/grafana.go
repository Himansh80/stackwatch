// Tier 6 v2 — Grafana Prometheus datasource adapter (M4).
//
//   GET  /api/v1/grafana/search?query=<prefix>
//   POST /api/v1/grafana/query  body: {query, from, to, step}
//
// These match Grafana's built-in Prometheus datasource protocol so
// existing Grafana instances can connect to StackWatch without writing
// a custom plugin. Auth via JWT in the Authorization header (use the
// user's API key in Grafana's "Basic Auth" field with `Bearer ` prefix).
package handler

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/promql"
)

// GrafanaSearch returns metric names matching the prefix (or all if empty).
// Grafana calls this to populate the query-builder metric dropdown.
func GrafanaSearch(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			c.JSON(401, gin.H{"error": "tenant not in context"})
			return
		}
		query := c.Query("query")
		// Grafana convention: empty query = list ALL metrics.
		// Non-empty query = LIKE prefix (or exact match if no wildcard).
		// We just do LIKE with the prefix; add % for substring matches.
		like := query
		if like != "" && !strings.Contains(like, "%") && !strings.Contains(like, "_") {
			like = like + "%"
		}

		var rows interface {
			Close()
			Next() bool
			Scan(...interface{}) error
		}
		var err error
		if like == "" {
			rows, err = pool.Pgx().Query(c.Request.Context(),
				`SELECT DISTINCT metric_name FROM metric_points
				 WHERE tenant_id = $1
				 ORDER BY metric_name LIMIT 1000`, tenantID)
		} else {
			rows, err = pool.Pgx().Query(c.Request.Context(),
				`SELECT DISTINCT metric_name FROM metric_points
				 WHERE tenant_id = $1 AND metric_name LIKE $2
				 ORDER BY metric_name LIMIT 1000`, tenantID, like)
		}
		if err != nil {
			c.JSON(500, gin.H{"error": "query: " + err.Error()})
			return
		}
		defer rows.Close()

		out := []string{}
		for rows.Next() {
			var m string
			if err := rows.Scan(&m); err == nil && m != "" {
				out = append(out, m)
			}
		}
		// Grafana expects a JSON array of strings (or {label: ...} objects)
		c.JSON(200, out)
	}
}

// grafanaQueryRequest matches what Grafana sends for /api/v1/query.
type grafanaQueryRequest struct {
	Query    string `json:"query"`
	From     int64  `json:"from"`     // unix seconds (not ms)
	To       int64  `json:"to"`       // unix seconds
	Step     int64  `json:"step"`     // seconds
	Instant  bool   `json:"instant"`  // for /api/v1/query (instant query)
}

// GrafanaQuery runs a PromQL-lite query and returns Grafana-format result.
//
// Note: Grafana sends unix SECONDS, our /prom/query uses unix MS.
// We convert here.
func GrafanaQuery(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			c.JSON(401, gin.H{"error": "tenant not in context"})
			return
		}

		var req grafanaQueryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "bad json: " + err.Error()})
			return
		}
		if req.Query == "" {
			c.JSON(400, gin.H{"error": "query is required"})
			return
		}

		step := req.Step * 1000 // Grafana s → our ms
		if step <= 0 {
			step = 30000
		}
		fromMs := req.From * 1000
		toMs := req.To * 1000
		if fromMs == 0 || toMs == 0 {
			now := time.Now()
			fromMs = now.Add(-1 * time.Hour).UnixMilli()
			toMs = now.UnixMilli()
		}

		q, err := promql.Parse(req.Query)
		if err != nil {
			c.JSON(400, gin.H{"error": "parse: " + err.Error()})
			return
		}

		baseSQL, args := q.ToSQL(step)
		from := time.UnixMilli(fromMs)
		to := time.UnixMilli(toMs)
		args = append(args, tenantID, from, to)
		tenantIdx := len(args) - 2
		fromIdx := len(args) - 1
		toIdx := len(args)
		extras := " AND tenant_id = $" + grafanaItoa(tenantIdx) +
			" AND ts >= $" + grafanaItoa(fromIdx) +
			" AND ts < $" + grafanaItoa(toIdx)
		var fullSQL string
		if strings.Contains(baseSQL, "GROUP BY") {
			fullSQL = strings.Replace(baseSQL, "GROUP BY", extras+" GROUP BY", 1)
		} else {
			fullSQL = strings.Replace(baseSQL, "ORDER BY", extras+" ORDER BY", 1)
		}

		rows, err := pool.Pgx().Query(c.Request.Context(), fullSQL, args...)
		if err != nil {
			c.JSON(500, gin.H{"error": "query: " + err.Error()})
			return
		}
		defer rows.Close()

		// Grafana expects: data.result = [{target: query, datapoints: [[ms, value]]}]
		// We return Prometheus matrix (same as /prom/query) since Grafana's
		// Prometheus datasource speaks that natively.
		type metric struct {
			Metric map[string]string `json:"metric"`
			Values [][2]interface{}   `json:"values"`
		}
		var single metric
		single.Metric = map[string]string{"__name__": q.Metric}
		for k, v := range q.Labels {
			single.Metric[k] = v
		}
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
		c.JSON(200, gin.H{
			"data": gin.H{
				"resultType": "matrix",
				"result":     []metric{single},
			},
			"status": "success",
		})
	}
}

// itoa — avoid importing strconv in this file (already used elsewhere).
func grafanaItoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
