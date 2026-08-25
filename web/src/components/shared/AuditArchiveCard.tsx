import { memo } from 'react';
import { motion, buttonSpring, useReducedMotion } from '../../lib/motion';

/**
 * AuditArchiveCard — Tier 9.4 (Phase 4) Datadog-style card for a
 * single audit_log_archive row returned by
 * GET /api/v1/enterprise/audit/archives.
 *
 * Mirrors the auditArchiveRow JSON shape (handlers_audit_archive_
 * types.go): id, batch_id, event_count, size_bytes, period_start,
 * period_end, status, created_at. Purely DISPLAY — the parent
 * (AuditSection) owns the fetch lifecycle.
 *
 * Layout:
 *   - Top row: dash-sev "ARCHIVE" badge + batch_id + status pill
 *     ('running' = amber, 'completed' = green, 'failed' = red).
 *   - Stats row: event count | formatted size | period range | age.
 *   - Download button (disabled when status !== 'completed').
 *
 * Tokens reuses --surface / --border / --accent / --green / --amber
 * / --red and the .threat-card-* / .dash-status classes already
 * used by the RBAC, SCIM and SSO sections. No new CSS.
 */

export interface AuditArchiveRow {
  id: string;
  tenant_id: string;
  batch_id: string;
  event_count: number;
  size_bytes: number;
  period_start: string;
  period_end: string;
  status: 'running' | 'completed' | 'failed' | string;
  created_at: string;
}

interface AuditArchiveCardProps {
  archive: AuditArchiveRow;
  busy?: boolean;
  onDownload?: (id: string) => void;
}

const STATUS_CLASS: Record<string, string> = {
  running: 'dash-status-stale',
  completed: 'dash-status-up',
  failed: 'dash-status-down',
};

function formatBytes(bytes: number): string {
  if (!bytes || bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.min(units.length - 1, Math.floor(Math.log10(bytes) / 3));
  const scaled = bytes / Math.pow(1000, i);
  // 1 decimal for KB+, integer for B.
  const display = i === 0 ? String(Math.round(scaled)) : scaled.toFixed(1);
  return `${display} ${units[i]}`;
}

function formatDateTime(iso: string): string {
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return iso;
  return new Date(t).toISOString().replace('T', ' ').replace(/\..+$/, ' UTC');
}

function relativeTime(iso: string): string {
  if (!iso) return '—';
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return iso;
  const diff = Math.max(0, Date.now() - t);
  const sec = Math.floor(diff / 1000);
  if (sec < 60) return `${sec}s ago`;
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min}m ago`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr}h ago`;
  const day = Math.floor(hr / 24);
  return `${day}d ago`;
}

function AuditArchiveCardInner({
  archive,
  busy = false,
  onDownload,
}: AuditArchiveCardProps) {
  const reduce = useReducedMotion();
  const statusClass = STATUS_CLASS[archive.status] ?? 'dash-status-stale';
  const canDownload = archive.status === 'completed' && !!onDownload;

  const handleDownload = () => {
    if (canDownload && onDownload) onDownload(archive.id);
  };

  return (
    <article className="threat-card" aria-label={`Audit archive ${archive.batch_id}`}>
      <div className="threat-card-top">
        <span
          className="dash-sev dash-sev-low"
          style={{ background: 'var(--accent-soft)' }}
          title="Audit archive"
        >
          ARCHIVE
        </span>
        <strong className="threat-card-type">{archive.batch_id}</strong>
        <span
          className={`dash-status ${statusClass}`}
          style={{ marginLeft: 'auto' }}
        >
          <span className="dash-status-dot" aria-hidden="true" />
          {archive.status}
        </span>
      </div>

      <p className="threat-card-desc">
        <code style={{ fontSize: 11 }}>
          {formatDateTime(archive.period_start)} → {formatDateTime(archive.period_end)}
        </code>
      </p>

      <div className="threat-card-meta">
        <span>
          events{' '}
          <strong style={{ color: 'var(--text)' }}>
            {archive.event_count.toLocaleString()}
          </strong>
        </span>
        <span style={{ marginLeft: 12 }}>
          size{' '}
          <strong style={{ color: 'var(--text)' }}>
            {formatBytes(archive.size_bytes)}
          </strong>
        </span>
        <span style={{ marginLeft: 12 }} title={archive.created_at}>
          created{' '}
          <strong style={{ color: 'var(--text)' }}>
            {relativeTime(archive.created_at)}
          </strong>
        </span>
        {onDownload ? (
          <motion.button
            type="button"
            className="threat-card-resolve-btn"
            onClick={handleDownload}
            disabled={!canDownload || busy}
            title={
              canDownload
                ? `Download ${archive.batch_id}.json.gz`
                : `Archive is ${archive.status}; only completed archives can be downloaded`
            }
            style={{ marginLeft: 'auto' }}
            whileHover={!canDownload || reduce ? undefined : buttonSpring.whileHover}
            whileTap={!canDownload || reduce ? undefined : buttonSpring.whileTap}
            transition={buttonSpring.transition}
          >
            Download
          </motion.button>
        ) : null}
      </div>
    </article>
  );
}

const AuditArchiveCard = memo(AuditArchiveCardInner);
export default AuditArchiveCard;
