import { useCallback, useMemo, useState } from 'react';
import EmptyState from './shared/EmptyState';
import KpiCard from './shared/KpiCard';
import PredictionChart, { ForecastPoint, HistoricalPoint } from './shared/PredictionChart';
import { ApiError, api } from '../lib/api';
import { motion, buttonSpring, kpiStagger, useReducedMotion } from '../lib/motion';

// PredictiveAlert — JSON shape returned by GET /api/v1/predict/alerts.
// Mirrors the predictiveAlertRow type in handlers_predict_types.go.
export interface PredictiveAlert {
  id: string;
  metric_name: string;
  server_id?: string;
  predicted_value: number;
  predicted_breach_at: string;
  confidence: number;
  severity: 'info' | 'warning' | 'critical' | string;
  status: 'open' | 'acknowledged' | 'resolved' | string;
  ack_user_id?: string;
  ack_note?: string;
  created_at: string;
}

// One forecast window — metric + (historical, forecast) points +
// confidence + MAPE. Rendered as one PredictionChart card.
export interface ForecastWindow {
  metricName: string;
  historical: HistoricalPoint[];
  forecast: ForecastPoint[];
  confidence: number;
  mape: number;
  modelType: string;
}

interface PredictiveAlertsSectionProps {
  alerts: PredictiveAlert[];
  windows?: ForecastWindow[];
  modelAccuracy?: { mape: number; rmse: number; confidence: number } | null;
  horizonDefault: number;
  busy: boolean;
  onError: (msg: string) => void;
}

