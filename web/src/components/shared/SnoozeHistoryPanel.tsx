import { useEffect, useState } from 'react';
import EmptyState from './EmptyState';
import { ApiError, api } from '../../lib/api';
import { motion, buttonSpring, useReducedMotion } from '../../lib/motion';

/**
 * SnoozeHistoryPanel — Tier 8.4 (Phase 4) UI for the recent-snoozes
 * table + "Snooze alert" action modal. Extracted from
 * NoiseReductionSection so that file stays under the 400-LOC cap.
 *
 * Renders:
 *   - heading + subhead
 *   - top-10 snooze list (EmptyState when empty)
 *   - "+ Snooze alert" CTA + modal (POST /noise/snooze)
 *
 * Props:
 *   - snoozes: list returned by GET /noise/history
 *   - busy:    combined busy flag from the parent
 *   - onError: callback for error reporting
 *   - onChanged: re-fetch the snooze list after a successful create
 */

export interface SnoozeRow {
  id: string;
  alert_id: string;
  user_id: string;
  duration_seconds: number;
  expires_at: string;
  reason?: string | null;
  active: boolean;
  created_at: string;
}

interface SnoozeHistoryPanelProps {
  snoozes: SnoozeRow[];
  busy: boolean;
  onError: (msg: string) => void;
  onChanged: () => void;
}

function windowLabel(seconds: number): string {
  if (seconds >= 604800) return '7d';
  if (seconds >= 86400) return '24h';
  if (seconds >= 21600) return '6h';
  if (seconds >= 3600) return '1h';
  if (seconds >= 900) return '15m';
  if (seconds >= 300) return '5m';
  return `${seconds}s`;
}

function shortId(id: string): string {
  return id.length > 8 ? `${id.slice(0, 8)}…` : id;
}

export default function SnoozeHistoryPanel({
  snoozes,
  busy,
  onError,
  onChanged,
}: SnoozeHistoryPanelProps) {
  const reduce = useReducedMotion();
  const [sectionBusy, setSectionBusy] = useState(false);
  const [modal, setModal] = useState(false);
  const [alertID, setAlertID] = useState('');
  const [duration, setDuration] = useState('3600');
  const [reason, setReason] = useState('');

  const isBusy = busy || sectionBusy;

  // Close modal on Escape.
  useEffect(() => {
    if (!modal) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setModal(false);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [modal]);

  const submit = async () => {
    const trimmed = alertID.trim();
    const d = parseInt(duration, 10);
    if (!trimmed) {
      onError('Alert ID is required.');
      return;
    }
    if (Number.isNaN(d) || d < 60) {
      onError('Duration must be at least 60 seconds.');
      return;
    }
    setSectionBusy(true);
    try {
      await api('POST', '/api/v1/noise/snooze', {
        alert_id: trimmed,
        duration_seconds: d,
        reason: reason.trim(),
      });
      setModal(false);
      setAlertID('');
      setDuration('3600');
      setReason('');
      onChanged();
    } catch (cause) {
      onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    } finally {
      setSectionBusy(false);
    }
  };

  return (
    <section className="dash-section">
      <span className="dash-eyebrow">Snooze log</span>
      <h2 className="dash-section-title">Recent snoozes</h2>
      <p
        className="dash-section-sub"
        style={{ color: 'var(--text-muted)', fontSize: 12, marginTop: 0 }}
      >
        Manual snoozes stamp a snooze_log row with <code>expires_at</code>.
        While <code>expires_at &gt; now()</code>, the alert is treated as
        suppressed. Most recent 10 are shown below.
      </p>
      {snoozes.length === 0 ? (
        <EmptyState
          illustration={<span style={{ fontSize: 36 }}>☾</span>}
          headline="No snoozes yet"
          subhead="When you snooze an alert, it lands here with an expiry time. Use the snooze action to silence a known-noisy alert for a specific window."
        />
      ) : (
        <div
          className="threat-card-list"
          style={{ maxHeight: 320, overflowY: 'auto', paddingRight: 4 }}
        >
          {snoozes.slice(0, 10).map((s) => (
            <div key={s.id} className="threat-card" style={{ padding: '10px 14px' }}>
              <div className="threat-card-top">
                <span
                  className={`dash-status ${
                    s.active ? 'dash-status-stale' : 'dash-status-up'
                  }`}
                >
                  <span className="dash-status-dot" aria-hidden="true" />
                  {s.active ? 'active' : 'expired'}
                </span>
                <code style={{ fontSize: 12, color: 'var(--text)' }}>
                  alert {shortId(s.alert_id)}
                </code>
                <span className="threat-card-time" title={s.created_at}>
                  {new Date(s.created_at).toLocaleString()}
                </span>
              </div>
              <p className="threat-card-desc" style={{ fontSize: 12 }}>
                {windowLabel(s.duration_seconds)} ·{' '}
                expires {new Date(s.expires_at).toLocaleString()}
                {s.reason ? <> · <em>{s.reason}</em></> : null}
              </p>
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
          + Snooze alert
        </motion.button>
      </div>

      {modal ? (
        <div className="slow-query-explain-modal" role="dialog" aria-modal="true">
          <div className="slow-query-explain-modal-head">
            <strong>Snooze alert</strong>
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
              void submit();
            }}
          >
            <label>
              <span>Alert ID (UUID)</span>
              <input
                type="text"
                required
                value={alertID}
                onChange={(e) => setAlertID(e.target.value)}
                placeholder="uuid-of-anomaly-or-predictive-alert"
              />
            </label>
            <label>
              <span>Duration (seconds, min 60, max 604800)</span>
              <input
                type="number"
                min={60}
                max={604800}
                step={60}
                value={duration}
                onChange={(e) => setDuration(e.target.value)}
              />
            </label>
            <label>
              <span>Reason (optional)</span>
              <input
                type="text"
                maxLength={2048}
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="known-noisy metric, in maintenance window"
              />
            </label>
            <p style={{ color: 'var(--text-muted)', fontSize: 11, margin: 0 }}>
              Snooze is stamped from your JWT — you can&apos;t snooze on behalf of another user.
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
                disabled={isBusy || !alertID.trim()}
              >
                {sectionBusy ? 'Snoozing…' : 'Snooze'}
              </button>
            </div>
          </form>
        </div>
      ) : null}
    </section>
  );
}
