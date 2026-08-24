import { ReactNode } from 'react';

/**
 * SlaBadge — small pill showing uptime % vs target + optional response ms.
 *
 * Color tier:
 *   uptime >= target      → green   (SLA met)
 *   uptime >= target - 1  → amber   (within 1pp of breach)
 *   uptime <  target - 1  → red     (SLA breach)
 *
 * Response ms adds a second dim:
 *   responseMs > targetMs → flips the amber/red boundary one tier harsher
 *   when uptime alone is borderline.
 *
 * Used by SyntheticsPage test rows and the SLA detail card. Honors
 * design tokens from base.css (--green, --amber, --red with their
 * -soft + -text variants).
 */
export interface SlaBadgeProps {
  uptimePct: number;
  targetPct?: number;
  responseMs?: number;
  targetMs?: number;
  /** Optional trailing node (e.g. last-run timestamp). */
  hint?: ReactNode;
}

type Tone = 'green' | 'amber' | 'red';

function pickTone(uptimePct: number, targetPct: number, responseMs?: number, targetMs?: number): Tone {
  const baseTone: Tone =
    uptimePct >= targetPct
      ? 'green'
      : uptimePct >= targetPct - 1
      ? 'amber'
      : 'red';
  // Response-time breach worsens amber/red.
  if (responseMs !== undefined && targetMs !== undefined && responseMs > targetMs) {
    if (baseTone === 'green') return 'amber';
    return 'red';
  }
  return baseTone;
}

export default function SlaBadge({
  uptimePct,
  targetPct = 99.9,
  responseMs,
  targetMs,
  hint,
}: SlaBadgeProps) {
  const tone = pickTone(uptimePct, targetPct, responseMs, targetMs);
  const cls =
    tone === 'green'
      ? 'dash-sla-badge dash-sla-badge-green'
      : tone === 'amber'
      ? 'dash-sla-badge dash-sla-badge-amber'
      : 'dash-sla-badge dash-sla-badge-red';
  return (
    <span className={cls} title={`SLA target: ${targetPct}% uptime`}>
      <span className="dash-sla-badge-uptime">
        {uptimePct.toFixed(2)}%
      </span>
      {responseMs !== undefined ? (
        <span className="dash-sla-badge-ms">{responseMs}ms</span>
      ) : null}
      {hint ? <span className="dash-sla-badge-hint">{hint}</span> : null}
    </span>
  );
}