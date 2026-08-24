import { useCallback } from 'react';
import AnomalyChart, { AnomalyEvent } from './shared/AnomalyChart';
import EmptyState from './shared/EmptyState';
import KpiCard from './shared/KpiCard';
import TrainAnomalyModelModal from './shared/TrainAnomalyModelModal';
import { ApiError, api } from '../lib/api';
import { motion, buttonSpring, kpiStagger, useReducedMotion } from '../lib/motion';

/**
 * AnomaliesSection — Tier 8.1 (D9 Phase 1) UI surface extracted from
 * IntelligencePage so the page can stay under the 400-LOC cap and
 * the Anomalies content can be rendered inside the Anomalies tab of
 * the Phase 5 unified dashboard.
 *
 * Owns:
 *   - The KPI strip (today / critical / acknowledged)
 *   - The AnomalyChart with severity legend + empty state
 *   - The "Latest 10 events" list with acknowledge buttons
 *   - The "Train new model" modal trigger (POST /anomaly/train)
 *
 * The parent (IntelligencePage) feeds in the loaded events list +
 * models count + busy flag + setError callback, and calls onChanged()
 * after a successful ack so it can re-fetch the list.
 *
 * Motion: existing exports (kpiStagger, buttonSpring) — no new
 * variants. Reuses existing tokens (--surface, --border, --red,
 * --amber, --green) via inline styles.
 */

export interface AnomaliesSectionProps {
  events: AnomalyEvent[];
  modelCount: number;
  busy: boolean;
  onError: (msg: string) => void;
  onChanged: () => void;
  onOpenTrainModal: () => void;
  trainingOpen: boolean;
  onCloseTrainModal: () => void;
}

export default function AnomaliesSection({
  events,
  modelCount,
  busy,
  onError,
  onChanged,
  onOpenTrainModal,
  trainingOpen,
  onCloseTrainModal,
}: AnomaliesSectionProps) {
  const reduce = useReducedMotion();

  // Acknowledge an anomaly event. Mirrors Phase 1 behavior; the
  // parent re-fetches everything via onChanged() so the chart +
  // list + KPI strip stay in sync.
  const ack = useCallback(
    async (eventId: string) => {
      try {
        await api('POST', '/api/v1/anomaly/ack', {
          event_id: eventId,
          note: 'acknowledged from intelligence page',
        });
        onChanged();
      } catch (cause) {
        onError(
          cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message,
        );
      }
    },
    [onChanged, onError],
  );

  // KPI strip — derived client-side from the loaded events.
  // today = events in last 24h; critical = events with severity=critical;
  // acknowledged = events with acknowledged=true. Computed locally
  // so the strip doesn't need a separate /summary endpoint.
  const counts = (() => {
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
  })();

  return (
    <>
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
        <KpiCard
          label="Models"
          value={modelCount}
          status={modelCount > 0 ? 'up' : 'neutral'}
          accent="cyan"
        />
      </motion.div>

      <section className="dash-section">
        <span className="dash-eyebrow">Detections</span>
        <h2 className="dash-section-title">Recent anomalies</h2>
        <p
          className="dash-section-sub"
          style={{ color: 'var(--text-muted)', fontSize: 12, marginTop: 0 }}
        >
          Each circle is a detected anomaly. Color encodes severity (red=critical,
          amber=warning, green=info). Acknowledge noise to dim the dot in the chart.
        </p>
        {events.length === 0 ? (
          <EmptyState
            illustration={<span style={{ fontSize: 36 }}>⌬</span>}
            headline="No anomalies detected yet"
            subhead="Train a model on a metric, then call /anomaly/detect to start flagging outliers. The chart will populate as events are recorded."
          />
        ) : (
          <div
            className="anomaly-chart-card"
            style={{
              background: 'var(--surface)',
              border: '1px solid var(--border)',
              borderRadius: 'var(--radius-lg)',
              padding: 16,
            }}
          >
            <AnomalyChart events={events} />
          </div>
        )}
        <div style={{ marginTop: 12 }}>
          <motion.button
            type="button"
            className="empty-state-cta"
            onClick={onOpenTrainModal}
            whileHover={reduce ? undefined : buttonSpring.whileHover}
            whileTap={reduce ? undefined : buttonSpring.whileTap}
            transition={buttonSpring.transition}
            disabled={busy}
          >
            + Train new model
          </motion.button>
        </div>
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
                  <span
                    className={`dash-status dash-status-${
                      e.severity === 'critical'
                        ? 'down'
                        : e.severity === 'warning'
                          ? 'stale'
                          : 'up'
                    }`}
                  >
                    <span className="dash-status-dot" aria-hidden="true" />
                    {e.severity}
                  </span>
                  <strong className="threat-card-type">{e.metric_name}</strong>
                  <span className="threat-card-time" title={e.ts}>
                    {new Date(e.ts).toLocaleString()}
                  </span>
                </div>
                <p className="threat-card-desc">
                  score <strong>{e.anomaly_score.toFixed(2)}</strong> · observed{' '}
                  <code>{e.observed_value.toFixed(2)}</code> · expected [
                  {e.expected_range_low.toFixed(2)},{' '}
                  {e.expected_range_high.toFixed(2)}]
                </p>
                <div className="threat-card-meta">
                  <span className="threat-card-meta-pill">
                    <span className="threat-card-meta-label">server</span>
                    <code>
                      {e.server_id ? `${e.server_id.slice(0, 8)}…` : 'tenant-wide'}
                    </code>
                  </span>
                  {e.acknowledged ? (
                    <span
                      className="threat-card-resolve-btn"
                      style={{ color: 'var(--green)' }}
                    >
                      ✓ acknowledged
                    </span>
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

      {trainingOpen ? (
        <TrainAnomalyModelModal
          busy={busy}
          onClose={onCloseTrainModal}
          onSaved={() => {
            onCloseTrainModal();
            onChanged();
          }}
          onError={onError}
        />
      ) : null}
    </>
  );
}