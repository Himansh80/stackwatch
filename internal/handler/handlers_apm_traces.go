// Tier 7 — APM (D2) — Trace ingest + retrieve.
//
//	POST /api/v1/apm/traces            — ingest trace (OTLP-shaped JSON)
//	GET  /api/v1/apm/traces/:trace_id  — get trace + all spans (capped at 500)
//
// Trace ingest auto-creates the root service if missing for this tenant.
package handler

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// apmTraceReq is the OTLP-shaped JSON body for trace ingest.
type apmTraceReq struct {
	TraceID     string       `json:"trace_id" binding:"required,min=1,max=64"`
	RootSpanID  string       `json:"root_span_id" binding:"required,min=1,max=32"`
	ServiceName string       `json:"service_name" binding:"required,min=1,max=128"`
	DurationUS  int64        `json:"duration_us"`
	Status      string       `json:"status" binding:"max=32"`
	StartedAt   time.Time    `json:"started_at"`
	Spans       []apmSpanReq `json:"spans"`
}

// apmSpanReq is the per-span shape inside a trace.
type apmSpanReq struct {
	SpanID       string          `json:"span_id" binding:"required,min=1,max=32"`
	ParentSpanID string          `json:"parent_span_id" binding:"max=32"`
	ServiceName  string          `json:"service_name"`
	Name         string          `json:"name" binding:"required,min=1,max=256"`
	DurationUS   int64           `json:"duration_us"`
	Status       string          `json:"status" binding:"max=32"`
	StartedAt    time.Time       `json:"started_at"`
	Attributes   json.RawMessage `json:"attributes"`
}

// IngestAPMTrace inserts a trace + N spans atomically (best-effort). The
// service referenced by the root span is auto-created on demand if it
// doesn't yet exist for this tenant.
func IngestAPMTrace(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r apmTraceReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		if r.Status == "" {
			r.Status = "ok"
		}
		if r.StartedAt.IsZero() {
			r.StartedAt = time.Now()
		}
		ctx := c.Request.Context()
		// Ensure root service exists.
		rootSvc, err := upsertAPMService(ctx, pool, tenantID, r.ServiceName, "", "", "")
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		// Insert trace row.
		var traceUUID uuid.UUID
		err = pool.Pgx().QueryRow(ctx,
			`INSERT INTO apm_traces (tenant_id, service_id, trace_id, root_span_id, duration_us, status, started_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 RETURNING id`,
			tenantID, rootSvc, r.TraceID, r.RootSpanID, r.DurationUS, r.Status, r.StartedAt,
		).Scan(&traceUUID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		// Insert spans (best-effort — collect per-span errors).
		insertedSpans := 0
		for _, s := range r.Spans {
			svc := rootSvc
			if s.ServiceName != "" && s.ServiceName != r.ServiceName {
				sid, serr := upsertAPMService(ctx, pool, tenantID, s.ServiceName, "", "", "")
				if serr == nil {
					svc = sid
				}
			}
			status := s.Status
			if status == "" {
				status = "ok"
			}
			startedAt := s.StartedAt
			if startedAt.IsZero() {
				startedAt = r.StartedAt
			}
			attrs := s.Attributes
			if len(attrs) == 0 || string(attrs) == "null" {
				attrs = json.RawMessage(`{}`)
			}
			_, serr := pool.Pgx().Exec(ctx,
				`INSERT INTO apm_spans
				 (tenant_id, trace_id, span_id, parent_span_id, service_id,
				  name, duration_us, status, started_at, attributes)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb)`,
				tenantID, r.TraceID, s.SpanID, nullIfEmpty(s.ParentSpanID),
				svc, s.Name, s.DurationUS, status, startedAt, string(attrs))
			if serr == nil {
				insertedSpans++
			}
		}
		kernel.RespondCreated(c, gin.H{
			"trace_id":       r.TraceID,
			"service_id":     rootSvc.String(),
			"trace_uuid":     traceUUID.String(),
			"inserted_spans": insertedSpans,
			"total_spans":    len(r.Spans),
		})
	}
}

// GetAPMTrace returns one trace (root) + all spans ordered by start time.
func GetAPMTrace(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		traceID := c.Param("trace_id")
		if traceID == "" {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		ctx := c.Request.Context()
		var (
			id, tr, root, dur, stat, startedAt, ingestedAt, svcID, svcName string
		)
		err := pool.Pgx().QueryRow(ctx,
			`SELECT t.id::text, t.trace_id, t.root_span_id, t.duration_us::text, t.status,
			        t.started_at::text, t.ingested_at::text,
			        s.id::text, s.name
			 FROM apm_traces t
			 JOIN apm_services s ON s.id = t.service_id
			 WHERE t.tenant_id = $1 AND t.trace_id = $2
			 ORDER BY t.ingested_at DESC
			 LIMIT 1`,
			tenantID, traceID,
		).Scan(&id, &tr, &root, &dur, &stat, &startedAt, &ingestedAt, &svcID, &svcName)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		trace := gin.H{
			"id":           id,
			"trace_id":     tr,
			"root_span_id": root,
			"duration_us":  dur,
			"status":       stat,
			"started_at":   startedAt,
			"ingested_at":  ingestedAt,
			"service_id":   svcID,
			"service_name": svcName,
		}
		// Spans.
		rows, err := pool.Pgx().Query(ctx,
			`SELECT span_id, COALESCE(parent_span_id, ''), service_id::text, name, duration_us,
			        status, started_at::text, attributes::text
			 FROM apm_spans
			 WHERE tenant_id = $1 AND trace_id = $2
			 ORDER BY started_at ASC
			 LIMIT 500`,
			tenantID, traceID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		spans := []gin.H{}
		for rows.Next() {
			var spanID, parent, svc, name, status, started, attrs string
			var dur int64
			if err := rows.Scan(&spanID, &parent, &svc, &name, &dur, &status, &started, &attrs); err != nil {
				continue
			}
			spans = append(spans, gin.H{
				"span_id":        spanID,
				"parent_span_id": parent,
				"service_id":     svc,
				"name":           name,
				"duration_us":    dur,
				"status":         status,
				"started_at":     started,
				"attributes":     json.RawMessage(attrs),
			})
		}
		kernel.RespondOK(c, gin.H{
			"trace": trace,
			"spans": spans,
			"total": len(spans),
		})
	}
}
