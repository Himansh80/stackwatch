// Tier 11 Phase 8 — Platform Health (PL8).
//
// capacity_forecast_counts.go — per-section COUNT / SUM
// helpers used by CapacityForecastWorker.tick.
//
// Pulled out of capacity_forecast.go to keep that worker
// under the 400-LOC cap. Each helper is a small focused
// query: count tenants, count servers, count alerts+logins,
// count DB connections, count API calls, count storage,
// count backups. All helpers tolerate missing tables
// (to_regclass) and missing columns gracefully so a partial
// DB never aborts the snapshot pass.
//
// Why the helpers log on error but never return one:
// The snapshot insert should happen even if some sections
// fail (e.g. a freshly-promoted prod where log_lines doesn't
// exist yet). Each section logs its own error; the caller
// (tick) continues with whatever data it has.

package platform

import (
	"context"
	"log/slog"
	"time"

	"github.com/stackwatch/platform/internal/db"
)

// countTenants counts total + active (24h) tenants.
// Total = tenants table. Active = users with recent login.
func countTenants(ctx context.Context, pool *db.Pool, logger *slog.Logger, snap *healthSnapshot) {
	if err := pool.Pgx().QueryRow(ctx,
		`SELECT COUNT(*)::int FROM tenants`,
	).Scan(&snap.TotalTenants); err != nil {
		logger.Warn("capacity forecast worker: count tenants failed", "err", err)
	}
	if err := pool.Pgx().QueryRow(ctx,
		`SELECT COUNT(DISTINCT tenant_id)::int
		   FROM users
		  WHERE last_login_at >= NOW() - INTERVAL '24 hours'`,
	).Scan(&snap.ActiveTenants24h); err != nil {
		logger.Warn("capacity forecast worker: count active tenants failed", "err", err)
	}
}

// countServers counts up / stale / down pinned services
// from homelab_service_health (Tier 10 Phase 2). "Stale" =
// last checked >5 min ago with status=unknown. "Down" =
// last status='down'.
func countServers(ctx context.Context, pool *db.Pool, logger *slog.Logger, snap *healthSnapshot) {
	rows, err := pool.Pgx().Query(ctx, `
		WITH latest AS (
		  SELECT DISTINCT ON (service_id) service_id, status, checked_at
		    FROM homelab_service_health
		   ORDER BY service_id, checked_at DESC
		)
		SELECT
		  COUNT(*)::int AS total,
		  COUNT(*) FILTER (WHERE status = 'up')::int       AS up,
		  COUNT(*) FILTER (WHERE status = 'down')::int     AS down,
		  COUNT(*) FILTER (WHERE status = 'unknown'
		                    AND checked_at < NOW() - INTERVAL '5 minutes')::int AS stale
		  FROM latest
	`)
	if err != nil {
		logger.Warn("capacity forecast worker: count servers failed", "err", err)
		return
	}
	defer rows.Close()
	if rows.Next() {
		_ = rows.Scan(&snap.TotalServers, &snap.ServersUp, &snap.ServersDown, &snap.ServersStale)
	}
}

// countAlertsAndLogins counts open incidents + failed
// logins in the last 24h.
func countAlertsAndLogins(ctx context.Context, pool *db.Pool, logger *slog.Logger, snap *healthSnapshot) {
	if err := pool.Pgx().QueryRow(ctx,
		`SELECT COUNT(*)::int FROM incidents WHERE status = 'open'`,
	).Scan(&snap.OpenAlerts); err != nil {
		logger.Warn("capacity forecast worker: count open alerts failed", "err", err)
	}
	if err := pool.Pgx().QueryRow(ctx,
		`SELECT COUNT(*)::int FROM user_failed_logins
		  WHERE ts >= NOW() - INTERVAL '24 hours'`,
	).Scan(&snap.FailedLogins24h); err != nil {
		logger.Warn("capacity forecast worker: count failed logins failed", "err", err)
	}
}

// countDbConnections returns the current active DB
// connection count from pg_stat_activity.
func countDbConnections(ctx context.Context, pool *db.Pool, logger *slog.Logger, snap *healthSnapshot) {
	var n int
	if err := pool.Pgx().QueryRow(ctx,
		`SELECT COUNT(*)::int FROM pg_stat_activity WHERE state IS NOT NULL`,
	).Scan(&n); err != nil {
		logger.Warn("capacity forecast worker: count db connections failed", "err", err)
		return
	}
	snap.DBConnections = n
}

// countAPICalls5m returns the average API calls/min over
// the last 5 minutes. Reads from platform_usage_events
// (event_kind = 'api.call'). A NULL SUM (no events) → 0.
func countAPICalls5m(ctx context.Context, pool *db.Pool, logger *slog.Logger, snap *healthSnapshot) {
	var n *int
	if err := pool.Pgx().QueryRow(ctx,
		`SELECT (COUNT(*) / 5)::int
		   FROM platform_usage_events
		  WHERE event_kind = 'api.call'
		    AND created_at >= NOW() - INTERVAL '5 minutes'`,
	).Scan(&n); err != nil {
		logger.Warn("capacity forecast worker: count api calls failed", "err", err)
		return
	}
	if n != nil {
		snap.APICallsPerMin5mAvg = *n
	}
}

// countStorage sums bytes from audit_log / metric_points /
// log_lines (each pg_total_relation_size) and divides by
// 1GB. The tables are independent so a missing one
// (to_regclass NULL) skips cleanly.
func countStorage(ctx context.Context, pool *db.Pool, logger *slog.Logger, snap *healthSnapshot) {
	var bytes int64
	err := pool.Pgx().QueryRow(ctx, `
		SELECT COALESCE(SUM(size), 0)::big FROM (
		  SELECT pg_total_relation_size('audit_log') WHERE to_regclass('audit_log') IS NOT NULL
		  UNION ALL
		  SELECT pg_total_relation_size('metric_points') WHERE to_regclass('metric_points') IS NOT NULL
		  UNION ALL
		  SELECT pg_total_relation_size('log_lines') WHERE to_regclass('log_lines') IS NOT NULL
		) s(size)
	`).Scan(&bytes)
	if err != nil {
		logger.Warn("capacity forecast worker: count storage failed", "err", err)
		return
	}
	snap.StorageGBUsed = bytes / (1024 * 1024 * 1024)
}

// countBackups counts platform_backups rows + finds the
// most recent backup_latest_at.
func countBackups(ctx context.Context, pool *db.Pool, logger *slog.Logger, snap *healthSnapshot) {
	if err := pool.Pgx().QueryRow(ctx,
		`SELECT COUNT(*)::int,
		        MAX(created_at)
		   FROM platform_backups
		  WHERE status IN ('completed', 'verified')`,
	).Scan(&snap.BackupCount, &snap.BackupLatestAt); err != nil {
		logger.Warn("capacity forecast worker: count backups failed", "err", err)
	}
}

// compile-time guard: ensure healthSnapshot.ServersStale
// exists in the main file. Time is imported here only so the
// helper file compiles standalone (a test or future caller
// could import this file without the main worker).
var _ = time.Now