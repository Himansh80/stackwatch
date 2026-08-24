import { useCallback, useEffect, useMemo, useState } from 'react';
import { ApiError, api, getToken } from '../lib/api';
import { useLogout } from '../lib/useLogout';
import AppSidebar from '../components/AppSidebar';
import EmptyState from '../components/shared/EmptyState';
import IncidentCard, { Incident } from '../components/shared/IncidentCard';
import KpiCard from '../components/shared/KpiCard';
import { motion, buttonSpring, kpiStagger, pageEnter, useReducedMotion } from '../lib/motion';

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
  const logout = useLogout();
  const reduce = useReducedMotion();
  const [tab, setTab] = useState<Tab>('open');
  const [error, setError] = useState('');
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [sevFilter, setSevFilter] = useState<SeverityFilter>('all');

  const [creating, setCreating] = useState(false);
  const [createTitle, setCreateTitle] = useState('');
  const [createDesc, setCreateDesc] = useState('');
  const [createSev, setCreateSev] = useState<CreateSeverity>('sev3');
  const [createBusy, setCreateBusy] = useState(false);

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
      setIncidents(r.incidents || []);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, [tab, sevFilter]);

  useEffect(() => {
    setError('');
    void loadIncidents();
  }, [loadIncidents]);

  const declare = useCallback(async () => {
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

  const tabSummary = `${incidents.length} incident${incidents.length === 1 ? '' : 's'}`;

  return (
    <div className="dash-app">
      <AppSidebar
        active="incidents"
        onLogout={logout}
        show={['dashboard', 'billing', 'profile', 'settings', 'proxmox', 'truenas', 'incidents']}
      />
      <main className="dash-main">
        <header className="dash-topbar">
          <div className="dash-greeting">
            <span className="dash-greeting-eyebrow">Operations</span>
            <div className="dash-greeting-row">
              <strong className="dash-greeting-text">Incidents</strong>
              <span className="dash-greeting-clock">
                <span className="dash-greeting-clock-time">{tabSummary}</span>
              </span>
            </div>
          </div>
          <div className="dash-top-actions">
            <motion.button
              type="button"
              className="empty-state-cta"
              onClick={() => setCreating(true)}
              whileHover={reduce ? undefined : buttonSpring.whileHover}
              whileTap={reduce ? undefined : buttonSpring.whileTap}
              transition={buttonSpring.transition}
            >
              + Declare incident
            </motion.button>
          </div>
        </header>

        <div className="logs-tabs" role="tablist">
          {(['open', 'acknowledged', 'resolved', 'all'] as Tab[]).map((t) => (
            <button
              key={t}
              role="tab"
              type="button"
              aria-selected={tab === t}
              className={`logs-tab ${tab === t ? 'logs-tab-active' : ''}`}
              onClick={() => setTab(t)}
            >
              {t === 'all' ? 'All' : t.charAt(0).toUpperCase() + t.slice(1)}
            </button>
          ))}
        </div>

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
            />
            <KpiCard
              label="Sev1 (all-time view)"
              value={sev1KpiValue}
              status={sev1KpiValue > 0 ? 'crit' : 'up'}
              accent={sev1KpiValue > 0 ? 'red' : 'green'}
            />
            <KpiCard
              label="MTTR"
              value={mttrLabel}
              status={counts.mttrMs === 0 ? 'neutral' : counts.mttrMs > 4 * 60 * 60 * 1000 ? 'crit' : 'up'}
              accent={counts.mttrMs === 0 ? 'indigo' : counts.mttrMs > 4 * 60 * 60 * 1000 ? 'red' : 'green'}
            />
          </motion.div>

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

        {creating ? (
          <div className="slow-query-explain-modal" role="dialog" aria-modal="true">
            <div className="slow-query-explain-modal-head">
              <strong>Declare incident</strong>
              <button
                type="button"
                className="slow-query-explain-close"
                onClick={() => setCreating(false)}
                aria-label="Close"
              >
                ✕
              </button>
            </div>
            <form
              className="incident-create-form"
              onSubmit={(e) => {
                e.preventDefault();
                void declare();
              }}
            >
              <label>
                <span>Title</span>
                <input
                  type="text"
                  required
                  maxLength={256}
                  value={createTitle}
                  onChange={(e) => setCreateTitle(e.target.value)}
                  placeholder="Database primary is unreachable"
                />
              </label>
              <label>
                <span>Description</span>
                <textarea
                  rows={4}
                  maxLength={8192}
                  value={createDesc}
                  onChange={(e) => setCreateDesc(e.target.value)}
                  placeholder="Optional context for the war room."
                />
              </label>
              <label>
                <span>Severity</span>
                <select
                  value={createSev}
                  onChange={(e) => setCreateSev(e.target.value as CreateSeverity)}
                >
                  <option value="sev1">SEV1 — Critical</option>
                  <option value="sev2">SEV2 — High</option>
                  <option value="sev3">SEV3 — Medium</option>
                  <option value="sev4">SEV4 — Low</option>
                </select>
              </label>
              <div className="incident-create-actions">
                <button
                  type="button"
                  className="dash-icon-button"
                  onClick={() => setCreating(false)}
                  disabled={createBusy}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="empty-state-cta"
                  disabled={createBusy || !createTitle.trim()}
                >
                  {createBusy ? 'Declaring…' : 'Declare'}
                </button>
              </div>
            </form>
          </div>
        ) : null}
      </main>
    </div>
  );
}
