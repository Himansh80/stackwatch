import { useMemo, useState } from 'react';
import { motion, sparklineDraw, useReducedMotion } from '../../lib/motion';

/**
 * AnomalyEvent — one row from GET /api/v1/anomaly/events.
 *
 * Field shapes match the server JSON exactly so this component is
 * drop-in for any consumer that calls the endpoint.
 */
export interface AnomalyEvent {
  id: string;
  model_id?: string;
  server_id?: string;
  metric_name: string;
  anomaly_score: number;
  severity: 'info' | 'warning' | 'critical' | string;
  observed_value: number;
  expected_range_low: number;
  expected_range_high: number;
  ts: string;
  acknowledged: boolean;
  ack_note?: string;
  ack_user_id?: string;
}

interface AnomalyChartProps {
  /** Events to plot, newest first (matches server ORDER BY ts DESC). */
  events: AnomalyEvent[];
  /** Optional chart height in px. Default 220. */
  height?: number;
}

/**
 * Map severity to a CSS variable consumed by the inline `style` block.
 * We deliberately reference the existing design tokens (`--red`,
 * `--amber`, `--green`) defined in styles/profile.css rather than
 * hardcoding hex. The fallback string at the bottom of the map keeps
 * TypeScript happy if someone passes an unknown severity.
 */
const SEVERITY_TO_VAR: Record<string, string> = {
  critical: 'var(--red)',
  warning: 'var(--amber)',
  info: 'var(--green)',
};
const fallbackColor = 'var(--text-muted, #6b7280)';

const severityColor = (sev: string): string => SEVERITY_TO_VAR[sev] ?? fallbackColor;

/**
 * AnomalyChart — Datadog-style scatter of anomaly_score over time.
 *
 * Layout:
 *   - X-axis: timestamps (oldest left, newest right). When the events
 *     array is newest-first we reverse it for the plot so time flows
 *     left → right (matches what on-call engineers expect).
 *   - Y-axis: anomaly_score (0..10 fixed domain with the red threshold
 *     line drawn at 3 = sigma). Auto-rescales if the max score > 10.
 *   - Each event is a circle, colored by severity (red=critical,
 *     amber=warning, green=info). Acknowledged events render at
 *     40% opacity with a thin stroke.
 *   - Hover any circle → tooltip with ts + score + observed_value.
 *
 * Why an SVG and not a charting library: avoids adding any
 * dependencies (the spec explicitly bans new deps) and matches the
 * pattern used by other lightweight visualizations in the app.
 */
