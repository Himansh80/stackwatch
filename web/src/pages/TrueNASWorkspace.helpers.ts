import type { TNRow } from '../lib/truenas';

export function value(row: TNRow, key: string): string {
  const raw = row[key];
  if (raw === null || raw === undefined) return '—';
  if (typeof raw === 'object') return JSON.stringify(raw);
  return String(raw);
}

export function readNumber(row: TNRow, ...keys: string[]): number {
  for (const k of keys) {
    const v = row[k];
    if (typeof v === 'number' && Number.isFinite(v)) return v;
    if (typeof v === 'string') {
      const n = Number(v);
      if (!Number.isNaN(n)) return n;
    }
  }
  return 0;
}

export function poolHealthTone(row: TNRow): 'ok' | 'warn' | 'crit' | 'unknown' {
  const status = String(row.status ?? row.health ?? '').toLowerCase();
  if (status.includes('online') || status === 'ok' || status === 'healthy') return 'ok';
  if (status.includes('degraded')) return 'warn';
  if (status.includes('offline') || status.includes('faulted') || status.includes('error')) return 'crit';
  return 'unknown';
}

export function rowsFrom(payload: unknown): TNRow[] {
  if (Array.isArray(payload)) return payload as TNRow[];
  if (!payload || typeof payload !== 'object') return [];
  const record = payload as Record<string, unknown>;
  for (const key of ['data', 'rows', 'items', 'pools', 'datasets', 'shares', 'snapshots', 'disks', 'users', 'groups', 'services', 'tasks']) {
    if (Array.isArray(record[key])) return record[key] as TNRow[];
    if (record[key] && typeof record[key] === 'object') {
      const nested = rowsFrom(record[key]);
      if (nested.length) return nested;
    }
  }
  return [record];
}

export function formatBytesLocal(bytes: number): string {
  if (!bytes) return '—';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let v = bytes;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i += 1;
  }
  return `${v.toFixed(v >= 100 ? 0 : 1)} ${units[i]}`;
}

/**
 * filterRows — client-side filter for the live upstream response
 * table. The JSON-RPC payload is small enough that filtering in
 * place is fine — no need for a backend roundtrip per keystroke.
 */
export function filterRows(rows: TNRow[], search: string): TNRow[] {
  if (!search.trim()) return rows;
  const needle = search.toLowerCase();
  return rows.filter((row) =>
    Object.values(row).some((value) => {
      if (value === null || value === undefined) return false;
      if (typeof value === 'object') {
        try { return JSON.stringify(value).toLowerCase().includes(needle); }
        catch { return false; }
      }
      return String(value).toLowerCase().includes(needle);
    }),
  );
}