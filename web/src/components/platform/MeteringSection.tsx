import { useEffect, useMemo, useState } from 'react';
import { ApiError, api, getToken } from '../../lib/api';
import { motion, kpiEnter, kpiStagger, pageEnter, useReducedMotion } from '../../lib/motion';
import EmptyState from '../shared/EmptyState';
import KpiCard from '../shared/KpiCard';

// Tier 11 Phase 2 — Usage Metering (PL2).
//
// MeteringSection renders the metering surface inside the
// unified PlatformPage (Phase 8 lands the surrounding 6-tab
// shell; today Phase 2 exports this component so a quick
// integration can drop it into any tab container).
//
// Layout (top → bottom):
//   - Header    : "Usage" + period dropdown + Export CSV button
//   - KPI strip : 4 cards — total events / top kind / estimated
//                 cost / distinct kinds observed
//   - Bar chart : per-kind bars (simple SVG, no recharts dep) so
//                 we stay inside the no-new-deps Tier 11 rule
//   - Detail    : per-kind counts/sums/cost with mini sparklines
//   - Empty     : helpful onboarding when no events exist
//
// Motion : pageEnter on the section wrapper; kpiStagger on the
// KPI strip; kpiEnter on each card. No new variants — reuses the
// existing motion primitives from src/lib/motion.tsx.
//
// Data sources:
//   GET /api/v1/platform/usage/current  — totals + per-kind rollup
//   GET /api/v1/platform/usage/history  — bucket series for chart
//   GET /api/v1/platform/usage/export?format=csv — raw events
//
// Auth: every load bails early when no JWT is present so we
// don't spam a 401 from the platform-admin login flow.

type Period = 'day' | 'week' | 'month';

interface KindSummary {
  event_kind: string;
  count: number;
  sum_quantity: number;
  cost_usd: number;
}

interface CurrentResp {
  period: Period;
  range_start: string;
  range_end: string;
  total_count: number;
  top_event_kind: string;
  top_event_count: number;
  estimated_cost: number;
  kinds: KindSummary[];
}

interface HistoryResp {
  periods: number;
  buckets: string[];
  series: Record<string, Array<{ bucket_ts: string; sum_quantity: number; count: number }>>;
}

interface MeteringSectionProps {
  /** True when the caller's role can reach the super_admin summary surface. */
  isPlatformAdmin: boolean;
}

/**
 * Per-kind colour palette. Static (no theme hook today) so the
 * chart stays stable across hot-reload — themes are a Phase 8 polish.
 * Each kind maps to a CSS variable already declared in styles.css.
 */
const KIND_COLOR: Record<string, string> = {
  'server.created':           'var(--accent-cyan, #22d3ee)',
  'server.deleted':           'var(--accent-violet, #8b5cf6)',
  'alert.fired':              'var(--accent-amber, #f59e0b)',
  'alert.resolved':           'var(--accent-emerald, #10b981)',
  'api.call':                 'var(--accent-indigo, #6366f1)',
  'storage.gb.hour':          'var(--accent-pink, #ec4899)',
  'dashboard.panel.rendered': 'var(--accent-blue, #3b82f6)',
  'login.success':            'var(--accent-emerald, #10b981)',
  'login.failure':            'var(--accent-red, #ef4444)',
};

const formatUSD = (n: number): string =>
  n === 0 ? '$0.00' : `$${n.toFixed(n < 0.01 ? 4 : 2)}`;

const formatQty = (n: number): string =>
  n >= 1000 ? `${(n / 1000).toFixed(1)}k` : `${n}`;

