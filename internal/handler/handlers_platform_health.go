// Tier 11 Phase 8 — Platform Health (PL8).
//
// handlers_platform_health.go — 5 super_admin endpoints:
//
//	GET /api/v1/platform/health/summary            — HealthSummary
//	GET /api/v1/platform/health/regions            — HealthRegionsSummary
//	GET /api/v1/platform/health/tenants/top         — HealthTopTenants
//	GET /api/v1/platform/health/capacity/forecast  — HealthCapacityForecast
//	GET /api/v1/platform/health/alerts             — HealthAlerts
//
// Shared types live in handlers_platform_health_types.go.
// Heavy helpers live in:
//   handlers_platform_health_forecast.go — forecastExhaustion
//   handlers_platform_health_alerts.go  — buildPlatformAlerts
//
// Summary returns the latest platform_health_snapshots row
// plus the last 24 hourly rows for the trend sparkline.
// Regions aggregates platform_regions counts by kind + status.
// TopTenants returns the top-N tenants by 30-day usage.
// CapacityForecast extrapolates a simple linear model from
// the last 30 days of platform_health_snapshots for disk /
// db connections / api throughput. Alerts aggregates active
// platform-level alerts from the latest snapshot row.
//
// All endpoints are PROTECTED + super_admin gated. Static
// paths are registered BEFORE any future :id sibling in
// mountPlatformRoutes so Gin's tree-router matches the
// literal suffix first.

package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ------------------------------------------------------------------
// healthSummarySelect is the shared SELECT column list for
// platform_health_snapshots rows. Both the latest + trend
// branches of HealthSummary reuse it (the SCAN below would
// drift between the two if they were duplicated).
// ------------------------------------------------------------------
const healthSummarySelect = `
   id, snapshot_at, overall_status,
   api_up, db_up, ingest_up, alert_engine_up, ai_engine_up, web_terminal_up,
   total_tenants, active_tenants_24h, total_servers,
   servers_up, servers_stale, servers_down,
   open_alerts, failed_logins_24h,
   db_connections, api_calls_per_min_5m_avg,
   storage_gb_used, backup_count, backup_latest_at`

// scanHealthSummaryRow scans one row from a pgx.Rows
// iterator. Returns the row formatted for JSON marshalling.
func scanHealthSummaryRow(rows pgx.Rows) (healthSummaryRow, error) {
	var (
		r            healthSummaryRow
		snapshotAt   time.Time
		backupLatest *time.Time
	)
	if err := rows.Scan(
		&r.ID, &snapshotAt, &r.OverallStatus,
		&r.APIUp, &r.DBUp, &r.IngestUp, &r.AlertEngineUp, &r.AIEngineUp, &r.WebTerminalUp,
		&r.TotalTenants, &r.ActiveTenants24h, &r.TotalServers,
		&r.ServersUp, &r.ServersStale, &r.ServersDown,
		&r.OpenAlerts, &r.FailedLogins24h,
		&r.DBConnections, &r.APICallsPerMin5mAvg,
		&r.StorageGBUsed, &r.BackupCount, &backupLatest,
	); err != nil {
		return r, err
	}
	r.SnapshotAt = snapshotAt.UTC().Format(time.RFC3339)
	if backupLatest != nil {
		r.BackupLatestAt = backupLatest.UTC().Format(time.RFC3339)
	}
	return r, nil
}

// ------------------------------------------------------------------
// Super-admin endpoint: GET /api/v1/platform/health/summary
// ------------------------------------------------------------------

