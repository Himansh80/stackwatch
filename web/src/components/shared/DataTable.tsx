/**
 * DataTable.tsx — shared DataTable component (sortable, paginated, accessible).
 *
 * Tier 20: replaces all 22 medium + 1 low table drifts.
 *
 * Visual contract (from spec §3.3):
 *   - Container: var(--color-surface), var(--radius-lg), 1px border
 *   - Header row: var(--color-surface-elevated), 11px uppercase
 *   - Body rows: 14px, tabular-nums on numeric, 48px height
 *   - Hover: var(--color-surface-hover)
 *   - Sortable headers: arrow icons (▲ ▼ ↕)
 *   - Pagination: bottom-right, "Showing X–Y of Z"
 *   - Loading: <SkeletonRow> shared component
 *   - Empty: <EmptyState> shared component (passed as prop)
 *   - Sticky header: position: sticky; top: 0
 *
 * Accessibility (from spec §3.3):
 *   - Real <table> semantics: <thead>, <tbody>, <tr>, <th>, <td>
 *   - Sortable headers as <button> inside <th> with aria-sort
 *   - Row click: <button> inside <td> or whole row has role=button
 *   - Pagination: <nav aria-label="Pagination">
 */
import { ReactNode, useMemo, useState, Key, KeyboardEvent as ReactKeyboardEvent } from 'react';
import { SkeletonRow } from './SkeletonRow';
import EmptyState from './EmptyState';

export type SortDirection = 'asc' | 'desc';
export type CellAlign = 'left' | 'center' | 'right';

export interface Column<T> {
  key: string;
  header: string;
  render?: (row: T) => ReactNode;
  sortable?: boolean;
  width?: string | number;
  align?: CellAlign;
  /** Optional CSS class for cells in this column. */
  className?: string;
}

export interface DataTableProps<T> {
  columns: Column<T>[];
  rows: T[];
  loading?: boolean;
  emptyState?: ReactNode;
  pageSize?: number;
  currentPage?: number;
  onPageChange?: (page: number) => void;
  sortBy?: string;
  sortDir?: SortDirection;
  onSort?: (key: string, dir: SortDirection) => void;
  onRowClick?: (row: T) => void;
  rowKey: (row: T) => Key;
  stickyHeader?: boolean;
  emptyTitle?: string;
  emptyDescription?: string;
  className?: string;
}