export default function MeteringSection({ isPlatformAdmin }: MeteringSectionProps) {
  const reduce = useReducedMotion();
  const [period, setPeriod] = useState<Period>('month');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [current, setCurrent] = useState<CurrentResp | null>(null);
  const [history, setHistory] = useState<HistoryResp | null>(null);

  // requireAuth — same pattern as HomelabPage. Bails to a
  // friendly error rather than letting the request 401 in the wild.
  const requireAuth = (): boolean => {
    if (!getToken()) {
      setError('Sign in to view usage metering.');
      return false;
    }
    return true;
  };

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      if (!requireAuth()) return;
      setBusy(true);
      setError('');
      try {
        const [c, h] = await Promise.all([
          api<CurrentResp>('GET', `/api/v1/platform/usage/current?period=${period}`),
          api<HistoryResp>('GET', `/api/v1/platform/usage/history?periods=14`),
        ]);
        if (cancelled) return;
        setCurrent(c);
        setHistory(h);
      } catch (cause) {
        if (!cancelled) {
          setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
        }
      } finally {
        if (!cancelled) setBusy(false);
      }
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, [period]);

  // Sorted top-kinds for the bar chart. Sort by count desc so
  // the longest bar always leads. Memoized to avoid recomputation
  // when the period dropdown is open / busy state flips.
  const sortedKinds = useMemo(() => {
    if (!current) return [];
    return [...current.kinds].sort((a, b) => b.count - a.count).slice(0, 8);
  }, [current]);

  const maxKindCount = useMemo(
    () => sortedKinds.reduce((m, k) => Math.max(m, k.count), 0),
    [sortedKinds],
  );

  // Export-CSV handler. Uses fetch + Blob so the browser
  // downloads with the right filename (a plain `<a download>`
  // would lose the auth header).
  const handleExport = async () => {
    if (!requireAuth()) return;
    try {
      const token = getToken();
      const resp = await fetch(
        `/api/v1/platform/usage/export?format=csv&limit=10000`,
        { headers: token ? { Authorization: `Bearer ${token}` } : undefined },
      );
      if (!resp.ok) {
        setError(`Export failed: HTTP ${resp.status}`);
        return;
      }
      const blob = await resp.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      const cd = resp.headers.get('content-disposition');
      const m = cd?.match(/filename="([^"]+)"/);
      a.download = m?.[1] || 'usage-export.csv';
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Export failed');
    }
  };

  const showEmpty = !busy && !error && current && current.total_count === 0;

  return (
    <motion.section
      className="metering-section"
      initial="hidden"
      animate="show"
      variants={pageEnter}
    >
      <header className="metering-header">
        <div>
          <h2 className="metering-title">Usage</h2>
          <p className="metering-sub">
            Platform metering — what billable events your tenant has produced{' '}
            {period === 'day'
              ? 'in the last 24 hours'
              : period === 'week'
                ? 'in the last 7 days'
                : 'in the last 30 days'}
            .
          </p>
        </div>
        <div className="metering-actions">
          <label className="metering-period">
            <span className="metering-period-label">Period</span>
            <select
              value={period}
              onChange={(e) => setPeriod(e.target.value as Period)}
              disabled={busy}
              aria-label="Select period"
            >
              <option value="day">Day</option>
              <option value="week">Week</option>
              <option value="month">Month</option>
            </select>
          </label>
          <motion.button
            type="button"
            className="metering-export"
            onClick={handleExport}
            disabled={busy}
            whileHover={reduce ? undefined : { y: -1 }}
            whileTap={reduce ? undefined : { scale: 0.98 }}
          >
            Export CSV
          </motion.button>
        </div>
      </header>

      {error ? <div className="metering-error" role="alert">{error}</div> : null}

      <motion.div className="dash-metric-strip" initial="hidden" animate="show" variants={kpiStagger}>
        <motion.div variants={kpiEnter}>
          <KpiCard
            label="Total events"
            value={current?.total_count ?? 0}
            status={current && current.total_count > 0 ? 'up' : 'neutral'}
            accent="cyan"
          />
        </motion.div>
        <motion.div variants={kpiEnter}>
          <KpiCard
            label="Top kind"
            value={current?.top_event_count ?? 0}
            status={current && current.top_event_count > 0 ? 'up' : 'neutral'}
            accent="indigo"
          />
        </motion.div>
        <motion.div variants={kpiEnter}>
          <KpiCard
            label="Estimated cost"
            value={formatUSD(current?.estimated_cost ?? 0)}
            status={'neutral'}
            accent="amber"
          />
        </motion.div>
        <motion.div variants={kpiEnter}>
          <KpiCard
            label="Distinct kinds"
            value={current?.kinds.length ?? 0}
            status={'neutral'}
            accent="violet"
          />
        </motion.div>
      </motion.div>

      {showEmpty ? (
        <EmptyState
          illustration={<span aria-hidden="true">📊</span>}
          headline="No usage events yet"
          subhead="Once a server is created, an alert is fired, or a dashboard panel renders, the event lands here. The hourly worker rolls events up so totals stay fast even at millions of rows."
        />
      ) : null}

      {current && current.kinds.length > 0 ? (
        <>
          <div className="metering-chart" role="img" aria-label="Events by kind">
            <div className="metering-chart-title">Events by kind</div>
            <div className="metering-bars">
              {sortedKinds.map((k) => {
                const pct = maxKindCount > 0 ? Math.round((k.count / maxKindCount) * 100) : 0;
                const color = KIND_COLOR[k.event_kind] || 'var(--accent-cyan, #22d3ee)';
                return (
                  <div className="metering-bar-row" key={k.event_kind}>
                    <div className="metering-bar-label" title={k.event_kind}>
                      {k.event_kind}
                    </div>
                    <div className="metering-bar-track">
                      <motion.div
                        className="metering-bar-fill"
                        initial={{ width: 0 }}
                        animate={{ width: `${pct}%` }}
                        transition={{ duration: reduce ? 0 : 0.5, ease: [0.16, 1, 0.3, 1] }}
                        style={{ background: color }}
                        title={`${k.count} events`}
                      />
                    </div>
                    <div className="metering-bar-val">{formatQty(k.count)}</div>
                  </div>
                );
              })}
            </div>
          </div>

          <div className="metering-detail">
            <div className="metering-detail-title">Detail by kind</div>
            <table className="metering-detail-table">
              <thead>
                <tr>
                  <th>Kind</th>
                  <th>Count</th>
                  <th>Sum qty</th>
                  <th>Cost</th>
                  <th>Trend (14d)</th>
                </tr>
              </thead>
              <tbody>
                {sortedKinds.map((k) => {
                  const series = history?.series[k.event_kind] || [];
                  const counts = series.map((p) => p.count);
                  const maxC = counts.reduce((m, n) => Math.max(m, n), 0) || 1;
                  return (
                    <tr key={k.event_kind}>
                      <td>
                        <span
                          className="metering-kind-dot"
                          style={{ background: KIND_COLOR[k.event_kind] || 'var(--accent-cyan, #22d3ee)' }}
                          aria-hidden="true"
                        />
                        {k.event_kind}
                      </td>
                      <td>{k.count.toLocaleString()}</td>
                      <td>{k.sum_quantity.toFixed(k.sum_quantity < 1 ? 4 : 2)}</td>
                      <td>{formatUSD(k.cost_usd)}</td>
                      <td>
                        <svg width="120" height="22" viewBox="0 0 120 22" aria-hidden="true">
                          {counts.length === 0 ? (
                            <text x="0" y="15" fontSize="10" fill="currentColor" opacity="0.4">
                              no data
                            </text>
                          ) : (
                            counts.map((c, i) => {
                              const x = (i / Math.max(counts.length - 1, 1)) * 116 + 2;
                              const h = (c / maxC) * 18;
                              return (
                                <rect
                                  key={i}
                                  x={x}
                                  y={20 - h}
                                  width={Math.max(2, 116 / Math.max(counts.length, 1) - 1)}
                                  height={h}
                                  fill={KIND_COLOR[k.event_kind] || 'var(--accent-cyan, #22d3ee)'}
                                  opacity="0.75"
                                />
                              );
                            })
                          )}
                        </svg>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
            {isPlatformAdmin ? (
              <p className="metering-admin-hint">
                Platform admins can also read the cross-tenant summary at
                <code> GET /api/v1/platform/usage/summary</code>.
              </p>
            ) : null}
          </div>
        </>
      ) : null}

      {current && current.top_event_kind ? (
        <p className="metering-foot">
          Top event this period: <strong>{current.top_event_kind}</strong> ({current.top_event_count} events).
        </p>
      ) : null}
    </motion.section>
  );
}
