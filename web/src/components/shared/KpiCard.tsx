import { ReactNode } from 'react';
import { motion, cardLift, kpiEnter, useReducedMotion } from '../../lib/motion';
import Sparkline from '../dashboard/Sparkline';

export type KpiAccent = 'cyan' | 'indigo' | 'green' | 'amber' | 'red' | 'violet';
export type KpiStatus = 'up' | 'down' | 'stale' | 'crit' | 'neutral';

interface KpiCardProps {
  /** Short uppercase eyebrow above the value. */
  label: string;
  /** The big numeric or short string value. */
  value: string | number;
  /** Optional subtext rendered under the value (12px, muted). */
  delta?: string;
  /** Optional icon node rendered in the colored bubble. */
  icon?: ReactNode;
  /** Status drives the top-stripe + icon bubble tone mapping.
   *  `up` defaults to cyan (neutral positive); `down`/`crit` = red; `stale` = amber. */
  status?: KpiStatus;
  /** Accent overrides the default status→tone mapping. */
  accent?: KpiAccent;
  /** Optional sparkline series; rendered in the same tone as the accent. */
  sparkline?: number[];
}

/** Map accent → CSS class suffix used by `.dash-metric-*` rules in
 *  styles/profile.css. Kept tiny so all visual tokens stay in CSS. */
const ACCENT_TO_CLASS: Record<KpiAccent, string> = {
  cyan: 'cyan',
  indigo: 'indigo',
  green: 'green',
  amber: 'amber',
  red: 'red',
  violet: 'indigo', // reuses indigo rule — accent stays in tokens
};

/** Default tone chosen by status when no `accent` is given. Mirrors
 *  Datadog convention: green=ok, amber=stale, red=down/crit, cyan=neutral. */
const STATUS_TO_ACCENT: Record<KpiStatus, KpiAccent> = {
  up: 'cyan',
  neutral: 'cyan',
  stale: 'amber',
  down: 'red',
  crit: 'red',
};

/**
 * KpiCard — shared KPI tile used on Dashboard, Profile, and any future
 * page that wants a metric strip.
 *
 * Visual contract (from 002-polish spec §"Component additions"):
 *   - radius `var(--radius-lg)`, surface `var(--surface)`-to-`--surface-2)`
 *     gradient, 1px border `--border`
 *   - 3px left edge stripe in `accent` color
 *   - hover: `cardLift` (`translateY(-1px)` + shadow L2 within 150ms)
 *   - label uppercase 10px, value 30px mono, hint 11px muted
 *
 * Motion: parent `kpiStagger` should pass `kpiEnter` to each child. The
 * card itself uses `kpiEnter` so a single KpiCard still animates in
 * when rendered standalone. Honors `prefers-reduced-motion` via
 * `useReducedMotion()` from `lib/motion`.
 */
export default function KpiCard({
  label,
  value,
  delta,
  icon,
  status = 'neutral',
  accent,
  sparkline,
}: KpiCardProps) {
  const reduce = useReducedMotion();
  const resolvedAccent: KpiAccent = accent ?? STATUS_TO_ACCENT[status];
  const toneClass = ACCENT_TO_CLASS[resolvedAccent];
  return (
    <motion.article
      className={`dash-metric dash-metric-${toneClass}`}
      variants={kpiEnter}
      whileHover={reduce ? undefined : cardLift}
    >
      <span className="dash-metric-stripe" aria-hidden="true" />
      {(icon || label) && (
        <div className="dash-metric-top">
          {icon ? <span className="dash-metric-icon">{icon}</span> : null}
          {label ? <span className="dash-metric-label">{label}</span> : null}
        </div>
      )}
      <div className="dash-metric-row">
        <strong>{value}</strong>
        {sparkline && sparkline.length ? (
          <Sparkline values={sparkline} tone={toneClass === 'cyan' ? 'cyan' : (toneClass as 'indigo' | 'green' | 'amber' | 'red')} />
        ) : null}
      </div>
      {delta ? <span className="dash-metric-hint">{delta}</span> : null}
    </motion.article>
  );
}
