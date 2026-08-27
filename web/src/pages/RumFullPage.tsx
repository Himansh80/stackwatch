import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ApiError, api, getToken } from '../lib/api';
import KpiCard from '../components/shared/KpiCard';
import EmptyState from '../components/shared/EmptyState';
import StatusPill from '../components/shared/StatusPill';
import ErrorGroupCard, { ErrorGroup } from '../components/shared/ErrorGroupCard';
import { motion, kpiStagger, pageEnter } from '../lib/motion';

/**
 * RumFullPage — RUM overview at /rum.
 *
 * Top: KPI strip (active sessions / total sessions / errors / p75 LCP)
 * Middle: sessions list table (sortable by started_at DESC, last seen,
 *         errors count, page views)
 * Bottom: error groups table (sortable by occurrence_count)
 *
 * Sidebar nav item: "RUM" → /rum (set in AppSidebar).
 *
 * Motion: pageEnter on the page, kpiStagger on the KPI strip.
 * Honors `prefers-reduced-motion` via the shared useReducedMotion
 * primitives (pageEnter + kpiEnter are no-ops on reduce).
 */

interface RUMSession {
  session_id: string;
  url: string;
  started_at: string;
  last_seen: string;
  page_views: number;
  errors: number;
  web_vitals: number;
  resources: number;
  interactions: number;
  long_tasks: number;
}

type SessionSort = 'last_seen' | 'errors' | 'page_views' | 'started_at';

export default function RumFullPage() {
  const nav = useNavigate();
  const [sessions, setSessions] = useState<RUMSession[]>([]);
  const [errorGroups, setErrorGroups] = useState<ErrorGroup[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [sessionSort, setSessionSort] = useState<SessionSort>('last_seen');
  const [window, _setWindow] = useState<'1h' | '24h' | '7d'>('24h');

  const load = useCallback(async () => {
    if (!getToken()) {
      setError('Sign in to view RUM.');
      setLoading(false);
      return;
    }
    setError('');
    setLoading(true);
    try {
      const [sessionsRes, groupsRes] = await Promise.all([
        api<{ sessions?: RUMSession[] }>('GET', `/api/v1/rum/sessions?time_range=${window}&limit=50`),
        api<{ groups?: ErrorGroup[] }>('GET', '/api/v1/rum/error-groups?limit=20'),
      ]);
      setSessions(sessionsRes.sessions || []);
      setErrorGroups(groupsRes.groups || []);
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message || 'Unable to load RUM data.';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, [window]);

  useEffect(() => { void load(); }, [load]);

  const sortedSessions = useMemo(() => {
    const arr = [...sessions];
    arr.sort((a, b) => {
      switch (sessionSort) {
        case 'errors':       return (b.errors || 0) - (a.errors || 0);
        case 'page_views':   return (b.page_views || 0) - (a.page_views || 0);
        case 'started_at':   return (b.started_at || '').localeCompare(a.started_at || '');
        case 'last_seen':
        default:             return (b.last_seen || '').localeCompare(a.last_seen || '');
      }
    });
    return arr;
  }, [sessions, sessionSort]);

  // KPIs from the in-memory data. Active = at least one event in the
  // last 5 minutes; p75 LCP is approximated from the count of slow
  // resources (we don't have web-vitals rollups in Phase 3).
  const kpis = useMemo(() => {
    const now = Date.now();
    const recent = sessions.filter((s) => {
      const t = Date.parse(s.last_seen);
      return Number.isFinite(t) && now - t < 5 * 60_000;
    }).length;
    const totalErrors = errorGroups.reduce((acc, g) => acc + (g.occurrence_count || 0), 0);
    return {
      active: recent,
      total: sessions.length,
      errors: totalErrors,
      lcp: sessions.length === 0 ? '—' : '—', // reserved for Phase 4 web-vitals rollup
    };
  }, [sessions, errorGroups]);

  return (
        <motion.div className="dash-page" initial="hidden" animate="show" variants={pageEnter}>
          <section className="dash-section">
            <span className="dash-eyebrow">Live</span>
            <h2 className="dash-section-title">Real-user observability</h2>
            <motion.div className="dash-kpi-strip" variants={kpiStagger} initial="hidden" animate="show">
              <KpiCard label="Active (5m)" value={kpis.active} accent="green" />
              <KpiCard label="Sessions" value={kpis.total} accent="indigo" />
              <KpiCard label="Errors" value={kpis.errors} accent="red" />
              <KpiCard label="p75 LCP" value={kpis.lcp} accent="amber" />
            </motion.div>
          </section>

          {error ? (
            <div className="dash-error" role="alert">{error}</div>
          ) : null}

          <section className="dash-section">
            <span className="dash-eyebrow">Sessions</span>
            <h2 className="dash-section-title">Recent browser sessions</h2>
            {loading ? (
              <p className="dash-section-lede">Loading sessions…</p>
            ) : sortedSessions.length === 0 ? (
              <EmptyState
                illustration={<span style={{ fontSize: 36 }}>◎</span>}
                headline="No sessions captured yet"
                subhead={`No browser sessions in the last ${window}. Install the RUM snippet and POST events to /api/v1/rum/* to populate this dashboard.`}
              />
            ) : (
              <table className="sw-table">
                <thead>
                  <tr>
                    <th>Session</th>
                    <th>URL</th>
                    <th>
                      <button
                        type="button"
                        className="sw-button"
                        onClick={() => setSessionSort('last_seen')}
                      >
                        Last seen
                      </button>
                    </th>
                    <th>
                      <button
                        type="button"
                        className="sw-button"
                        onClick={() => setSessionSort('page_views')}
                      >
                        Views
                      </button>
                    </th>
                    <th>Resources</th>
                    <th>Vitals</th>
                    <th>
                      <button
                        type="button"
                        className="sw-button"
                        onClick={() => setSessionSort('errors')}
                      >
                        Errors
                      </button>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {sortedSessions.map((s) => (
                    <tr
                      key={s.session_id}
                      className="sw-row-clickable"
                      onClick={() => nav(`/rum/session?id=${encodeURIComponent(s.session_id)}`)}
                    >
                      <td><code>{s.session_id.slice(0, 12)}…</code></td>
                      <td title={s.url}>{s.url ? s.url.slice(0, 48) : '—'}</td>
                      <td>{s.last_seen ? new Date(s.last_seen).toLocaleString() : '—'}</td>
                      <td>{s.page_views}</td>
                      <td>{s.resources}</td>
                      <td>{s.web_vitals}</td>
                      <td>{s.errors > 0 ? <StatusPill status="crit" label={`×${s.errors}`} size="sm" /> : <StatusPill status="ok" label="0" size="sm" />}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </section>

          <section className="dash-section">
            <span className="dash-eyebrow">Errors</span>
            <h2 className="dash-section-title">Top error groups</h2>
            {errorGroups.length === 0 ? (
              <p className="dash-section-lede">No error groups yet — once JS errors come in they&apos;ll appear here, sorted by occurrence count.</p>
            ) : (
              <div className="rum-error-grid">
                {errorGroups.map((g) => (
                  <ErrorGroupCard key={g.id} group={g} />
                ))}
              </div>
            )}
          </section>
        </motion.div>
  );
}
