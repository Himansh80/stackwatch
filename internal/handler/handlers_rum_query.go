// Tier 7 — RUM Full (D4) — Query endpoints.
//
//	GET /api/v1/rum/sessions                — list distinct sessions
//	                                          (deduped by session_id) for
//	                                          the caller's tenant, sorted
//	                                          by most recent activity
//	GET /api/v1/rum/sessions/:id            — session detail (header + counts)
//	GET /api/v1/rum/sessions/:id/waterfall  — chronological resource list
//	GET /api/v1/rum/error-groups            — list error groups (sorted
//	                                          by occurrence_count DESC)
//
// Sessions are derived from the rum_* event tables since Phase 3 has
// no session-start beacon. We deduplicate by session_id and aggregate
// counts via subqueries so the list endpoint stays cheap.
package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// rumTimeRange parses the optional ?time_range= query parameter into
// a Postgres interval string. Defaults to "24 hours" when missing or
// unrecognised — keeps the default list size manageable.
func rumTimeRange(c *gin.Context) string {
	switch strings.ToLower(strings.TrimSpace(c.Query("time_range"))) {
	case "1h", "1hour":
		return "1 hour"
	case "7d", "7days", "week":
		return "7 days"
	case "30d", "30days", "month":
		return "30 days"
	case "24h", "24hours", "1d", "day", "":
		return "24 hours"
	default:
		return "24 hours"
	}
}

// rumLimit clamps the optional ?limit= param to a sane range. Capped
// at 200 so a misbehaving client can't pull the whole table.
func rumLimit(c *gin.Context, def int) int {
	v := c.Query("limit")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	if n > 200 {
		return 200
	}
	return n
}

// rumSessionRow is the on-the-wire shape for one session in the list.
type rumSessionRow struct {
	SessionID    string `json:"session_id"`
	FirstURL     string `json:"url"`
	FirstSeen    string `json:"started_at"`
	LastSeen     string `json:"last_seen"`
	PageViews    int    `json:"page_views"`
	Errors       int    `json:"errors"`
	WebVitals    int    `json:"web_vitals"`
	Resources    int    `json:"resources"`
	Interactions int    `json:"interactions"`
	LongTasks    int    `json:"long_tasks"`
}

// ListRUMSessions returns the deduped session list for the tenant.
// Aggregates come from the event tables directly (no separate
// rum_sessions row required) so the endpoint works the moment the
// first ingest lands.
func ListRUMSessions(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		interval := rumTimeRange(c)
		limit := rumLimit(c, 50)
		ctx := c.Request.Context()
		// Pull a session row per distinct session_id, picking the
		// earliest URL we saw + the latest timestamp across all
		// event tables for this tenant within the window.
		rows, err := pool.Pgx().Query(ctx,
			`WITH session_bounds AS (
			   SELECT session_id,
			          MIN(ts) AS first_seen,
			          MAX(ts) AS last_seen
			   FROM (
			     SELECT session_id, ts FROM rum_web_vitals
			       WHERE tenant_id = $1 AND ts > now() - $2::interval
			     UNION ALL
			     SELECT session_id, ts FROM rum_resources
			       WHERE tenant_id = $1 AND ts > now() - $2::interval
			     UNION ALL
			     SELECT session_id, ts FROM rum_interactions
			       WHERE tenant_id = $1 AND ts > now() - $2::interval
			     UNION ALL
			     SELECT session_id, ts FROM rum_long_tasks
			       WHERE tenant_id = $1 AND ts > now() - $2::interval
			   ) u
			   GROUP BY session_id
			 )
			 SELECT
			   b.session_id,
			   COALESCE(MIN(r.url) FILTER (WHERE r.url IS NOT NULL), '') AS first_url,
			   b.first_seen::text,
			   b.last_seen::text,
			   (SELECT COUNT(*) FROM rum_web_vitals  w WHERE w.tenant_id=$1 AND w.session_id=b.session_id) AS web_vitals,
			   (SELECT COUNT(*) FROM rum_resources    r WHERE r.tenant_id=$1 AND r.session_id=b.session_id) AS resources,
			   (SELECT COUNT(*) FROM rum_interactions i WHERE i.tenant_id=$1 AND i.session_id=b.session_id) AS interactions,
			   (SELECT COUNT(*) FROM rum_long_tasks   l WHERE l.tenant_id=$1 AND l.session_id=b.session_id) AS long_tasks,
			   0 AS errors
			 FROM session_bounds b
			 LEFT JOIN rum_resources r
			   ON r.tenant_id = $1 AND r.session_id = b.session_id
			 GROUP BY b.session_id, b.first_seen, b.last_seen
			 ORDER BY b.last_seen DESC
			 LIMIT $3`,
			tenantID, interval, limit,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []rumSessionRow{}
		for rows.Next() {
			var s rumSessionRow
			if err := rows.Scan(&s.SessionID, &s.FirstURL, &s.FirstSeen,
				&s.LastSeen, &s.WebVitals, &s.Resources, &s.Interactions,
				&s.LongTasks, &s.Errors); err != nil {
				continue
			}
			out = append(out, s)
		}
		kernel.RespondOK(c, gin.H{
			"sessions": out,
			"total":    len(out),
			"window":   interval,
		})
	}
}

