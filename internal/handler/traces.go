// Tier 6 v5 — Distributed Tracing (M5).
//
//   POST /api/v1/traces                ingest a single span (or batch)
//   POST /api/v1/traces/otlp           ingest OTLP-format payload
//   GET  /api/v1/traces                list recent spans (paged)
//   GET  /api/v1/traces/:trace_id      get full trace (all spans)
//   GET  /api/v1/traces/services/summary  per-service aggregate stats
//
// Simplified shape (matches what most agents send):
//   { trace_id, span_id, parent_id, service, name, kind, duration_ms, status, ts }
// OTLP shape (resource_spans -> scope_spans -> spans) is also accepted as a
// convenience for off-the-shelf OpenTelemetry SDKs.
//
// Trace IDs are 16-byte hex (32 chars). Span IDs are 8-byte hex (16 chars).
package handler

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// allowedTraceKinds are the standard OTel span kinds plus 'unspecified'.
var allowedTraceKinds = map[string]bool{
	"unspecified": true,
	"internal":    true,
	"server":      true,
	"client":      true,
	"producer":    true,
	"consumer":    true,
}

// allowedTraceStatuses are the standard OTel span statuses.
var allowedTraceStatuses = map[string]bool{
	"unset":  true,
	"ok":     true,
	"error":  true,
}

// SpanInput is the JSON shape for ingesting a single span.
type SpanInput struct {
	TraceID    string          `json:"trace_id" binding:"required,len=32"`
	SpanID     string          `json:"span_id" binding:"required,len=16"`
	ParentID   string          `json:"parent_id"`
	Service    string          `json:"service" binding:"required,min=1,max=128"`
	Name       string          `json:"name" binding:"required,min=1,max=256"`
	Kind       string          `json:"kind"`
	DurationMS int             `json:"duration_ms"`
	Status     string          `json:"status"`
	TS         time.Time       `json:"ts"`
	Attributes json.RawMessage `json:"attributes"`
}

// BatchSpanInput wraps an array of spans for batch ingest.
type BatchSpanInput struct {
	Spans []SpanInput `json:"spans" binding:"required,min=1,max=500,dive"`
}

