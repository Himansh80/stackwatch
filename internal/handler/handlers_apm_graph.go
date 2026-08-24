// Tier 7 — APM (D2) — Service map + flame graph.
//
//	GET /api/v1/apm/service-map              — graph nodes + edges
//	GET /api/v1/apm/services/:id/flame-graph — flame graph data
//
// Service map is computed from the last 5 minutes of span traffic.
// Edges are inferred from cross-service parent/child span pairs.
// Flame graph returns the latest N traces' spans with depth info so
// the frontend can render an SVG flame graph.
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// APMServiceMap returns graph data (nodes + edges) computed from the
// last 5 minutes of span traffic. Edges are inferred from parent-child
// span relationships that cross services.
func APMServiceMap(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		ctx := c.Request.Context()
		// Nodes: every service with any span activity in the window.
		rows, err := pool.Pgx().Query(ctx,
			`SELECT s.id::text, s.name,
			        COUNT(sp.id)::float / 300.0 AS request_rate
			 FROM apm_services s
			 LEFT JOIN apm_spans sp
			   ON sp.service_id = s.id
			  AND sp.started_at > NOW() - INTERVAL '5 minutes'
			 WHERE s.tenant_id = $1
			 GROUP BY s.id, s.name
			 ORDER BY s.name`,
			tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		nodes := []gin.H{}
		for rows.Next() {
			var id, name string
			var rr float64
			if err := rows.Scan(&id, &name, &rr); err != nil {
				continue
			}
			nodes = append(nodes, gin.H{
				"id":           id,
				"name":         name,
				"request_rate": rr,
			})
		}
		// Edges: child span's service → parent span's service.
		// Aggregate per pair over the 5m window.
		erows, err := pool.Pgx().Query(ctx,
			`SELECT child_s.id::text AS source, parent_s.id::text AS target,
			        COUNT(*) AS request_count,
			        COUNT(*) FILTER (WHERE child.status = 'error') AS error_count
			 FROM apm_spans child
			 JOIN apm_spans parent
			   ON parent.tenant_id = child.tenant_id
			  AND parent.trace_id = child.trace_id
			  AND parent.span_id = child.parent_span_id
			 JOIN apm_services child_s ON child_s.id = child.service_id
			 JOIN apm_services parent_s ON parent_s.id = parent.service_id
			 WHERE child.tenant_id = $1
			   AND child.parent_span_id IS NOT NULL AND child.parent_span_id != ''
			   AND child_s.id != parent_s.id
			   AND child.started_at > NOW() - INTERVAL '5 minutes'
			 GROUP BY child_s.id, parent_s.id`,
			tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer erows.Close()
		edges := []gin.H{}
		for erows.Next() {
			var source, target string
			var rc, ec int64
			if err := erows.Scan(&source, &target, &rc, &ec); err != nil {
				continue
			}
			edges = append(edges, gin.H{
				"source":        source,
				"target":        target,
				"request_count": rc,
				"error_count":   ec,
			})
		}
		kernel.RespondOK(c, gin.H{
			"nodes": nodes,
			"edges": edges,
		})
	}
}

// APMFlameGraph returns the latest N traces' spans for a service in a
// flame-graph-ready shape: each span's depth in the tree (computed via
// iterative parent-walk). Capped at 500 spans per request.
func APMFlameGraph(pool *db.Pool) gin.HandlerFunc {
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
		ctx := c.Request.Context()
		rows, err := pool.Pgx().Query(ctx,
			`WITH recent AS (
			   SELECT DISTINCT trace_id FROM apm_spans
			   WHERE tenant_id = $1 AND service_id = $2
			   ORDER BY trace_id DESC
			   LIMIT 20
			 )
			 SELECT s.span_id, COALESCE(s.parent_span_id, '') AS parent,
			        s.name, s.duration_us, s.status, s.started_at::text,
			        s.service_id::text
			 FROM apm_spans s
			 JOIN recent r ON r.trace_id = s.trace_id
			 WHERE s.tenant_id = $1
			 ORDER BY s.trace_id, s.started_at
			 LIMIT 500`,
			tenantID, id)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		type spanRow struct {
			SpanID    string
			Parent    string
			Name      string
			DurUS     int64
			Status    string
			StartedAt string
			ServiceID string
		}
		spans := []spanRow{}
		for rows.Next() {
			var s spanRow
			if err := rows.Scan(&s.SpanID, &s.Parent, &s.Name, &s.DurUS, &s.Status, &s.StartedAt, &s.ServiceID); err != nil {
				continue
			}
			spans = append(spans, s)
		}
		// Assign depth via iterative walk per trace.
		depthBySpan := map[string]int{}
		for _, s := range spans {
			depth := 0
			cur := s.Parent
			for i := 0; i < 32 && cur != ""; i++ {
				if d, ok := depthBySpan[cur]; ok {
					depth = d + 1
					break
				}
				// Parent not in batch — treat as root.
				cur = ""
			}
			depthBySpan[s.SpanID] = depth
		}
		// Emit with depth attached.
		out := []gin.H{}
		for _, s := range spans {
			out = append(out, gin.H{
				"span_id":        s.SpanID,
				"parent_span_id": s.Parent,
				"name":           s.Name,
				"duration_us":    s.DurUS,
				"status":         s.Status,
				"started_at":     s.StartedAt,
				"service_id":     s.ServiceID,
				"depth":          depthBySpan[s.SpanID],
			})
		}
		kernel.RespondOK(c, gin.H{
			"service_id": id.String(),
			"spans":      out,
			"total":      len(out),
			"cap":        500,
		})
	}
}
