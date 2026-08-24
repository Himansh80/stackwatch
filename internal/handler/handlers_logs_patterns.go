// Tier 7 — Log Management Full (D3) — Log pattern aggregation.
//
//	GET /api/v1/logs/patterns  — return detected log patterns
//
// A "pattern" is a 100-char prefix of a log message aggregated by
// service. Returns rows shaped like:
//
//	{ service, pattern, sample_count, last_seen_at }
//
// Source data:
//   1. The `log_patterns` table written by ingest pipelines (preferred)
//   2. Live aggregation from the legacy Tier 6 v1 `logs` table when no
//      patterns have been recorded yet
//
// The legacy `logs` table may not exist yet (it's part of Tier 6 v1
// but not all installations have it). When it doesn't, the endpoint
// returns an empty array with total=0 — never a 500.
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// logPatternRow is the canonical pattern shape returned to clients.
type logPatternRow struct {
	Service      string `json:"service"`
	Pattern      string `json:"pattern"`
	SampleCount  int    `json:"sample_count"`
	LastSeenAt   string `json:"last_seen_at"`
}

// ListLogPatterns returns detected log patterns for the tenant.
// Tries log_patterns first, falls back to live aggregation from the
// legacy `logs` table, then to an empty list if neither exists.
func ListLogPatterns(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		ctx := c.Request.Context()
		// Try the canonical log_patterns table first.
		rows, err := pool.Pgx().Query(ctx,
			`SELECT service, pattern, sample_count, last_seen_at::text
			 FROM log_patterns
			 WHERE tenant_id = $1
			 ORDER BY sample_count DESC, last_seen_at DESC
			 LIMIT 100`,
			tenantID)
		if err == nil {
			defer rows.Close()
			out := []logPatternRow{}
			for rows.Next() {
				var p logPatternRow
				if err := rows.Scan(&p.Service, &p.Pattern, &p.SampleCount, &p.LastSeenAt); err != nil {
					continue
				}
				out = append(out, p)
			}
			if len(out) > 0 {
				kernel.RespondOK(c, gin.H{"patterns": out, "total": len(out), "source": "log_patterns"})
				return
			}
		}
		// Fallback: live aggregation from the Tier 6 v1 `logs` table.
		// If the table doesn't exist, the Query call returns an error
		// and we degrade to an empty list rather than a 500.
		lrows, lerr := pool.Pgx().Query(ctx,
			`SELECT service, substring(message from 1 for 100) AS pattern,
			        count(*) AS sample_count, max(ts)::text AS last_seen
			 FROM logs
			 WHERE tenant_id = $1
			   AND ts > NOW() - INTERVAL '24 hours'
			 GROUP BY service, substring(message from 1 for 100)
			 ORDER BY sample_count DESC
			 LIMIT 100`,
			tenantID)
		if lerr != nil {
			// Table missing or query failed — return empty, not 500.
			if strings.Contains(lerr.Error(), "does not exist") ||
				strings.Contains(lerr.Error(), "relation") {
				kernel.RespondOK(c, gin.H{"patterns": []logPatternRow{}, "total": 0, "source": "empty"})
				return
			}
			kernel.RespondError(c, lerr)
			return
		}
		defer lrows.Close()
		out := []logPatternRow{}
		for lrows.Next() {
			var p logPatternRow
			if err := lrows.Scan(&p.Service, &p.Pattern, &p.SampleCount, &p.LastSeenAt); err != nil {
				continue
			}
			out = append(out, p)
		}
		kernel.RespondOK(c, gin.H{"patterns": out, "total": len(out), "source": "logs"})
	}
}