export default function AnomalyChart({ events, height = 220 }: AnomalyChartProps) {
  const reduce = useReducedMotion();
  const [hover, setHover] = useState<{ e: AnomalyEvent; x: number; y: number } | null>(null);

  // Reverse so oldest is left, newest is right (events arrive newest-first).
  const ordered = useMemo(() => [...events].reverse(), [events]);

  // Compute plot bounds. Y domain is 0..10 unless the data exceeds
  // that — the threshold line at 3σ stays in the same relative spot
  // by clamping it into the rendered Y range.
  const { yMax, xPoints, plotted } = useMemo(() => {
    if (ordered.length === 0) {
      return { yMax: 10, xPoints: [] as { e: AnomalyEvent; x: number; y: number }[], plotted: [] as AnomalyEvent[] };
    }
    const scores = ordered.map((e) => Math.abs(e.anomaly_score)).filter((s) => Number.isFinite(s));
    const observedMax = scores.length ? Math.max(...scores) : 0;
    const computedYMax = Math.max(10, Math.ceil(observedMax + 1));
    // X is normalized to 0..1; the SVG viewBox stretches it to 1000 wide.
    const xPts = ordered.map((e, i) => ({
      e,
      x: ordered.length === 1 ? 500 : (i / (ordered.length - 1)) * 1000,
      y: computedYMax === 0 ? 0 : (1 - Math.min(Math.abs(e.anomaly_score) / computedYMax, 1)) * 200,
    }));
    return { yMax: computedYMax, xPoints: xPts, plotted: ordered };
  }, [ordered]);

  const thresholdY = yMax === 0 ? 0 : (1 - Math.min(3 / yMax, 1)) * 200;

  if (plotted.length === 0) {
    return (
      <div className="anomaly-chart-empty" style={{ minHeight: height, display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-muted, #6b7280)' }}>
        No anomalies yet. Train a model and feed it some traffic to see detections.
      </div>
    );
  }

  return (
    <div className="anomaly-chart-wrap" style={{ position: 'relative' }}>
      <svg
        viewBox="0 0 1000 240"
        preserveAspectRatio="none"
        role="img"
        aria-label={`Anomaly score over time — ${plotted.length} events`}
        style={{ width: '100%', height }}
        onMouseLeave={() => setHover(null)}
      >
        {/* Threshold line at 3σ — dashed red across the whole chart. */}
        <line
          x1="0"
          x2="1000"
          y1={thresholdY}
          y2={thresholdY}
          stroke="var(--red)"
          strokeWidth="1"
          strokeDasharray="6 4"
          opacity="0.5"
        />
        <text x="6" y={thresholdY - 4} fontSize="10" fill="var(--red)" opacity="0.8">
          3σ threshold
        </text>

        {/* X axis baseline. */}
        <line x1="0" x2="1000" y1="200" y2="200" stroke="var(--border)" strokeWidth="1" />

        {/* Connecting line between consecutive points — Datadog style. */}
        {xPoints.length > 1 ? (
          <motion.polyline
            points={xPoints.map((p) => `${p.x},${p.y}`).join(' ')}
            fill="none"
            stroke="var(--border)"
            strokeWidth="1"
            strokeOpacity="0.4"
            initial={reduce ? 'show' : 'hidden'}
            animate="show"
            variants={sparklineDraw}
          />
        ) : null}

        {/* Each event as a circle. */}
        {xPoints.map(({ e, x, y }) => {
          const fill = severityColor(e.severity);
          const r = e.severity === 'critical' ? 6 : e.severity === 'warning' ? 5 : 4;
          const opacity = e.acknowledged ? 0.4 : 1;
          return (
            <g key={e.id}>
              <circle
                cx={x}
                cy={y}
                r={r}
                fill={fill}
                opacity={opacity}
                stroke={e.acknowledged ? fill : 'var(--surface, #0e0e10)'}
                strokeWidth="1.5"
                onMouseEnter={() => setHover({ e, x, y })}
                style={{ cursor: 'pointer' }}
              >
                <title>
                  {`${e.metric_name} • score ${e.anomaly_score.toFixed(2)} • value ${e.observed_value.toFixed(2)}`}
                </title>
              </circle>
            </g>
          );
        })}

        {/* Y axis labels (0 and yMax). */}
        <text x="0" y="12" fontSize="10" fill="var(--text-muted, #6b7280)">
          {yMax.toFixed(0)}σ
        </text>
        <text x="0" y="198" fontSize="10" fill="var(--text-muted, #6b7280)">
          0σ
        </text>
      </svg>

      {hover ? (
        <div
          className="anomaly-chart-tooltip"
          style={{
            position: 'absolute',
            left: `calc(${(hover.x / 1000) * 100}% + 8px)`,
            top: `calc(${(hover.y / 200) * 100}% + 4px)`,
            background: 'var(--surface, #0e0e10)',
            border: '1px solid var(--border)',
            borderRadius: 6,
            padding: '6px 10px',
            fontSize: 11,
            color: 'var(--text, #e5e7eb)',
            pointerEvents: 'none',
            whiteSpace: 'nowrap',
            boxShadow: '0 4px 12px rgba(0,0,0,0.4)',
            zIndex: 2,
          }}
        >
          <div style={{ fontWeight: 600 }}>{hover.e.metric_name}</div>
          <div style={{ opacity: 0.8 }}>{new Date(hover.e.ts).toLocaleString()}</div>
          <div>score: <strong style={{ color: severityColor(hover.e.severity) }}>{hover.e.anomaly_score.toFixed(2)}</strong> ({hover.e.severity})</div>
          <div>observed: {hover.e.observed_value.toFixed(2)}</div>
          <div>range: [{hover.e.expected_range_low.toFixed(2)}, {hover.e.expected_range_high.toFixed(2)}]</div>
          {hover.e.acknowledged ? <div style={{ color: 'var(--green)' }}>✓ acknowledged</div> : null}
        </div>
      ) : null}
    </div>
  );
}
