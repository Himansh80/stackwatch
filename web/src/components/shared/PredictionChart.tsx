import { useMemo, useState } from 'react';
import { motion, sparklineDraw, useReducedMotion } from '../../lib/motion';

/**
 * Historical data point — last 7 days of metric_points (or whatever
 * the caller passes). Field shapes match the server JSON exactly.
 */
export interface HistoricalPoint {
  ts: string;
  value: number;
}

/**
 * Forecast point — one hourly prediction returned from POST
 * /api/v1/predict/forecast. p10/p50/p90 are the 10th/50th/90th
 * percentiles of the forecast distribution (±1.28σ around the
 * linear point estimate).
 */
export interface ForecastPoint {
  ts: string;
  p10: number;
  p50: number;
  p90: number;
}

interface PredictionChartProps {
  /** Historical series (oldest first OR newest first — we sort it). */
  historical: HistoricalPoint[];
  /** Forecast series (oldest first, matches server output). */
  forecast: ForecastPoint[];
  /** Optional chart height in px. Default 220. */
  height?: number;
  /** Optional metric name for the tooltip header. */
  metricName?: string;
}

/**
 * PredictionChart — Datadog-style prediction band with p10/p50/p90.
 *
 * Layout:
 *   - X-axis: timestamps (oldest left, newest right). We sort both
 *     arrays so the chart always flows left → right even if the
 *     caller passes newest-first historical data.
 *   - Y-axis: auto-scaled from min(historical ∪ p10) to
 *     max(historical ∪ p90).
 *   - Historical: solid line in `--accent` color.
 *   - Forecast p50: dashed line in `--accent` color (continuation
 *     of the historical line so the eye sees one continuous trend).
 *   - Forecast band: shaded area between p10 and p90 (using
 *     `--accent-soft` so it's the same accent but at 15% opacity).
 *   - Hover any point → tooltip with timestamp + historical value
 *     OR forecast p10/p50/p90.
 *
 * No external chart library — pure SVG, matches the AnomalyChart
 * pattern. Reuses existing tokens (--accent, --accent-soft, --border,
 * --text) so no hardcoded hex.
 */
