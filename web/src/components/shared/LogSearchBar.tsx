import { FormEvent, useState } from 'react';
import Button from './Button';
import Input from './Input';
import Select from './Select';

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
 * Tier 20 Phase G: refactored to shared Button + Input + Select.
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
      <Input
        type="search"
        label="Query"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="search messages…"
        aria-label="log query"
        maxLength={1024}
      />
      <Select
        label="Level"
        value={level}
        onChange={(e) => setLevel(e.target.value)}
        aria-label="log level"
        options={[
          { value: '', label: 'any' },
          { value: 'error', label: 'error' },
          { value: 'warn', label: 'warn' },
          { value: 'info', label: 'info' },
          { value: 'debug', label: 'debug' },
        ]}
      />
      <Input
        type="text"
        label="Service"
        value={service}
        onChange={(e) => setService(e.target.value)}
        placeholder="api"
        aria-label="service filter"
        maxLength={128}
      />
      <Select
        label="Range"
        value={timeRange}
        onChange={(e) => setTimeRange(e.target.value as LogFilters['timeRange'])}
        aria-label="time range"
        options={[
          { value: '3600', label: 'last 1h' },
          { value: '86400', label: 'last 24h' },
          { value: '604800', label: 'last 7d' },
          { value: 'custom', label: 'custom' },
        ]}
      />
      <Button type="submit" variant="primary" size="md">
        Search
      </Button>
    </form>
  );
}
