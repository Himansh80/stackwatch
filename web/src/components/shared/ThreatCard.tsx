/**
 * ThreatCard — compact card showing one security_threats row.
 *
 * Used on SecurityPage → Threats tab. Renders:
 *   - severity pill (low=muted, medium=amber, high=red, critical=red+pulsing)
 *   - threat type + source IP + description
 *   - detected_at relative time ("2 min ago")
 *   - resolve button (placeholder — wire to backend when write endpoint lands)
 *
 * When `resolved_at` is set, the card renders muted/strikethrough so
 * resolved threats visibly recede. Honors design tokens (--green,
 * --amber, --red with their -soft/-text variants). No hex literals.
 */
export interface ThreatCardProps {
  threat: {
    id: string;
    severity: 'low' | 'medium' | 'high' | 'critical' | string;
    threat_type: string;
    source_ip?: string | null;
    user_id?: string | null;
    description: string;
    detected_at: string;
    resolved_at?: string | null;
  };
  onResolve?: (id: string) => void;
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

export default function ThreatCard({ threat, onResolve }: ThreatCardProps) {
  const sev = (threat.severity || 'low').toLowerCase();
  const toneClass =
    sev === 'critical'
      ? 'dash-sev dash-sev-critical'
      : sev === 'high'
      ? 'dash-sev dash-sev-high'
      : sev === 'medium'
      ? 'dash-sev dash-sev-medium'
      : 'dash-sev dash-sev-low';
  const resolved = Boolean(threat.resolved_at);
  return (
    <article className={`threat-card ${resolved ? 'threat-card-resolved' : ''}`}>
      <div className="threat-card-top">
        <span className={toneClass}>{sev}</span>
        <strong className="threat-card-type">{threat.threat_type}</strong>
        <span className="threat-card-time" title={threat.detected_at}>
          {relativeTime(threat.detected_at)}
        </span>
      </div>
      <p className="threat-card-desc">{threat.description}</p>
      <div className="threat-card-meta">
        {threat.source_ip ? (
          <span className="threat-card-meta-pill">
            <span className="threat-card-meta-label">src</span>
            <code>{threat.source_ip}</code>
          </span>
        ) : null}
        {threat.user_id ? (
          <span className="threat-card-meta-pill">
            <span className="threat-card-meta-label">user</span>
            <code>{threat.user_id.slice(0, 8)}</code>
          </span>
        ) : null}
        {resolved ? (
          <span className="dash-status dash-status-good threat-card-resolved-pill">
            <span className="dash-status-dot" aria-hidden="true" />
            resolved
          </span>
        ) : onResolve ? (
          <button
            type="button"
            className="sw-button sw-button-secondary threat-card-resolve-btn"
            onClick={() => onResolve(threat.id)}
          >
            Resolve
          </button>
        ) : null}
      </div>
    </article>
  );
}