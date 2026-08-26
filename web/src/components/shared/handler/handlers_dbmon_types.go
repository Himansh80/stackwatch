// Tier 7 Phase 3 — DB Monitoring (D9). Shared types.
//
// All queries honor tenant_id from the JWT. The materialized view
// database_slow_queries is the public read surface; database_queries
// is the raw ingest. Allowed database values are enforced at the
// API edge so the column never holds garbage.
package handler

import "time"

// allowedDBMonDatabases — defense in depth at the API edge.
// Spec defines these four engines; anything else is rejected.
var allowedDBMonDatabases = map[string]bool{
	"postgres": true, "mysql": true, "mariadb": true, "mongodb": true,
}

// dbmonSlowQueryRow is the row shape returned by
// GET /api/v1/database/slow-queries and GET /database/queries/top.
// Maps directly to the columns of the database_slow_queries
// materialized view (plus query_text via ANY_VALUE).
type dbmonSlowQueryRow struct {
	QueryHash       string  `json:"query_hash"`
	QueryText       string  `json:"query_text"`
	Database        string  `json:"database"`
	TotalCount      int     `json:"total_count"`
	TotalDurationMS int64   `json:"total_duration_ms"`
	MaxDurationMS   int64   `json:"max_duration_ms"`
	AvgDurationMS   int64   `json:"avg_duration_ms"`
	LastSeen        *string `json:"last_seen,omitempty"`
}

// dbmonQueryReq is the JSON body for POST /database/query-stats.
// Agents post one row per executed query. query_hash is a stable
// hash of the normalized text so the materialized view can roll
// up identical queries across hosts.
type dbmonQueryReq struct {
	Database        string `json:"database"         binding:"required,oneof=postgres mysql mariadb mongodb"`
	QueryHash       string `json:"query_hash"       binding:"required,min=1,max=128"`
	QueryText       string `json:"query_text"       binding:"required,min=1,max=8192"`
	DurationMS      int    `json:"duration_ms"      binding:"required,min=0"`
	RowsExamined    int    `json:"rows_examined"    binding:"min=0"`
	RowsReturned    int    `json:"rows_returned"    binding:"min=0"`
	UserID          string `json:"user_id"          binding:"max=128"`
	ApplicationName string `json:"application_name" binding:"max=128"`
}

// dbmonConnectionPoolRow is one row of database_connection_pools.
// One row per (database, host, port, snapshot time). The UI shows
// the latest row per (database, host, port).
type dbmonConnectionPoolRow struct {
	ID        string `json:"id"`
	Database  string `json:"database"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	PoolSize  int    `json:"pool_size"`
	Active    int    `json:"active"`
	Idle      int    `json:"idle"`
	Waiting   int    `json:"waiting"`
	CheckedAt string `json:"checked_at"`
}

// Compile-time guard: keep time import even when only used elsewhere.
var _ = time.RFC3339