// IngestSpan inserts one or more spans.
//
// Body: { spans: [...] }  OR  [ ... ] (raw array)  OR  {...} (single span)
// Tenant ID: from JWT (auth required).
func IngestSpan(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		raw, err := c.GetRawData()
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		// Try array first, then batch wrapper, then single span.
		var spans []SpanInput
		if len(raw) > 0 && raw[0] == '[' {
			if err := json.Unmarshal(raw, &spans); err != nil {
				kernel.RespondError(c, kernel.ErrBadRequest)
				return
			}
		} else if len(raw) > 0 && raw[0] == '{' {
			// Try single span first (no "spans" key); else treat as batch wrapper.
			var probe map[string]json.RawMessage
			if err := json.Unmarshal(raw, &probe); err == nil {
				if _, hasSpans := probe["spans"]; hasSpans {
					var batch BatchSpanInput
					if err := json.Unmarshal(raw, &batch); err != nil {
						kernel.RespondError(c, kernel.ErrBadRequest)
						return
					}
					spans = batch.Spans
				} else {
					var single SpanInput
					if err := json.Unmarshal(raw, &single); err != nil {
						kernel.RespondError(c, kernel.ErrBadRequest)
						return
					}
					// Manual validation since single-span branch bypasses binding tags.
					if len(single.TraceID) != 32 || len(single.SpanID) != 16 ||
						single.Service == "" || single.Name == "" {
						kernel.RespondError(c, kernel.ErrBadRequest)
						return
					}
					spans = []SpanInput{single}
				}
			} else {
				kernel.RespondError(c, kernel.ErrBadRequest)
				return
			}
		} else {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		if len(spans) == 0 {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		if len(spans) > 500 {
			spans = spans[:500]
		}
		ctx := c.Request.Context()
		inserted := 0
		for _, s := range spans {
			kind := s.Kind
			if kind == "" {
				kind = "internal"
			}
			if !allowedTraceKinds[kind] {
				kind = "internal"
			}
			status := s.Status
			if status == "" {
				status = "ok"
			}
			if !allowedTraceStatuses[status] {
				status = "ok"
			}
			ts := s.TS
			if ts.IsZero() {
				ts = time.Now()
			}
			attrs := s.Attributes
			if len(attrs) == 0 || string(attrs) == "null" {
				attrs = json.RawMessage(`{}`)
			}
			_, err := pool.Pgx().Exec(ctx,
				`INSERT INTO traces (tenant_id, trace_id, span_id, parent_id, service, name, kind, duration_ms, status, ts, attributes)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb)`,
				tenantID, s.TraceID, s.SpanID, s.ParentID, s.Service, s.Name,
				kind, s.DurationMS, status, ts, string(attrs))
			if err == nil {
				inserted++
			}
		}
		kernel.RespondCreated(c, gin.H{
			"received":  len(spans),
			"inserted":  inserted,
			"rejected":  len(spans) - inserted,
		})
	}
}

// IngestOTLP accepts OTLP-format payloads (the standard OpenTelemetry format).
//
// Body shape:
//   { resourceSpans: [ { resource: {...}, scopeSpans: [ { spans: [...] } ] } ] }
// Simplified — we don't extract resource attributes, just pull span fields.
func IngestOTLP(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		raw, err := c.GetRawData()
		if err != nil || len(raw) == 0 {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var otlp struct {
			ResourceSpans []struct {
				ScopeSpans []struct {
					Spans []struct {
						TraceID    string          `json:"traceId"`
						SpanID     string          `json:"spanId"`
						ParentID   string          `json:"parentSpanId"`
						Name       string          `json:"name"`
						Kind       int             `json:"kind"` // 1=internal, 2=server, 3=client, 4=producer, 5=consumer
						StartUnix  int64           `json:"startTimeUnixNano"`
						EndUnix    int64           `json:"endTimeUnixNano"`
						Status     struct {
							Code int `json:"code"` // 0=unset, 1=ok, 2=error
						} `json:"status"`
						Attributes []struct {
							Key   string          `json:"key"`
							Value struct {
								StringVal string `json:"stringValue"`
							} `json:"value"`
						} `json:"attributes"`
					} `json:"spans"`
				} `json:"scopeSpans"`
			} `json:"resourceSpans"`
		}
		if err := json.Unmarshal(raw, &otlp); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		kindNames := map[int]string{1: "internal", 2: "server", 3: "client", 4: "producer", 5: "consumer"}
		statusNames := map[int]string{0: "unset", 1: "ok", 2: "error"}
		ctx := c.Request.Context()
		inserted := 0
		var firstService string
		for _, rs := range otlp.ResourceSpans {
			for _, ss := range rs.ScopeSpans {
				for _, s := range ss.Spans {
					attrs := map[string]string{}
					for _, a := range s.Attributes {
						attrs[a.Key] = a.Value.StringVal
					}
					if firstService == "" {
						if v, ok := attrs["service.name"]; ok {
							firstService = v
						}
					}
					attrBytes, _ := json.Marshal(attrs)
					if len(s.TraceID) != 32 || len(s.SpanID) != 16 {
						continue
					}
					durMS := int((s.EndUnix - s.StartUnix) / 1_000_000)
					ts := time.Unix(0, s.StartUnix)
					_, err := pool.Pgx().Exec(ctx,
						`INSERT INTO traces (tenant_id, trace_id, span_id, parent_id, service, name, kind, duration_ms, status, ts, attributes)
						 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb)`,
						tenantID, s.TraceID, s.SpanID, s.ParentID,
						defaultStr(firstService, "unknown_service"), s.Name,
						defaultStr(kindNames[s.Kind], "internal"),
						durMS, defaultStr(statusNames[s.Status.Code], "ok"),
						ts, string(attrBytes))
					if err == nil {
						inserted++
					}
				}
			}
		}
		kernel.RespondCreated(c, gin.H{
			"received_resource_spans": len(otlp.ResourceSpans),
			"inserted":                inserted,
		})
	}
}

func defaultStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// ListTraces returns recent spans (paged, time-ordered).
//
// Query: ?limit=N (default 50, max 500), ?service=foo, ?kind=server
func ListTraces(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		if limit <= 0 || limit > 500 {
			limit = 50
		}
		service := c.Query("service")
		kind := c.Query("kind")
		args := []any{tenantID, limit}
		q := `SELECT trace_id, span_id, COALESCE(parent_id, ''), service, name, kind, duration_ms, status, ts::text, attributes::text
		      FROM traces WHERE tenant_id = $1`
		if service != "" {
			q += " AND service = $3"
			args = append(args, service)
		}
		if kind != "" {
			if len(args) == 3 {
				q += " AND kind = $4"
			} else {
				q += " AND kind = $3"
			}
			args = append(args, kind)
		}
		q += " ORDER BY ts DESC LIMIT $2"
		rows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []gin.H{}
		for rows.Next() {
			var (
				traceID, spanID, parentID, service, name, kind, ts, attrs string
				duration                                                    int
				status                                                      string
			)
			if err := rows.Scan(&traceID, &spanID, &parentID, &service, &name, &kind, &duration, &status, &ts, &attrs); err != nil {
				continue
			}
			out = append(out, gin.H{
				"trace_id":    traceID,
				"span_id":     spanID,
				"parent_id":   parentID,
				"service":     service,
				"name":        name,
				"kind":        kind,
				"duration_ms": duration,
				"status":      status,
				"ts":          ts,
				"attributes":  json.RawMessage(attrs),
			})
		}
		kernel.RespondOK(c, gin.H{"spans": out, "total": len(out)})
	}
}

