import { useMemo } from 'react';
import { motion, sparklineDraw, useReducedMotion } from '../../lib/motion';

/**
 * ResourceWaterfall — chronological list of network requests.
 *
 * Visual contract (003-tier7-datadog-parity §"ResourceWaterfall.tsx"):
 *   - one row per resource
 *   - URL truncated to 40ch (hover shows full URL)
 *   - resource type pill (JS / CSS / IMG / XHR / FONT / OTHER)
 *   - duration bar (width = % of longest request) colored by status
 *   - size formatted in KB / MB (0/unknown renders as —)
 *   - status pill (200=green, 3xx=neutral, 4xx=amber, 5xx=red)
 *   - time offset relative to the row's ts (mm:ss from session start)
 *   - each bar uses sparklineDraw variant for entry
 *
 * Motion: each bar uses `sparklineDraw` for entry; honors
 * `prefers-reduced-motion` via `useReducedMotion()`.
 */

export interface WaterfallResource {
  id: string;
  url: string;
  resource_type: string;
  duration_ms: number;
  size_bytes: number;
  status: number;
  /** ISO-8601 timestamp from the API. */
  ts: string;
}

interface ResourceWaterfallProps {
  resources: WaterfallResource[];
}

function formatBytes(b: number): string {
  if (!Number.isFinite(b) || b <= 0) return '—';
  if (b >= 1_048_576) return `${(b / 1_048_576).toFixed(2)} MB`;
  if (b >= 1024) return `${(b / 1024).toFixed(1)} KB`;
  return `${b} B`;
}

function formatOffset(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) return '0:00';
  const totalSec = Math.floor(ms / 1000);
  const m = Math.floor(totalSec / 60);
  const s = totalSec % 60;
  return `${m}:${s.toString().padStart(2, '0')}`;
}

function statusTone(status: number): 'good' | 'neutral' | 'warn' | 'bad' {
  if (!status) return 'neutral';
  if (status >= 500) return 'bad';
  if (status >= 400) return 'warn';
  if (status >= 300) return 'neutral';
  return 'good';
}

const KNOWN_TYPES = new Set([
  'script', 'stylesheet', 'image', 'xhr', 'fetch', 'font', 'document', 'other',
]);

function typeLabel(t: string): string {
  const lower = (t || '').toLowerCase();
  if (!KNOWN_TYPES.has(lower)) return 'OTHER';
  if (lower === 'stylesheet') return 'CSS';
  if (lower === 'script') return 'JS';
  return lower.toUpperCase();
}

export default function ResourceWaterfall({ resources }: ResourceWaterfallProps) {
  const reduce = useReducedMotion();
  const { maxDur, baseTs } = useMemo(() => {
    if (!resources.length) return { maxDur: 1, baseTs: '' };
    const max = Math.max(...resources.map((r) => Math.max(r.duration_ms, 1)), 1);
    // Earliest timestamp = session t=0; we use that as the offset base.
    const sortedTs = resources.map((r) => r.ts).filter(Boolean).sort();
    const base = sortedTs[0] || '';
    return { maxDur: max, baseTs: base };
  }, [resources]);

  if (!resources.length) {
    return (
      <div className="rum-waterfall-empty" role="status">
        No network requests captured yet for this session.
      </div>
    );
  }
  const baseMs = baseTs ? Date.parse(baseTs) : 0;

  return (
    <div className="rum-waterfall" role="table" aria-label="Session resource waterfall">
      <div className="rum-waterfall-header" role="row">
        <span role="columnheader">URL</span>
        <span role="columnheader">Type</span>
        <span role="columnheader">Duration</span>
        <span role="columnheader">Size</span>
        <span role="columnheader">Status</span>
        <span role="columnheader">Time</span>
      </div>
      <div className="rum-waterfall-rows" role="rowgroup">
        {resources.map((r) => {
          const tone = statusTone(r.status);
          const pct = (r.duration_ms / maxDur) * 100;
          const tsMs = r.ts ? Date.parse(r.ts) : 0;
          const offsetMs = baseMs && tsMs ? Math.max(0, tsMs - baseMs) : 0;
          const url = r.url || '';
          const trunc = url.length > 40 ? `${url.slice(0, 40)}…` : url;
          return (
            <motion.div
              key={r.id}
              role="row"
              className={`rum-waterfall-row rum-waterfall-row-${tone}`}
              initial="hidden"
              animate="show"
              variants={reduce ? undefined : sparklineDraw}
            >
              <span className="rum-waterfall-url" title={url} role="cell">
                {trunc}
              </span>
              <span className={`rum-waterfall-type rum-waterfall-type-${r.resource_type || 'other'}`} role="cell">
                {typeLabel(r.resource_type)}
              </span>
              <span className="rum-waterfall-duration" role="cell">
                <span
                  className="rum-waterfall-bar"
                  style={{ width: `${Math.max(pct, 2)}%` }}
                  aria-hidden="true"
                />
                <span className="rum-waterfall-duration-label">{r.duration_ms}ms</span>
              </span>
              <span className="rum-waterfall-size" role="cell">
                {formatBytes(r.size_bytes)}
              </span>
              <span className={`rum-waterfall-status rum-waterfall-status-${tone}`} role="cell">
                {r.status || '—'}
              </span>
              <span className="rum-waterfall-time" role="cell">
                {formatOffset(offsetMs)}
              </span>
            </motion.div>
          );
        })}
      </div>
    </div>
  );
}
