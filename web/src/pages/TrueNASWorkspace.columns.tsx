import StatusPill from '../components/shared/StatusPill';
import DataTable, { type Column } from '../components/shared/DataTable';
import type { TNRow } from '../lib/truenas';
import { formatBytesLocal, poolHealthTone, readNumber } from './TrueNASWorkspace.helpers';

/**
 * columnsFor — section-aware column set for the live upstream
 * response DataTable. Each TrueNAS section returns 3-4 sensible
 * columns built from the dynamic TNRow keys we know exist for
 * that endpoint (the row payload schema is documented in the
 * middleware handler for `/api/v1/truenas/call`).
 */
export function columnsFor(sectionId: string): Column<TNRow>[] {
  const idColumn: Column<TNRow> = {
    key: 'id',
    header: 'ID',
    render: (row) => <code>{String(row.id ?? row.name ?? '—')}</code>,
  };
  switch (sectionId) {
    case 'pools':
      return [
        { key: 'name', header: 'Pool', render: (row) => String(row.name ?? '—'), sortable: true },
        {
          key: 'status',
          header: 'Health',
          render: (row) => (
            <StatusPill status={poolHealthTone(row)} label={String(row.status ?? row.health ?? 'unknown')} size="sm" />
          ),
        },
        {
          key: 'size',
          header: 'Size',
          align: 'right',
          render: (row) => formatBytesLocal(readNumber(row, 'size', 'total')),
        },
        {
          key: 'allocated',
          header: 'Used',
          align: 'right',
          render: (row) => {
            const used = readNumber(row, 'allocated', 'used');
            const total = readNumber(row, 'size', 'total');
            const pct = total > 0 ? Math.round((used / total) * 100) : 0;
            return `${formatBytesLocal(used)} (${pct}%)`;
          },
        },
      ];
    case 'datasets':
      return [
        idColumn,
        { key: 'name', header: 'Dataset', render: (row) => String(row.name ?? '—'), sortable: true },
        {
          key: 'available',
          header: 'Available',
          align: 'right',
          render: (row) => formatBytesLocal(readNumber(row, 'available', 'avail')),
        },
      ];
    case 'nfs':
    case 'smb':
      return [
        idColumn,
        { key: 'path', header: 'Path', render: (row) => String(row.path ?? row.name ?? '—') },
        {
          key: 'enabled',
          header: 'State',
          render: (row) => {
            const enabled = row.enabled !== false && row.enabled !== 'false';
            return <StatusPill status={enabled ? 'up' : 'down'} label={enabled ? 'enabled' : 'disabled'} size="sm" />;
          },
        },
      ];
    case 'iscsi':
      return [
        idColumn,
        { key: 'name', header: 'Target', render: (row) => String(row.name ?? '—') },
        { key: 'type', header: 'Type', render: (row) => String(row.type ?? '—') },
      ];
    case 'snapshots':
      return [
        idColumn,
        { key: 'name', header: 'Snapshot', render: (row) => String(row.name ?? '—') },
        {
          key: 'creation',
          header: 'Created',
          render: (row) => {
            const t = row.creation ?? row.created_at;
            if (!t) return '—';
            const d = new Date(String(t));
            return Number.isNaN(d.getTime()) ? '—' : d.toLocaleString();
          },
        },
      ];
    case 'disks':
      return [
        idColumn,
        { key: 'name', header: 'Disk', render: (row) => String(row.name ?? '—') },
        {
          key: 'temperature',
          header: 'Temp',
          align: 'right',
          render: (row) => `${readNumber(row, 'temperature', 'temp')}°C`,
        },
      ];
    case 'users':
    case 'system':
    case 'cloud':
    default:
      return [
        idColumn,
        { key: 'name', header: 'Name', render: (row) => String(row.name ?? '—') },
      ];
  }
}

// Re-export Column type for callers that need to extend or annotate.
export type { Column } from '../components/shared/DataTable';
// Keep DataTable in the dependency graph so module resolution mirrors
// the original file's import set even when Column is type-only.
export { DataTable };