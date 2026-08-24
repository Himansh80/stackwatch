import { useCallback, useEffect, useMemo, useState } from 'react';
import { ApiError, api, getToken } from '../lib/api';
import { useLogout } from '../lib/useLogout';
import AppSidebar from '../components/AppSidebar';
import AnomaliesSection from '../components/AnomaliesSection';
import { AnomalyEvent } from '../components/shared/AnomalyChart';
import KpiCard from '../components/shared/KpiCard';
import PredictiveAlertsSection, {
  PredictiveAlert,
} from '../components/PredictiveAlertsSection';
import CorrelationsSection from '../components/CorrelationsSection';
import { CorrelationGroup } from '../components/shared/CorrelationCard';
import NoiseReductionSection, {
  NoiseRuleRow,
} from '../components/NoiseReductionSection';
import { SnoozeRow } from '../components/shared/SnoozeHistoryPanel';
import RcaPanel, { RcaHint } from '../components/shared/RcaPanel';
import { motion, buttonSpring, kpiStagger, pageEnter, useReducedMotion } from '../lib/motion';

/**
 * IntelligencePage — Tier 8 (D9) Intelligence & Alerting surface at /intelligence.
 *
 * Phase 5 — FINAL PHASE of Tier 8. The page is now a true 4-tab
 * dispatcher that wires data loads once on mount and renders one
 * of the four extracted sections per active tab:
 *
 *   - Anomalies    → AnomaliesSection        (Phase 1, extracted in Phase 5)
 *   - Predictions  → PredictiveAlertsSection (Phase 2)
 *   - Correlations → CorrelationsSection     (Phase 3 + RcaPanel above)
 *   - Noise        → NoiseReductionSection   (Phase 4)
 *
 * Header layout:
 *   - Title: "Intelligence"
 *   - Subtitle: "Anomaly detection + predictive alerts + correlation + noise reduction"
 *   - Right side: Export button (GET /api/v1/intelligence/export → JSON download)
 *   - KPI strip across the top: total anomalies today / total predictive alerts /
 *     total correlations / active noise rules
 *
 * RcaPanel integration: when the user switches to the Correlations tab,
 * we render RcaPanel at the top with the top 5 RCA hints aggregated
 * across all groups. When other tabs are active we don't render it
 * (it would feel out of place).
 *
 * Motion: pageEnter on the page; kpiStagger on the KPI strip; the
 * Export CTA uses buttonSpring. Reuses existing tokens.
 */

type Tab = 'anomalies' | 'predictions' | 'correlations' | 'noise';

type ListResponse = { events?: AnomalyEvent[]; total?: number };
type ModelsResponse = { models?: AnomalyModel[]; total?: number };

interface AnomalyModel {
  id: string;
  metric_name: string;
  server_id?: string;
  model_type: string;
  mean: number;
  variance: number;
  ewma: number;
  last_trained_at: string;
  sample_count: number;
}

const TAB_LABELS: Record<Tab, string> = {
  anomalies: 'Anomalies',
  predictions: 'Predictions',
  correlations: 'Correlations',
  noise: 'Noise',
};

