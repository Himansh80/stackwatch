import { useState } from 'react';

/**
 * Reusable list-page filter bar.
 *
 * Pattern from Datadog / Better Stack / Linear: every list view has
 * a search input + status filter chips + an optional actions slot on
 * the right. This component provides that consistent header so we
 * don't have to rebuild it on every page.
 *
 * Props:
 *   - search + onSearchChange: controlled text query
 *   - chips: array of {id, label, count?, active?, onClick}
 *   - placeholder: search input placeholder
 *   - actions: optional ReactNode for right-side buttons (e.g. "Add")
 *   - onClear: optional callback when the X clear button is pressed
 */

export interface FilterChip {
  id: string;
  label: string;
  count?: number;
  active?: boolean;
  onClick: () => void;
}

interface FilterBarProps {
  search: string;
  onSearchChange: (next: string) => void;
  chips?: FilterChip[];
  placeholder?: string;
  actions?: React.ReactNode;
  onClear?: () => void;
  className?: string;
  ariaLabel?: string;
}

export default function FilterBar({
  search,
  onSearchChange,
  chips = [],
  placeholder = 'Search...',
  actions,
  onClear,
  className,
  ariaLabel = 'Filter results',
}: FilterBarProps) {
  const [focused, setFocused] = useState(false);
  const showClear = search.length > 0 || chips.some((c) => c.active);

  return <div className={`sw-filter-bar ${className ?? ''}`} role="search" aria-label={ariaLabel}>
    <div className={`sw-filter-search ${focused ? 'sw-filter-search-focus' : ''}`}>
      <span className="sw-filter-search-icon" aria-hidden="true">⌕</span>
      <input
        type="search"
        value={search}
        onChange={(e) => onSearchChange(e.target.value)}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        placeholder={placeholder}
        aria-label={`${ariaLabel} search`}
        autoComplete="off"
        spellCheck={false}
      />
      {search.length > 0 && <button
        type="button"
        className="sw-filter-search-clear"
        onClick={() => { onSearchChange(''); onClear?.(); }}
        aria-label="Clear search"
        title="Clear"
      >×</button>}
    </div>
    {chips.length > 0 && <div className="sw-filter-chips" role="group" aria-label="Status filters">
      {chips.map((chip) => (
        <button
          key={chip.id}
          type="button"
          className={`sw-filter-chip ${chip.active ? 'sw-filter-chip-active' : ''}`}
          onClick={chip.onClick}
          aria-pressed={chip.active || false}
        >
          <span>{chip.label}</span>
          {chip.count !== undefined && <span className="sw-filter-chip-count">{chip.count}</span>}
        </button>
      ))}
    </div>}
    {actions && <div className="sw-filter-actions">{actions}</div>}
    {showClear && onClear && chips.some((c) => c.active) && <button
      type="button"
      className="sw-filter-reset"
      onClick={onClear}
    >Clear filters</button>}
  </div>;
}
