// Tier 11 Phase 5 — Backup / Restore (PL5).
//
// BackupSection's row table + KPI card subcomponents. Split
// out of BackupSection.tsx so each file stays under the
// 400-LOC cap.
import { motion, useReducedMotion } from '../../lib/motion';

interface BackupRow {
  id: string;
  tenant_id: string;
  created_by_user_id?: string;
  backup_kind: 'manual' | 'scheduled' | 'system';
  status: 'pending' | 'running' | 'completed' | 'failed';
  file_path?: string;
  file_size_bytes: number;
  uncompressed_bytes: number;
  table_count: number;
  row_count: number;
  encryption_algo: string;
  sha256_plaintext?: string;
  sha256_ciphertext?: string;
  started_at: string;
  completed_at?: string;
  error_message?: string;
  expires_at?: string;
  created_at: string;
}

export const STATUS_COLORS: Record<string, string> = {
  pending: 'bg-amber-500/10 text-amber-300 border-amber-500/30',
  running: 'bg-blue-500/10 text-blue-300 border-blue-500/30 animate-pulse',
  completed: 'bg-emerald-500/10 text-emerald-300 border-emerald-500/30',
  failed: 'bg-rose-500/10 text-rose-300 border-rose-500/30',
};

interface KpiCardProps {
  label: string;
  value: string;
  hint?: string;
}

export function KpiCard({ label, value, hint }: KpiCardProps) {
  return (
    <motion.div
      variants={!useReducedMotion() ? undefined : undefined}
      className="rounded-lg border border-border bg-card p-4"
    >
      <div className="text-xs uppercase tracking-wider text-muted-foreground">
        {label}
      </div>
      <div className="mt-1 text-2xl font-semibold text-foreground">
        {value}
      </div>
      {hint && (
        <div className="mt-1 text-xs text-muted-foreground">{hint}</div>
      )}
    </motion.div>
  );
}

interface BackupTableProps {
  rows: BackupRow[];
  canMutate: boolean;
  downloadingId: string | null;
  onDownload: (id: string) => void;
  onDelete: (id: string) => void;
}

export function BackupTable({
  rows,
  canMutate,
  downloadingId,
  onDownload,
  onDelete,
}: BackupTableProps) {
  if (rows.length === 0) return null;
  return (
    <div className="overflow-x-auto rounded-lg border border-border">
      <table className="w-full text-sm">
        <thead className="bg-muted/50 text-xs uppercase tracking-wider text-muted-foreground">
          <tr>
            <th className="px-3 py-2 text-left">Started</th>
            <th className="px-3 py-2 text-left">Kind</th>
            <th className="px-3 py-2 text-left">Status</th>
            <th className="px-3 py-2 text-right">Size</th>
            <th className="px-3 py-2 text-right">Rows</th>
            <th className="px-3 py-2 text-right">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {rows.map((r) => (
            <tr key={r.id} className="bg-card hover:bg-card/70">
              <td className="px-3 py-2 text-foreground">
                {formatAge(r.started_at)}
              </td>
              <td className="px-3 py-2 text-muted-foreground">
                {r.backup_kind}
              </td>
              <td className="px-3 py-2">
                <span
                  className={`inline-block rounded-full border px-2 py-0.5 text-xs ${
                    STATUS_COLORS[r.status] ?? STATUS_COLORS.pending
                  }`}
                >
                  {r.status}
                </span>
                {r.error_message && (
                  <div className="mt-1 text-xs text-rose-300">
                    {r.error_message}
                  </div>
                )}
              </td>
              <td className="px-3 py-2 text-right text-muted-foreground">
                {formatBytes(r.file_size_bytes)}
              </td>
              <td className="px-3 py-2 text-right text-muted-foreground">
                {r.row_count.toLocaleString()}
              </td>
              <td className="px-3 py-2 text-right">
                <button
                  onClick={() => onDownload(r.id)}
                  disabled={r.status !== 'completed' || downloadingId === r.id}
                  className="mr-2 rounded-md border border-border bg-card px-2 py-1 text-xs text-foreground hover:bg-muted disabled:opacity-50"
                >
                  {downloadingId === r.id ? 'Downloading…' : 'Download'}
                </button>
                {canMutate && (
                  <button
                    onClick={() => onDelete(r.id)}
                    className="rounded-md border border-rose-500/30 bg-rose-500/10 px-2 py-1 text-xs text-rose-300 hover:bg-rose-500/20"
                  >
                    Delete
                  </button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function formatAge(iso?: string): string {
  if (!iso) return 'never';
  const ms = Date.now() - new Date(iso).getTime();
  if (ms < 0) return 'just now';
  const min = Math.floor(ms / 60000);
  if (min < 1) return 'just now';
  if (min < 60) return `${min}m ago`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr}h ago`;
  const day = Math.floor(hr / 24);
  return `${day}d ago`;
}

function formatBytes(b: number): string {
  if (b === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.min(Math.floor(Math.log(b) / Math.log(1024)), units.length - 1);
  const num = b / Math.pow(1024, i);
  return `${num.toFixed(num < 10 ? 2 : 1)} ${units[i]}`;
}
