import { memo } from 'react';
import { motion, buttonSpring, useReducedMotion } from '../../lib/motion';

/**
 * ComplianceReportCard — Tier 9.5 (Phase 5) Datadog-style card
 * for a single compliance_reports row returned by
 * GET /api/v1/enterprise/compliance/reports.
 *
 * Mirrors the complianceReportRow JSON shape
 * (handlers_compliance_types.go): id, framework, period_start,
 * period_end, status, artifact_path, created_at, completed_at,
 * error_message. Purely DISPLAY — the parent (ComplianceSection)
 * owns the fetch lifecycle.
 *
 * Layout:
 *   - Top row: framework badge (soc2=red, iso27001=blue,
 *     hipaa=purple, pci=amber, gdpr=green) + status pill
 *     (pending=muted, running=amber, completed=green, failed=red).
 *   - Stats row: period range | created_at (relative) |
 *     completed_at (relative, if available).
 *   - Download button (only when status='completed' AND parent
 *     provided onDownload).
 *   - Error message (red, only when status='failed').
 *
 * Tokens reuses --surface / --border / --accent / --green / --amber
 * / --red / .threat-card-* / .dash-status classes already used by
 * the RBAC, SCIM, SSO and Audit sections. No new CSS.
 */

export interface ComplianceReportRow {
  id: string;
  tenant_id: string;
  framework: 'soc2' | 'iso27001' | 'hipaa' | 'pci' | 'gdpr' | string;
  period_start: string;
  period_end: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | string;
  artifact_path?: string;
  scheduled_id?: string;
  requested_by?: string;
  created_at: string;
  completed_at?: string;
  error_message?: string;
}

interface ComplianceReportCardProps {
  report: ComplianceReportRow;
  busy?: boolean;
  onDownload?: (id: string) => void;
}

const STATUS_CLASS: Record<string, string> = {
  pending: 'dash-status-stale',
  running: 'dash-status-stale',
  completed: 'dash-status-up',
  failed: 'dash-status-down',
};

// Framework badge colors map — matches the spec §"Story 5" badges.
// Background tint via var(--*-soft) for a soft-pill appearance.
const FRAMEWORK_STYLE: Record<string, { bg: string; label: string }> = {
  soc2: { bg: 'rgba(239, 68, 68, 0.16)', label: 'SOC2' },
  iso27001: { bg: 'rgba(59, 130, 246, 0.16)', label: 'ISO 27001' },
  hipaa: { bg: 'rgba(168, 85, 247, 0.16)', label: 'HIPAA' },
  pci: { bg: 'rgba(245, 158, 11, 0.16)', label: 'PCI' },
  gdpr: { bg: 'rgba(16, 185, 129, 0.16)', label: 'GDPR' },
};

function formatDateTime(iso: string): string {
  if (!iso) return '—';
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

function ComplianceReportCardInner({
  report,
  busy = false,
  onDownload,
}: ComplianceReportCardProps) {
  const reduce = useReducedMotion();
  const statusClass = STATUS_CLASS[report.status] ?? 'dash-status-stale';
  const fwStyle =
    FRAMEWORK_STYLE[report.framework] ?? {
      bg: 'var(--accent-soft)',
      label: report.framework.toUpperCase(),
    };
  const canDownload = report.status === 'completed' && !!onDownload;

  const handleDownload = () => {
    if (canDownload && onDownload) onDownload(report.id);
  };

  return (
    <article
      className="threat-card"
      aria-label={`Compliance report ${report.framework} ${report.id}`}
    >
      <div className="threat-card-top">
        <span
          className="dash-sev dash-sev-low"
          style={{ background: fwStyle.bg }}
          title={`Framework: ${report.framework}`}
        >
          {fwStyle.label}
        </span>
        <strong className="threat-card-type">
          {report.framework.toUpperCase()} report
        </strong>
        <span
          className={`dash-status ${statusClass}`}
          style={{ marginLeft: 'auto' }}
        >
          <span className="dash-status-dot" aria-hidden="true" />
          {report.status}
        </span>
      </div>

      <p className="threat-card-desc">
        <code style={{ fontSize: 11 }}>
          {formatDateTime(report.period_start)} →{' '}
          {formatDateTime(report.period_end)}
        </code>
      </p>

      <div className="threat-card-meta">
        <span>
          created{' '}
          <strong style={{ color: 'var(--text)' }}>
            {relativeTime(report.created_at)}
          </strong>
        </span>
        {report.completed_at ? (
          <span style={{ marginLeft: 12 }} title={report.completed_at}>
            completed{' '}
            <strong style={{ color: 'var(--text)' }}>
              {relativeTime(report.completed_at)}
            </strong>
          </span>
        ) : null}
        {onDownload ? (
          <motion.button
            type="button"
            className="threat-card-resolve-btn"
            onClick={handleDownload}
            disabled={!canDownload || busy}
            title={
              canDownload
                ? `Download ${report.framework} report`
                : `Report is ${report.status}; only completed reports can be downloaded`
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

      {report.status === 'failed' && report.error_message ? (
        <div
          style={{
            marginTop: 8,
            padding: '6px 10px',
            background: 'rgba(239, 68, 68, 0.08)',
            border: '1px solid rgba(239, 68, 68, 0.4)',
            borderRadius: 'var(--radius-md)',
            color: 'var(--red, #ef4444)',
            fontSize: 11,
            fontFamily: 'ui-monospace, SFMono-Regular, monospace',
          }}
          role="alert"
        >
          {report.error_message}
        </div>
      ) : null}
    </article>
  );
}

const ComplianceReportCard = memo(ComplianceReportCardInner);
export default ComplianceReportCard;
