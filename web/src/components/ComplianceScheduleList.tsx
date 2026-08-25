import {
  motion,
  buttonSpring,
  useReducedMotion,
} from '../lib/motion';
import { ComplianceScheduleRow } from './ComplianceSection';
import { formatDateTime, relativeTime } from './ComplianceSectionHelpers';

/**
 * ComplianceScheduleList — Tier 9.5 (Phase 5) presentational
 * component that renders the schedule list + per-row delete
 * action. Lives outside ComplianceSection.tsx so the section
 * component stays under the 400-LOC modular cap.
 *
 * Dumb component: it owns no state of its own. The parent
 * (ComplianceSection) supplies the rows + an `onDelete(id)`
 * callback that hits the DELETE endpoint and refreshes the
 * parent's list.
 *
 * Motion: reuses existing exports (buttonSpring) — no new
 * variants.
 */

interface ComplianceScheduleListProps {
  schedules: ComplianceScheduleRow[];
  busy: boolean;
  onDelete: (id: string) => void;
}

export default function ComplianceScheduleList({
  schedules,
  busy,
  onDelete,
}: ComplianceScheduleListProps) {
  const reduce = useReducedMotion();
  if (schedules.length === 0) {
    return (
      <div className="threat-card-list">
        <div
          className="threat-card"
          style={{
            textAlign: 'center',
            padding: 24,
            color: 'var(--text-muted)',
          }}
        >
          <strong style={{ color: 'var(--text)' }}>No recurring schedules</strong>
          <p style={{ margin: '8px 0 16px', fontSize: 12 }}>
            Click <strong>+ Schedule report</strong> to configure a recurring
            compliance report.
          </p>
        </div>
      </div>
    );
  }
  return (
    <div className="threat-card-list">
      {schedules.map((s) => (
        <article
          key={s.id}
          className="threat-card"
          aria-label={`Schedule ${s.framework}`}
        >
          <div className="threat-card-top">
            <span
              className="dash-sev dash-sev-low"
              style={{ background: 'var(--accent-soft)' }}
              title={`Framework: ${s.framework}`}
            >
              {s.framework.toUpperCase()}
            </span>
            <strong className="threat-card-type">
              {s.frequency} schedule
            </strong>
            <span
              className={`dash-status ${
                s.enabled ? 'dash-status-up' : 'dash-status-stale'
              }`}
              style={{ marginLeft: 'auto' }}
            >
              <span className="dash-status-dot" aria-hidden="true" />
              {s.enabled ? 'enabled' : 'paused'}
            </span>
          </div>
          <p className="threat-card-desc">
            <span style={{ fontSize: 11 }}>
              next run <code>{formatDateTime(s.next_run_at)}</code> (
              {relativeTime(s.next_run_at)})
            </span>
          </p>
          <div className="threat-card-meta">
            <span>
              recipients{' '}
              <strong style={{ color: 'var(--text)' }}>
                {s.recipients.length}
              </strong>
            </span>
            <span style={{ marginLeft: 12 }}>
              created{' '}
              <strong style={{ color: 'var(--text)' }}>
                {relativeTime(s.created_at)}
              </strong>
            </span>
            <motion.button
              type="button"
              className="threat-card-resolve-btn"
              onClick={() => onDelete(s.id)}
              disabled={busy}
              title="Delete this schedule"
              style={{ marginLeft: 'auto' }}
              whileHover={reduce ? undefined : buttonSpring.whileHover}
              whileTap={reduce ? undefined : buttonSpring.whileTap}
              transition={buttonSpring.transition}
            >
              Delete
            </motion.button>
          </div>
        </article>
      ))}
    </div>
  );
}