export default function DataTable<T>({
  columns,
  rows,
  loading = false,
  emptyState,
  pageSize = 25,
  currentPage: controlledPage,
  onPageChange,
  sortBy: controlledSortBy,
  sortDir: controlledSortDir,
  onSort,
  onRowClick,
  rowKey,
  stickyHeader = false,
  emptyTitle = 'No data',
  emptyDescription = 'There are no items to display.',
  className = '',
}: DataTableProps<T>) {
  // Uncontrolled state (only used if controlled props not provided)
  const [internalPage, setInternalPage] = useState(0);
  const [internalSortBy, setInternalSortBy] = useState<string | undefined>();
  const [internalSortDir, setInternalSortDir] = useState<SortDirection>('asc');

  const currentPage = controlledPage ?? internalPage;
  const sortBy = controlledSortBy ?? internalSortBy;
  const sortDir = controlledSortDir ?? internalSortDir;

  // Pagination
  const totalRows = rows.length;
  const totalPages = Math.max(1, Math.ceil(totalRows / pageSize));
  const safePage = Math.min(currentPage, totalPages - 1);
  const pageStart = safePage * pageSize;
  const pageEnd = Math.min(pageStart + pageSize, totalRows);

  const visibleRows = useMemo(
    () => rows.slice(pageStart, pageEnd),
    [rows, pageStart, pageEnd]
  );

  const handlePageChange = (nextPage: number) => {
    const clamped = Math.max(0, Math.min(nextPage, totalPages - 1));
    if (onPageChange) onPageChange(clamped);
    else setInternalPage(clamped);
  };

  // Sort
  const handleSort = (key: string) => {
    const col = columns.find((c) => c.key === key);
    if (!col?.sortable) return;
    let newDir: SortDirection = 'asc';
    if (sortBy === key) {
      newDir = sortDir === 'asc' ? 'desc' : 'asc';
    }
    if (onSort) onSort(key, newDir);
    else {
      setInternalSortBy(key);
      setInternalSortDir(newDir);
    }
  };

  // Sort indicator glyph
  const sortGlyph = (key: string) => {
    if (sortBy !== key) return '↕';
    return sortDir === 'asc' ? '↑' : '↓';
  };

  const ariaSortFor = (key: string): 'ascending' | 'descending' | 'none' => {
    if (sortBy !== key) return 'none';
    return sortDir === 'asc' ? 'ascending' : 'descending';
  };

  const containerClasses = [
    'data-table-container',
    stickyHeader ? 'data-table-sticky' : '',
    className,
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <div className={containerClasses}>
      <table className="data-table" role="table">
        <thead>
          <tr>
            {columns.map((col) => {
              const alignStyle =
                col.align === 'right'
                  ? { textAlign: 'right' as const }
                  : col.align === 'center'
                  ? { textAlign: 'center' as const }
                  : undefined;
              const widthStyle = col.width
                ? { width: typeof col.width === 'number' ? `${col.width}px` : col.width }
                : undefined;
              const cellStyle = { ...alignStyle, ...widthStyle };
              if (col.sortable) {
                return (
                  <th
                    key={col.key}
                    className={`data-table-th data-table-th-sortable ${col.className ?? ''}`}
                    aria-sort={ariaSortFor(col.key)}
                    style={cellStyle}
                    scope="col"
                  >
                    <button
                      type="button"
                      className="data-table-sort-btn"
                      onClick={() => handleSort(col.key)}
                    >
                      <span>{col.header}</span>
                      <span className="data-table-sort-glyph" aria-hidden="true">
                        {sortGlyph(col.key)}
                      </span>
                    </button>
                  </th>
                );
              }
              return (
                <th
                  key={col.key}
                  className={`data-table-th ${col.className ?? ''}`}
                  style={cellStyle}
                  scope="col"
                >
                  {col.header}
                </th>
              );
            })}
          </tr>
        </thead>
        <tbody>
          {loading ? (
            // Loading: render skeleton rows
            <tr>
              <td colSpan={columns.length} className="data-table-loading-cell">
                <SkeletonRow columns={columns.length} />
              </td>
            </tr>
          ) : visibleRows.length === 0 ? (
            // Empty: render EmptyState
            <tr>
              <td colSpan={columns.length} className="data-table-empty-cell">
                {emptyState ?? (
                  <EmptyState
                    headline={emptyTitle}
                    subhead={emptyDescription}
                  />
                )}
              </td>
            </tr>
          ) : (
            // Data rows
            visibleRows.map((row, idx) => {
              const key = rowKey(row);
              const rowClass = `data-table-row ${
                onRowClick ? 'data-table-row-clickable' : ''
              }`;
              const onClick = onRowClick
                ? () => onRowClick(row)
                : undefined;
              const onKeyDown = onRowClick
                ? (e: ReactKeyboardEvent<HTMLTableRowElement>) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault();
                      onRowClick(row);
                    }
                  }
                : undefined;
              const tabIndex = onRowClick ? 0 : undefined;
              const rowRole = onRowClick ? 'button' : undefined;
              return (
                <tr
                  key={key}
                  className={rowClass}
                  onClick={onClick}
                  onKeyDown={onKeyDown}
                  tabIndex={tabIndex}
                  role={rowRole}
                  aria-rowindex={idx + 2}
                >
                  {columns.map((col) => {
                    const alignStyle =
                      col.align === 'right'
                        ? { textAlign: 'right' as const }
                        : col.align === 'center'
                        ? { textAlign: 'center' as const }
                        : undefined;
                    return (
                          <td
                            key={col.key}
                            className={`data-table-td ${col.className ?? ''}`}
                            style={alignStyle}
                          >
                            {col.render ? col.render(row) : String((row as Record<string, unknown>)[col.key] ?? '')}
                          </td>
                        );
                  })}
                </tr>
              );
            })
          )}
        </tbody>
      </table>

      {/* Pagination */}
      {!loading && totalRows > pageSize && (
        <nav className="data-table-pagination" aria-label="Pagination">
          <span className="data-table-pagination-info">
            Showing <strong>{pageStart + 1}</strong>–<strong>{pageEnd}</strong> of{' '}
            <strong>{totalRows}</strong>
          </span>
          <div className="data-table-pagination-buttons">
            <button
              type="button"
              className="data-table-pagination-btn"
              onClick={() => handlePageChange(safePage - 1)}
              disabled={safePage === 0}
              aria-label="Previous page"
            >
              ‹ Prev
            </button>
            <span className="data-table-pagination-current">
              Page {safePage + 1} of {totalPages}
            </span>
            <button
              type="button"
              className="data-table-pagination-btn"
              onClick={() => handlePageChange(safePage + 1)}
              disabled={safePage >= totalPages - 1}
              aria-label="Next page"
            >
              Next ›
            </button>
          </div>
        </nav>
      )}
    </div>
  );
}