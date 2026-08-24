import { useState } from 'react';
import { motion, buttonSpring, useReducedMotion } from '../../lib/motion';

/**
 * CorrelationCard — Tier 8.3 (D9 Phase 3) UI surface for a single
 * correlation group. Mirrors the alertCorrelationRow JSON shape
 * returned by GET /api/v1/correlations/groups (the optional
 * TopRCAHint field is what feeds the quote-block at the bottom).
 *
 * Layout (Datadog-style):
 *   - Header: short correlation_id + member count badge + auto/manual tag
 *   - Similarity score rendered as a thin progress bar (0..1 → 0..100%)
 *   - Root alert badge + list of member alerts as small chips
 *   - Top RCA hint as a quote-block with confidence indicator
 *   - Footer: thumbs up / down feedback buttons + expand toggle
 *
 * The card itself is presentational — the parent
 * CorrelationsSection owns the API calls and decides what to do
 * when feedback is submitted. Reuses existing tokens (--red,
 * --amber, --accent, --surface, --border) via the inline styles.
 */

export interface CorrelationMember {
  alert_id: string;
  source: 'anomaly' | 'predict' | string;
  severity: 'info' | 'warning' | 'critical' | string;
  metric_name: string;
  server_id?: string;
  detected_at: string;
}

export interface CorrelationRCAHint {
  likely_root: string;
  confidence: number;
  reasoning: string;
}

export interface CorrelationGroup {
  id: string;
  correlation_id: string;
  root_alert_id: string;
  member_alert_ids: string[];
  member_count: number;
  similarity_score: number;
  auto_detected: boolean;
  created_at: string;
  top_rca_hint?: CorrelationRCAHint | null;
  members?: CorrelationMember[];
}

interface CorrelationCardProps {
  group: CorrelationGroup;
  busy?: boolean;
  onFeedback?: (groupId: string, useful: boolean) => void;
  onExpand?: (groupId: string) => void;
}

const SEVERITY_COLOR: Record<string, string> = {
  critical: 'var(--red)',
  warning: 'var(--amber)',
  info: 'var(--green)',
};

function severityColor(sev: string): string {
  return SEVERITY_COLOR[sev] ?? 'var(--text-muted, #6b7280)';
}

function shortId(id: string): string {
  return id.length > 8 ? `${id.slice(0, 8)}…` : id;
}

/**
 * Render the similarity score as a thin progress bar. score is
 * 0..1; we clamp + scale to 0..100% so the bar fills proportionally.
 */
function SimilarityBar({ score }: { score: number }) {
  const pct = Math.max(0, Math.min(100, score * 100));
  const tone =
    score >= 0.85
      ? 'var(--green)'
      : score >= 0.6
        ? 'var(--amber)'
        : 'var(--red)';
  return (
    <div
      aria-label={`similarity ${pct.toFixed(0)}%`}
      style={{
        height: 4,
        width: '100%',
        background: 'var(--surface-2, var(--surface))',
        borderRadius: 999,
        overflow: 'hidden',
        border: '1px solid var(--border)',
      }}
    >
      <div
        style={{
          height: '100%',
          width: `${pct}%`,
          background: tone,
          transition: 'width 200ms ease-out',
        }}
      />
    </div>
  );
}