// GetTraceByID returns all spans for a single trace.
func GetTraceByID(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		traceID := c.Param("trace_id")
		if len(traceID) != 32 {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT trace_id, span_id, COALESCE(parent_id, ''), service, name, kind, duration_ms, status, ts::text, attributes::text
			 FROM traces WHERE tenant_id = $1 AND trace_id = $2 ORDER BY ts`,
			tenantID, traceID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []gin.H{}
		rootCount := 0
		errCount := 0
		totalDur := 0
		for rows.Next() {
			var (
				tID, sID, pID, svc, nm, kd, ts, attrs string
				dur                                   int
				stat                                  string
			)
			if err := rows.Scan(&tID, &sID, &pID, &svc, &nm, &kd, &dur, &stat, &ts, &attrs); err != nil {
				continue
			}
			if pID == "" {
				rootCount++
			}
			if stat == "error" {
				errCount++
			}
			totalDur += dur
			out = append(out, gin.H{
				"trace_id":    tID,
				"span_id":     sID,
				"parent_id":   pID,
				"service":     svc,
				"name":        nm,
				"kind":        kd,
				"duration_ms": dur,
				"status":      stat,
				"ts":          ts,
				"attributes":  json.RawMessage(attrs),
			})
		}
		kernel.RespondOK(c, gin.H{
			"trace_id":         traceID,
			"spans":            out,
			"total_spans":      len(out),
			"root_spans":       rootCount,
			"error_spans":      errCount,
			"total_duration_ms": totalDur,
		})
	}
}

// TraceServiceSummary returns per-service aggregate stats.
//
// Useful for showing "this service has 80 traces / 12 errors / avg 250ms"
// style dashboards.
func TraceServiceSummary(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT service,
			        COUNT(*)                              AS span_count,
			        COUNT(DISTINCT trace_id)              AS trace_count,
			        COUNT(*) FILTER (WHERE status='error') AS error_count,
			        COALESCE(AVG(duration_ms)::int, 0)    AS avg_ms,
			        COALESCE(MAX(duration_ms), 0)         AS p100_ms
			 FROM traces
			 WHERE tenant_id = $1 AND ts > NOW() - INTERVAL '1 hour'
			 GROUP BY service
			 ORDER BY span_count DESC`,
			tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []gin.H{}
		for rows.Next() {
			var (
				svc      string
				count    int64
				traceN   int64
				errN     int64
				avg, p100 int
			)
			if err := rows.Scan(&svc, &count, &traceN, &errN, &avg, &p100); err != nil {
				continue
			}
			out = append(out, gin.H{
				"service":      svc,
				"span_count":   count,
				"trace_count":  traceN,
				"error_count":  errN,
				"avg_ms":       avg,
				"p100_ms":      p100,
			})
		}
		// Verify pgx.ErrNoRows is handled correctly (no rows is OK, just empty).
		if rows.Err() != nil && rows.Err() != pgx.ErrNoRows {
			kernel.RespondError(c, rows.Err())
			return
		}
		kernel.RespondOK(c, gin.H{"services": out, "total": len(out)})
	}
}
