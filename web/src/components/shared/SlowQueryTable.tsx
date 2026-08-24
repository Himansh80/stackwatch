/**
 * SlowQueryTable — compact table for one row of database_slow_queries.
 *
 * Used on DatabasePage → "Slow Queries" tab. Renders:
 *   - Query (truncated to 80 chars + title for full text)
 *   - Database pill (pg / my / ma / md)
 *   - Total Count (monospace number)
 *   - Total Duration (formatted: "5.2s")
 *   - Max Duration (formatted)
 *   - Avg Duration (formatted; amber >100ms, red >1s)
 *   - Last seen (relative time, only when present)
 *
 * Design tokens only (--green, --amber, --red, --muted, etc.).
 * No hex literals. Reuses .dash-status and .dash-status-* tones.
 */

export interface SlowQuery {
  query_hash: string;
  query_text: string;
  database?: string;
  total_count: number;
  total_duration_ms: number;
  max_duration_ms: number;
  avg_duration_ms: number;
  last_seen?: string | null;
}

export interface SlowQueryTableProps {
  queries: SlowQuery[];
  /** Optional click handler — if provided, rows become buttons. */
  onSelect?: (q: SlowQuery) => void;
}

const QUERY_PREVIEW_CHARS = 80;

const DATABASE_LABEL: Record<string, string> = {
  postgres: 'PG',
  mysql: 'MY',
  mariadb: 'MA',
  mongodb: 'MD',
};

function databaseLabel(d: string): string {
  return DATABASE_LABEL[d.toLowerCase()] ?? d.slice(0, 2).toUpperCase();
}

function formatMs(ms: number): string {
  if (!ms || ms <= 0) return '—';
  if (ms < 1000) return `${ms}ms`;
  const sec = ms / 1000;
  if (sec < 60) return `${sec.toFixed(sec < 10 ? 2 : 1)}s`;
  const min = Math.floor(sec / 60);
  return `${min}m ${Math.round(sec % 60)}s`;
}

function formatNumber(n: number): string {
  if (!n || n <= 0) return '0';
  if (n < 1000) return `${n}`;
  if (n < 1_000_000) return `${(n / 1000).toFixed(n < 10_000 ? 1 : 0)}k`;
  return `${(n / 1_000_000).toFixed(1)}M`;
}

function relativeTime(iso: string | null | undefined): string {
  if (!iso) return '—';
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return iso;
  const diff = Math.max(0, Date.now() - t);
  const sec = Math.floor(diff / 1000);
  if (sec < 60) return `${sec}s ago`;
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min}m ago`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr}h ago`;
  const day = Math.floor(hr / 24);
  return `${day}d ago`;
}

function avgTone(ms: number): string {
  if (ms >= 1000) return 'slow-query-avg slow-query-avg-bad';
  if (ms >= 100) return 'slow-query-avg slow-query-avg-warn';
  return 'slow-query-avg slow-query-avg-good';
}

export default function SlowQueryTable({ queries, onSelect }: SlowQueryTableProps) {
  if (queries.length === 0) {
    return (
      <div className="slow-query-table-empty">
        No slow queries yet.
      </div>
    );
  }
  return (
    <table className="dash-table slow-query-table">
      <thead>
        <tr>
          <th>Query</th>
          <th>Database</th>
          <th>Total</th>
          <th>Total duration</th>
          <th>Max</th>
          <th>Avg</th>
          <th>Last seen</th>
        </tr>
      </thead>
      <tbody>
        {queries.map((q) => {
          const preview = (q.query_text || '').length > QUERY_PREVIEW_CHARS
            ? `${q.query_text.slice(0, QUERY_PREVIEW_CHARS)}…`
            : q.query_text || '—';
          const row = (
            <>
              <td className="slow-query-cell-query">
                <code title={q.query_text || ''}>{preview}</code>
              </td>
              <td>
                <span className="dash-status dash-status-muted">
                  <span className="dash-status-dot" aria-hidden="true" />
                  {databaseLabel(q.database || '')}
                </span>
              </td>
              <td><code>{formatNumber(q.total_count)}</code></td>
              <td>{formatMs(q.total_duration_ms)}</td>
              <td>{formatMs(q.max_duration_ms)}</td>
              <td><span className={avgTone(q.avg_duration_ms)}>{formatMs(q.avg_duration_ms)}</span></td>
              <td><small>{relativeTime(q.last_seen ?? null)}</small></td>
            </>
          );
          if (onSelect) {
            return (
              <tr
                key={q.query_hash}
                className="slow-query-row-clickable"
                onClick={() => onSelect(q)}
              >
                {row}
              </tr>
            );
          }
          return (
            <tr key={q.query_hash}>
              {row}
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}