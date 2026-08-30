// Tier 7 — RUM Full (D4) — Ingest endpoints.
//
//	Six POST endpoints, one per RUM data category:
//
//	POST /api/v1/rum/web-vitals        — per-page vitals (LCP/FID/CLS/FCP/TTFB)
//	POST /api/v1/rum/resource-timings  — per-network-request timing row
//	POST /api/v1/rum/interactions      — click/input/scroll events
//	POST /api/v1/rum/long-tasks        — browser-flagged long tasks (>50ms)
//	POST /api/v1/rum/errors            — JS errors, dedup'd by fingerprint
//	POST /api/v1/rum/heatmap           — click coordinates (synthetic row
//	                                     in rum_resources for now — Phase 3
//	                                     keeps the data plane uniform)
//
// All endpoints honor JWT.tenant_id. Error groups UPSERT on
// (tenant_id, fingerprint) so the second hit on the same error bumps
// occurrence_count + last_seen_at in one round-trip.
package handler

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// allowedRUMVitals is the Web Vitals allow-list — defense in depth so
// the metric column only carries one of the five values we expect.
var allowedRUMVitals = map[string]bool{
	"lcp": true, "fid": true, "cls": true, "fcp": true, "ttfb": true,
}

// allowedRUMResourceTypes is a coarse allow-list. The SDK is expected
// to send one of these; we accept anything short but log / normalize
// empty values to "other" so the column never carries empty strings.
var allowedRUMResourceTypes = map[string]bool{
	"document": true, "script": true, "stylesheet": true, "image": true,
	"xhr": true, "fetch": true, "font": true, "other": true,
}

// allowedRUMRatings is the Web Vitals rating partition. Anything else
// gets coerced to "needs-improvement" so the UI's filter still works.
var allowedRUMRatings = map[string]bool{
	"good": true, "needs-improvement": true, "poor": true,
}

// rumWebVitalReq is the body shape for POST /rum/web-vitals.
type rumWebVitalReq struct {
	SessionID string  `json:"session_id" binding:"required,min=1,max=128"`
	URL       string  `json:"url" binding:"required,min=1,max=2048"`
	Metric    string  `json:"metric" binding:"required,min=1,max=16"`
	Value     float64 `json:"value"`
	Rating    string  `json:"rating" binding:"max=32"`
	TS        string  `json:"ts"` // optional RFC3339
}

