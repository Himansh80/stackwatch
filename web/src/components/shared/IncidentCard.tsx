/**
 * IncidentCard — compact card showing one incidents row.
 *
 * Used on IncidentsPage. Renders:
 *   - Severity badge (sev1=red+pulse, sev2=red, sev3=amber, sev4=muted)
 *   - Title (16px bold)
 *   - Status badge (open=red, acknowledged=amber, resolved=green)
 *   - Commander name (if set, else "Unassigned")
 *   - Duration (formatted: "2h 34m")
 *   - "View details" link
 *
 * Design tokens only (--green, --amber, --red, --muted, etc).
 * No hex literals. Reuses .dash-status and .dash-sev-* tones from
 * dashboard.css so the visual language stays consistent with the
 * rest of the app.
 */

export type IncidentSeverity = 'sev1' | 'sev2' | 'sev3' | 'sev4' | string;
export type IncidentStatus = 'open' | 'acknowledged' | 'resolved' | string;

export interface Incident {
  id: string;
  title: string;
  description?: string;
  severity: IncidentSeverity;
  status: IncidentStatus;
  commander_id?: string | null;
  started_at: string;
  resolved_at?: string | null;
  postmortem_id?: string | null;
}

export interface IncidentCardProps {
  incident: Incident;
  /** Click handler — if provided, the whole card becomes a button. */
  onSelect?: (incident: Incident) => void;
}

const SEVERITY_LABEL: Record<string, string> = {
  sev1: 'SEV1',
  sev2: 'SEV2',
  sev3: 'SEV3',
  sev4: 'SEV4',
};

const SEVERITY_CLASS: Record<string, string> = {
  sev1: 'dash-sev dash-sev-critical',
  sev2: 'dash-sev dash-sev-high',
  sev3: 'dash-sev dash-sev-medium',
  sev4: 'dash-sev dash-sev-low',
};

function severityClass(sev: string): string {
  const k = (sev || '').toLowerCase();
  return SEVERITY_CLASS[k] ?? 'dash-sev dash-sev-low';
}

function severityLabel(sev: string): string {
  const k = (sev || '').toLowerCase();
  return SEVERITY_LABEL[k] ?? sev.toUpperCase();
}

function statusTone(status: string): string {
  switch ((status || '').toLowerCase()) {
    case 'resolved':
      return 'dash-status dash-status-good';
    case 'acknowledged':
      return 'dash-status dash-status-warn';
    case 'open':
    default:
      return 'dash-status dash-status-bad';
  }
}

function shortCommander(id: string | null | undefined): string {
  if (!id) return 'Unassigned';
  return id.length > 8 ? `${id.slice(0, 8)}…` : id;
}

function formatDuration(ms: number): string {
  if (!ms || ms <= 0) return '—';
  const sec = Math.floor(ms / 1000);
  if (sec < 60) return `${sec}s`;
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min}m ${sec % 60}s`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr}h ${min % 60}m`;
  const day = Math.floor(hr / 24);
  return `${day}d ${hr % 24}h`;
}

function durationMs(start: string, end: string | null | undefined): number {
  if (!start) return 0;
  const startT = new Date(start).getTime();
  if (Number.isNaN(startT)) return 0;
  const endT = end ? new Date(end).getTime() : Date.now();
  if (Number.isNaN(endT)) return 0;
  return Math.max(0, endT - startT);
}

export default function IncidentCard({ incident, onSelect }: IncidentCardProps) {
  const dur = formatDuration(durationMs(incident.started_at, incident.resolved_at));
  const resolved = (incident.status || '').toLowerCase() === 'resolved';
  return (
    <article className={`threat-card${resolved ? ' threat-card-resolved' : ''}`}>
      <div className="threat-card-top">
        <span className={severityClass(incident.severity)}>{severityLabel(incident.severity)}</span>
        <strong className="threat-card-type">{incident.title || 'Untitled incident'}</strong>
        <span className="threat-card-time" title={incident.started_at}>
          {incident.started_at ? new Date(incident.started_at).toLocaleString() : '—'}
        </span>
      </div>
      {incident.description ? (
        <p className="threat-card-desc">{incident.description}</p>
      ) : null}
      <div className="threat-card-meta">
        <span className={statusTone(incident.status)}>
          <span className="dash-status-dot" aria-hidden="true" />
          {incident.status}
        </span>
        <span className="threat-card-meta-pill">
          <span className="threat-card-meta-label">commander</span>
          <code>{shortCommander(incident.commander_id)}</code>
        </span>
        <span className="threat-card-meta-pill">
          <span className="threat-card-meta-label">{resolved ? 'duration' : 'elapsed'}</span>
          <code>{dur}</code>
        </span>
        {onSelect ? (
          <button
            type="button"
            className="threat-card-resolve-btn"
            onClick={() => onSelect(incident)}
          >
            View details →
          </button>
        ) : null}
      </div>
    </article>
  );
}