// GetRUMSession returns the session detail — header info plus counts
// for each child category. The deep data (vitals / resources / etc.)
// is fetched by /rum/sessions/:id/waterfall and the session page
// loads it directly to keep this endpoint snappy.
func GetRUMSession(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		sessionID := c.Param("id")
		if sessionID == "" {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		ctx := c.Request.Context()
		// Header: earliest event timestamp + sample URL. Only
		// rum_resources carries a URL — pull that separately to
		// avoid a UNION column-shape mismatch with rum_interactions
		// + rum_long_tasks (which don't have a url column).
		var firstSeen, lastSeen string
		err := pool.Pgx().QueryRow(ctx,
			`SELECT
			   MIN(ts)::text,
			   MAX(ts)::text
			 FROM (
			   SELECT ts FROM rum_web_vitals   WHERE tenant_id = $1 AND session_id = $2
			   UNION ALL
			   SELECT ts FROM rum_resources     WHERE tenant_id = $1 AND session_id = $2
			   UNION ALL
			   SELECT ts FROM rum_interactions  WHERE tenant_id = $1 AND session_id = $2
			   UNION ALL
			   SELECT ts FROM rum_long_tasks    WHERE tenant_id = $1 AND session_id = $2
			 ) u`,
			tenantID, sessionID,
		).Scan(&firstSeen, &lastSeen)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		// First URL — pull from rum_resources (the only event table
		// that carries a url column). Returns empty when the session
		// had no resource timings (vitals-only sessions still get
		// the timestamp header above).
		var firstURL string
		_ = pool.Pgx().QueryRow(ctx,
			`SELECT COALESCE(MIN(url), '') FROM rum_resources
			 WHERE tenant_id = $1 AND session_id = $2`,
			tenantID, sessionID,
		).Scan(&firstURL)
		// Counts per category (cheap, indexed).
		var (
			nVitals, nResources, nInteractions, nLongTasks int
		)
		if err := pool.Pgx().QueryRow(ctx,
			`SELECT
			   (SELECT COUNT(*) FROM rum_web_vitals   WHERE tenant_id = $1 AND session_id = $2),
			   (SELECT COUNT(*) FROM rum_resources     WHERE tenant_id = $1 AND session_id = $2),
			   (SELECT COUNT(*) FROM rum_interactions  WHERE tenant_id = $1 AND session_id = $2),
			   (SELECT COUNT(*) FROM rum_long_tasks    WHERE tenant_id = $1 AND session_id = $2)`,
			tenantID, sessionID,
		).Scan(&nVitals, &nResources, &nInteractions, &nLongTasks); err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, gin.H{
			"session_id":   sessionID,
			"first_url":    firstURL,
			"started_at":   firstSeen,
			"last_seen":    lastSeen,
			"web_vitals":   nVitals,
			"resources":    nResources,
			"interactions": nInteractions,
			"long_tasks":   nLongTasks,
			"errors":       0, // error fingerprint lookup lives on the page; rare
		})
	}
}

