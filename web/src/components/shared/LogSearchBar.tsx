import { FormEvent, useState } from 'react';

export interface LogFilters {
  /** Free-text query string. */
  query: string;
  /** Optional level filter: error | warn | info | debug. Empty = any. */
  level: string;
  /** Optional service name filter. Empty = any. */
  service: string;
  /** Time range in seconds, picked from the time-range menu. */
  timeRange: '3600' | '86400' | '604800' | 'custom';
  /** When timeRange === 'custom', callers may set this in seconds. */
  customSeconds?: number;
}

interface LogSearchBarProps {
  onSearch: (q: string, filters: LogFilters) => void;
  initialFilters?: LogFilters;
}

/**
 * LogSearchBar — search input + level / service / time-range filters.
 *
 * Submits on Enter or button click. All inputs are uncontrolled-once-
 * submitted (caller owns the resulting state via onSearch).
 *
 * Visual contract: reuses the same form-row spacing as
 * .apm-register-form so the Logs page sits flush next to APM.
 */
export default function LogSearchBar({ onSearch, initialFilters }: LogSearchBarProps) {
  const [query, setQuery] = useState(initialFilters?.query ?? '');
  const [level, setLevel] = useState(initialFilters?.level ?? '');
  const [service, setService] = useState(initialFilters?.service ?? '');
  const [timeRange, setTimeRange] = useState<LogFilters['timeRange']>(
    initialFilters?.timeRange ?? '86400'
  );

  const submit = (event: FormEvent) => {
    event.preventDefault();
    onSearch(query, { query, level, service, timeRange });
  };

  return (
    <form className="log-search-bar" onSubmit={submit} role="search">
      <label className="log-search-bar-field">
        <span>Query</span>
        <input
          type="search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="search messages…"
          aria-label="log query"
          maxLength={1024}
        />
      </label>
      <label className="log-search-bar-field">
        <span>Level</span>
        <select
          value={level}
          onChange={(e) => setLevel(e.target.value)}
          aria-label="log level"
        >
          <option value="">any</option>
          <option value="error">error</option>
          <option value="warn">warn</option>
          <option value="info">info</option>
          <option value="debug">debug</option>
        </select>
      </label>
      <label className="log-search-bar-field">
        <span>Service</span>
        <input
          type="text"
          value={service}
          onChange={(e) => setService(e.target.value)}
          placeholder="api"
          aria-label="service filter"
          maxLength={128}
        />
      </label>
      <label className="log-search-bar-field">
        <span>Range</span>
        <select
          value={timeRange}
          onChange={(e) => setTimeRange(e.target.value as LogFilters['timeRange'])}
          aria-label="time range"
        >
          <option value="3600">last 1h</option>
          <option value="86400">last 24h</option>
          <option value="604800">last 7d</option>
          <option value="custom">custom</option>
        </select>
      </label>
      <button type="submit" className="sw-button sw-button-primary">
        Search
      </button>
    </form>
  );
}