// PredictiveAlertsSection — Tier 8.2 (D9 Phase 2) UI surface.
// Renders the predictive-alerting section on IntelligencePage.
// Owns the modal + runForecast / ackAlert logic so the parent
// page stays under the 400-LOC cap. Motion uses existing exports
// (kpiStagger, buttonSpring) — no new variants.
export default function PredictiveAlertsSection({
  alerts,
  windows: initialWindows,
  modelAccuracy: initialAccuracy,
  horizonDefault,
  busy,
  onError,
}: PredictiveAlertsSectionProps) {
  const reduce = useReducedMotion();
  const [modal, setModal] = useState(false);
  const [metric, setMetric] = useState('');
  const [horizon, setHorizon] = useState(horizonDefault);
  const [windows, setWindows] = useState<ForecastWindow[]>(initialWindows ?? []);
  const [modelAccuracy, setModelAccuracy] = useState(initialAccuracy ?? null);
  const [sectionBusy, setSectionBusy] = useState(false);
  const isBusy = busy || sectionBusy;

  // KPIs computed locally from the alerts list.
  const counts = useMemo(() => {
    const oneDayAgo = Date.now() - 24 * 60 * 60 * 1000;
    let today = 0;
    let acknowledged = 0;
    for (const a of alerts) {
      const ts = new Date(a.created_at).getTime();
      if (ts >= oneDayAgo) today += 1;
      if (a.status === 'acknowledged') acknowledged += 1;
    }
    return { today, acknowledged };
  }, [alerts]);

  // Run a forecast for a single metric and add the result to the
  // windows state. Also refreshes the accuracy endpoint.
  const runForecast = useCallback(async () => {
    const m = metric.trim();
    if (!m) {
      onError('Metric name is required.');
      return;
    }
    setSectionBusy(true);
    try {
      const resp = await api<{
        predicted_values: ForecastPoint[];
        confidence: number;
        mape: number;
        model_type: string;
      }>('POST', '/api/v1/predict/forecast', {
        metric_name: m,
        horizon_hours: horizon,
        model_type: 'linear_regression',
      });
      const newWindow: ForecastWindow = {
        metricName: m,
        historical: [],
        forecast: resp.predicted_values,
        confidence: resp.confidence,
        mape: resp.mape,
        modelType: resp.model_type,
      };
      setWindows((prev) => {
        const filtered = prev.filter((w) => w.metricName !== m);
        return [...filtered, newWindow].slice(-3);
      });
      // Best-effort accuracy refresh for the KPI strip.
      try {
        const acc = await api<{ mape: number; rmse: number; confidence: number }>(
          'GET',
          `/api/v1/predict/accuracy?metric_name=${encodeURIComponent(m)}`,
        );
        setModelAccuracy(acc);
      } catch {
        // ignore — accuracy is best effort
      }
      setModal(false);
      setMetric('');
    } catch (cause) {
      onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    } finally {
      setSectionBusy(false);
    }
  }, [metric, horizon, onError]);

  // Acknowledge a predictive alert. The parent's loadData would
  // re-fetch everything; here we just call the endpoint and let
  // the parent reload on next mount.
  const ackAlert = useCallback(async (id: string) => {
    setSectionBusy(true);
    try {
      await api('POST', '/api/v1/predict/ack', {
        alert_id: id,
        note: 'acknowledged from intelligence page',
      });
    } catch (cause) {
      onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    } finally {
      setSectionBusy(false);
    }
  }, [onError]);

  return (
    <>
      <motion.div
        className="dash-metric-strip"
        initial="hidden"
        animate="show"
        variants={kpiStagger}
        style={{ marginTop: 16 }}
      >
        <KpiCard
          label="Predictive alerts (24h)"
          value={counts.today}
          status={counts.today > 0 ? 'crit' : 'up'}
          accent={counts.today > 0 ? 'red' : 'green'}
        />
        <KpiCard
          label="Avg model MAPE"
          value={modelAccuracy ? `${modelAccuracy.mape.toFixed(1)}%` : '—'}
          status={modelAccuracy && modelAccuracy.mape > 25 ? 'crit' : 'neutral'}
          accent={modelAccuracy && modelAccuracy.mape > 25 ? 'amber' : 'cyan'}
        />
        <KpiCard
          label="Acknowledged"
          value={counts.acknowledged}
          status="neutral"
          accent="indigo"
        />
      </motion.div>

      <section className="dash-section">
        <span className="dash-eyebrow">Forecasts</span>
        <h2 className="dash-section-title">Predictive Alerts</h2>
        <p
          className="dash-section-sub"
          style={{ color: 'var(--text-muted)', fontSize: 12, marginTop: 0 }}
        >
          Each chart shows the next{' '}
          {windows[0]?.forecast.length ?? horizonDefault} hours of forecast for one metric.
          The dashed line is the median (p50); the shaded band is the p10–p90 confidence
          interval. Click &quot;Generate forecast&quot; below to add a metric.
        </p>
        {windows.length === 0 ? (
          <EmptyState
            illustration={<span style={{ fontSize: 36 }}>◷</span>}
            headline="No forecasts generated yet"
            subhead={`Click "Generate forecast" to fit a linear regression over the last 7 days of a metric and project it forward.`}
          />
        ) : (
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fit, minmax(360px, 1fr))',
              gap: 16,
            }}
          >
            {windows.map((w) => (
              <div
                key={w.metricName}
                className="prediction-chart-card"
                style={{
                  background: 'var(--surface)',
                  border: '1px solid var(--border)',
                  borderRadius: 'var(--radius-lg)',
                  padding: 16,
                }}
              >
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'baseline',
                    marginBottom: 8,
                  }}
                >
                  <strong style={{ color: 'var(--text)' }}>{w.metricName}</strong>
                  <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                    {w.forecast.length}h · MAPE {w.mape.toFixed(1)}% · conf{' '}
                    {(w.confidence * 100).toFixed(0)}%
                  </span>
                </div>
                <PredictionChart
                  historical={w.historical}
                  forecast={w.forecast}
                  height={200}
                  metricName={w.metricName}
                />
              </div>
            ))}
          </div>
        )}

        <div style={{ marginTop: 12 }}>
          <motion.button
            type="button"
            className="empty-state-cta"
            onClick={() => setModal(true)}
            whileHover={reduce ? undefined : buttonSpring.whileHover}
            whileTap={reduce ? undefined : buttonSpring.whileTap}
            transition={buttonSpring.transition}
            disabled={isBusy}
          >
            + Generate forecast
          </motion.button>
        </div>
      </section>

      {alerts.length > 0 ? (
        <section className="dash-section">
          <span className="dash-eyebrow">Predictive event log</span>
          <h2 className="dash-section-title">Latest predictive alerts</h2>
          <div className="threat-card-list">
            {alerts.slice(0, 10).map((a) => (
              <div
                key={a.id}
                className="threat-card"
                style={a.status === 'acknowledged' ? { opacity: 0.5 } : undefined}
              >
                <div className="threat-card-top">
                  <span
                    className={`dash-status dash-status-${
                      a.severity === 'critical'
                        ? 'down'
                        : a.severity === 'warning'
                          ? 'stale'
                          : 'up'
                    }`}
                  >
                    <span className="dash-status-dot" aria-hidden="true" />
                    {a.severity}
                  </span>
                  <strong className="threat-card-type">{a.metric_name}</strong>
                  <span className="threat-card-time" title={a.created_at}>
                    {new Date(a.created_at).toLocaleString()}
                  </span>
                </div>
                <p className="threat-card-desc">
                  predicted breach <strong>{a.predicted_value.toFixed(2)}</strong> at{' '}
                  <code>{new Date(a.predicted_breach_at).toLocaleString()}</code> ·
                  confidence <strong>{(a.confidence * 100).toFixed(0)}%</strong>
                </p>
                <div className="threat-card-meta">
                  <span className="threat-card-meta-pill">
                    <span className="threat-card-meta-label">server</span>
                    <code>{a.server_id ? `${a.server_id.slice(0, 8)}…` : 'tenant-wide'}</code>
                  </span>
                  <span className="threat-card-meta-pill">
                    <span className="threat-card-meta-label">status</span>
                    <code>{a.status}</code>
                  </span>
                  {a.status === 'acknowledged' ? (
                    <span className="threat-card-resolve-btn" style={{ color: 'var(--green)' }}>
                      ✓ acknowledged
                    </span>
                  ) : (
                    <button
                      type="button"
                      className="threat-card-resolve-btn"
                      onClick={() => void ackAlert(a.id)}
                      disabled={isBusy}
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

      {modal ? (
        <div className="slow-query-explain-modal" role="dialog" aria-modal="true">
          <div className="slow-query-explain-modal-head">
            <strong>Generate forecast</strong>
            <button
              type="button"
              className="slow-query-explain-close"
              onClick={() => setModal(false)}
              aria-label="Close"
            >
              ✕
            </button>
          </div>
          <form
            className="notebook-create-form"
            onSubmit={(e) => {
              e.preventDefault();
              void runForecast();
            }}
          >
            <label>
              <span>Metric name</span>
              <input
                type="text"
                required
                maxLength={256}
                value={metric}
                onChange={(e) => setMetric(e.target.value)}
                placeholder="disk.used_pct"
              />
            </label>
            <label>
              <span>Horizon (hours)</span>
              <select
                value={horizon}
                onChange={(e) => setHorizon(parseInt(e.target.value, 10))}
              >
                <option value={6}>6h</option>
                <option value={12}>12h</option>
                <option value={24}>24h (recommended)</option>
                <option value={48}>48h</option>
                <option value={72}>3 days</option>
                <option value={168}>1 week</option>
              </select>
            </label>
            <p style={{ color: 'var(--text-muted)', fontSize: 11, margin: 0 }}>
              Linear regression over the last 7 days of metric_points. The fit computes
              slope + intercept, then projects the next N hours with p10/p90 bands
              (±1.28σ). If the forecast&apos;s p90 crosses 90% (configurable), an alert is
              created automatically.
            </p>
            <div className="incident-create-actions">
              <button
                type="button"
                className="dash-icon-button"
                onClick={() => setModal(false)}
                disabled={isBusy}
              >
                Cancel
              </button>
              <button
                type="submit"
                className="empty-state-cta"
                disabled={isBusy || !metric.trim()}
              >
                {sectionBusy ? 'Forecasting…' : 'Forecast'}
              </button>
            </div>
          </form>
        </div>
      ) : null}
    </>
  );
}