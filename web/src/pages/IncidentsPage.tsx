import { useCallback, useEffect, useMemo, useState } from 'react';
import { ApiError, api, getToken } from '../lib/api';
import EmptyState from '../components/shared/EmptyState';
import IncidentCard, { Incident } from '../components/shared/IncidentCard';
import KpiCard from '../components/shared/KpiCard';
import StatusPill from '../components/shared/StatusPill';
import TimeSeriesChart from '../components/shared/TimeSeriesChart';
import { motion, pageEnter, kpiStagger, useReducedMotion } from '../lib/motion';

type Tab = 'open' | 'acknowledged' | 'resolved' | 'all';
type SeverityFilter = 'all' | 'sev1' | 'sev2' | 'sev3' | 'sev4';
type CreateSeverity = 'sev1' | 'sev2' | 'sev3' | 'sev4';

/**
 * IncidentsPage — Tier 7.9 (D10) Service Management surface at /incidents.
 *
 * Layout:
 *   - Top: 3 KpiCards (open count / sev1 count / MTTR)
 *   - Middle: tabbed view (Open / Acknowledged / Resolved / All)
 *     - Each tab: list of IncidentCards + severity dropdown
 *   - "Declare incident" button at top-right (opens a form modal)
 *
 * Sidebar nav: "Incidents" added in AppSidebar under observability.
 *
 * Motion: pageEnter on the page; kpiStagger on the KPI strip so the
 * three cards enter with a 50ms gap. The Declare button uses the
 * shared buttonSpring. Reuses existing tokens + classes.
 */
