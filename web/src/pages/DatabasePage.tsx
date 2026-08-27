import { useCallback, useEffect, useState } from 'react';
import { ApiError, api, getToken } from '../lib/api';
import EmptyState from '../components/shared/EmptyState';
import KpiCard from '../components/shared/KpiCard';
import SlowQueryTable, { SlowQuery } from '../components/shared/SlowQueryTable';
import StatusPill from '../components/shared/StatusPill';
import { motion, kpiStagger, pageEnter } from '../lib/motion';

interface ConnectionPool {
  id: string;
  database: string;
  host: string;
  port: number;
  pool_size: number;
  active: number;
  idle: number;
  waiting: number;
  checked_at: string;
}

interface Explain {
  query_hash: string;
  query_text: string;
  database: string;
  total_executions: number;
  avg_duration_ms: number;
  suggestion: string;
  note: string;
}

type Tab = 'slow' | 'pools' | 'top';
type DbFilter = 'all' | 'postgres' | 'mysql' | 'mariadb' | 'mongodb';

/**
 * DatabasePage — Tier 7.8 (D9) DB Monitoring surface at /database.
 *
 * Layout:
 *   - Top: 3 KpiCards (queries today / avg duration / slow queries count)
 *   - Middle: tabbed view (Slow Queries / Connection Pools / Top Queries)
 *     - Slow tab: SlowQueryTable + database dropdown + optional explain modal
 *     - Pools tab: card grid (one card per (db, host, port))
 *     - Top tab: top queries by frequency
 *   - Explain modal opens when a row is clicked on the slow-queries tab.
 *
 * Sidebar nav: "Database" added in AppSidebar under observability.
 *
 * Motion: pageEnter on the page; kpiStagger on the KPI strip. Reuses
 * existing tokens + classes. No new motion variants.
 */