export default function PredictionChart({
  historical,
  forecast,
  height = 220,
  metricName,
}: PredictionChartProps) {
  const reduce = useReducedMotion();
  const [hover, setHover] = useState<
    | { kind: 'historical'; ts: string; value: number; x: number; y: number }
    | { kind: 'forecast'; pt: ForecastPoint; x: number; y: number }
    | null
  >(null);

  // Sort historical oldest-first. The server returns DESC but the
  // chart needs ASC for left-to-right time flow.
  const histAsc = useMemo(
    () =>
      [...historical].sort((a, b) => new Date(a.ts).getTime() - new Date(b.ts).getTime()),
    [historical],
  );

  // Compute bounds + per-point screen coords.
  const { yMin, yMax, histPts, p10Pts, p90Pts, p50Pts } = useMemo(() => {
    const allYs: number[] = [];
    for (const h of histAsc) allYs.push(h.value);
    for (const f of forecast) {
      allYs.push(f.p10, f.p50, f.p90);
    }
    if (allYs.length === 0) {
      return {
        yMin: 0,
        yMax: 100,
        histPts: [] as { ts: string; value: number; x: number; y: number }[],
        p10Pts: [] as { x: number; y: number }[],
        p50Pts: [] as { x: number; y: number }[],
        p90Pts: [] as { x: number; y: number }[],
      };
    }
    const lo = Math.min(...allYs);
    const hi = Math.max(...allYs);
    // Pad the bounds so points never sit on the chart edge.
    const span = Math.max(hi - lo, 1);
    const y0 = lo - span * 0.08;
    const y1 = hi + span * 0.08;

    // X mapping: oldest historical on the left, newest forecast on
    // the right. Combine into one timeline so the forecast continues
    // where the historical ends.
    const allTs: number[] = [
      ...histAsc.map((h) => new Date(h.ts).getTime()),
      ...forecast.map((f) => new Date(f.ts).getTime()),
    ];
    const tMin = allTs.length ? Math.min(...allTs) : 0;
    const tMax = allTs.length ? Math.max(...allTs) : 1;
    const tSpan = tMax - tMin || 1;
    const xOf = (ts: string) => ((new Date(ts).getTime() - tMin) / tSpan) * 1000;
    const yOf = (v: number) => ((y1 - v) / (y1 - y0)) * 200;

    const hPts = histAsc.map((h) => ({
      ts: h.ts,
      value: h.value,
      x: xOf(h.ts),
      y: yOf(h.value),
    }));
    const p10P = forecast.map((f) => ({ x: xOf(f.ts), y: yOf(f.p10) }));
    const p50P = forecast.map((f) => ({ x: xOf(f.ts), y: yOf(f.p50) }));
    const p90P = forecast.map((f) => ({ x: xOf(f.ts), y: yOf(f.p90) }));

    return {
      yMin: y0,
      yMax: y1,
      histPts: hPts,
      p10Pts: p10P,
      p50Pts: p50P,
      p90Pts: p90P,
    };
  }, [histAsc, forecast]);

  // Path data for the confidence band (closed polygon p10 → p90).
  const bandPath = useMemo(() => {
    if (p10Pts.length === 0) return '';
    const top = p90Pts.map((p) => `${p.x},${p.y}`).join(' L ');
    const bottom = [...p10Pts]
      .reverse()
      .map((p) => `${p.x},${p.y}`)
      .join(' L ');
    return `M ${top} L ${bottom} Z`;
  }, [p10Pts, p90Pts]);

  const histPath = useMemo(
    () =>
      histPts.length > 0
        ? histPts.map((p) => `${p.x},${p.y}`).join(' L ')
        : '',
    [histPts],
  );
  const p50Path = useMemo(
    () =>
      p50Pts.length > 0
        ? p50Pts.map((p) => `${p.x},${p.y}`).join(' L ')
        : '',
    [p50Pts],
  );

  // Early-return AFTER all hooks (Rules of Hooks) — render the
  // empty-state when there's nothing to chart.
  if (histAsc.length === 0 && forecast.length === 0) {
    return (
      <div
        className="prediction-chart-empty"
        style={{
          minHeight: height,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          color: 'var(--text-muted, #6b7280)',
        }}
      >
        No data yet. POST /api/v1/predict/forecast to generate a forecast.
      </div>
    );
  }

  return (
    <div className="prediction-chart-wrap" style={{ position: 'relative' }}>
      <svg
        viewBox="0 0 1000 240"
        preserveAspectRatio="none"
        role="img"
        aria-label={`Prediction band — ${histAsc.length} historical + ${forecast.length} forecast points`}
        style={{ width: '100%', height }}
        onMouseLeave={() => setHover(null)}
      >
        {/* Confidence band: shaded area between p10 and p90. */}
        {bandPath ? (
          <motion.path
            d={bandPath}
            fill="var(--accent-soft, rgba(34, 211, 238, 0.15))"
            stroke="none"
            initial={reduce ? 'show' : 'hidden'}
            animate="show"
            variants={sparklineDraw}
          />
        ) : null}

        {/* Historical line — solid accent. */}
        {histPath ? (
          <motion.path
            d={`M ${histPath}`}
            fill="none"
            stroke="var(--accent)"
            strokeWidth="2"
            initial={reduce ? 'show' : 'hidden'}
            animate="show"
            variants={sparklineDraw}
          />
        ) : null}

        {/* Forecast p50 — dashed accent. */}
        {p50Path ? (
          <motion.path
            d={`M ${p50Path}`}
            fill="none"
            stroke="var(--accent)"
            strokeWidth="2"
            strokeDasharray="6 4"
            opacity="0.85"
            initial={reduce ? 'show' : 'hidden'}
            animate="show"
            variants={sparklineDraw}
          />
        ) : null}

        {/* Baseline. */}
        <line x1="0" x2="1000" y1="200" y2="200" stroke="var(--border)" strokeWidth="1" />

        {/* Historical points (hoverable). */}
        {histPts.map((p) => (
          <circle
            key={`h-${p.ts}`}
            cx={p.x}
            cy={p.y}
            r="3"
            fill="var(--accent)"
            stroke="var(--surface, #0e0e10)"
            strokeWidth="1"
            onMouseEnter={() =>
              setHover({ kind: 'historical', ts: p.ts, value: p.value, x: p.x, y: p.y })
            }
            style={{ cursor: 'pointer' }}
          >
            <title>{`${new Date(p.ts).toLocaleString()} • ${p.value.toFixed(2)}`}</title>
          </circle>
        ))}

        {/* Forecast p50 markers (hoverable). */}
        {p50Pts.map((p, i) => {
          const pt = forecast[i];
          return (
            <circle
              key={`f-${pt.ts}`}
              cx={p.x}
              cy={p.y}
              r="3"
              fill="var(--accent)"
              stroke="var(--surface, #0e0e10)"
              strokeWidth="1"
              opacity="0.85"
              onMouseEnter={() => setHover({ kind: 'forecast', pt, x: p.x, y: p.y })}
              style={{ cursor: 'pointer' }}
            >
              <title>{`${new Date(pt.ts).toLocaleString()} • p50 ${pt.p50.toFixed(2)} [${pt.p10.toFixed(2)}, ${pt.p90.toFixed(2)}]`}</title>
            </circle>
          );
        })}

        {/* Y axis labels. */}
        <text x="0" y="12" fontSize="10" fill="var(--text-muted, #6b7280)">
          {yMax.toFixed(1)}
        </text>
        <text x="0" y="198" fontSize="10" fill="var(--text-muted, #6b7280)">
          {yMin.toFixed(1)}
        </text>
      </svg>

      {hover ? (
        <div
          className="prediction-chart-tooltip"
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
          {metricName ? <div style={{ fontWeight: 600 }}>{metricName}</div> : null}
          <div style={{ opacity: 0.8 }}>{new Date(hover.kind === 'historical' ? hover.ts : hover.pt.ts).toLocaleString()}</div>
          {hover.kind === 'historical' ? (
            <div>
              value: <strong style={{ color: 'var(--accent)' }}>{hover.value.toFixed(2)}</strong>
            </div>
          ) : (
            <>
              <div>
                p50: <strong style={{ color: 'var(--accent)' }}>{hover.pt.p50.toFixed(2)}</strong>
              </div>
              <div>
                p10/p90: [{hover.pt.p10.toFixed(2)}, {hover.pt.p90.toFixed(2)}]
              </div>
            </>
          )}
        </div>
      ) : null}
    </div>
  );
}