// HealthSummary returns the latest platform_health_snapshots
// row plus the last 24 hourly rows for the trend sparkline.
//
// 200 → healthSummaryResp{latest, trend}.
// 401 → no JWT. 403 → not super_admin.
func HealthSummary(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsUser, ok := userFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		if claimsUser.Role != "super_admin" {
			kernel.RespondErrorWithCode(c, http.StatusForbidden,
				"forbidden", "super_admin role required")
			return
		}

		ctx := c.Request.Context()
		out := healthSummaryResp{Trend: []healthSummaryRow{}}

		// 1) Latest snapshot — LIMIT 1, ORDER BY snapshot_at DESC.
		rows, err := pool.Pgx().Query(ctx,
			`SELECT `+healthSummarySelect+`
			   FROM platform_health_snapshots
			  ORDER BY snapshot_at DESC
			  LIMIT 1`)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if rows.Next() {
			r, scanErr := scanHealthSummaryRow(rows)
			if scanErr != nil {
				rows.Close()
				kernel.RespondError(c, scanErr)
				return
			}
			out.Latest = r
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}

		// 2) Last 24 hourly snapshots for the sparkline.
		rows, err = pool.Pgx().Query(ctx,
			`SELECT `+healthSummarySelect+`
			   FROM platform_health_snapshots
			  ORDER BY snapshot_at DESC
			  LIMIT 24`)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		for rows.Next() {
			r, scanErr := scanHealthSummaryRow(rows)
			if scanErr != nil {
				kernel.RespondError(c, scanErr)
				return
			}
			out.Trend = append(out.Trend, r)
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, out)
	}
}

// ------------------------------------------------------------------
// Super-admin endpoint: GET /api/v1/platform/health/regions
// ------------------------------------------------------------------

// HealthRegionsSummary returns aggregated per-kind + per-
// status counts for the platform_regions catalog. The
// dashboard renders a single tile from this response.
//
// 200 → healthRegionSummaryResp{...}.
func HealthRegionsSummary(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsUser, ok := userFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		if claimsUser.Role != "super_admin" {
			kernel.RespondErrorWithCode(c, http.StatusForbidden,
				"forbidden", "super_admin role required")
			return
		}

		var out healthRegionSummaryResp
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT
			   COUNT(*)::int AS total,
			   COUNT(*) FILTER (WHERE region_kind = 'primary')::int   AS primary,
			   COUNT(*) FILTER (WHERE region_kind = 'replica')::int   AS replica,
			   COUNT(*) FILTER (WHERE region_kind = 'standby')::int   AS standby,
			   COUNT(*) FILTER (WHERE last_health_status = 'up')::int       AS up,
			   COUNT(*) FILTER (WHERE last_health_status = 'degraded')::int AS degraded,
			   COUNT(*) FILTER (WHERE last_health_status = 'down')::int     AS down
			 FROM platform_regions
			 WHERE is_active = true`,
		).Scan(
			&out.TotalRegions,
			&out.PrimaryRegions, &out.ReplicaRegions, &out.StandbyRegions,
			&out.UpRegions, &out.DegradedRegions, &out.DownRegions,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, out)
	}
}

// ------------------------------------------------------------------
// Super-admin endpoint: GET /api/v1/platform/health/tenants/top
// ------------------------------------------------------------------

// HealthTopTenants returns the top-N tenants by 30-day usage
// event volume, sorted DESC. The `?n=` query param caps at
// healthTopTenantsMaxN (50) to bound the SQL.
//
// 200 → healthTopTenantResp{top: [...]}.
// 400 → invalid ?n= (non-integer / out-of-range).
func HealthTopTenants(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsUser, ok := userFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		if claimsUser.Role != "super_admin" {
			kernel.RespondErrorWithCode(c, http.StatusForbidden,
				"forbidden", "super_admin role required")
			return
		}

		n := healthTopTenantsDefaultN
		if raw := c.Query("n"); raw != "" {
			parsed, perr := strconv.Atoi(raw)
			if perr != nil || parsed < 1 || parsed > healthTopTenantsMaxN {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest,
					"bad_request",
					fmt.Sprintf("?n= must be 1..%d", healthTopTenantsMaxN))
				return
			}
			n = parsed
		}

		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT t.id::text,
			        t.name,
			        COALESCE(t.plan, 'free') AS plan,
			        COALESCE(SUM(a.count), 0)::bigint AS total_events,
			        (SELECT COUNT(*) FROM servers s WHERE s.tenant_id = t.id)::int AS server_count
			   FROM tenants t
			   LEFT JOIN platform_usage_aggregates a
			         ON a.tenant_id = t.id
			        AND a.bucket_ts >= NOW() - INTERVAL '30 days'
			  GROUP BY t.id, t.name, t.plan
			  ORDER BY total_events DESC
			  LIMIT $1`, n)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := healthTopTenantResp{Top: []healthTopTenantRow{}}
		for rows.Next() {
			var r healthTopTenantRow
			if err := rows.Scan(&r.TenantID, &r.Name, &r.Plan, &r.TotalEvents, &r.ServerCount); err != nil {
				kernel.RespondError(c, err)
				return
			}
			out.Top = append(out.Top, r)
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, out)
	}
}

