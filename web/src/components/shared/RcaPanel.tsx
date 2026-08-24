import { motion } from '../../lib/motion';

/**
 * RcaHint — one row from GET /api/v1/correlations/rca/:alert_id
 * (or nested inside an alertCorrelationRow.top_rca_hint).
 *
 * Matches the JSON shape returned by handlers_correlations.go so
 * this component is drop-in for any consumer that calls the
 * endpoint. We render `similar_past_incidents` as a flat array
 * (the handler returns it as JSONB → string[] in JSON) and compute
 * a `similar_count` badge in the parent.
 */
export interface RcaHint {
  id?: string;
  alert_id?: string;
  likely_root: string;
  confidence: number;       // 0..1
  reasoning: string;
  similar_past_incidents?: unknown[];
  created_at?: string;
}

interface RcaPanelProps {
  hints: RcaHint[];
  /** Max number of hints to render. Default 5 (spec). */
  limit?: number;
  /** Optional header label override. Default "Root cause analysis". */
  title?: string;
}

/**
 * Map a 0..1 confidence to a CSS token color. Mirrors the
 * CorrelationCard similarity-bar palette (green > 0.7, amber 0.4-0.7,
 * red < 0.4) so the dashboard reads consistently.
 */
function confidenceTone(c: number): { bg: string; label: string } {
  if (c >= 0.7) return { bg: 'var(--green)', label: 'high' };
  if (c >= 0.4) return { bg: 'var(--amber)', label: 'medium' };
  return { bg: 'var(--red)', label: 'low' };
}

function truncate(s: string, n: number): string {
  if (s.length <= n) return s;
  return `${s.slice(0, n).trimEnd()}…`;
}

/**
 * RcaPanel — Datadog-style root-cause-analysis panel.
 *
 * Renders the top N RCA hints (default 5) for the current alert /
 * correlation context. Each row shows:
 *   - Likely-root badge (left)
 *   - Confidence progress bar (Datadog-style thin filled bar, color
 *     graded by confidence tier)
 *   - Reasoning preview (first 80 chars + native title attribute
 *     for the full string on hover)
 *   - Similar past incidents count badge (e.g. "3 similar")
 *
 * Empty state: when `hints` is empty, render a one-line "no RCA
 * hints yet" message so the panel doesn't disappear (Datadog keeps
 * the surface mounted so users see the feature exists).
 *
 * Reuses existing tokens (--green, --amber, --red, --surface,
 * --border) via inline styles. Motion: reuses kpiStagger from the
 * shared motion module so the dashboard row animation matches the
 * other section strips — no new variants introduced.
 */
