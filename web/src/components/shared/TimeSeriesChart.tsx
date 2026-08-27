import { useMemo, useState } from 'react';
import { motion } from '../../lib/motion';

export type ChartColor = 'cyan' | 'indigo' | 'green' | 'amber' | 'red' | 'violet';

interface TimeSeriesChartProps {
  /** Numeric values; chronological order. Must have at least 2 entries. */
  values: number[];
  /** Optional labels for x-axis (same length as values). Defaults to indices. */
  labels?: string[];
  /** Color theme for the line + fill. */
  color?: ChartColor;
  /** Unit suffix shown next to the current value (e.g. "%", "MB"). */
  unit?: string;
  /** Optional title shown above the chart. */
  title?: string;
  /** Fixed pixel height of the SVG. */
  height?: number;
  /** Optional empty-state message when values is empty. */
  emptyMessage?: string;
}

const COLOR_VARS: Record<ChartColor, { stroke: string; fill: string }> = {
  cyan:   { stroke: '#22d3ee', fill: 'rgba(34,211,238,0.18)' },
  indigo: { stroke: '#818cf8', fill: 'rgba(129,140,248,0.18)' },
  green:  { stroke: '#10b981', fill: 'rgba(16,185,129,0.18)' },
  amber:  { stroke: '#f59e0b', fill: 'rgba(245,158,11,0.18)' },
  red:    { stroke: '#ef4444', fill: 'rgba(239,68,68,0.18)' },
  violet: { stroke: '#a78bfa', fill: 'rgba(167,139,250,0.18)' },
};

/**
 * TimeSeriesChart — minimal SVG line chart with gradient fill, hover
 * crosshair, and min/max/avg readout. Used on Overview page for the
 * resource trend section (CPU / Memory / Disk).
 *
 * Why SVG and not a library:
 *   - Zero deps
 *   - Crisp at any DPR
 *   - Easy to theme via CSS variables
 *   - Small footprint (~3KB)
 *
 * Visual contract:
 *   - 100% width, fixed height (default 120px)
 *   - Gradient fill below the line
 *   - Hover crosshair with timestamp + value
 *   - Min/max/avg shown below the chart
 *   - Empty state: 3-line copy + illustration glyph
 */
export default function TimeSeriesChart({
  values,
  labels,
  color = 'cyan',
  unit = '',
  title,
  height = 120,
  emptyMessage = 'Waiting for live data…',
}: TimeSeriesChartProps) {
  const [hoverIdx, setHoverIdx] = useState<number | null>(null);
  const W = 600;
  const H = height;
  const P = 8; // padding inside the SVG

  const stats = useMemo(() => {
    if (!values.length) return null;
    let min = values[0];
    let max = values[0];
    let sum = 0;
    for (const v of values) {
      if (v < min) min = v;
      if (v > max) max = v;
      sum += v;
    }
    return {
      min,
      max,
      avg: sum / values.length,
      current: values[values.length - 1],
    };
  }, [values]);

  if (!values.length || values.length < 2 || !stats) {
    return (
      <div className="ts-chart ts-chart-empty" style={{ height: H }}>
        <span className="ts-chart-empty-glyph" aria-hidden="true">⌁</span>
        <p>{emptyMessage}</p>
      </div>
    );
  }

  const range = stats.max - stats.min || 1;
  const points = values.map((v, i) => {
    const x = P + (i * (W - P * 2)) / (values.length - 1);
    const y = P + (H - P * 2) * (1 - (v - stats.min) / range);
    return `${x},${y}`;
  });
  const linePath = `M ${points.join(' L ')}`;
  const fillPath = `${linePath} L ${P + (W - P * 2)},${H - P} L ${P},${H - P} Z`;
  const colors = COLOR_VARS[color];

  const handleMove = (e: React.MouseEvent<SVGSVGElement>) => {
    const svg = e.currentTarget;
    const rect = svg.getBoundingClientRect();
    const xRel = ((e.clientX - rect.left) / rect.width) * W;
    const idx = Math.round(((xRel - P) / (W - P * 2)) * (values.length - 1));
    if (idx >= 0 && idx < values.length) setHoverIdx(idx);
  };
  const handleLeave = () => setHoverIdx(null);

  const hover = hoverIdx != null ? {
    x: P + (hoverIdx * (W - P * 2)) / (values.length - 1),
    value: values[hoverIdx],
    label: labels?.[hoverIdx] ?? `#${hoverIdx + 1}`,
  } : null;

  return (
    <div className="ts-chart">
      {title ? <h4 className="ts-chart-title">{title}</h4> : null}
      <div className="ts-chart-svg-wrap">
        <svg
          className="ts-chart-svg"
          viewBox={`0 0 ${W} ${H}`}
          preserveAspectRatio="none"
          role="img"
          aria-label={`${title ?? 'Chart'}: ${stats.current.toFixed(1)}${unit}`}
          onMouseMove={handleMove}
          onMouseLeave={handleLeave}
        >
          <defs>
            <linearGradient id={`ts-fill-${color}`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={colors.stroke} stopOpacity="0.45" />
              <stop offset="100%" stopColor={colors.stroke} stopOpacity="0" />
            </linearGradient>
          </defs>
          <motion.path
            d={fillPath}
            fill={`url(#ts-fill-${color})`}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.4 }}
          />
          <motion.path
            d={linePath}
            stroke={colors.stroke}
            strokeWidth="1.5"
            fill="none"
            strokeLinejoin="round"
            strokeLinecap="round"
            initial={{ pathLength: 0 }}
            animate={{ pathLength: 1 }}
            transition={{ duration: 0.6 }}
          />
          {hover ? (
            <g>
              <line
                x1={hover.x}
                x2={hover.x}
                y1={P}
                y2={H - P}
                stroke="rgba(255,255,255,0.18)"
                strokeWidth="1"
                strokeDasharray="2 2"
              />
              <circle cx={hover.x} cy={
                P + (H - P * 2) * (1 - (hover.value - stats.min) / range)
              } r="3" fill={colors.stroke} stroke="white" strokeWidth="1.5" />
            </g>
          ) : null}
        </svg>
        {hover ? (
          <div
            className="ts-chart-tooltip"
            style={{
              left: `${(hover.x / W) * 100}%`,
            }}
          >
            <span className="ts-chart-tooltip-label">{hover.label}</span>
            <strong>{hover.value.toFixed(1)}{unit}</strong>
          </div>
        ) : null}
      </div>
      <div className="ts-chart-stats">
        <span className="ts-chart-current">
          <strong>{stats.current.toFixed(1)}{unit}</strong>
          <em>current</em>
        </span>
        <span className="ts-chart-min"><strong>{stats.min.toFixed(1)}{unit}</strong><em>min</em></span>
        <span className="ts-chart-avg"><strong>{stats.avg.toFixed(1)}{unit}</strong><em>avg</em></span>
        <span className="ts-chart-max"><strong>{stats.max.toFixed(1)}{unit}</strong><em>max</em></span>
      </div>
    </div>
  );
}