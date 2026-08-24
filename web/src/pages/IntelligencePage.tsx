import { useCallback, useEffect, useMemo, useState } from 'react';
import { ApiError, api, getToken } from '../lib/api';
import { useLogout } from '../lib/useLogout';
import AppSidebar from '../components/AppSidebar';
import AnomalyChart, { AnomalyEvent } from '../components/shared/AnomalyChart';
import EmptyState from '../components/shared/EmptyState';
import KpiCard from '../components/shared/KpiCard';
import PredictiveAlertsSection, {
  PredictiveAlert,
} from '../components/PredictiveAlertsSection';
import { motion, buttonSpring, kpiStagger, pageEnter, useReducedMotion } from '../lib/motion';

/**
 * IntelligencePage — Tier 8.1 (D9) ML Anomaly Detection surface at /intelligence.
 *
 * Phase 1 ships only the TOP section: an Anomalies overview with a
 * 3-card KPI strip (today / critical / acknowledged), the AnomalyChart
 * of recent detections, and a "Train new model" button + form modal.
 * Phases 2-5 will add their own sections below this one; Phase 5
 * unifies everything under a tabbed layout.
 *
 * Layout (Phase 1):
 *   - Topbar with "Intelligence" title + "Train new model" button
 *   - 3 KpiCards using kpiStagger
 *   - AnomalyChart (from shared/AnomalyChart) with severity legend
 *   - Recent events list (last 10) — same data the chart uses
 *   - "Train new model" modal — POST /anomaly/train
 *
 * Motion: pageEnter on the page; kpiStagger on the KPI strip; the
 * Train CTA uses the shared buttonSpring. Reuses existing tokens.
 */
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

