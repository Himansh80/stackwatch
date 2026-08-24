import { useMemo, useState } from 'react';
import { motion, sparklineDraw, useReducedMotion } from '../../lib/motion';

/**
 * FlameGraph — SVG flame graph renderer.
 *
 * Renders an array of spans as horizontal bars whose width is the
 * span's duration as a fraction of the longest span. Each row is a
 * depth level (0 = root). Spans are colored by service (hash → token)
 * so cross-service spans are visually distinct.
 *
 * Visual contract (003-tier7-datadog-parity §"FlameGraph.tsx"):
 *   - SVG (no D3 dep)
 *   - row height 18px, gap 2px
 *   - each bar uses design-token colors (--accent / --green / --amber /
 *     --red / --indigo / --violet via a stable hash → tone map)
 *   - hover tooltip shows span name + duration_us
 *   - motion: each bar uses sparklineDraw variant for entry
 *
 * Honors `prefers-reduced-motion` via `useReducedMotion()`.
 */

export interface FlameSpan {
  span_id: string;
  name: string;
  duration_us: number;
  status: string;
  service_id: string;
  depth: number;
}

interface FlameGraphProps {
  spans: FlameSpan[];
  /** Total SVG width in px (bars are sized relative to this). */
  width?: number;
  /** Optional click handler — drills into a span. */
  onSpanClick?: (span: FlameSpan) => void;
}

// Stable hash of a string to a 0..5 tone index.
function toneFor(key: string): number {
  let h = 0;
  for (let i = 0; i < key.length; i++) {
    h = (h * 31 + key.charCodeAt(i)) | 0;
  }
  return Math.abs(h) % 6;
}

const TONES = ['cyan', 'green', 'amber', 'red', 'indigo', 'violet'] as const;

function formatDur(us: number): string {
  if (!Number.isFinite(us) || us <= 0) return '—';
  if (us >= 1_000_000) return `${(us / 1_000_000).toFixed(2)}s`;
  if (us >= 1_000) return `${(us / 1_000).toFixed(1)}ms`;
  return `${us}µs`;
}

export default function FlameGraph({
  spans,
  width = 880,
  onSpanClick,
}: FlameGraphProps) {
  const reduce = useReducedMotion();
  const [hover, setHover] = useState<FlameSpan | null>(null);

  // Pre-compute layout — used by the bars below.
  const layout = useMemo(() => {
    const rowH = 18;
    const gap = 2;
    if (!spans.length) return { rows: 0, totalUS: 0, height: 0, maxDepth: 0, rowH, gap };
    const maxUS = Math.max(...spans.map((s) => Math.max(s.duration_us, 1)), 1);
    const maxDepth = Math.max(...spans.map((s) => s.depth), 0);
    const height = (maxDepth + 1) * (rowH + gap) + 8;
    return { rows: maxDepth + 1, totalUS: maxUS, height, maxDepth, rowH, gap };
  }, [spans]);

  if (!spans.length) {
    return (
      <div className="apm-flame-empty" role="status">
        No spans yet — ingest a trace to see its flame graph.
      </div>
    );
  }

  const usableWidth = width - 24; // 12px padding each side
  return (
    <div className="apm-flame" role="img" aria-label="Flame graph">
      <svg
        viewBox={`0 0 ${width} ${layout.height}`}
        width="100%"
        preserveAspectRatio="xMinYMid meet"
        style={{ maxWidth: width }}
      >
        {spans.map((s) => {
          const tone = TONES[toneFor(s.service_id + ':' + s.name)];
          const w = Math.max((s.duration_us / layout.totalUS) * usableWidth, 2);
          const x = 12;
          const y = s.depth * (layout.rowH + layout.gap) + 4;
          const isError = s.status === 'error';
          const cls = `apm-flame-bar apm-flame-bar-${tone}${isError ? ' apm-flame-bar-error' : ''}`;
          return (
            <motion.g
              key={s.span_id}
              onMouseEnter={() => setHover(s)}
              onMouseLeave={() => setHover(null)}
              onClick={() => onSpanClick && onSpanClick(s)}
              style={{ cursor: onSpanClick ? 'pointer' : 'default' }}
              initial="hidden"
              animate="show"
              variants={reduce ? undefined : sparklineDraw}
            >
              <motion.rect
                x={x}
                y={y}
                width={w}
                height={layout.rowH}
                rx={3}
                ry={3}
                className={cls}
                variants={reduce ? undefined : sparklineDraw}
              />
              {w > 36 ? (
                <text
                  x={x + 6}
                  y={y + layout.rowH / 2 + 4}
                  className="apm-flame-label"
                  fill="currentColor"
                >
                  {s.name.length > Math.floor(w / 6) ? s.name.slice(0, Math.floor(w / 6) - 1) + '…' : s.name}
                </text>
              ) : null}
            </motion.g>
          );
        })}
      </svg>
      {hover ? (
        <div className="apm-flame-tooltip" role="tooltip">
          <strong>{hover.name}</strong>
          <span>{formatDur(hover.duration_us)}</span>
          <span className="apm-flame-tooltip-status">{hover.status}</span>
        </div>
      ) : null}
    </div>
  );
}
