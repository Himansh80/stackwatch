import { useState } from 'react';

export interface LogEntryData {
  /** ISO timestamp string from the backend. */
  ts: string;
  /** 'error' | 'warn' | 'info' | 'debug' | other. */
  level: string;
  /** Originating service name. */
  service: string;
  /** Free-form log message. */
  message: string;
  /** Optional structured fields. */
  attributes?: Record<string, unknown>;
}

interface LogEntryProps {
  entry: LogEntryData;
}

/**
 * LogEntry — single Datadog-style log row.
 *
 * Layout (left-to-right):
 *   - timestamp (mono 11px, muted)
 *   - level pill (ERROR=red, WARN=amber, INFO=cyan, DEBUG=muted)
 *   - service name (mono 12px, indigo)
 *   - message (truncated to 200ch in collapsed mode)
 *
 * Hover expands the message and surfaces the attributes JSON in a
 * popover-style pre block. No external state needed — expansion is
 * per-row.
 */
export default function LogEntry({ entry }: LogEntryProps) {
  const [expanded, setExpanded] = useState(false);
  const level = (entry.level || 'info').toLowerCase();
  const hasMore = (entry.message?.length ?? 0) > 200;
  const display = expanded || !hasMore ? entry.message : `${entry.message.slice(0, 200)}…`;
  const ts = entry.ts ? new Date(entry.ts).toLocaleString() : '—';
  const hasAttrs = !!entry.attributes && Object.keys(entry.attributes).length > 0;
  return (
    <div
      className="log-entry"
      data-level={level}
      onMouseEnter={() => setExpanded(true)}
      onMouseLeave={() => setExpanded(false)}
    >
      <span className="log-entry-ts">{ts}</span>
      <span className={`log-entry-level log-entry-level-${level}`}>{level.toUpperCase()}</span>
      <span className="log-entry-service">{entry.service || 'unknown'}</span>
      <span className="log-entry-message" title={expanded ? entry.message : undefined}>
        {display}
      </span>
      {expanded && hasAttrs ? (
        <pre className="log-entry-attrs" aria-label="log attributes">
          {JSON.stringify(entry.attributes, null, 2)}
        </pre>
      ) : null}
    </div>
  );
}
