/**
 * StatusPill — small colored badge used in host rows.
 *
 * Color derives from the host's status string ('good' / 'warn' /
 * 'bad'). The default maps to a muted neutral tone.
 */
export default function StatusPill({ status }: { status: string }) {
  const tone =
    status === 'online' || status === 'good' || status === 'up'
      ? 'good'
      : status === 'warn' || status === 'degraded'
      ? 'warn'
      : status === 'bad' || status === 'down' || status === 'offline'
      ? 'bad'
      : 'good';
  const label = status || 'unknown';
  return (
    <span className={`dash-status dash-status-${tone}`}>
      <span className="dash-status-dot" />
      {label}
    </span>
  );
}