export default function DatabasePage() {
  const [tab, setTab] = useState<Tab>('slow');
  const [error, setError] = useState('');

  const [slowQueries, setSlowQueries] = useState<SlowQuery[]>([]);
  const [topQueries, setTopQueries] = useState<SlowQuery[]>([]);
  const [pools, setPools] = useState<ConnectionPool[]>([]);
  const [dbFilter, setDbFilter] = useState<DbFilter>('all');

  const [explain, setExplain] = useState<Explain | null>(null);
  const [explainLoading, setExplainLoading] = useState(false);

  const requireAuth = (): boolean => {
    if (!getToken()) {
      setError('Sign in to view Database Monitoring.');
      return false;
    }
    return true;
  };

  const loadSlow = useCallback(async () => {
    if (!requireAuth()) return;
    try {
      const params = new URLSearchParams();
      if (dbFilter !== 'all') params.set('database', dbFilter);
      const qs = params.toString();
      const r = await api<{ slow_queries?: SlowQuery[] }>('GET', `/api/v1/database/slow-queries${qs ? `?${qs}` : ''}`);
      setSlowQueries(r.slow_queries || []);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, [dbFilter]);

  const loadTop = useCallback(async () => {
    if (!requireAuth()) return;
    try {
      const params = new URLSearchParams();
      if (dbFilter !== 'all') params.set('database', dbFilter);
      const qs = params.toString();
      const r = await api<{ queries?: SlowQuery[] }>('GET', `/api/v1/database/queries/top${qs ? `?${qs}` : ''}`);
      setTopQueries(r.queries || []);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, [dbFilter]);

  const loadPools = useCallback(async () => {
    if (!requireAuth()) return;
    try {
      const r = await api<{ pools?: ConnectionPool[] }>('GET', '/api/v1/database/connection-pool');
      setPools(r.pools || []);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, []);

  useEffect(() => {
    setError('');
    if (tab === 'slow') void loadSlow();
    else if (tab === 'top') void loadTop();
    else void loadPools();
  }, [tab, loadSlow, loadTop, loadPools]);

  const handleSelect = useCallback(async (q: SlowQuery) => {
    setExplainLoading(true);
    try {
      const r = await api<Explain>('GET', `/api/v1/database/query-explain?query_hash=${encodeURIComponent(q.query_hash)}`);
      setExplain(r);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    } finally {
      setExplainLoading(false);
    }
  }, []);

  const totalExec = slowQueries.reduce((sum, q) => sum + (q.total_count || 0), 0);
  const weightedMs = slowQueries.reduce((sum, q) => sum + (q.avg_duration_ms || 0) * (q.total_count || 0), 0);
  const avgMs = totalExec > 0 ? Math.round(weightedMs / totalExec) : 0;
  const slowCount = slowQueries.filter((q) => q.avg_duration_ms >= 1000).length;

  const tabSummary = tab === 'slow'
    ? `${slowQueries.length} slow quer${slowQueries.length === 1 ? 'y' : 'ies'}`
    : tab === 'top'
      ? `${topQueries.length} frequent quer${topQueries.length === 1 ? 'y' : 'ies'}`
      : `${pools.length} pool${pools.length === 1 ? '' : 's'}`;

  return (
        <motion.div className="dash-page" initial="hidden" animate="show" variants={pageEnter}>
          {error ? <div className="dash-error" role="alert">{error}</div> : null}

          <motion.div
            className="dash-metric-strip"
            initial="hidden"
            animate="show"
            variants={kpiStagger}
          >
            <KpiCard
              label="Queries tracked"
              value={totalExec}
              status={totalExec > 0 ? 'up' : 'neutral'}
              accent="cyan"
            />
            <KpiCard
              label="Avg duration"
              value={avgMs < 1000 ? `${avgMs}ms` : `${(avgMs / 1000).toFixed(2)}s`}
              status={avgMs === 0 ? 'neutral' : avgMs >= 1000 ? 'crit' : avgMs >= 100 ? 'stale' : 'up'}
              accent={avgMs >= 1000 ? 'red' : avgMs >= 100 ? 'amber' : 'green'}
            />
            <KpiCard
              label="Slow (≥1s avg)"
              value={slowCount}
              status={slowCount > 0 ? 'crit' : 'up'}
              accent={slowCount > 0 ? 'red' : 'green'}
            />
          </motion.div>

          <div className="synth-filter-row">
            <label>
              <span>Database</span>
              <select
                value={dbFilter}
                onChange={(e) => setDbFilter(e.target.value as DbFilter)}
              >
                <option value="all">All</option>
                <option value="postgres">Postgres</option>
                <option value="mysql">MySQL</option>
                <option value="mariadb">MariaDB</option>
                <option value="mongodb">MongoDB</option>
              </select>
            </label>
          </div>

          {tab === 'slow' ? (
            <section className="dash-section">
              <span className="dash-eyebrow">Slow Queries</span>
              <h2 className="dash-section-title">Aggregated by query hash (avg desc)</h2>
              {slowQueries.length === 0 ? (
                <EmptyState
                  illustration={<span style={{ fontSize: 36 }}>⊕</span>}
                  headline="No slow queries yet"
                  subhead="Query stats appear here when the agent POSTs to /api/v1/database/query-stats. Each row aggregates all calls sharing the same query_hash."
                />
              ) : (
                <SlowQueryTable queries={slowQueries} onSelect={handleSelect} />
              )}
              {explainLoading ? (
                <p className="slow-query-explain-loading">Loading explain…</p>
              ) : null}
              {explain ? (
                <div className="slow-query-explain-modal" role="dialog" aria-modal="true">
                  <div className="slow-query-explain-modal-head">
                    <strong>Explain · {explain.database} · {explain.query_hash.slice(0, 12)}</strong>
                    <button
                      type="button"
                      className="slow-query-explain-close"
                      onClick={() => setExplain(null)}
                    >
                      ✕
                    </button>
                  </div>
                  <pre className="slow-query-explain-text">{explain.query_text}</pre>
                  <div className="slow-query-explain-stats">
                    <span><small>Executions</small><strong>{explain.total_executions}</strong></span>
                    <span><small>Avg duration</small><strong>{explain.avg_duration_ms}ms</strong></span>
                  </div>
                  <p className="slow-query-explain-suggestion">{explain.suggestion}</p>
                  <small className="slow-query-explain-note">{explain.note}</small>
                </div>
              ) : null}
            </section>
          ) : null}

          {tab === 'pools' ? (
            <section className="dash-section">
              <span className="dash-eyebrow">Connection Pools</span>
              <h2 className="dash-section-title">Latest pool snapshot per (db, host, port)</h2>
              {pools.length === 0 ? (
                <EmptyState
                  illustration={<span style={{ fontSize: 36 }}>≋</span>}
                  headline="No pool snapshots yet"
                  subhead="Connection pool snapshots stream here when the agent POSTs to /api/v1/database/connection-pool (one row per check)."
                />
              ) : (
                <div className="pool-card-list">
                  {pools.map((p) => {
                    const utilization = p.pool_size > 0 ? Math.round((p.active / p.pool_size) * 100) : 0;
                    return (
                      <article key={p.id} className="pool-card">
                        <div className="pool-card-head">
                          <StatusPill status="unknown" label={p.database} size="sm" />
                          <strong className="pool-card-host"><code>{p.host}:{p.port}</code></strong>
                        </div>
                        <div className="pool-card-metrics">
                          <div>
                            <small>Pool size</small>
                            <strong>{p.pool_size}</strong>
                          </div>
                          <div>
                            <small>Active</small>
                            <strong>{p.active}</strong>
                          </div>
                          <div>
                            <small>Idle</small>
                            <strong>{p.idle}</strong>
                          </div>
                          <div>
                            <small>Waiting</small>
                            <strong className={p.waiting > 0 ? 'pool-card-warn' : ''}>{p.waiting}</strong>
                          </div>
                        </div>
                        <div className="pool-card-bar" title={`${utilization}% utilized`}>
                          <div className="pool-card-bar-fill" style={{ width: `${Math.min(100, utilization)}%` }} />
                        </div>
                        <small className="pool-card-checked">checked {new Date(p.checked_at).toLocaleTimeString()}</small>
                      </article>
                    );
                  })}
                </div>
              )}
            </section>
          ) : null}

          {tab === 'top' ? (
            <section className="dash-section">
              <span className="dash-eyebrow">Top Queries</span>
              <h2 className="dash-section-title">Most-frequent queries (count desc)</h2>
              {topQueries.length === 0 ? (
                <EmptyState
                  illustration={<span style={{ fontSize: 36 }}>∑</span>}
                  headline="No frequent queries yet"
                  subhead="The top-queries view ranks by total_count. Query stats appear here once the agent POSTs to /api/v1/database/query-stats."
                />
              ) : (
                <SlowQueryTable queries={topQueries} />
              )}
            </section>
          ) : null}
        </motion.div>
  );
}