export default function CorrelationCard({
  group,
  busy,
  onFeedback,
  onExpand,
}: CorrelationCardProps) {
  const reduce = useReducedMotion();
  const [expanded, setExpanded] = useState(false);
  const [feedbackSent, setFeedbackSent] = useState<null | 'up' | 'down'>(null);

  const handleExpand = () => {
    setExpanded((prev) => !prev);
    if (onExpand && !expanded) onExpand(group.correlation_id);
  };

  const handleFeedback = (useful: boolean) => {
    if (feedbackSent) return;
    setFeedbackSent(useful ? 'up' : 'down');
    if (onFeedback) onFeedback(group.correlation_id, useful);
  };

  const hint = group.top_rca_hint;

  return (
    <motion.article
      className="threat-card"
      whileHover={reduce ? undefined : { y: -1 }}
      transition={buttonSpring.transition}
      style={{ position: 'relative' }}
    >
      <div className="threat-card-top">
        <span
          className={`dash-status ${
            group.auto_detected ? 'dash-status-up' : 'dash-status-stale'
          }`}
        >
          <span className="dash-status-dot" aria-hidden="true" />
          {group.auto_detected ? 'auto' : 'manual'}
        </span>
        <strong className="threat-card-type">
          group {shortId(group.correlation_id)}
        </strong>
        <span className="threat-card-time" title={group.created_at}>
          {new Date(group.created_at).toLocaleString()}
        </span>
      </div>

      <p
        className="threat-card-desc"
        style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}
      >
        <span>
          <strong style={{ color: 'var(--red)' }}>{group.member_count}</strong> members · root{' '}
          <code style={{ color: 'var(--red)' }}>{shortId(group.root_alert_id)}</code>
        </span>
        <span style={{ flex: 1, minWidth: 120 }}>
          <SimilarityBar score={group.similarity_score} />
        </span>
        <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
          sim {(group.similarity_score * 100).toFixed(0)}%
        </span>
      </p>

      {/* Member chips. Show root + first few member ids as
          short pills. Click "Show all" to reveal the rest. */}
      <div
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          gap: 6,
          marginTop: 6,
          marginBottom: hint ? 8 : 0,
        }}
      >
        {[group.root_alert_id, ...group.member_alert_ids]
          .slice(0, expanded ? undefined : 4)
          .map((id, idx) => (
            <span
              key={`${id}-${idx}`}
              className="threat-card-meta-pill"
              title={id}
              style={{
                borderColor: idx === 0 ? 'var(--red)' : 'var(--border)',
                borderWidth: idx === 0 ? 1.5 : 1,
              }}
            >
              <span className="threat-card-meta-label">{idx === 0 ? 'root' : 'm'}</span>
              <code>{shortId(id)}</code>
            </span>
          ))}
        {!expanded && group.member_alert_ids.length > 3 ? (
          <button
            type="button"
            onClick={handleExpand}
            className="threat-card-resolve-btn"
            style={{ border: 'none', background: 'transparent', cursor: 'pointer' }}
          >
            +{group.member_alert_ids.length - 3} more
          </button>
        ) : null}
      </div>

      {/* Expanded members list with severity colors. */}
      {expanded && group.members && group.members.length > 0 ? (
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: 4,
            marginBottom: hint ? 8 : 0,
            padding: 8,
            background: 'var(--surface-2, var(--surface))',
            border: '1px solid var(--border)',
            borderRadius: 8,
          }}
        >
          {group.members.map((m) => (
            <div
              key={m.alert_id}
              style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 12 }}
            >
              <span
                style={{
                  width: 8,
                  height: 8,
                  borderRadius: '50%',
                  background: severityColor(m.severity),
                  flexShrink: 0,
                }}
                aria-hidden="true"
              />
              <span style={{ color: 'var(--text)' }}>{m.metric_name}</span>
              <code style={{ color: 'var(--text-muted)', fontSize: 11 }}>
                {shortId(m.alert_id)}
              </code>
              <span style={{ color: 'var(--text-muted)', fontSize: 11, marginLeft: 'auto' }}>
                {m.severity} · {m.source}
              </span>
            </div>
          ))}
        </div>
      ) : null}

      {/* Top RCA hint as a quote-block with confidence. */}
      {hint ? (
        <blockquote
          style={{
            margin: 0,
            padding: '8px 12px',
            borderLeft: `3px solid ${
              hint.confidence >= 0.7
                ? 'var(--accent)'
                : hint.confidence >= 0.4
                  ? 'var(--amber)'
                  : 'var(--red)'
            }`,
            background: 'var(--surface-2, var(--surface))',
            borderRadius: 6,
            color: 'var(--text)',
            fontSize: 13,
          }}
        >
          <div
            style={{
              display: 'flex',
              alignItems: 'baseline',
              gap: 8,
              marginBottom: 4,
            }}
          >
            <strong style={{ color: 'var(--accent)' }}>{hint.likely_root}</strong>
            <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
              confidence {(hint.confidence * 100).toFixed(0)}%
            </span>
          </div>
          <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>{hint.reasoning}</div>
        </blockquote>
      ) : null}

      <div className="threat-card-meta" style={{ marginTop: 10 }}>
        <button
          type="button"
          className="threat-card-resolve-btn"
          onClick={handleExpand}
          disabled={busy}
        >
          {expanded ? 'Collapse ↑' : 'Expand ↓'}
        </button>
        {feedbackSent === 'up' ? (
          <span style={{ color: 'var(--green)', fontSize: 12 }}>✓ marked helpful</span>
        ) : feedbackSent === 'down' ? (
          <span style={{ color: 'var(--amber)', fontSize: 12 }}>✓ marked not helpful</span>
        ) : (
          <>
            <button
              type="button"
              className="threat-card-resolve-btn"
              onClick={() => handleFeedback(true)}
              disabled={busy}
              aria-label="Mark helpful"
              title="This correlation was helpful"
            >
              👍
            </button>
            <button
              type="button"
              className="threat-card-resolve-btn"
              onClick={() => handleFeedback(false)}
              disabled={busy}
              aria-label="Mark not helpful"
              title="This correlation was wrong / unhelpful"
            >
              👎
            </button>
          </>
        )}
      </div>
    </motion.article>
  );
}