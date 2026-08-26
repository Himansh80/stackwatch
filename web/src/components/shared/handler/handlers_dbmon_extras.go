// Tier 7 Phase 3 — DB Monitoring (D9). Connection pool snapshot
// read + write, and the explain-plan helper.
//
// Routes:
//
//	GET    /api/v1/database/connection-pool     — latest pool snapshot per (db,host,port)
//	POST   /api/v1/database/connection-pool     — ingest a pool snapshot (agent write)
//	GET    /api/v1/database/query-explain       — explain for a query_hash (heuristic)
//
// The query-explain endpoint doesn't actually run EXPLAIN against a
// real database — StackWatch only sees the hash + normalized text,
// not a live connection. It returns a heuristic suggestion so the UI
// has something useful to show (e.g. "this is a SELECT, consider
// adding an index on the first filtered column").
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ListDBMonConnectionPools returns the most-recent pool snapshot per
// (database, host, port). DISTINCT ON collapses the history down to
// one row per (db, host, port) without needing a window function.
func ListDBMonConnectionPools(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT DISTINCT ON (database, host, port)
			        id::text, database, host, port, pool_size, active,
			        idle, waiting, checked_at::text
			 FROM database_connection_pools
			 WHERE tenant_id = $1
			 ORDER BY database, host, port, checked_at DESC`,
			tenantID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []dbmonConnectionPoolRow{}
		for rows.Next() {
			var p dbmonConnectionPoolRow
			if err := rows.Scan(&p.ID, &p.Database, &p.Host, &p.Port,
				&p.PoolSize, &p.Active, &p.Idle, &p.Waiting, &p.CheckedAt); err != nil {
				continue
			}
			out = append(out, p)
		}
		kernel.RespondOK(c, gin.H{"pools": out, "total": len(out)})
	}
}

// dbmonPoolReq is the POST body for the connection-pool ingest.
type dbmonPoolReq struct {
	Database string `json:"database" binding:"required,oneof=postgres mysql mariadb mongodb"`
	Host     string `json:"host"     binding:"required,min=1,max=256"`
	Port     int    `json:"port"     binding:"required,min=1,max=65535"`
	PoolSize int    `json:"pool_size" binding:"min=0"`
	Active   int    `json:"active"    binding:"min=0"`
	Idle     int    `json:"idle"      binding:"min=0"`
	Waiting  int    `json:"waiting"   binding:"min=0"`
}

// IngestDBMonConnectionPool records one pool snapshot for the
// caller's tenant. Returns 201 with the new row id.
func IngestDBMonConnectionPool(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r dbmonPoolReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var id string
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO database_connection_pools
			   (tenant_id, database, host, port, pool_size, active, idle, waiting)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 RETURNING id::text`,
			tenantID, r.Database, strings.TrimSpace(r.Host), r.Port,
			r.PoolSize, r.Active, r.Idle, r.Waiting,
		).Scan(&id)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondCreated(c, gin.H{
			"id":       id,
			"database": r.Database,
			"host":     r.Host,
			"port":     r.Port,
		})
	}
}

// ExplainDBMonQuery returns a heuristic explain plan for the query
// identified by ?query_hash=X. StackWatch doesn't have a live DB
// connection per host, so we can't run real EXPLAIN — instead we
// return the stored query_text + a suggestion based on the SQL
// shape (SELECT vs UPDATE vs INSERT etc.).
func ExplainDBMonQuery(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		queryHash := strings.TrimSpace(c.Query("query_hash"))
		if queryHash == "" {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var queryText string
		var database string
		var totalCount int64
		var avgMS int64
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT ANY_VALUE(query_text), ANY_VALUE(database),
			        COUNT(*), (SUM(duration_ms)::numeric / NULLIF(COUNT(*), 0))::BIGINT
			 FROM database_queries
			 WHERE tenant_id = $1 AND query_hash = $2`,
			tenantID, queryHash,
		).Scan(&queryText, &database, &totalCount, &avgMS)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		suggestion := explainHeuristic(queryText, avgMS)
		kernel.RespondOK(c, gin.H{
			"query_hash":        queryHash,
			"query_text":        queryText,
			"database":          database,
			"total_executions":  totalCount,
			"avg_duration_ms":   avgMS,
			"suggestion":        suggestion,
			"note":              "Heuristic suggestion only; real EXPLAIN requires a live database connection.",
		})
	}
}

// explainHeuristic returns a one-line suggestion based on the first
// SQL keyword and the avg duration. It is intentionally simple —
// Datadog's UI shows similar placeholder copy until it can reach
// the live database.
func explainHeuristic(query string, avgMS int64) string {
	trimmed := strings.TrimSpace(query)
	upper := strings.ToUpper(trimmed)
	switch {
	case strings.HasPrefix(upper, "SELECT"):
		if avgMS > 1000 {
			return "Long-running SELECT. Consider adding an index on the leading WHERE column, or partitioning the table."
		}
		if avgMS > 100 {
			return "SELECT exceeds 100ms. Verify that filtered columns are indexed and not wrapped in functions."
		}
		return "SELECT under 100ms. Looks healthy — no immediate action needed."
	case strings.HasPrefix(upper, "UPDATE"), strings.HasPrefix(upper, "DELETE"):
		if avgMS > 1000 {
			return "Mutating query is slow. Check for missing indexes on WHERE columns and consider batching the writes."
		}
		return "Mutation under 1s. Verify the row count being touched matches expectations."
	case strings.HasPrefix(upper, "INSERT"):
		return "Insert — verify the table is not near its fillfactor and that any FK targets are indexed."
	case strings.HasPrefix(upper, "BEGIN"), strings.HasPrefix(upper, "COMMIT"):
		return "Transaction control statement — check for missing COMMIT or long-held locks."
	default:
		return "Unknown statement type. Inspect the query_text for the actual SQL."
	}
}