export default function RcaPanel({
  hints,
  limit = 5,
  title = 'Root cause analysis',
}: RcaPanelProps) {
  const top = hints.slice(0, limit);

  if (top.length === 0) {
    return (
      <section
        className="rca-panel"
        style={{
          background: 'var(--surface)',
          border: '1px solid var(--border)',
          borderRadius: 'var(--radius-lg)',
          padding: 16,
        }}
      >
        <div
          className="rca-panel-head"
          style={{
            display: 'flex',
            alignItems: 'baseline',
            justifyContent: 'space-between',
            marginBottom: 8,
          }}
        >
          <strong style={{ color: 'var(--text)' }}>{title}</strong>
          <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
            top {limit} suggestions
          </span>
        </div>
        <p
          style={{
            color: 'var(--text-muted)',
            fontSize: 12,
            margin: 0,
          }}
        >
          No RCA hints yet. Hints appear here when an alert is part of a correlation
          group (Tier 8.3 background correlator) or when a manual correlation is
          created.
        </p>
      </section>
    );
  }

  return (
    <section
      className="rca-panel"
      style={{
        background: 'var(--surface)',
        border: '1px solid var(--border)',
        borderRadius: 'var(--radius-lg)',
        padding: 16,
      }}
    >
      <div
        className="rca-panel-head"
        style={{
          display: 'flex',
          alignItems: 'baseline',
          justifyContent: 'space-between',
          marginBottom: 12,
        }}
      >
        <strong style={{ color: 'var(--text)' }}>{title}</strong>
        <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
          top {top.length} suggestion{top.length === 1 ? '' : 's'}
        </span>
      </div>

      <motion.ul
        className="rca-panel-list"
        style={{
          listStyle: 'none',
          padding: 0,
          margin: 0,
          display: 'flex',
          flexDirection: 'column',
          gap: 10,
        }}
        initial="hidden"
        animate="show"
        variants={{
          hidden: { opacity: 0 },
          show: {
            opacity: 1,
            transition: { staggerChildren: 0.04 },
          },
        }}
      >
        {top.map((h, idx) => {
          const tone = confidenceTone(h.confidence);
          const pct = Math.max(0, Math.min(100, h.confidence * 100));
          const similarCount = Array.isArray(h.similar_past_incidents)
            ? h.similar_past_incidents.length
            : 0;
          return (
            <motion.li
              key={h.id ?? `${h.likely_root}-${idx}`}
              className="rca-panel-row"
              style={{
                display: 'grid',
                gridTemplateColumns: 'minmax(140px, 1fr) minmax(160px, 2fr) auto',
                gap: 12,
                alignItems: 'center',
                padding: '8px 10px',
                background: 'var(--surface-2, rgba(255,255,255,0.02))',
                border: '1px solid var(--border)',
                borderRadius: 'var(--radius-md, 8px)',
              }}
              variants={{
                hidden: { opacity: 0, y: 4 },
                show: { opacity: 1, y: 0 },
              }}
            >
              {/* Likely-root badge */}
              <span
                className="rca-panel-badge"
                style={{
                  fontFamily: 'var(--mono, monospace)',
                  fontSize: 12,
                  fontWeight: 700,
                  color: 'var(--text)',
                  letterSpacing: 0.2,
                }}
                title={h.likely_root}
              >
                {h.likely_root}
              </span>

              {/* Reasoning preview + confidence bar */}
              <div
                className="rca-panel-reason"
                style={{ display: 'flex', flexDirection: 'column', gap: 6, minWidth: 0 }}
              >
                <span
                  style={{
                    color: 'var(--text-muted)',
                    fontSize: 12,
                    whiteSpace: 'nowrap',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                  }}
                  title={h.reasoning}
                >
                  {truncate(h.reasoning, 80)}
                </span>
                <div
                  className="rca-panel-bar"
                  aria-label={`confidence ${pct.toFixed(0)}% (${tone.label})`}
                  style={{
                    height: 4,
                    width: '100%',
                    background: 'var(--surface, rgba(255,255,255,0.04))',
                    borderRadius: 999,
                    overflow: 'hidden',
                    border: '1px solid var(--border)',
                  }}
                >
                  <div
                    style={{
                      height: '100%',
                      width: `${pct}%`,
                      background: tone.bg,
                      transition: 'width 200ms ease',
                    }}
                  />
                </div>
              </div>

              {/* Similar count badge + confidence pill */}
              <div
                className="rca-panel-meta"
                style={{ display: 'flex', alignItems: 'center', gap: 8, flexShrink: 0 }}
              >
                <span
                  className="threat-card-meta-pill"
                  title={`${similarCount} similar past incident${similarCount === 1 ? '' : 's'}`}
                  style={{
                    padding: '2px 8px',
                    background: 'var(--surface-2, rgba(255,255,255,0.04))',
                    border: '1px solid var(--border)',
                    borderRadius: 999,
                    fontSize: 11,
                    color: 'var(--text-muted)',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {similarCount > 0 ? `${similarCount} similar` : 'no similar'}
                </span>
                <span
                  style={{
                    fontSize: 11,
                    fontWeight: 700,
                    color: tone.bg,
                    fontVariantNumeric: 'tabular-nums',
                    minWidth: 36,
                    textAlign: 'right',
                  }}
                  title={`confidence ${pct.toFixed(0)}%`}
                >
                  {pct.toFixed(0)}%
                </span>
              </div>
            </motion.li>
          );
        })}
      </motion.ul>
    </section>
  );
}