import { ProxmoxResource } from '../../lib/proxmox';

/**
 * TrendChart — line chart of current CPU usage across resources.
 *
 * Renders an SVG with gridlines, area fill, polyline, and data dots.
 * Shows a polished empty state when no resources have reported yet.
 */
export default function TrendChart({ resources }: { resources: ProxmoxResource[] }) {
  const values = resources
    .map((row) => Number(row.cpu))
    .filter((number) => Number.isFinite(number));
  if (!values.length) {
    return (
      <div className="dash-chart-empty">
        <div className="dash-empty-illustration" aria-hidden="true">
          <svg viewBox="0 0 120 80" fill="none" stroke="currentColor" strokeWidth="1.2">
            <rect x="6" y="14" width="108" height="56" rx="6" opacity=".35" />
            <path d="M14 60 L34 44 L54 50 L74 32 L94 38 L106 28" strokeLinecap="round" strokeLinejoin="round" />
            <circle cx="34" cy="44" r="2" />
            <circle cx="54" cy="50" r="2" />
            <circle cx="74" cy="32" r="2" />
            <circle cx="94" cy="38" r="2" />
          </svg>
        </div>
        <strong>Waiting for live resource data</strong>
        <small>Connect a Proxmox host to populate this view.</small>
      </div>
    );
  }
  const max = Math.max(...values, 1);
  const points = values
    .map((number, index) => {
      const x = 18 + (index / Math.max(values.length - 1, 1)) * 464;
      const y = 158 - (number / max) * 124;
      return `${x},${y}`;
    })
    .join(' ');
  return (
    <div className="dash-chart-wrap">
      <svg viewBox="0 0 500 190" role="img" aria-label="Current workload CPU values">
        <defs>
          <linearGradient id="dashArea" x1="0" x2="0" y1="0" y2="1">
            <stop offset="0" stopColor="#38bdf8" stopOpacity=".34" />
            <stop offset="1" stopColor="#38bdf8" stopOpacity="0" />
          </linearGradient>
        </defs>
        <line x1="18" y1="34" x2="482" y2="34" className="dash-grid-line" />
        <line x1="18" y1="96" x2="482" y2="96" className="dash-grid-line" />
        <line x1="18" y1="158" x2="482" y2="158" className="dash-grid-line" />
        <polygon points={`18,158 ${points} 482,158`} fill="url(#dashArea)" />
        <polyline points={points} fill="none" stroke="#38bdf8" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />
        {values.map((number, index) => {
          const x = 18 + (index / Math.max(values.length - 1, 1)) * 464;
          const y = 158 - (number / max) * 124;
          return (
            <circle
              key={`${number}-${index}`}
              cx={x}
              cy={y}
              r="3.5"
              fill="#0b1220"
              stroke="#67e8f9"
              strokeWidth="2"
            />
          );
        })}
      </svg>
      <div className="dash-chart-axis">
        <span>0%</span>
        <span>Current workload CPU by resource</span>
        <span>100%</span>
      </div>
    </div>
  );
}