// ------------------------------------------------------------------
// Super-admin endpoint: GET /api/v1/platform/health/capacity/forecast
// ------------------------------------------------------------------

// HealthCapacityForecast returns projected capacity
// exhaustion dates for disk / db connections / api throughput.
// Extrapolated from the last 30 days of
// platform_health_snapshots via a simple linear model.
//
// 200 → healthForecastResp{forecast_days, ...}.
func HealthCapacityForecast(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsUser, ok := userFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		if claimsUser.Role != "super_admin" {
			kernel.RespondErrorWithCode(c, http.StatusForbidden,
				"forbidden", "super_admin role required")
			return
		}

		ctx := c.Request.Context()
		diskFullAt := forecastExhaustion(ctx, pool,
			"storage_gb_used", 10_000) // 10 TB default cap
		dbConnFullAt := forecastExhaustion(ctx, pool,
			"db_connections", 100) // 100 connections default cap
		apiThroughputFullAt := forecastExhaustion(ctx, pool,
			"api_calls_per_min_5m_avg", 100_000) // 100k req/min cap

		kernel.RespondOK(c, healthForecastResp{
			ForecastDays:                 30,
			ProjectedDiskFullAt:          diskFullAt,
			ProjectedDBConnectionsFullAt: dbConnFullAt,
			ProjectedAPIThroughputFullAt: apiThroughputFullAt,
		})
	}
}

// ------------------------------------------------------------------
// Super-admin endpoint: GET /api/v1/platform/health/alerts
// ------------------------------------------------------------------

// HealthAlerts returns the active platform-level alerts
// aggregated from the latest platform_health_snapshots row.
// Three severity tiers: critical / warning / info.
//
// 200 → healthAlertsResp{alerts: [...]}.
// 401 → no JWT. 403 → not super_admin.
func HealthAlerts(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsUser, ok := userFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		if claimsUser.Role != "super_admin" {
			kernel.RespondErrorWithCode(c, http.StatusForbidden,
				"forbidden", "super_admin role required")
			return
		}

		var (
			overall          string
			apiUp, dbUp      bool
			ingestUp, aUp    bool
			aiUp, wtUp       bool
			serversDown      int
			serversStale     int
			openAlerts       int
			failedLogins24h  int
			backupCount      int
			activeTenants24h int
			snapshotAt       time.Time
		)
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT overall_status,
			        api_up, db_up, ingest_up, alert_engine_up, ai_engine_up, web_terminal_up,
			        servers_down, servers_stale,
			        open_alerts, failed_logins_24h,
			        backup_count, active_tenants_24h,
			        snapshot_at
			   FROM platform_health_snapshots
			  ORDER BY snapshot_at DESC
			  LIMIT 1`,
		).Scan(
			&overall, &apiUp, &dbUp, &ingestUp, &aUp, &aiUp, &wtUp,
			&serversDown, &serversStale,
			&openAlerts, &failedLogins24h,
			&backupCount, &activeTenants24h,
			&snapshotAt,
		)
		if err != nil && err != pgx.ErrNoRows {
			kernel.RespondError(c, err)
			return
		}
		if err == pgx.ErrNoRows {
			// No snapshot yet → empty alerts list (the worker
			// hasn't run). kernel.RespondOK enforces the
			// empty-slice shape via the make() in the type.
			kernel.RespondOK(c, healthAlertsResp{Alerts: []healthAlertRow{}})
			return
		}
		kernel.RespondOK(c, buildPlatformAlerts(platformAlertInput{
			Overall:          overall,
			APIUp:            apiUp,
			DBUp:             dbUp,
			IngestUp:         ingestUp,
			AlertEngineUp:    aUp,
			AIEngineUp:       aiUp,
			WebTerminalUp:    wtUp,
			ServersDown:      serversDown,
			ServersStale:     serversStale,
			OpenAlerts:       openAlerts,
			FailedLogins24h:  failedLogins24h,
			BackupCount:      backupCount,
			ActiveTenants24h: activeTenants24h,
			SnapshotAt:       snapshotAt,
		}))
	}
}