// IngestRUMWebVital inserts one vital. The metric is allow-listed; the
// rating is coerced to a known bucket; ts defaults to now() if missing.
func IngestRUMWebVital(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r rumWebVitalReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		m := strings.ToLower(strings.TrimSpace(r.Metric))
		if !allowedRUMVitals[m] {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		rating := strings.ToLower(strings.TrimSpace(r.Rating))
		if !allowedRUMRatings[rating] {
			rating = "needs-improvement"
		}
		ts := time.Now().UTC()
		if r.TS != "" {
			if parsed, perr := time.Parse(time.RFC3339, r.TS); perr == nil {
				ts = parsed.UTC()
			}
		}
		_, err := pool.Pgx().Exec(c.Request.Context(),
			`INSERT INTO rum_web_vitals
			   (tenant_id, session_id, url, metric, value, rating, ts)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			tenantID, r.SessionID, r.URL, m, r.Value, rating, ts,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondCreated(c, gin.H{
			"session_id": r.SessionID,
			"metric":     m,
			"rating":     rating,
		})
	}
}

// rumResourceReq is the body shape for POST /rum/resource-timings.
type rumResourceReq struct {
	SessionID    string `json:"session_id" binding:"required,min=1,max=128"`
	URL          string `json:"url" binding:"required,min=1,max=2048"`
	ResourceType string `json:"resource_type" binding:"max=32"`
	DurationMS   int64  `json:"duration_ms" binding:"min=0"`
	SizeBytes    int64  `json:"size_bytes"`
	Status       int    `json:"status"`
	TS           string `json:"ts"`
}

// IngestRUMResource inserts one network resource timing row.
func IngestRUMResource(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r rumResourceReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		rt := strings.ToLower(strings.TrimSpace(r.ResourceType))
		if rt == "" {
			rt = "other"
		}
		if !allowedRUMResourceTypes[rt] {
			rt = "other"
		}
		ts := time.Now().UTC()
		if r.TS != "" {
			if parsed, perr := time.Parse(time.RFC3339, r.TS); perr == nil {
				ts = parsed.UTC()
			}
		}
		_, err := pool.Pgx().Exec(c.Request.Context(),
			`INSERT INTO rum_resources
			   (tenant_id, session_id, url, resource_type, duration_ms, size_bytes, status, ts)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			tenantID, r.SessionID, r.URL, rt, r.DurationMS,
			nullIfEmptyInt(r.SizeBytes), r.Status, ts,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondCreated(c, gin.H{
			"session_id": r.SessionID,
			"url":        r.URL,
			"type":       rt,
		})
	}
}

// rumInteractionReq is the body shape for POST /rum/interactions.
type rumInteractionReq struct {
	SessionID  string `json:"session_id" binding:"required,min=1,max=128"`
	Action     string `json:"action" binding:"required,min=1,max=32"`
	Target     string `json:"target" binding:"max=512"`
	DurationMS int64  `json:"duration_ms"`
	TS         string `json:"ts"`
}

// IngestRUMInteraction inserts one user interaction.
func IngestRUMInteraction(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r rumInteractionReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		ts := time.Now().UTC()
		if r.TS != "" {
			if parsed, perr := time.Parse(time.RFC3339, r.TS); perr == nil {
				ts = parsed.UTC()
			}
		}
		_, err := pool.Pgx().Exec(c.Request.Context(),
			`INSERT INTO rum_interactions
			   (tenant_id, session_id, action, target, duration_ms, ts)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			tenantID, r.SessionID, strings.ToLower(r.Action),
			nullIfEmpty(r.Target), nullIfEmptyInt(r.DurationMS), ts,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondCreated(c, gin.H{
			"session_id": r.SessionID,
			"action":     strings.ToLower(r.Action),
		})
	}
}

// rumLongTaskReq is the body shape for POST /rum/long-tasks.
type rumLongTaskReq struct {
	SessionID  string `json:"session_id" binding:"required,min=1,max=128"`
	DurationMS int64  `json:"duration_ms" binding:"required,min=1"`
	TS         string `json:"ts"`
}

// IngestRUMLongTask inserts one long-task record. The SDK is expected
// to filter to >50ms; we re-check at the API edge.
func IngestRUMLongTask(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r rumLongTaskReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		if r.DurationMS < 50 {
			// Reject below the long-task threshold; SDK should
			// already be filtering but we defend in depth.
			kernel.RespondOK(c, gin.H{"skipped": true, "reason": "below_threshold"})
			return
		}
		ts := time.Now().UTC()
		if r.TS != "" {
			if parsed, perr := time.Parse(time.RFC3339, r.TS); perr == nil {
				ts = parsed.UTC()
			}
		}
		_, err := pool.Pgx().Exec(c.Request.Context(),
			`INSERT INTO rum_long_tasks (tenant_id, session_id, duration_ms, ts)
			 VALUES ($1, $2, $3, $4)`,
			tenantID, r.SessionID, r.DurationMS, ts,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondCreated(c, gin.H{
			"session_id":  r.SessionID,
			"duration_ms": r.DurationMS,
		})
	}
}

// rumErrorReq is the body shape for POST /rum/errors. fingerprint is
// mandatory — the SDK computes it from the stack trace.
type rumErrorReq struct {
	SessionID   string `json:"session_id" binding:"required,min=1,max=128"`
	Fingerprint string `json:"fingerprint" binding:"required,min=1,max=128"`
	Message     string `json:"message" binding:"required,min=1,max=2048"`
	StackTrace  string `json:"stack_trace" binding:"max=32768"`
	Service     string `json:"service" binding:"max=128"`
	Source      string `json:"source" binding:"max=64"`
	URL         string `json:"url" binding:"max=2048"`
}

// IngestRUMError UPSERTs into rum_error_groups. Existing rows see
// occurrence_count++ and last_seen_at refreshed.
func IngestRUMError(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r rumErrorReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var id string
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO rum_error_groups
			   (tenant_id, fingerprint, message, stack_trace, service, source,
			    occurrence_count, first_seen_at, last_seen_at)
			 VALUES ($1, $2, $3, $4, $5, $6, 1, now(), now())
			 ON CONFLICT (tenant_id, fingerprint) DO UPDATE
			   SET occurrence_count = rum_error_groups.occurrence_count + 1,
			       last_seen_at = now(),
			       message = EXCLUDED.message
			 RETURNING id::text`,
			tenantID, r.Fingerprint, r.Message,
			nullIfEmpty(r.StackTrace),
			nullIfEmpty(r.Service),
			nullIfEmpty(r.Source),
		).Scan(&id)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondCreated(c, gin.H{
			"error_group_id": id,
			"fingerprint":    r.Fingerprint,
		})
	}
}

// rumHeatmapReq is the body shape for POST /rum/heatmap. Phase 3
// records the click as a synthetic rum_resources row so the
// data-plane schema stays uniform (one row per "interesting" event).
// A future phase can introduce a dedicated rum_clicks table once we
// know which fields the heatmap UI actually needs.
type rumHeatmapReq struct {
	SessionID      string  `json:"session_id" binding:"required,min=1,max=128"`
	URL            string  `json:"url" binding:"required,min=1,max=2048"`
	X              float64 `json:"x"`
	Y              float64 `json:"y"`
	ViewportWidth  int     `json:"viewport_width"`
	ViewportHeight int     `json:"viewport_height"`
}

// IngestRUMHeatmap records one click event as a rum_resources row.
func IngestRUMHeatmap(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r rumHeatmapReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		// Encode x/y + viewport into a JSON attributes blob and
		// round-trip it through the URL as a sentinel-marker so the
		// session-detail page can pull it back out. Phase 3 keeps a
		// uniform data plane; a later migration will move this to a
		// real attributes jsonb column.
		attrsJSON, _ := json.Marshal(gin.H{
			"x": r.X, "y": r.Y,
			"vw": r.ViewportWidth, "vh": r.ViewportHeight,
			"kind": "heatmap",
		})
		marker := r.URL + "#hm=" + string(attrsJSON)
		_, err := pool.Pgx().Exec(c.Request.Context(),
			`INSERT INTO rum_resources
			   (tenant_id, session_id, url, resource_type, duration_ms, size_bytes, status, ts)
			 VALUES ($1, $2, $3, 'other', 0, NULL, 200, now())`,
			tenantID, r.SessionID, marker,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondCreated(c, gin.H{
			"session_id": r.SessionID,
			"recorded":   true,
		})
	}
}

// nullIfEmptyInt returns nil for 0 (we treat 0 = "unknown" for
// optional numeric fields) and the value itself otherwise.
func nullIfEmptyInt(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}