// GetRUMSessionWaterfall returns the session's network resources +
// interactions + long tasks + vitals all in chronological order so
// the session-detail page can render a single waterfall timeline.
func GetRUMSessionWaterfall(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		sessionID := c.Param("id")
		if sessionID == "" {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		limit := rumLimit(c, 500)
		ctx := c.Request.Context()
		// Resources (the waterfall's primary content).
		rRows, err := pool.Pgx().Query(ctx,
			`SELECT id::text, url, COALESCE(resource_type, 'other'),
			        duration_ms, COALESCE(size_bytes, 0), COALESCE(status, 0),
			        ts::text
			 FROM rum_resources
			 WHERE tenant_id = $1 AND session_id = $2
			 ORDER BY ts ASC
			 LIMIT $3`,
			tenantID, sessionID, limit,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rRows.Close()
		resources := []gin.H{}
		for rRows.Next() {
			var id, url, rtype, ts string
			var dur, size, status int64
			if err := rRows.Scan(&id, &url, &rtype, &dur, &size, &status, &ts); err != nil {
				continue
			}
			resources = append(resources, gin.H{
				"id":            id,
				"url":           url,
				"resource_type": rtype,
				"duration_ms":   dur,
				"size_bytes":    size,
				"status":        status,
				"ts":            ts,
			})
		}
		// Web vitals (in same order; the page renders them in a
		// separate sub-pane).
		vRows, err := pool.Pgx().Query(ctx,
			`SELECT id::text, url, metric, value, COALESCE(rating, ''), ts::text
			 FROM rum_web_vitals
			 WHERE tenant_id = $1 AND session_id = $2
			 ORDER BY ts ASC
			 LIMIT $3`,
			tenantID, sessionID, limit,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer vRows.Close()
		vitals := []gin.H{}
		for vRows.Next() {
			var id, url, metric, rating, ts string
			var value float64
			if err := vRows.Scan(&id, &url, &metric, &value, &rating, &ts); err != nil {
				continue
			}
			vitals = append(vitals, gin.H{
				"id":     id,
				"url":    url,
				"metric": metric,
				"value":  value,
				"rating": rating,
				"ts":     ts,
			})
		}
		// Interactions + long tasks — separate arrays.
		iRows, err := pool.Pgx().Query(ctx,
			`SELECT id::text, action, COALESCE(target, ''),
			        COALESCE(duration_ms, 0), ts::text
			 FROM rum_interactions
			 WHERE tenant_id = $1 AND session_id = $2
			 ORDER BY ts ASC
			 LIMIT $3`,
			tenantID, sessionID, limit,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer iRows.Close()
		interactions := []gin.H{}
		for iRows.Next() {
			var id, action, target, ts string
			var dur int64
			if err := iRows.Scan(&id, &action, &target, &dur, &ts); err != nil {
				continue
			}
			interactions = append(interactions, gin.H{
				"id":          id,
				"action":      action,
				"target":      target,
				"duration_ms": dur,
				"ts":          ts,
			})
		}
		lRows, err := pool.Pgx().Query(ctx,
			`SELECT id::text, duration_ms, ts::text
			 FROM rum_long_tasks
			 WHERE tenant_id = $1 AND session_id = $2
			 ORDER BY ts ASC
			 LIMIT $3`,
			tenantID, sessionID, limit,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer lRows.Close()
		longTasks := []gin.H{}
		for lRows.Next() {
			var id, ts string
			var dur int64
			if err := lRows.Scan(&id, &dur, &ts); err != nil {
				continue
			}
			longTasks = append(longTasks, gin.H{
				"id":          id,
				"duration_ms": dur,
				"ts":          ts,
			})
		}
		kernel.RespondOK(c, gin.H{
			"session_id":   sessionID,
			"resources":    resources,
			"vitals":       vitals,
			"interactions": interactions,
			"long_tasks":   longTasks,
		})
	}
}

//
// ListRUMErrorGroups + pgxRows live in handlers_rum_errors.go (split
// to keep this file under the 400-LOC modularity cap).

