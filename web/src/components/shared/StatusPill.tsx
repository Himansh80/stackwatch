export type StatusValue = 'up' | 'stale' | 'down' | 'unknown' | 'ok' | 'warn' | 'crit' | (string & {});

interface StatusPillProps {
  /** Status drives the dot color. */
  status: StatusValue;
  /** Optional label. If omitted, the status string itself is rendered. */
  label?: string;
  /** Size preset. */
  size?: 'sm' | 'md' | 'lg';
  /** Show only the dot (no label). */
  iconOnly?: boolean;
  /** Optional aria-label override for accessibility. */
  ariaLabel?: string;
}

const STATUS_LABEL: Record<StatusValue, string> = {
  up: 'Online',
  ok: 'OK',
  stale: 'Stale',
  down: 'Offline',
  warn: 'Warning',
  crit: 'Critical',
  unknown: 'Unknown',
};

/**
 * StatusPill — colored dot + label, sized sm/md/lg.
 *
 * Used for server status, alert severity, host health, etc.
 * Single source of truth for status color mapping across the app.
 *
 * Visual contract:
 *   - dot: 8px (sm) / 10px (md) / 12px (lg), filled with status color
 *   - label: 11px (sm) / 12px (md) / 14px (lg), uppercase, letter-spaced
 *   - pill: inline-flex, transparent bg, 8px gap
 *   - "Online" → green; "Stale" → amber; "Offline" → red; "Critical" → red;
 *     "Warning" → amber; "OK" → green; "Unknown" → gray
 */
export default function StatusPill({
  status,
  label,
  size = 'md',
  iconOnly = false,
  ariaLabel,
}: StatusPillProps) {
  const resolved = STATUS_LABEL[status];
  const text = label ?? resolved;
  return (
    <span
      className={`status-pill status-pill-${size} status-pill-${status}`}
      role="status"
      aria-label={ariaLabel ?? text}
    >
      <span className="status-pill-dot" aria-hidden="true" />
      {iconOnly ? null : <span className="status-pill-label">{text}</span>}
    </span>
  );
}