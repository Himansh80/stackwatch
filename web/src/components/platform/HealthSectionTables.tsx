// Tier 11 Phase 8 — Platform Health (PL8).
//
// HealthSectionTables — bottom half of the health dashboard:
// top-tenants table, capacity-forecast panel, and active-
// alerts table. Lifted out of HealthSection.tsx to keep that
// file under the 400-LOC cap while sharing types via re-export
// (HealthSection.tsx imports the types from here so callers
// can keep their existing import shape).
//
// Why the types live here and not in HealthSection.tsx:
//   They're shared with HealthSection (so they're declared
//   once) AND they're consumed internally by the table
//   renderers — keeping both in this file avoids a circular
//   import. HealthSection.tsx re-exports them via `export type`
//   so external callers see the same public surface.

export interface HealthSummaryRow {
  id: string;
  snapshot_at: string;
  overall_status: 'up' | 'degraded' | 'down' | 'unknown';
  api_up: boolean;
  db_up: boolean;
  ingest_up: boolean;
  alert_engine_up: boolean;
  ai_engine_up: boolean;
  web_terminal_up: boolean;
  total_tenants: number;
  active_tenants_24h: number;
  total_servers: number;
  servers_up: number;
  servers_stale: number;
  servers_down: number;
  open_alerts: number;
  failed_logins_24h: number;
  db_connections: number;
  api_calls_per_min_5m_avg: number;
  storage_gb_used: number;
  backup_count: number;
  backup_latest_at: string;
}

export interface HealthSummaryResp {
  latest: HealthSummaryRow;
  trend: HealthSummaryRow[];
}

export interface HealthRegionsResp {
  total_regions: number;
  primary_regions: number;
  replica_regions: number;
  standby_regions: number;
  up_regions: number;
  degraded_regions: number;
  down_regions: number;
}

export interface HealthTopTenant {
  tenant_id: string;
  name: string;
  plan: string;
  total_events: number;
  server_count: number;
}

export interface HealthForecastResp {
  forecast_days: number;
  projected_disk_full_at: string;
  projected_db_connections_full_at: string;
  projected_api_throughput_full_at: string;
}

export interface HealthAlertRow {
  severity: 'critical' | 'warning' | 'info';
  message: string;
  created_at: string;
  link: string;
}

function severityColor(s: string): string {
  switch (s) {
    case 'critical':
      return 'var(--accent-red, #ef4444)';
    case 'warning':
      return 'var(--accent-amber, #f59e0b)';
    case 'info':
      return 'var(--accent-cyan, #22d3ee)';
    default:
      return 'var(--text-muted, #94a3b8)';
  }
}

interface TablesProps {
  top: HealthTopTenant[];
  forecast: HealthForecastResp | null;
  alerts: HealthAlertRow[];
}

export default function HealthSectionTables({ top, forecast, alerts }: TablesProps) {
  return (
    <>
      {/* Top tenants by 30-day usage. */}
      <section className="kpi-card">
        <h3 className="text-sm font-semibold mb-2">Top tenants (30-day usage)</h3>
        {top.length === 0 ? (
          <p className="text-sm opacity-60">No usage events recorded yet.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left opacity-70">
                  <th className="py-1 pr-3">Tenant</th>
                  <th className="py-1 pr-3">Plan</th>
                  <th className="py-1 pr-3 text-right">Events (30d)</th>
                  <th className="py-1 pr-3 text-right">Servers</th>
                </tr>
              </thead>
              <tbody>
                {top.map((row) => (
                  <tr key={row.tenant_id} className="border-t border-white/5">
                    <td className="py-1 pr-3">{row.name}</td>
                    <td className="py-1 pr-3">{row.plan}</td>
                    <td className="py-1 pr-3 text-right">{row.total_events.toLocaleString()}</td>
                    <td className="py-1 pr-3 text-right">{row.server_count}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      {/* Capacity forecast. */}
      {forecast ? (
        <section className="kpi-card">
          <h3 className="text-sm font-semibold mb-2">
            Capacity forecast (next {forecast.forecast_days} days)
          </h3>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 text-sm">
            <div>
              <span className="opacity-70">Disk full at</span>
              <div className="font-mono">{forecast.projected_disk_full_at || 'no signal'}</div>
            </div>
            <div>
              <span className="opacity-70">DB connections full at</span>
              <div className="font-mono">{forecast.projected_db_connections_full_at || 'no signal'}</div>
            </div>
            <div>
              <span className="opacity-70">API throughput full at</span>
              <div className="font-mono">{forecast.projected_api_throughput_full_at || 'no signal'}</div>
            </div>
          </div>
        </section>
      ) : null}

      {/* Active alerts. */}
      <section className="kpi-card">
        <h3 className="text-sm font-semibold mb-2">Active alerts ({alerts.length})</h3>
        {alerts.length === 0 ? (
          <p className="text-sm opacity-60">No active alerts. ✅</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left opacity-70">
                  <th className="py-1 pr-3">Severity</th>
                  <th className="py-1 pr-3">Message</th>
                  <th className="py-1 pr-3">As of</th>
                </tr>
              </thead>
              <tbody>
                {alerts.map((a, i) => (
                  <tr key={i} className="border-t border-white/5">
                    <td className="py-1 pr-3">
                      <span
                        className="dash-status-pill"
                        style={{ background: 'transparent', color: severityColor(a.severity) }}
                      >
                        {a.severity.toUpperCase()}
                      </span>
                    </td>
                    <td className="py-1 pr-3">{a.message}</td>
                    <td className="py-1 pr-3 font-mono text-xs opacity-70">
                      {new Date(a.created_at).toLocaleTimeString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </>
  );
}