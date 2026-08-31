import { useEffect, useMemo, useState } from 'react';
import { ApiError, api, getToken } from '../../lib/api';
import {
  motion,
  pageEnter,
  kpiStagger,
  kpiEnter,
} from '../../lib/motion';
import Button from '../shared/Button';
import EmptyState from '../shared/EmptyState';
import KpiCard from '../shared/KpiCard';
import HealthSectionTables, {
  HealthAlertRow,
  HealthForecastResp,
  HealthRegionsResp,
  HealthSummaryResp,
  HealthTopTenant,
} from './HealthSectionTables';

// Tier 11 Phase 8 — Platform Health (PL8).
//
// HealthSection renders the operator-facing health dashboard.
// Used by the super_admin-only PlatformPage tab (Phase 8
// finalises the 6-tab customer-facing shell; PL8 stays
// operator-only).
//
// Layout: header + KPI strip + trend + service grid +
// regions card + tables (top tenants / forecast / alerts).
// Types live in HealthSectionTables.tsx (which re-exports
// them). Tables themselves render there too — this file
// only handles the "above-the-fold" panels + auth gate +
// auto-refresh.
//
// Motion : pageEnter on section wrapper; kpiStagger + kpiEnter
// on KPI strip. No new variants — reuses existing exports.

export type {
  HealthAlertRow,
  HealthForecastResp,
  HealthRegionsResp,
  HealthSummaryResp,
  HealthTopTenant,
};

const SERVICE_ROWS: Array<{ key: keyof HealthSummaryResp['latest']; label: string }> = [
  { key: 'api_up', label: 'API gateway' },
  { key: 'db_up', label: 'Postgres' },
  { key: 'ingest_up', label: 'Ingest worker' },
  { key: 'alert_engine_up', label: 'Alert engine' },
  { key: 'ai_engine_up', label: 'AI engine' },
  { key: 'web_terminal_up', label: 'Web terminal' },
];

interface HealthSectionProps {
  /** True when caller is super_admin and may read /health/* endpoints. */
  canRead: boolean;
}

