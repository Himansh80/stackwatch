import { motion, statusPulse, useReducedMotion } from '../../lib/motion';

/**
 * StatusPill — small colored badge used in host rows.
 *
 * Color derives from the host's status string ('good' / 'warn' /
 * 'bad'). When the resolved tone is 'bad' the dot pulses via the
 * shared `statusPulse` variant — Datadog's "the room is on fire"
 * signal. The pulse is suppressed when the user has reduced-motion
 * enabled so the dot stays solid.
 */
export default function StatusPill({ status }: { status: string }) {
  const reduce = useReducedMotion();
  const tone =
    status === 'online' || status === 'good' || status === 'up'
      ? 'good'
      : status === 'warn' || status === 'degraded'
      ? 'warn'
      : status === 'bad' || status === 'down' || status === 'offline' || status === 'crit'
      ? 'bad'
      : 'good';
  const label = status || 'unknown';
  // Only the 'bad' tone gets the breathing dot — warn keeps a
  // static dot so we don't visually shout at every degraded host.
  const Dot = tone === 'bad' && !reduce ? motion.span : 'span';
  const dotProps =
    tone === 'bad' && !reduce
      ? { animate: statusPulse.animate, transition: statusPulse.transition }
      : {};
  return (
    <span className={`dash-status dash-status-${tone}`}>
      <Dot className="dash-status-dot" aria-hidden="true" {...dotProps} />
      {label}
    </span>
  );
}