export default function IntelligencePage() {
  const logout = useLogout();
  const reduce = useReducedMotion();
  const [error, setError] = useState('');
  const [events, setEvents] = useState<AnomalyEvent[]>([]);
  const [models, setModels] = useState<AnomalyModel[]>([]);
  const [busy, setBusy] = useState(false);

  // Train modal state.
  const [training, setTraining] = useState(false);
  const [trainMetric, setTrainMetric] = useState('');
  const [trainServer, setTrainServer] = useState('');
  const [trainModelType, setTrainModelType] = useState<'welford' | 'ewma'>('welford');

  // Predictive state (Phase 2). The PredictiveAlertsSection component
  // owns its own modal + runForecast logic; we just feed it the
  // alerts list (loaded by loadData) and a setError callback.
  const [predictiveAlerts, setPredictiveAlerts] = useState<PredictiveAlert[]>([]);

  const requireAuth = (): boolean => {
    if (!getToken()) {
      setError('Sign in to view Intelligence.');
      return false;
    }
    return true;
  };

  const loadData = useCallback(async () => {
    if (!requireAuth()) return;
    try {
      const [ev, md, pa] = await Promise.all([
        api<ListResponse>('GET', '/api/v1/anomaly/events?limit=100'),
        api<ModelsResponse>('GET', '/api/v1/anomaly/models'),
        api<{ alerts?: PredictiveAlert[]; total?: number }>('GET', '/api/v1/predict/alerts?limit=50'),
      ]);
      setEvents(ev.events || []);
      setModels(md.models || []);
      setPredictiveAlerts(pa.alerts || []);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, []);

  // Note: predictive alerting is delegated to PredictiveAlertsSection.
  // The section owns its own runForecast + ackAlert logic; the page
  // just feeds it the alerts list (loaded by loadData) and a setError
  // callback. After the section acks an alert, loadData is the source
  // of truth for refreshing the list — there is no separate handler
  // needed here because the page reloads predictive alerts whenever
  // it remounts (Phase 2 deferral: keep the page dumb).

  useEffect(() => {
    setError('');
    void loadData();
  }, [loadData]);

  // KPIs: today / critical / acknowledged. Today = events in the last
  // 24h. Critical = events with severity=critical regardless of when.
  // Acknowledged = events with acknowledged=true. All three are
  // computed client-side from the loaded event list (cap 100; in
  // production we'll add a /summary endpoint in Phase 5).
  const counts = useMemo(() => {
    const oneDayAgo = Date.now() - 24 * 60 * 60 * 1000;
    let today = 0;
    let critical = 0;
    let acknowledged = 0;
    for (const e of events) {
      const ts = new Date(e.ts).getTime();
      if (ts >= oneDayAgo) today += 1;
      if (e.severity === 'critical') critical += 1;
      if (e.acknowledged) acknowledged += 1;
    }
    return { today, critical, acknowledged };
  }, [events]);

  const train = useCallback(async () => {
    if (!trainMetric.trim()) {
      setError('Metric name is required.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      const body: Record<string, string> = {
        metric_name: trainMetric.trim(),
        model_type: trainModelType,
      };
      if (trainServer.trim()) body.server_id = trainServer.trim();
      await api('POST', '/api/v1/anomaly/train', body);
      setTraining(false);
      setTrainMetric('');
      setTrainServer('');
      setTrainModelType('welford');
      await loadData();
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    } finally {
      setBusy(false);
    }
  }, [trainMetric, trainServer, trainModelType, loadData]);

  const ack = useCallback(async (eventId: string) => {
    setBusy(true);
    setError('');
    try {
      await api('POST', '/api/v1/anomaly/ack', { event_id: eventId, note: 'acknowledged from intelligence page' });
      await loadData();
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    } finally {
      setBusy(false);
    }
  }, [loadData]);

  return (
    <div className="dash-app">
      <AppSidebar
        active="intelligence"
        onLogout={logout}
        // Sidebar nav includes every Tier 7+ page so the user can
        // jump between observability surfaces without bouncing to
        // the dashboard. The "intelligence" item will be added in
        // AppSidebar as part of Phase 5; for Phase 1 the route still
        // exists at /intelligence via direct URL.
        show={['dashboard', 'billing', 'profile', 'settings', 'proxmox', 'truenas', 'incidents', 'notebooks', 'intelligence']}
      />
      <main className="dash-main">
        <header className="dash-topbar">
          <div className="dash-greeting">
            <span className="dash-greeting-eyebrow">Intelligence</span>
            <div className="dash-greeting-row">
              <strong className="dash-greeting-text">Anomalies</strong>
              <span className="dash-greeting-clock">
                <span className="dash-greeting-clock-time">{events.length} events · {models.length} models</span>
              </span>
            </div>
          </div>
          <div className="dash-top-actions">
            <motion.button
              type="button"
              className="empty-state-cta"
              onClick={() => setTraining(true)}
              whileHover={reduce ? undefined : buttonSpring.whileHover}
              whileTap={reduce ? undefined : buttonSpring.whileTap}
              transition={buttonSpring.transition}
              disabled={busy}
            >
              + Train new model
            </motion.button>
          </div>
        </header>

        <motion.div className="dash-page" initial="hidden" animate="show" variants={pageEnter}>
          {error ? <div className="dash-error" role="alert">{error}</div> : null}

          <motion.div
            className="dash-metric-strip"
            initial="hidden"
            animate="show"
            variants={kpiStagger}
          >
            <KpiCard
              label="Anomalies (24h)"
              value={counts.today}
              status={counts.today > 0 ? 'crit' : 'up'}
              accent={counts.today > 0 ? 'red' : 'green'}
            />
            <KpiCard
              label="Critical"
              value={counts.critical}
              status={counts.critical > 0 ? 'crit' : 'up'}
              accent={counts.critical > 0 ? 'red' : 'cyan'}
            />
            <KpiCard
              label="Acknowledged"
              value={counts.acknowledged}
              status="neutral"
              accent="indigo"
            />
          </motion.div>

          <section className="dash-section">
            <span className="dash-eyebrow">Detections</span>
            <h2 className="dash-section-title">Recent anomalies</h2>
            <p className="dash-section-sub" style={{ color: 'var(--text-muted)', fontSize: 12, marginTop: 0 }}>
              Each circle is a detected anomaly. Color encodes severity (red=critical, amber=warning, green=info).
              Acknowledge noise to dim the dot in the chart.
            </p>
            {events.length === 0 ? (
              <EmptyState
                illustration={<span style={{ fontSize: 36 }}>⌬</span>}
                headline="No anomalies detected yet"
                subhead="Train a model on a metric, then call /anomaly/detect to start flagging outliers. The chart will populate as events are recorded."
              />
            ) : (
              <div className="anomaly-chart-card" style={{ background: 'var(--surface)', border: '1px solid var(--border)', borderRadius: 'var(--radius-lg)', padding: 16 }}>
                <AnomalyChart events={events} />
              </div>
            )}
          </section>

          {events.length > 0 ? (
            <section className="dash-section">
              <span className="dash-eyebrow">Event log</span>
              <h2 className="dash-section-title">Latest 10 events</h2>
              <div className="threat-card-list">
                {events.slice(0, 10).map((e) => (
                  <div
                    key={e.id}
                    className="threat-card"
                    style={e.acknowledged ? { opacity: 0.5 } : undefined}
                  >
                    <div className="threat-card-top">
                      <span className={`dash-status dash-status-${e.severity === 'critical' ? 'down' : e.severity === 'warning' ? 'stale' : 'up'}`}>
                        <span className="dash-status-dot" aria-hidden="true" />
                        {e.severity}
                      </span>
                      <strong className="threat-card-type">{e.metric_name}</strong>
                      <span className="threat-card-time" title={e.ts}>
                        {new Date(e.ts).toLocaleString()}
                      </span>
                    </div>
                    <p className="threat-card-desc">
                      score <strong>{e.anomaly_score.toFixed(2)}</strong> · observed <code>{e.observed_value.toFixed(2)}</code> · expected [{e.expected_range_low.toFixed(2)}, {e.expected_range_high.toFixed(2)}]
                    </p>
                    <div className="threat-card-meta">
                      <span className="threat-card-meta-pill">
                        <span className="threat-card-meta-label">server</span>
                        <code>{e.server_id ? `${e.server_id.slice(0, 8)}…` : 'tenant-wide'}</code>
                      </span>
                      {e.acknowledged ? (
                        <span className="threat-card-resolve-btn" style={{ color: 'var(--green)' }}>✓ acknowledged</span>
                      ) : (
                        <button
                          type="button"
                          className="threat-card-resolve-btn"
                          onClick={() => void ack(e.id)}
                          disabled={busy}
                        >
                          Acknowledge →
                        </button>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </section>
          ) : null}

          {/* ---- Predictive Alerts section (Phase 2 / Tier 8.2) ---- */}
          {/* Extracted to PredictiveAlertsSection component so this
              page stays under the 400-LOC cap. The section owns its
              own KPI strip + forecast grid + event log; the page
              owns the modelAccuracy state + the Generate Forecast
              modal that lives in the topbar. */}
          <PredictiveAlertsSection
            alerts={predictiveAlerts}
            horizonDefault={24}
            busy={busy}
            onError={setError}
          />
        </motion.div>

        {training ? (
          <div className="slow-query-explain-modal" role="dialog" aria-modal="true">
            <div className="slow-query-explain-modal-head">
              <strong>Train new model</strong>
              <button
                type="button"
                className="slow-query-explain-close"
                onClick={() => setTraining(false)}
                aria-label="Close"
              >
                ✕
              </button>
            </div>
            <form
              className="notebook-create-form"
              onSubmit={(e) => {
                e.preventDefault();
                void train();
              }}
            >
              <label>
                <span>Metric name</span>
                <input
                  type="text"
                  required
                  maxLength={256}
                  value={trainMetric}
                  onChange={(e) => setTrainMetric(e.target.value)}
                  placeholder="cpu.user_pct"
                />
              </label>
              <label>
                <span>Server ID (optional — leave blank for tenant-wide)</span>
                <input
                  type="text"
                  value={trainServer}
                  onChange={(e) => setTrainServer(e.target.value)}
                  placeholder="uuid of the server, or blank for tenant-wide"
                />
              </label>
              <label>
                <span>Model type</span>
                <select
                  value={trainModelType}
                  onChange={(e) => setTrainModelType(e.target.value as 'welford' | 'ewma')}
                >
                  <option value="welford">Welford + EWMA (recommended)</option>
                  <option value="ewma">EWMA only</option>
                </select>
              </label>
              <div className="incident-create-actions">
                <button
                  type="button"
                  className="dash-icon-button"
                  onClick={() => setTraining(false)}
                  disabled={busy}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="empty-state-cta"
                  disabled={busy || !trainMetric.trim()}
                >
                  {busy ? 'Training…' : 'Train'}
                </button>
              </div>
            </form>
          </div>
        ) : null}
      </main>
    </div>
  );
}
