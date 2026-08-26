import LogEntry, { LogEntryData } from '../../components/shared/LogEntry';
import LogSearchBar from '../../components/shared/LogSearchBar';
import EmptyState from '../../components/shared/EmptyState';

interface LogSearchTabProps {
  entries: LogEntryData[];
  busy: boolean;
  error: string;
  onSearch: (q: string) => void;
}

export default function LogSearchTab({ entries, busy, error, onSearch }: LogSearchTabProps) {
  return (
    <section className="dash-section">
      <span className="dash-eyebrow">Search</span>
      <h2 className="dash-section-title">Log explorer</h2>
      <LogSearchBar onSearch={(q) => onSearch(q)} />
      {error ? <div className="dash-error" role="alert">{error}</div> : null}
      {busy ? (
        <p className="dash-section-lede">Searching…</p>
      ) : entries.length === 0 ? (
        <EmptyState
          illustration={<span style={{ fontSize: 36 }}>↹</span>}
          headline="No log entries"
          subhead="Run a search, or check the Patterns tab to see what's flowing through."
        />
      ) : (
        <div className="log-entry-list">
          {entries.map((entry, idx) => (
            <LogEntry key={`${entry.ts}-${idx}`} entry={entry} />
          ))}
        </div>
      )}
    </section>
  );
}
