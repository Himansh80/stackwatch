/**
 * DataTable.test.tsx — tests for shared DataTable component.
 *
 * Tier 20 AC-3: render, sort, pagination, empty, loading, row click.
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import DataTable, { type Column } from './DataTable';

interface TestRow {
  id: string;
  name: string;
  count: number;
}

const columns: Column<TestRow>[] = [
  { key: 'name', header: 'Name', sortable: true },
  { key: 'count', header: 'Count', sortable: true, align: 'right' },
  { key: 'status', header: 'Status' },
];

const rows: TestRow[] = [
  { id: '1', name: 'apple', count: 5 },
  { id: '2', name: 'banana', count: 3 },
  { id: '3', name: 'cherry', count: 8 },
];

describe('DataTable', () => {
  it('renders columns and rows', () => {
    render(
      <DataTable columns={columns} rows={rows} rowKey={(r) => r.id} />
    );
    expect(screen.getByText('Name')).toBeInTheDocument();
    expect(screen.getByText('apple')).toBeInTheDocument();
    expect(screen.getByText('banana')).toBeInTheDocument();
    expect(screen.getByText('cherry')).toBeInTheDocument();
  });

  it('uses real <table> semantics', () => {
    const { container: c } = render(
      <DataTable columns={columns} rows={rows} rowKey={(r) => r.id} />
    );
    expect(c.querySelector('table')).toBeInTheDocument();
    expect(c.querySelector('thead')).toBeInTheDocument();
    expect(c.querySelector('tbody')).toBeInTheDocument();
  });

  it('renders custom cell content via render prop', () => {
    const customColumns: Column<TestRow>[] = [
      {
        key: 'name',
        header: 'Name',
        render: (r) => <strong data-testid="custom">{r.name.toUpperCase()}</strong>,
      },
    ];
    render(
      <DataTable columns={customColumns} rows={[rows[0]]} rowKey={(r) => r.id} />
    );
    expect(screen.getByTestId('custom')).toHaveTextContent('APPLE');
  });

  it('calls onSort when sortable header clicked', () => {
    const onSort = vi.fn();
    render(
      <DataTable
        columns={columns}
        rows={rows}
        rowKey={(r) => r.id}
        onSort={onSort}
      />
    );
    fireEvent.click(screen.getByRole('button', { name: /Name/i }));
    expect(onSort).toHaveBeenCalledWith('name', 'asc');
  });

  it('toggles sort direction on second click', () => {
    const onSort = vi.fn();
    const { rerender } = render(
      <DataTable
        columns={columns}
        rows={rows}
        rowKey={(r) => r.id}
        onSort={onSort}
        sortBy="name"
        sortDir="asc"
      />
    );
    rerender(
      <DataTable
        columns={columns}
        rows={rows}
        rowKey={(r) => r.id}
        onSort={onSort}
        sortBy="name"
        sortDir="desc"
      />
    );
    fireEvent.click(screen.getByRole('button', { name: /Name/i }));
    expect(onSort).toHaveBeenCalledWith('name', 'asc');
  });

  it('renders empty state when no rows', () => {
    render(
      <DataTable
        columns={columns}
        rows={[]}
        rowKey={(r: TestRow) => r.id}
        emptyTitle="No servers"
        emptyDescription="Add your first server to get started"
      />
    );
    expect(screen.getByText('No servers')).toBeInTheDocument();
    expect(screen.getByText('Add your first server to get started')).toBeInTheDocument();
  });

  it('renders custom emptyState when provided', () => {
    render(
      <DataTable
        columns={columns}
        rows={[]}
        rowKey={(r: TestRow) => r.id}
        emptyState={<div data-testid="custom-empty">Custom empty</div>}
      />
    );
    expect(screen.getByTestId('custom-empty')).toBeInTheDocument();
  });

  it('renders skeleton when loading', () => {
    const { container } = render(
      <DataTable
        columns={columns}
        rows={[]}
        rowKey={(r: TestRow) => r.id}
        loading
      />
    );
    // SkeletonRow should be in the loading cell
    expect(container.querySelector('.skeleton-row, [data-testid="skeleton-row"]')).toBeTruthy();
  });

  it('paginates rows when more than pageSize', () => {
    const bigRows: TestRow[] = Array.from({ length: 30 }, (_, i) => ({
      id: String(i),
      name: `row-${i}`,
      count: i,
    }));
    render(
      <DataTable
        columns={columns}
        rows={bigRows}
        rowKey={(r) => r.id}
        pageSize={10}
      />
    );
    // Should show first 10 rows
    expect(screen.getByText('row-0')).toBeInTheDocument();
    expect(screen.getByText('row-9')).toBeInTheDocument();
    expect(screen.queryByText('row-10')).not.toBeInTheDocument();
  });

  it('calls onPageChange when Next clicked', () => {
    const onPageChange = vi.fn();
    const bigRows: TestRow[] = Array.from({ length: 30 }, (_, i) => ({
      id: String(i),
      name: `row-${i}`,
      count: i,
    }));
    render(
      <DataTable
        columns={columns}
        rows={bigRows}
        rowKey={(r) => r.id}
        pageSize={10}
        onPageChange={onPageChange}
      />
    );
    fireEvent.click(screen.getByRole('button', { name: /Next/i }));
    expect(onPageChange).toHaveBeenCalledWith(1);
  });

  it('calls onRowClick when row clicked and', () => {
    const onRowClick = vi.fn();
    render(
      <DataTable
        columns={columns}
        rows={rows}
        rowKey={(r) => r.id}
        onRowClick={onRowClick}
      />
    );
    fireEvent.click(screen.getByText('apple'));
    expect(onRowClick).toHaveBeenCalledWith(rows[0]);
  });

  it('row has role=button when onRowClick provided', () => {
    render(
      <DataTable
        columns={columns}
        rows={rows}
        rowKey={(r) => r.id}
        onRowClick={() => {}}
      />
    );
    const appleRow = screen.getByText('apple').closest('tr');
    expect(appleRow).toHaveAttribute('role', 'button');
    expect(appleRow).toHaveAttribute('tabindex', '0');
  });

  it('row has no role when onRowClick omitted', () => {
    render(<DataTable columns={columns} rows={rows} rowKey={(r) => r.id} />);
    const appleRow = screen.getByText('apple').closest('tr');
    expect(appleRow).not.toHaveAttribute('role');
  });
});