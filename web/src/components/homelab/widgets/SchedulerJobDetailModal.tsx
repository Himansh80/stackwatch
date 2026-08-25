import {
  schedulerRelativeTime,
  schedulerStatusColor,
  type SchedulerJob,
  type SchedulerRun,
} from './SchedulerWidget.types';

/**
 * SchedulerJobDetailModal — Tier 10 Phase 9 (H10).
 *
 * The detail modal extracted from SchedulerWidget so the
 * main widget stays under the 400-LOC cap. Shows the job's
 * URL, method, schedule, headers, body, and a compact
 * recent-runs list with status pill + latency + error
 * tooltip.
 *
 * The user can only CLOSE the modal from here — toggling
 * enabled, deleting, and triggering a manual run live in
 * the parent widget (where the action handlers already
 * exist). Closing the modal also re-fetches the parent's
 * runs list when triggered via the parent's `openDetail`
 * helper, so a manual run from the row shows up here
 * immediately on the next open.
 */

interface SchedulerJobDetailModalProps {
  job: SchedulerJob;
  runs: SchedulerRun[];
  onClose: () => void;
}

export default function SchedulerJobDetailModal({
  job,
  runs,
  onClose,
}: SchedulerJobDetailModalProps) {
  return (
    <div
      className="homelab-modal-backdrop"
      role="dialog"
      aria-modal="true"
      onClick={onClose}
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.6)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 1000,
      }}
    >
      <div
        className="homelab-modal-panel"
        onClick={(e) => e.stopPropagation()}
        style={{
          background: 'var(--surface-2)',
          borderRadius: 8,
          padding: 24,
          minWidth: 480,
          maxWidth: 640,
          width: '95%',
          maxHeight: '90vh',
          overflowY: 'auto',
        }}
      >
        <h3 style={{ marginTop: 0 }}>{job.name || '(unnamed)'}</h3>
        <small style={{ color: 'var(--muted)' }}>
          <code>{job.method}</code> {job.url}
        </small>
        <p>
          <strong>Schedule:</strong> <code>{job.schedule}</code>
        </p>
        <div>
          <strong>Headers:</strong>
          <pre
            style={{
              background: 'var(--surface-1)',
              padding: 8,
              borderRadius: 4,
              overflow: 'auto',
              fontSize: 12,
            }}
          >
            {Object.entries(job.headers)
              .map(([k, v]) => `${k}: ${v}`)
              .join('\n') || '(none)'}
          </pre>
        </div>
        {job.body ? (
          <div>
            <strong>Body:</strong>
            <pre
              style={{
                background: 'var(--surface-1)',
                padding: 8,
                borderRadius: 4,
                overflow: 'auto',
                fontSize: 12,
              }}
            >
              {job.body}
            </pre>
          </div>
        ) : null}
        <div>
          <strong>Recent runs ({runs.length})</strong>
          <div style={{ maxHeight: 200, overflowY: 'auto', marginTop: 8 }}>
            {runs.length === 0 ? (
              <small style={{ color: 'var(--muted)' }}>No runs yet.</small>
            ) : (
              runs.map((r) => (
                <div
                  key={r.id}
                  style={{
                    padding: 6,
                    borderBottom: '1px solid var(--border)',
                    fontSize: 12,
                  }}
                >
                  <span
                    style={{
                      background: schedulerStatusColor(r.status),
                      color: 'white',
                      padding: '1px 6px',
                      borderRadius: 3,
                      marginRight: 6,
                    }}
                  >
                    {r.status}
                  </span>
                  {r.http_status_code ? `${r.http_status_code} • ` : ''}
                  {r.duration_ms ? `${r.duration_ms}ms • ` : ''}
                  {schedulerRelativeTime(r.started_at)}
                  {r.error_message ? ` — ${r.error_message.slice(0, 60)}` : ''}
                </div>
              ))
            )}
          </div>
        </div>
        <div style={{ marginTop: 12, textAlign: 'right' }}>
          <button type="button" className="empty-state-cta" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
