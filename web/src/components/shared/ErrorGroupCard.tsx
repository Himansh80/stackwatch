import { useState } from 'react';
import { motion, useReducedMotion } from '../../lib/motion';

/**
 * ErrorGroupCard — a single deduped error bucket.
 *
 * Visual contract (003-tier7-datadog-parity §"ErrorGroupCard.tsx"):
 *   - card surface, border, radius reusing the shared tokens
 *   - error message: bold, 14px, --text
 *   - occurrence count badge (--accent-soft bg + --accent text)
 *   - last-seen relative time (e.g. "3m ago")
 *   - first 5 lines of stack trace in monospace, scrollable
 *     on hover the card expands to show the full stack
 *   - no motion when reduced-motion is on (uses useReducedMotion)
 *
 * Props match the shape returned by GET /api/v1/rum/error-groups.
 */

export interface ErrorGroup {
  id: string;
  fingerprint: string;
  message: string;
  stack_trace: string;
  service?: string;
  source?: string;
  occurrence_count: number;
  first_seen_at: string;
  last_seen_at: string;
}

interface ErrorGroupCardProps {
  group: ErrorGroup;
}

function relativeTime(iso: string, now: number): string {
  if (!iso) return '—';
  const t = Date.parse(iso);
  if (!Number.isFinite(t)) return '—';
  const diff = Math.max(0, now - t);
  if (diff < 60_000) return 'just now';
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
  return `${Math.floor(diff / 86_400_000)}d ago`;
}

function previewLines(stack: string, n: number): string[] {
  if (!stack) return [];
  return stack.split(/\r?\n/).slice(0, n);
}

export default function ErrorGroupCard({ group }: ErrorGroupCardProps) {
  const reduce = useReducedMotion();
  const [expanded, setExpanded] = useState(false);
  // Relative time recomputes once per minute — cheap rAF-free
  // interval; we don't need sub-second precision.
  const now = Date.now();
  const preview = previewLines(group.stack_trace, 5);
  const hasMore = group.stack_trace.split(/\r?\n/).length > preview.length;

  return (
    <motion.article
      className="rum-error-card"
      role="article"
      aria-label={`Error group ${group.fingerprint}`}
      initial={{ opacity: 0, y: 4 }}
      animate={{ opacity: 1, y: 0 }}
      transition={reduce ? { duration: 0 } : { duration: 0.22 }}
    >
      <header className="rum-error-card-header">
        <strong className="rum-error-card-message" title={group.message}>
          {group.message}
        </strong>
        <span className="rum-error-card-count" aria-label="Occurrence count">
          ×{group.occurrence_count}
        </span>
      </header>
      <div className="rum-error-card-meta">
        <span className="rum-error-card-service">{group.service || 'browser'}</span>
        <span className="rum-error-card-source">{group.source || 'js'}</span>
        <span className="rum-error-card-time" title={group.last_seen_at}>
          last seen {relativeTime(group.last_seen_at, now)}
        </span>
      </div>
      {group.stack_trace ? (
        <pre
          className={`rum-error-card-stack ${expanded ? 'rum-error-card-stack-expanded' : ''}`}
          aria-label="Stack trace"
        >
          {(expanded ? group.stack_trace.split(/\r?\n/) : preview).map((line, i) => (
            <span key={i} className="rum-error-card-stack-line">{line}</span>
          ))}
        </pre>
      ) : null}
      {hasMore ? (
        <button
          type="button"
          className="rum-error-card-expand"
          onClick={() => setExpanded((v) => !v)}
        >
          {expanded ? 'Show less' : `Show full stack (${group.stack_trace.split(/\r?\n/).length} lines)`}
        </button>
      ) : null}
    </motion.article>
  );
}