export default function IncidentsPage() {
  const _reduce = useReducedMotion();
  const [tab, setTab] = useState<Tab>('open');
  const [error, setError] = useState('');
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [sevFilter, setSevFilter] = useState<SeverityFilter>('all');
  // Rolling history for sparklines (last 24 polls = ~12 min window).
  const [openHistory, setOpenHistory] = useState<number[]>([]);
  const [sev1History, setSev1History] = useState<number[]>([]);
  const [rateHistory, setRateHistory] = useState<number[]>([]);

  const [_creating, setCreating] = useState(false);
  const [createTitle, setCreateTitle] = useState('');
  const [createDesc, setCreateDesc] = useState('');
  const [createSev, setCreateSev] = useState<CreateSeverity>('sev3');
  const [_createBusy, setCreateBusy] = useState(false);

  const requireAuth = (): boolean => {
    if (!getToken()) {
      setError('Sign in to view Incidents.');
      return false;
    }
    return true;
  };

  const loadIncidents = useCallback(async () => {
    if (!requireAuth()) return;
    try {
      const params = new URLSearchParams();
      if (tab !== 'all') params.set('status', tab);
      if (sevFilter !== 'all') params.set('severity', sevFilter);
      const qs = params.toString();
      const r = await api<{ incidents?: Incident[] }>(
        'GET',
        `/api/v1/incidents${qs ? `?${qs}` : ''}`,
      );
      const rows = r.incidents || [];
      setIncidents(rows);
      // Update rolling history — only count "open" / "sev1" so the
      // trend isn't polluted by tab/filter selections.
      const append = (prev: number[], next: number, max = 24) => {
        const out = [...prev, next];
        return out.length > max ? out.slice(out.length - max) : out;
      };
      setRateHistory((prev) => append(prev, rows.length));
      if (tab === 'open') setOpenHistory((prev) => append(prev, rows.length));
      const sev1 = rows.filter((i) => (i.severity || '').toLowerCase() === 'sev1').length;
      setSev1History((prev) => append(prev, sev1));
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, [tab, sevFilter]);

  useEffect(() => {
    setError('');
    void loadIncidents();
  }, [loadIncidents]);

  const _declare = useCallback(async () => {
    if (!createTitle.trim()) {
      setError('Title is required to declare an incident.');
      return;
    }
    setCreateBusy(true);
    setError('');
    try {
      await api('POST', '/api/v1/incidents', {
        title: createTitle.trim(),
        description: createDesc.trim(),
        severity: createSev,
      });
      setCreating(false);
      setCreateTitle('');
      setCreateDesc('');
      setCreateSev('sev3');
      await loadIncidents();
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    } finally {
      setCreateBusy(false);
    }
  }, [createTitle, createDesc, createSev, loadIncidents]);

  // KPIs computed client-side from the loaded list. The list is
  // already filtered server-side by tab/status, so "open count"
  // is just incidents.length when tab === 'open'. For "all" we
  // derive open/acknowledged/resolved counts from the payload.
  const counts = useMemo(() => {
    let open = 0;
    let ack = 0;
    let resolved = 0;
    let sev1 = 0;
    const durations: number[] = [];
    for (const inc of incidents) {
      const status = (inc.status || '').toLowerCase();
      if (status === 'open') open += 1;
      else if (status === 'acknowledged') ack += 1;
      else if (status === 'resolved') {
        resolved += 1;
        if (inc.started_at && inc.resolved_at) {
          const d = new Date(inc.resolved_at).getTime() - new Date(inc.started_at).getTime();
          if (d > 0) durations.push(d);
        }
      }
      if ((inc.severity || '').toLowerCase() === 'sev1') sev1 += 1;
    }
    const mttrMs = durations.length === 0 ? 0 : durations.reduce((a, b) => a + b, 0) / durations.length;
    return { open, ack, resolved, sev1, mttrMs };
  }, [incidents]);

  const mttrLabel = useMemo(() => {
    if (counts.mttrMs === 0) return '—';
    const sec = Math.floor(counts.mttrMs / 1000);
    if (sec < 60) return `${sec}s`;
    const min = Math.floor(sec / 60);
    if (min < 60) return `${min}m`;
    const hr = Math.floor(min / 60);
    return `${hr}h ${min % 60}m`;
  }, [counts.mttrMs]);

  // When viewing "Open", the open count IS the list size.
  // Otherwise it's derived from the full payload (best-effort).
  const openKpiValue = tab === 'open' ? incidents.length : counts.open;
  const sev1KpiValue = tab === 'all' ? counts.sev1 : incidents.filter((i) => (i.severity || '').toLowerCase() === 'sev1').length;


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
              label="Open"
              value={openKpiValue}
              status={openKpiValue > 0 ? 'crit' : 'up'}
              accent={openKpiValue > 0 ? 'red' : 'green'}
              sparkline={openHistory}
            />
            <KpiCard
              label="Sev1 (all-time view)"
              value={sev1KpiValue}
              status={sev1KpiValue > 0 ? 'crit' : 'up'}
              accent={sev1KpiValue > 0 ? 'red' : 'green'}
              sparkline={sev1History}
            />
            <KpiCard
              label="MTTR"
              value={mttrLabel}
              status={counts.mttrMs === 0 ? 'neutral' : counts.mttrMs > 4 * 60 * 60 * 1000 ? 'crit' : 'up'}
              accent={counts.mttrMs === 0 ? 'indigo' : counts.mttrMs > 4 * 60 * 60 * 1000 ? 'red' : 'green'}
            />
          </motion.div>

          {rateHistory.length > 1 && (
            <section className="inc-trend">
              <article className="dash-panel">
                <div className="dash-panel-head">
                  <div><span className="dash-eyebrow">Trend</span><h3>Incident rate ({tab === 'all' ? 'all' : tab})</h3></div>
                  <span className="dash-panel-context">last {rateHistory.length} polls</span>
                </div>
                <TimeSeriesChart values={rateHistory} color="red" height={100} emptyMessage="Waiting for data…" />
              </article>
            </section>
          )}

          <div className="synth-filter-row inc-tabs">
            {(['open', 'acknowledged', 'resolved', 'all'] as Tab[]).map((t) => (
              <button
                key={t}
                type="button"
                className={`inc-tab ${tab === t ? 'active' : ''}`}
                onClick={() => setTab(t)}
              >
                <StatusPill
                  status={t === 'open' && openKpiValue > 0 ? 'crit' : t === 'acknowledged' ? 'warn' : t === 'resolved' ? 'ok' : 'unknown'}
                  label={t.charAt(0).toUpperCase() + t.slice(1)}
                  size="sm"
                />
                <span className="inc-tab-count">
                  {t === 'open' ? openKpiValue : t === 'acknowledged' ? counts.ack : t === 'resolved' ? counts.resolved : incidents.length}
                </span>
              </button>
            ))}
          </div>

          <section className="dash-section">
            <span className="dash-eyebrow">Incidents</span>
            <h2 className="dash-section-title">
              {tab === 'all' ? 'All incidents' : `${tab.charAt(0).toUpperCase()}${tab.slice(1)} incidents`}
            </h2>
            <div className="synth-filter-row">
              <label>
                <span>Severity</span>
                <select
                  value={sevFilter}
                  onChange={(e) => setSevFilter(e.target.value as SeverityFilter)}
                >
                  <option value="all">All</option>
                  <option value="sev1">SEV1</option>
                  <option value="sev2">SEV2</option>
                  <option value="sev3">SEV3</option>
                  <option value="sev4">SEV4</option>
                </select>
              </label>
            </div>
            {incidents.length === 0 ? (
              <EmptyState
                illustration={<span style={{ fontSize: 36 }}>⚑</span>}
                headline={tab === 'open' ? 'No open incidents' : 'No incidents match these filters'}
                subhead="Incidents appear here when declared via POST /api/v1/incidents. Use the Declare button above to start a new incident — a war room and tasks can be attached once one exists."
              />
            ) : (
              <div className="threat-card-list">
                {incidents.map((inc) => (
                  <IncidentCard key={inc.id} incident={inc} />
                ))}
              </div>
            )}
          </section>
        </motion.div>
  );
}
