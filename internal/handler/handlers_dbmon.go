// Tier 7 Phase 3 — DB Monitoring (D9). Slow queries, top queries,
// and ingest endpoint.
//
// Routes:
//
//	GET    /api/v1/database/slow-queries      — top 50 slowest (avg desc)
//	GET    /api/v1/database/queries/top       — most-frequent (count desc)
//	POST   /api/v1/database/query-stats       — ingest one query stat
//
// The slow-queries and top-queries reads refresh the materialized
// view CONCURRENTLY first so the surface is always almost live.
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// refreshSlowQueries runs REFRESH MATERIALIZED VIEW CONCURRENTLY
// on database_slow_queries. Errors are swallowed because a stale
// read is better than a 500; the next call will retry the refresh.
// CONCURRENTLY requires the unique index defined in the migration.
func refreshSlowQueries(pool *db.Pool, c *gin.Context) {
	_, _ = pool.Pgx().Exec(c.Request.Context(),
		"REFRESH MATERIALIZED VIEW CONCURRENTLY database_slow_queries")
}

// ListDBMonSlowQueries returns the top slow queries by avg_duration_ms.
// Optional filters: ?database=postgres&limit=50 (default 50, max 200).
func ListDBMonSlowQueries(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		database := c.Query("database")
		limit := clampLimit(c.Query("limit"), 50, 200)

		// Refresh the materialized view before reading so the UI
		// shows near-real-time data. Best-effort.
		refreshSlowQueries(pool, c)

		args := []any{tenantID}
		q := `SELECT query_hash, query_text, database, total_count,
		             total_duration_ms, max_duration_ms, avg_duration_ms,
		             last_seen::text
		      FROM database_slow_queries
		      WHERE tenant_id = $1`
		if allowedDBMonDatabases[database] {
			args = append(args, database)
			q += " AND database = $" + itoa(len(args))
		}
		args = append(args, limit)
		q += " ORDER BY avg_duration_ms DESC LIMIT $" + itoa(len(args))

		rows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []dbmonSlowQueryRow{}
		for rows.Next() {
			var r dbmonSlowQueryRow
			var lastSeen *string
			if err := rows.Scan(&r.QueryHash, &r.QueryText, &r.Database,
				&r.TotalCount, &r.TotalDurationMS, &r.MaxDurationMS,
				&r.AvgDurationMS, &lastSeen); err != nil {
				continue
			}
			r.LastSeen = lastSeen
			out = append(out, r)
		}
		kernel.RespondOK(c, gin.H{"slow_queries": out, "total": len(out)})
	}
}

// ListDBMonTopQueries returns the most-frequent queries by total_count.
// Same filters as slow-queries but ordered by frequency instead of avg.
func ListDBMonTopQueries(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		database := c.Query("database")
		limit := clampLimit(c.Query("limit"), 50, 200)

		refreshSlowQueries(pool, c)

		args := []any{tenantID}
		q := `SELECT query_hash, query_text, database, total_count,
		             total_duration_ms, max_duration_ms, avg_duration_ms,
		             last_seen::text
		      FROM database_slow_queries
		      WHERE tenant_id = $1`
		if allowedDBMonDatabases[database] {
			args = append(args, database)
			q += " AND database = $" + itoa(len(args))
		}
		args = append(args, limit)
		q += " ORDER BY total_count DESC LIMIT $" + itoa(len(args))

		rows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []dbmonSlowQueryRow{}
		for rows.Next() {
			var r dbmonSlowQueryRow
			var lastSeen *string
			if err := rows.Scan(&r.QueryHash, &r.QueryText, &r.Database,
				&r.TotalCount, &r.TotalDurationMS, &r.MaxDurationMS,
				&r.AvgDurationMS, &lastSeen); err != nil {
				continue
			}
			r.LastSeen = lastSeen
			out = append(out, r)
		}
		kernel.RespondOK(c, gin.H{"queries": out, "total": len(out)})
	}
}

// IngestDBMonQueryStats inserts a single query stat row. Agents POST
// here for every query they observe (or a sampled subset).
// The materialized view picks up the new row on its next refresh,
// which is fire-and-forget on the next slow-queries GET.
func IngestDBMonQueryStats(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r dbmonQueryReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		userID := r.UserID
		if userID == "" {
			userID = ""
		}
		appName := r.ApplicationName
		if appName == "" {
			appName = ""
		}
		var id string
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO database_queries
			   (tenant_id, database, query_hash, query_text, duration_ms,
			    rows_examined, rows_returned, user_id, application_name)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''), NULLIF($9, ''))
			 RETURNING id::text`,
			tenantID, r.Database, r.QueryHash, r.QueryText, r.DurationMS,
			r.RowsExamined, r.RowsReturned, userID, appName,
		).Scan(&id)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondCreated(c, gin.H{
			"id":          id,
			"duration_ms": r.DurationMS,
			"query_hash":  r.QueryHash,
		})
	}
}