export default function HealthSection({ canRead }: HealthSectionProps) {
  const [summary, setSummary] = useState<HealthSummaryResp | null>(null);
  const [regions, setRegions] = useState<HealthRegionsResp | null>(null);
  const [top, setTop] = useState<HealthTopTenant[]>([]);
  const [forecast, setForecast] = useState<HealthForecastResp | null>(null);
  const [alerts, setAlerts] = useState<HealthAlertRow[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [lastUpdated, setLastUpdated] = useState('');

  const loadAll = async () => {
    if (!getToken()) {
      setError('Sign in to view platform health.');
      return;
    }
    setBusy(true);
    setError('');
    setSuccess('');
    try {
      const [s, r, t, f, a] = await Promise.all([
        api<HealthSummaryResp>('GET', '/api/v1/platform/health/summary'),
        api<HealthRegionsResp>('GET', '/api/v1/platform/health/regions'),
        api<{ top: HealthTopTenant[] }>('GET', '/api/v1/platform/health/tenants/top?n=10'),
        api<HealthForecastResp>('GET', '/api/v1/platform/health/capacity/forecast'),
        api<{ alerts: HealthAlertRow[] }>('GET', '/api/v1/platform/health/alerts'),
      ]);
      setSummary(s);
      setRegions(r);
      setTop(t.top ?? []);
      setForecast(f);
      setAlerts(a.alerts ?? []);
      setLastUpdated(new Date().toISOString());
      setSuccess('Health refreshed.');
    } catch (cause) {
      if (cause instanceof ApiError && cause.status === 401) {
        setError('Sign in to view platform health.');
      } else if (cause instanceof ApiError && cause.status === 403) {
        setError('Super-admin role required to view platform health.');
      } else {
        setError(
          cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message
        );
      }
    } finally {
      setBusy(false);
    }
  };

  useEffect(() => {
    if (!canRead) return;
    let cancelled = false;
    void (async () => {
      await loadAll();
      if (cancelled) return;
    })();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [canRead]);

  useEffect(() => {
    if (!canRead) return;
    const id = window.setInterval(() => {
      void loadAll();
    }, 60_000);
    return () => window.clearInterval(id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [canRead]);

  const overall = summary?.latest?.overall_status ?? 'unknown';
  const latest = summary?.latest ?? null;

  const sparkline = useMemo(() => {
    const xs = summary?.trend ?? [];
    if (xs.length < 2) return null;
    const width = 240;
    const height = 40;
    const max = Math.max(...xs.map((r) => r.total_tenants || 0), 1);
    const step = width / Math.max(xs.length - 1, 1);
    const points = xs
      .map(
        (r, i) =>
          `${(i * step).toFixed(1)},${(height - (r.total_tenants / max) * height).toFixed(1)}`
      )
      .join(' ');
    return { width, height, points };
  }, [summary]);

  if (!canRead) {
    return (
      <EmptyState
        illustration={<span aria-hidden="true">🔒</span>}
        headline="Super-admin only"
        subhead="The platform health dashboard is restricted to super_admin role."
      />
    );
  }

  return (
    <motion.div initial="hidden" animate="show" variants={pageEnter} className="space-y-6 p-2">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold">Platform Health</h2>
          <p className="text-sm opacity-70">
            Last updated: {lastUpdated ? new Date(lastUpdated).toLocaleTimeString() : 'never'}
            {latest ? ` · snapshot ${new Date(latest.snapshot_at).toLocaleTimeString()}` : null}
          </p>
        </div>
        <Button
          variant="primary"
          size="md"
          onClick={() => void loadAll()}
          disabled={busy}
          loading={busy}
        >
          {busy ? 'Refreshing…' : 'Refresh'}
        </Button>
      </header>

      {error ? <div className="callout-error" role="alert">{error}</div> : null}
      {success && !error ? <div className="callout-success" role="status">{success}</div> : null}

      <motion.section
        variants={kpiStagger}
        initial="hidden"
        animate="show"
        className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3"
      >
        <motion.div variants={kpiEnter}>
          <KpiCard label="Overall" value={overall.toUpperCase()} accent="cyan"
            status={overall === 'down' ? 'stale' : 'neutral'} />
        </motion.div>
        <motion.div variants={kpiEnter}>
          <KpiCard label="Active tenants (24h)" value={latest?.active_tenants_24h ?? 0} accent="indigo" />
        </motion.div>
        <motion.div variants={kpiEnter}>
          <KpiCard label="Servers up" value={latest?.servers_up ?? 0} accent="green" />
        </motion.div>
        <motion.div variants={kpiEnter}>
          <KpiCard label="Open alerts" value={latest?.open_alerts ?? 0} accent="amber"
            status={(latest?.open_alerts ?? 0) > 10 ? 'stale' : 'neutral'} />
        </motion.div>
        <motion.div variants={kpiEnter}>
          <KpiCard label="DB connections" value={latest?.db_connections ?? 0} accent="cyan" />
        </motion.div>
        <motion.div variants={kpiEnter}>
          <KpiCard label="API calls / min" value={latest?.api_calls_per_min_5m_avg ?? 0} accent="violet" />
        </motion.div>
      </motion.section>

      {sparkline ? (
        <section className="kpi-card">
          <h3 className="text-sm font-semibold">
            Active tenants trend (last {summary?.trend.length ?? 0} snapshots)
          </h3>
          <svg viewBox={`0 0 ${sparkline.width} ${sparkline.height}`}
            preserveAspectRatio="none"
            className="w-full h-12 mt-2"
            aria-label="active tenants trend">
            <polyline fill="none" stroke="var(--accent-cyan, #22d3ee)" strokeWidth={2}
              points={sparkline.points} />
          </svg>
        </section>
      ) : null}

      {latest ? (
        <section>
          <h3 className="text-sm font-semibold mb-2">Service status</h3>
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-2">
            {SERVICE_ROWS.map(({ key, label }) => {
              const ok = Boolean(latest[key]);
              return (
                <div key={key} className="kpi-card flex flex-col gap-1">
                  <span className="text-xs opacity-70">{label}</span>
                  <span className="dash-status-pill"
                    style={{
                      background: ok ? 'rgba(16,185,129,0.15)' : 'rgba(239,68,68,0.15)',
                      color: ok ? 'var(--accent-emerald, #10b981)' : 'var(--accent-red, #ef4444)',
                    }}>
                    <span className="dash-status-dot"
                      style={{
                        background: ok ? 'var(--accent-emerald, #10b981)' : 'var(--accent-red, #ef4444)',
                      }} />
                    {ok ? 'UP' : 'DOWN'}
                  </span>
                </div>
              );
            })}
          </div>
        </section>
      ) : null}

      {regions ? (
        <section className="kpi-card">
          <h3 className="text-sm font-semibold mb-2">Regions</h3>
          <div className="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-7 gap-2 text-sm">
            <span>Total: <strong>{regions.total_regions}</strong></span>
            <span>Primary: <strong>{regions.primary_regions}</strong></span>
            <span>Replica: <strong>{regions.replica_regions}</strong></span>
            <span>Standby: <strong>{regions.standby_regions}</strong></span>
            <span style={{ color: 'var(--accent-emerald, #10b981)' }}>Up: <strong>{regions.up_regions}</strong></span>
            <span style={{ color: 'var(--accent-amber, #f59e0b)' }}>Degraded: <strong>{regions.degraded_regions}</strong></span>
            <span style={{ color: 'var(--accent-red, #ef4444)' }}>Down: <strong>{regions.down_regions}</strong></span>
          </div>
        </section>
      ) : null}

      <HealthSectionTables top={top} forecast={forecast} alerts={alerts} />

      {!latest && !busy && !error ? (
        <EmptyState
          illustration={<span aria-hidden="true">📊</span>}
          headline="No health snapshots yet"
          subhead="The capacity-forecast worker hasn't run. It ticks hourly; the first snapshot should appear within a minute of boot."
        />
      ) : null}
    </motion.div>
  );
}