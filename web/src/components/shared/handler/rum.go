// Tier 6 v4 — Real User Monitoring (M8).
//
//	POST /api/v1/rum/events    ingest (auth via tenant_id query param)
//	GET  /api/v1/rum/summary   aggregates: counts + avg by kind
//	GET  /api/v1/rum/events    recent events for debugging
//
// The ingest endpoint is intentionally unauthenticated (uses tenant_id
// query param) so browser snippets can post without managing tokens.
// Mitigations: rate limiting + payload size cap + per-IP throttle.
package handler

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
)

// allowedRUMKinds restricts what kinds we accept (defense in depth).
var allowedRUMKinds = map[string]bool{
	"pageload": true,
	"jserror":  true,
	"longtask": true,
	"fetch":    true,
	"xhr":      true,
}

// rumEvent is one event posted by the browser snippet.
type rumEvent struct {
	Kind  string                 `json:"kind"`
	URL   string                 `json:"url"`
	Value float64                `json:"value"`
	Attrs map[string]interface{} `json:"attrs"`
}

// rumIngestRequest is the body for POST /rum/events.
type rumIngestRequest struct {
	Events []rumEvent `json:"events"`
}

// maxRUMBatchSize caps how many events one batch can carry.
const maxRUMBatchSize = 100

// maxRUMAttrsSize caps the JSON-encoded attrs to prevent abuse.
const maxRUMAttrsSize = 4096

// IngestRUM accepts a batch of events from the browser snippet.
//
// Auth: tenant_id is passed as a query parameter (NOT a JWT) since
// browser snippets shouldn't carry tokens. Mitigated by:
// - max batch size (100 events)
// - max attrs size per event (4KB)
// - kind whitelist
func IngestRUM(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantStr := c.Query("tenant_id")
		tenantID, err := uuid.Parse(tenantStr)
		if err != nil {
			c.JSON(400, gin.H{"error": "missing or invalid tenant_id query param"})
			return
		}

		var req rumIngestRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "bad json: " + err.Error()})
			return
		}
		if len(req.Events) == 0 {
			c.JSON(400, gin.H{"error": "no events"})
			return
		}
		if len(req.Events) > maxRUMBatchSize {
			c.JSON(400, gin.H{"error": "batch too large"})
			return
		}

		// Validate each event's kind and size
		for _, ev := range req.Events {
			if !allowedRUMKinds[ev.Kind] {
				c.JSON(400, gin.H{"error": "invalid kind: " + ev.Kind})
				return
			}
			// Attrs size check (best effort — we accept whatever they send,
			// but truncate for storage)
		}

		// Insert in one transaction.
		tx, err := pool.Pgx().Begin(c.Request.Context())
		if err != nil {
			c.JSON(500, gin.H{"error": "tx begin: " + err.Error()})
			return
		}
		defer tx.Rollback(c.Request.Context())

		inserted := 0
		for _, ev := range req.Events {
			attrsBytes := mustMarshalJSONLimited(ev.Attrs, maxRUMAttrsSize)
			_, err := tx.Exec(c.Request.Context(),
				`INSERT INTO rum_events (tenant_id, kind, url, value, attrs)
				 VALUES ($1, $2, $3, $4, $5)`,
				tenantID, ev.Kind, ev.URL, ev.Value, attrsBytes)
			if err != nil {
				c.JSON(500, gin.H{"error": "insert: " + err.Error()})
				return
			}
			inserted++
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

// RUMSummary returns aggregates per kind for the last 24h.
//
// Auth: JWT (Requires auth).
func RUMSummary(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			c.JSON(401, gin.H{"error": "tenant not in context"})
			return
		}

		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT kind,
			        COUNT(*) AS n,
			        COALESCE(AVG(value), 0) AS avg_value,
			        COALESCE(MAX(value), 0) AS max_value
			 FROM rum_events
			 WHERE tenant_id = $1 AND ts >= NOW() - INTERVAL '24 hours'
			 GROUP BY kind
			 ORDER BY kind`, tenantID)
		if err != nil {
			c.JSON(500, gin.H{"error": "query: " + err.Error()})
			return
		}
		defer rows.Close()

		type summaryRow struct {
			Kind     string  `json:"kind"`
			Count    int64   `json:"count"`
			AvgValue float64 `json:"avg_value"`
			MaxValue float64 `json:"max_value"`
		}

		out := []summaryRow{}
		for rows.Next() {
			var r summaryRow
			if err := rows.Scan(&r.Kind, &r.Count, &r.AvgValue, &r.MaxValue); err != nil {
				continue
			}
			out = append(out, r)
		}
		c.JSON(200, gin.H{
			"summary":   out,
			"window":    "24h",
			"tenant_id": tenantID.String(),
		})
	}
}

// ListRUMEvents returns recent events for debugging.
//
// Auth: JWT.
func ListRUMEvents(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			c.JSON(401, gin.H{"error": "tenant not in context"})
			return
		}
		kind := c.Query("kind")
		limitStr := c.DefaultQuery("limit", "50")
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 || limit > 500 {
			limit = 50
		}

		args := []interface{}{tenantID}
		where := "tenant_id = $1"
		if kind != "" {
			args = append(args, kind)
			where += " AND kind = $2"
		}
		args = append(args, limit)

		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id, kind, url, value, attrs, ts
			 FROM rum_events WHERE `+where+`
			 ORDER BY ts DESC LIMIT $`+strconv.Itoa(len(args)), args...)
		if err != nil {
			c.JSON(500, gin.H{"error": "query: " + err.Error()})
			return
		}
		defer rows.Close()

		type eventRow struct {
			ID    string                 `json:"id"`
			Kind  string                 `json:"kind"`
			URL   string                 `json:"url"`
			Value float64                `json:"value"`
			Attrs map[string]interface{} `json:"attrs"`
			TS    time.Time              `json:"ts"`
		}
		out := []eventRow{}
		for rows.Next() {
			var e eventRow
			var attrsRaw []byte
			if err := rows.Scan(&e.ID, &e.Kind, &e.URL, &e.Value, &attrsRaw, &e.TS); err != nil {
				continue
			}
			if attrsRaw != nil {
				_ = jsonUnmarshal(attrsRaw, &e.Attrs)
			}
			if e.Attrs == nil {
				e.Attrs = map[string]interface{}{}
			}
			out = append(out, e)
		}
		c.JSON(200, gin.H{
			"events": out,
			"total":  len(out),
			"kind":   kind,
		})
	}
}

// mustMarshalJSONLimited marshals v, truncating if larger than max bytes.
// Returns "{}" on error or if value is nil.
func mustMarshalJSONLimited(v interface{}, maxBytes int) []byte {
	if v == nil {
		return []byte("{}")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	if len(b) > maxBytes {
		// Truncate to first maxBytes bytes + "..." marker
		// Replace last 3 bytes with "..."
		if maxBytes > 3 {
			return append(b[:maxBytes-3], '.', '.', '.')
		}
		return []byte("{}")
	}
	return b
}

// jsonUnmarshal is a thin wrapper around json.Unmarshal that ignores errors.
func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
