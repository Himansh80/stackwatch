// Tier 11 Phase 8 — Platform Health (PL8).
//
// handlers_platform_health_types.go — JSON request / response
// shapes + the allowlist of allowed overall statuses for the 5
// health endpoints:
//
//	GET /api/v1/platform/health/summary            — HealthSummary         (super_admin)
//	GET /api/v1/platform/health/regions            — HealthRegionsSummary  (super_admin)
//	GET /api/v1/platform/health/tenants/top         — HealthTopTenants      (super_admin)
//	GET /api/v1/platform/health/capacity/forecast  — HealthCapacityForecast(super_admin)
//	GET /api/v1/platform/health/alerts             — HealthAlerts          (super_admin)
//
// The handlers themselves live in handlers_platform_health.go.
// Splitting the types out keeps that file under the 400-LOC cap
// while making the request/response surface easy to scan.
//
// Why these shapes:
//
//   - healthSummaryResp is a thin wrapper around the latest row
//     from platform_health_snapshots plus a 24-element trend
//     array (last 24 hourly snapshots). The trend is what the
//     Datadog-style sparkline below the KPI strip renders — the
//     widget auto-refreshes every 60s and re-fetches /summary.
//
//   - healthRegionSummaryResp aggregates per-kind counts
//     (primary/replica/standby × up/degraded/down) so the
//     operator dashboard can render a single region tile without
//     a second query.
//
//   - healthTopTenantResp mirrors platform_usage_aggregates +
//     tenants rows; sorted DESC by total_events.
//
//   - healthForecastResp is a flat struct of ISO8601 exhaustion
//     dates. A null means "no signal" (e.g. no servers).
//
//   - healthAlertsResp is a flat list of active platform-level
//     alerts (severity/message/created_at/link).
//
//   - allowedHealthStatuses is the allowlist of overall_status
//     values the CapacityForecastWorker may write into
//     platform_health_snapshots. Defence-in-depth.

package handler

// allowedHealthStatuses is the server-side allowlist of overall
// health status values. Mirrors what CapacityForecastWorker
// actually writes — anything else gets a 400 in the worker.
var allowedHealthStatuses = map[string]struct{}{
	"up":      {},
	"degraded": {},
	"down":    {},
	"unknown": {},
}

// healthSummaryResp is the JSON shape for GET
// /api/v1/platform/health/summary.
//
// `latest` is the most-recent row from platform_health_snapshots
// (or a zero-value struct if the worker hasn't run yet).
// `trend` is the last 24 hourly snapshots for the sparkline.
type healthSummaryResp struct {
	Latest healthSummaryRow   `json:"latest"`
	Trend  []healthSummaryRow `json:"trend"`
}

// healthSummaryRow mirrors a single row of
// platform_health_snapshots. JSON tags use snake_case to
// match the table column names — this lets a future refactor
// swap to pgx.CollectRows + json.Marshal without re-mapping.
type healthSummaryRow struct {
	ID                    string `json:"id"`
	SnapshotAt            string `json:"snapshot_at"`
	OverallStatus         string `json:"overall_status"`
	APIUp                 bool   `json:"api_up"`
	DBUp                  bool   `json:"db_up"`
	IngestUp              bool   `json:"ingest_up"`
	AlertEngineUp         bool   `json:"alert_engine_up"`
	AIEngineUp            bool   `json:"ai_engine_up"`
	WebTerminalUp         bool   `json:"web_terminal_up"`
	TotalTenants          int    `json:"total_tenants"`
	ActiveTenants24h      int    `json:"active_tenants_24h"`
	TotalServers          int    `json:"total_servers"`
	ServersUp             int    `json:"servers_up"`
	ServersStale          int    `json:"servers_stale"`
	ServersDown           int    `json:"servers_down"`
	OpenAlerts            int    `json:"open_alerts"`
	FailedLogins24h       int    `json:"failed_logins_24h"`
	DBConnections         int    `json:"db_connections"`
	APICallsPerMin5mAvg   int    `json:"api_calls_per_min_5m_avg"`
	StorageGBUsed         int64  `json:"storage_gb_used"`
	BackupCount           int    `json:"backup_count"`
	BackupLatestAt        string `json:"backup_latest_at"`
}

// healthRegionSummaryResp is the JSON shape for GET
// /api/v1/platform/health/regions. Aggregated counts so the
// operator dashboard can render a single region tile.
type healthRegionSummaryResp struct {
	TotalRegions   int `json:"total_regions"`
	PrimaryRegions int `json:"primary_regions"`
	ReplicaRegions int `json:"replica_regions"`
	StandbyRegions int `json:"standby_regions"`
	UpRegions      int `json:"up_regions"`
	DegradedRegions int `json:"degraded_regions"`
	DownRegions    int `json:"down_regions"`
}

// healthTopTenantResp is the JSON shape for GET
// /api/v1/platform/health/tenants/top.
type healthTopTenantResp struct {
	Top []healthTopTenantRow `json:"top"`
}

// healthTopTenantRow is one tenant in the top-N list.
// `plan` is read from tenants.plan; `total_events` is the
// sum of platform_usage_events.count for the last 30 days;
// `server_count` is the row count of servers for that tenant.
type healthTopTenantRow struct {
	TenantID    string `json:"tenant_id"`
	Name        string `json:"name"`
	Plan        string `json:"plan"`
	TotalEvents int64  `json:"total_events"`
	ServerCount int    `json:"server_count"`
}

// healthForecastResp is the JSON shape for GET
// /api/v1/platform/health/capacity/forecast. Every date is
// ISO8601; an empty string means "no signal" (worker has no
// data to extrapolate from).
type healthForecastResp struct {
	ForecastDays                  int    `json:"forecast_days"`
	ProjectedDiskFullAt           string `json:"projected_disk_full_at"`
	ProjectedDBConnectionsFullAt  string `json:"projected_db_connections_full_at"`
	ProjectedAPIThroughputFullAt  string `json:"projected_api_throughput_full_at"`
}

// healthAlertsResp is the JSON shape for GET
// /api/v1/platform/health/alerts. Flat list — an empty list
// serialises as [] (not null) so the dashboard renders an
// explicit "no active alerts" row.
type healthAlertsResp struct {
	Alerts []healthAlertRow `json:"alerts"`
}

// healthAlertRow is one active platform-level alert.
type healthAlertRow struct {
	Severity  string `json:"severity"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
	Link      string `json:"link"`
}

// healthTopTenantsDefaultN is the default value for `?n=` on
// the top-tenants endpoint. Matches the spec's `?n=10`
// default; capped at healthTopTenantsMaxN to bound the SQL.
const (
	healthTopTenantsDefaultN = 10
	healthTopTenantsMaxN     = 50
)