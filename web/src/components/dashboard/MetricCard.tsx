import { ReactNode } from 'react';
import Sparkline from './Sparkline';

/**
 * One of the four KPI cards on the Dashboard.
 *
 * Composed in `web/src/components/dashboard/MetricCard.tsx`. The
 * `icon` prop is a JSX node (an `<ServerIcon />` etc. from
 * `components/icons.tsx`) so the parent controls which icon to draw.
 *
 * Tone drives both the colored left stripe and the icon-bubble color
 * via CSS (`currentColor` in the icon stroke + `.dash-metric-<tone>`
 * class on the card root).
 */
export type MetricTone = 'cyan' | 'indigo' | 'green' | 'amber' | 'red';

interface MetricCardProps {
  label: string;
  value: string;
  hint: string;
  tone: MetricTone;
  icon: ReactNode;
  sparkline?: number[];
}

export default function MetricCard({ label, value, hint, tone, icon, sparkline }: MetricCardProps) {
  return (
    <article className={`dash-metric dash-metric-${tone}`}>
      <span className="dash-metric-stripe" aria-hidden="true" />
      <div className="dash-metric-top">
        <span className="dash-metric-icon">{icon}</span>
        <span className="dash-metric-label">{label}</span>
      </div>
      <div className="dash-metric-row">
        <strong>{value}</strong>
        {sparkline && <Sparkline values={sparkline} tone={tone} />}
      </div>
      <span className="dash-metric-hint">{hint}</span>
    </article>
  );
}