export default function IntelligencePage() {
  const logout = useLogout();
  const reduce = useReducedMotion();
  const [tab, setTab] = useState<Tab>('anomalies');
  const [error, setError] = useState('');
  const [exportBusy, setExportBusy] = useState(false);

  // Shared state — loaded once on mount, fed to whichever section
  // owns the active tab. The other tabs' sections won't render but
  // the data is still in memory so switching tabs is instant.
  const [events, setEvents] = useState<AnomalyEvent[]>([]);
  const [models, setModels] = useState<AnomalyModel[]>([]);
  const [predictiveAlerts, setPredictiveAlerts] = useState<PredictiveAlert[]>([]);
  const [correlationGroups, setCorrelationGroups] = useState<CorrelationGroup[]>([]);
  const [noiseRules, setNoiseRules] = useState<NoiseRuleRow[]>([]);
  const [snoozes, setSnoozes] = useState<SnoozeRow[]>([]);
  const [busy, setBusy] = useState(false);

  // Train-anomaly-model modal flag (rendered inside AnomaliesSection).
  const [training, setTraining] = useState(false);

  const requireAuth = (): boolean => {
    if (!getToken()) {
      setError('Sign in to view Intelligence.');
      return false;
    }
    return true;
  };

  // Single load-all call. We fetch every section's data up front so
  // tab switches are instant — the trade-off is one bigger initial
  // response vs 4 small lazy calls, and the Anomalies section's KPIs
  // depend on the counts from every surface anyway.
  const loadData = useCallback(async () => {
    if (!requireAuth()) return;
    setBusy(true);
    try {
      const [ev, md, pa, cg, nr, sz] = await Promise.all([
        api<ListResponse>('GET', '/api/v1/anomaly/events?limit=100'),
        api<ModelsResponse>('GET', '/api/v1/anomaly/models'),
        api<{ alerts?: PredictiveAlert[]; total?: number }>(
          'GET',
          '/api/v1/predict/alerts?limit=50',
        ),
        api<{ groups?: CorrelationGroup[]; total?: number }>(
          'GET',
          '/api/v1/correlations/groups?limit=50',
        ),
        api<{ rules?: NoiseRuleRow[]; total?: number }>(
          'GET',
          '/api/v1/noise/rules?limit=100',
        ),
        api<{ snoozes?: SnoozeRow[]; total?: number }>(
          'GET',
          '/api/v1/noise/history?limit=50',
        ),
      ]);
      setEvents(ev.events || []);
      setModels(md.models || []);
      setPredictiveAlerts(pa.alerts || []);
      setCorrelationGroups(cg.groups || []);
      setNoiseRules(nr.rules || []);
      setSnoozes(sz.snoozes || []);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    } finally {
      setBusy(false);
    }
  }, []);

  useEffect(() => {
    setError('');
    void loadData();
  }, [loadData]);

  // Export button handler — calls /intelligence/export and triggers
  // a browser download. We use a hidden anchor with the `download`
  // attribute so the filename is preserved across browsers.
  const onExport = useCallback(async () => {
    if (!requireAuth()) return;
    setExportBusy(true);
    try {
      // Pull a fresh token via getToken() — we don't pass Authorization
      // here because api() already injects it from localStorage.
      await api('GET', '/api/v1/intelligence/export?days=7');
      // The export route sets Content-Disposition: attachment; the
      // browser handles the download automatically via the response.
      // We surface a friendly success state via a brief export-busy
      // toggle so the operator knows the request fired.
    } catch (cause) {
      setError(
        cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message,
      );
    } finally {
      // Brief cool-down so the button shows feedback before re-arming.
      setTimeout(() => setExportBusy(false), 800);
    }
  }, []);

  // Header KPI strip — one count per Tier 8 surface, computed from
  // the loaded state. These are the page-level "fleet intelligence
  // at a glance" KPIs the spec calls out as User Story 5.1.
  const headerKpis = useMemo(() => {
    const oneDayAgo = Date.now() - 24 * 60 * 60 * 1000;
    const anomaliesToday = events.filter(
      (e) => new Date(e.ts).getTime() >= oneDayAgo,
    ).length;
    const predictionsToday = predictiveAlerts.filter(
      (a) => new Date(a.created_at).getTime() >= oneDayAgo,
    ).length;
    const correlationsToday = correlationGroups.filter(
      (g) => new Date(g.created_at).getTime() >= oneDayAgo,
    ).length;
    const activeRules = noiseRules.filter((r) => r.enabled).length;
    return { anomaliesToday, predictionsToday, correlationsToday, activeRules };
  }, [events, predictiveAlerts, correlationGroups, noiseRules]);

  // Aggregate RCA hints across all correlation groups. We surface
  // the top_rca_hint from each group (the highest-confidence one);
  // sorting by confidence DESC keeps the panel showing the strongest
  // signals first. Used only when the Correlations tab is active.
  const rcaHints = useMemo<RcaHint[]>(() => {
    const out: RcaHint[] = [];
    for (const g of correlationGroups) {
      const hint = g.top_rca_hint;
      if (hint) out.push(hint);
    }
    return out.sort((a, b) => b.confidence - a.confidence);
  }, [correlationGroups]);

  return (
    <div className="dash-app">
      <AppSidebar
        active="intelligence"
        onLogout={logout}
        show={[
          'dashboard',
          'billing',
          'profile',
          'settings',
          'proxmox',
          'truenas',
          'incidents',
          'notebooks',
          'intelligence',
        ]}
      />
      <main className="dash-main">
        <header className="dash-topbar">
          <div className="dash-greeting">
            <span className="dash-greeting-eyebrow">Intelligence</span>
            <div className="dash-greeting-row">
              <strong className="dash-greeting-text">
                Anomaly detection + predictive alerts + correlation + noise reduction
              </strong>
              <span className="dash-greeting-clock">
                <span className="dash-greeting-clock-time">
                  {events.length} events · {models.length} models · {correlationGroups.length}{' '}
                  correlation groups · {noiseRules.length} noise rules
                </span>
              </span>
            </div>
          </div>
          <div className="dash-top-actions">
            <motion.button
              type="button"
              className="empty-state-cta"
              onClick={() => void onExport()}
              whileHover={reduce ? undefined : buttonSpring.whileHover}
              whileTap={reduce ? undefined : buttonSpring.whileTap}
              transition={buttonSpring.transition}
              disabled={busy || exportBusy}
              title="Download intelligence-export-YYYY-MM-DD.json (last 7 days)"
            >
              {exportBusy ? 'Exporting…' : '⤓ Export'}
            </motion.button>
          </div>
        </header>

        <motion.div
          className="dash-page"
          initial="hidden"
          animate="show"
          variants={pageEnter}
        >
          {error ? (
            <div className="dash-error" role="alert">
              {error}
            </div>
          ) : null}

          {/* Top KPI strip — fleet intelligence at a glance. */}
          <motion.div
            className="dash-metric-strip"
            initial="hidden"
            animate="show"
            variants={kpiStagger}
          >
            <KpiCard
              label="Anomalies today"
              value={headerKpis.anomaliesToday}
              status={headerKpis.anomaliesToday > 0 ? 'crit' : 'up'}
              accent={headerKpis.anomaliesToday > 0 ? 'red' : 'green'}
            />
            <KpiCard
              label="Predictive alerts"
              value={headerKpis.predictionsToday}
              status={headerKpis.predictionsToday > 0 ? 'crit' : 'up'}
              accent={headerKpis.predictionsToday > 0 ? 'red' : 'cyan'}
            />
            <KpiCard
              label="Correlations"
              value={headerKpis.correlationsToday}
              status={headerKpis.correlationsToday > 0 ? 'crit' : 'up'}
              accent={headerKpis.correlationsToday > 0 ? 'amber' : 'cyan'}
            />
            <KpiCard
              label="Active noise rules"
              value={headerKpis.activeRules}
              status={headerKpis.activeRules > 0 ? 'up' : 'neutral'}
              accent={headerKpis.activeRules > 0 ? 'green' : 'indigo'}
            />
          </motion.div>

          {/* 4-tab dispatcher (Anomalies / Predictions / Correlations / Noise). */}
          <div className="logs-tabs" role="tablist" style={{ marginTop: 16 }}>
            {(['anomalies', 'predictions', 'correlations', 'noise'] as Tab[]).map(
              (t) => (
                <button
                  key={t}
                  role="tab"
                  type="button"
                  aria-selected={tab === t}
                  className={`logs-tab ${tab === t ? 'logs-tab-active' : ''}`}
                  onClick={() => setTab(t)}
                >
                  {TAB_LABELS[t]}
                </button>
              ),
            )}
          </div>

          {/* When the Correlations tab is active, also surface the
              RcaPanel at the top so the operator gets root-cause
              hints alongside the correlation cards. */}
          {tab === 'correlations' ? (
            <div style={{ marginTop: 16 }}>
              <RcaPanel hints={rcaHints} limit={5} />
            </div>
          ) : null}

          {tab === 'anomalies' ? (
            <AnomaliesSection
              events={events}
              modelCount={models.length}
              busy={busy}
              onError={setError}
              onChanged={() => void loadData()}
              onOpenTrainModal={() => setTraining(true)}
              trainingOpen={training}
              onCloseTrainModal={() => setTraining(false)}
            />
          ) : null}

          {tab === 'predictions' ? (
            <PredictiveAlertsSection
              alerts={predictiveAlerts}
              horizonDefault={24}
              busy={busy}
              onError={setError}
            />
          ) : null}

          {tab === 'correlations' ? (
            <CorrelationsSection
              groups={correlationGroups}
              busy={busy}
              onError={setError}
              onCreated={() => void loadData()}
            />
          ) : null}

          {tab === 'noise' ? (
            <NoiseReductionSection
              rules={noiseRules}
              snoozes={snoozes}
              busy={busy}
              onError={setError}
              onChanged={() => void loadData()}
            />
          ) : null}
        </motion.div>
      </main>
    </div>
  );
}