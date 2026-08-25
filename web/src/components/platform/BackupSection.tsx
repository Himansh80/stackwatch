import { useEffect, useState } from 'react';
import { ApiError, api, getToken } from '../../lib/api';
import { motion, pageEnter } from '../../lib/motion';
import EmptyState from '../shared/EmptyState';
import { BackupTable, KpiCard, STATUS_COLORS } from './BackupTable';

// Tier 11 Phase 5 — Backup / Restore (PL5).
//
// BackupSection renders the per-tenant manual-backup surface
// inside the unified PlatformPage (Phase 8 lands the 6-tab
// shell; today Phase 5 exports this component so a quick
// integration can drop it into any tab container).
//
// Layout (top → bottom):
//   - Header       : "Backups" + "Create backup" button
//   - KPI strip    : 4 tiles — total backups / last status
//                    / storage used / rows captured
//   - Backup list  : newest-first rows with status badge +
//                    size + duration + Download + Delete
//   - Empty state  : when no backups exist, friendly
//                    explanation + first-action CTA
//   - Schedule placeholder: dashed card pointing at the
//                    platform_backup_jobs DB row + the
//                    BackupSchedulerWorker (1h tick). A
//                    full schedule editor lands Phase 6.
//
// Data sources:
//   POST   /api/v1/platform/backup/create        — manual trigger
//   GET    /api/v1/platform/backup/list          — list this tenant
//   GET    /api/v1/platform/backup/:id/download  — ciphertext stream
//   DELETE /api/v1/platform/backup/:id           — unlink + delete
//   POST   /api/v1/platform/backup/restore       — multipart upload
//
// Subcomponents (KpiCard + BackupTable) live in
// BackupTable.tsx (kept under the 400-LOC cap by the split).
//
// Auth: every load bails early when no JWT is present so we
// don't spam a 401 from the platform-admin login flow.
//
// Motion : pageEnter on the section wrapper. No new variants —
// reuses the existing motion primitives from
// src/lib/motion.tsx.

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

interface BackupSummary {
  total_backups: number;
  last_backup_at?: string;
  last_backup_status?: string;
  total_storage_bytes: number;
  total_rows: number;
}

interface BackupList {
  backups: BackupRow[];
  summary: BackupSummary;
}

interface BackupSectionProps {
  /** When true, show the "Create backup" + "Restore" buttons.
   *  Super_admin and platform_admin are the two roles that can
   *  mutate backups; everyone else sees read-only. */
  canMutate: boolean;
}

export default function BackupSection({ canMutate }: BackupSectionProps) {
  const [list, setList] = useState<BackupList | null>(null);
  const [busy, setBusy] = useState(false);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState('');
  const [downloadingId, setDownloadingId] = useState<string | null>(null);
  const [restoreOpen, setRestoreOpen] = useState(false);

  const requireAuth = (): boolean => {
    if (!getToken()) {
      setError('Sign in to view backups.');
      return false;
    }
    return true;
  };

  const load = async () => {
    if (!requireAuth()) return;
    setBusy(true);
    setError('');
    try {
      const data = await api<BackupList>(
        'GET',
        '/api/v1/platform/backup/list'
      );
      setList(data);
    } catch (cause) {
      setError(
        cause instanceof ApiError
          ? cause.friendlyMessage
          : (cause as Error).message
      );
    } finally {
      setBusy(false);
    }
  };

  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Auto-poll while any backup is in-flight so the dashboard
  // reflects status transitions without a manual refresh.
  useEffect(() => {
    if (!list) return;
    const inFlight = list.backups.some(
      (b) => b.status === 'pending' || b.status === 'running'
    );
    if (!inFlight) return;
    const t = setInterval(() => {
      void load();
    }, 5000);
    return () => clearInterval(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [list]);

  const handleCreate = async () => {
    if (!requireAuth()) return;
    setCreating(true);
    setError('');
    try {
      await api('POST', '/api/v1/platform/backup/create', {});
      await load();
    } catch (cause) {
      setError(
        cause instanceof ApiError
          ? cause.friendlyMessage
          : (cause as Error).message
      );
    } finally {
      setCreating(false);
    }
  };

  const handleDownload = async (id: string) => {
    if (!getToken()) return;
    setDownloadingId(id);
    setError('');
    try {
      const res = await fetch(
        `/api/v1/platform/backup/${id}/download`,
        {
          method: 'GET',
          headers: {
            Authorization: `Bearer ${getToken()}`,
          },
        }
      );
      if (!res.ok) {
        const txt = await res.text();
        throw new Error(txt || `download failed (HTTP ${res.status})`);
      }
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `stackwatch-backup-${id}.tar.gz.enc`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } catch (cause) {
      setError((cause as Error).message || 'download failed');
    } finally {
      setDownloadingId(null);
    }
  };

  const handleDelete = async (id: string) => {
    if (!requireAuth()) return;
    if (
      !window.confirm(
        'Permanently delete this backup? This cannot be undone.'
      )
    ) {
      return;
    }
    setError('');
    try {
      await api('DELETE', `/api/v1/platform/backup/${id}`);
      await load();
    } catch (cause) {
      setError(
        cause instanceof ApiError
          ? cause.friendlyMessage
          : (cause as Error).message
      );
    }
  };

  const summary: BackupSummary = list?.summary ?? {
    total_backups: 0,
    total_storage_bytes: 0,
    total_rows: 0,
  };
  const backups: BackupRow[] = list?.backups ?? [];
  const showEmpty = !busy && backups.length === 0;

  return (
    <motion.div
      initial="hidden"
      animate="show"
      variants={pageEnter}
      className="space-y-6 p-2"
    >
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-foreground">Backups</h2>
          <p className="text-sm text-muted-foreground">
            Encrypted with AES-256-GCM, per-tenant HKDF-derived key.
          </p>
        </div>
        <div className="flex gap-2">
          {canMutate && (
            <button
              onClick={() => void handleCreate()}
              disabled={creating}
              className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow hover:bg-primary/90 disabled:opacity-50"
            >
              {creating ? 'Creating…' : 'Create backup'}
            </button>
          )}
          <button
            onClick={() => setRestoreOpen(true)}
            className="rounded-md border border-border bg-card px-3 py-2 text-sm text-muted-foreground"
            title="Restore is staged for a follow-up phase — see audit log"
          >
            Restore (staged)
          </button>
        </div>
      </div>

      {/* Error banner */}
      {error && (
        <div className="rounded-md border border-rose-500/30 bg-rose-500/10 px-3 py-2 text-sm text-rose-300">
          {error}
        </div>
      )}

      {/* KPI strip */}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard
          label="Total backups"
          value={summary.total_backups.toString()}
          hint={
            summary.last_backup_at
              ? `Last: ${formatAge(summary.last_backup_at)}`
              : 'none yet'
          }
        />
        <KpiCard
          label="Last status"
          value={summary.last_backup_status ?? '—'}
          hint={
            summary.last_backup_status === 'completed'
              ? 'healthy'
              : summary.last_backup_status
                ? 'needs attention'
                : 'no data'
          }
        />
        <KpiCard
          label="Storage used"
          value={formatBytes(summary.total_storage_bytes)}
          hint={`across ${summary.total_backups} files`}
        />
        <KpiCard
          label="Rows captured"
          value={summary.total_rows.toLocaleString()}
          hint="approx (per pg_stat)"
        />
      </div>

      {/* Backup list OR empty state */}
      {showEmpty ? (
        <EmptyState
          illustration={<span aria-hidden="true">🗄️</span>}
          headline="No backups yet"
          subhead={`Click 'Create backup' to encrypt a full snapshot of this tenant's data. The first run captures every platform_* table; subsequent runs are incremental-friendly.`}
        />
      ) : (
        <BackupTable
          rows={backups}
          canMutate={canMutate}
          downloadingId={downloadingId}
          onDownload={(id) => void handleDownload(id)}
          onDelete={(id) => void handleDelete(id)}
        />
      )}

      {/* Schedule placeholder (Phase 5 ships minimal info; full
          schedule editor lands in Phase 6). We render a
          informational card so the dashboard layout doesn't
          shift between phases. */}
      <div className="rounded-md border border-dashed border-border bg-card/50 p-4 text-sm text-muted-foreground">
        Scheduled backups: configured per tenant in{' '}
        <code className="rounded bg-muted px-1">platform_backup_jobs</code>.
        The{' '}
        <code className="rounded bg-muted px-1">
          BackupSchedulerWorker
        </code>{' '}
        runs every hour. Phase 6 adds the in-dashboard editor.
      </div>

      {/* Restore placeholder — modal opens but only the
          minimal "what would happen" notice is rendered.
          Phase 6 adds the actual upload + decrypt + pg_restore
          UX. */}
      {restoreOpen && (
        <div className="rounded-md border border-amber-500/30 bg-amber-500/5 p-4 text-sm text-amber-200">
          Restore is staged for a follow-up phase — the API
          endpoint already accepts multipart uploads and the
          audit log records every attempt. A safe in-app
          restore flow with a confirmation modal lands in
          Phase 6.{' '}
          <button
            onClick={() => setRestoreOpen(false)}
            className="ml-2 underline"
          >
            Dismiss
          </button>
        </div>
      )}
    </motion.div>
  );
}

// formatAge + formatBytes live here as well so this file is
// self-contained; BackupTable.tsx re-implements the same
// helpers for its own row rendering (no shared util so we
// avoid creating a tiny `formatters` module for one caller
// from each side).
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
  const i = Math.min(
    Math.floor(Math.log(b) / Math.log(1024)),
    units.length - 1
  );
  const num = b / Math.pow(1024, i);
  return `${num.toFixed(num < 10 ? 2 : 1)} ${units[i]}`;
}
// STATUS_COLORS is re-exported so an importer doesn't need to
// reach into BackupTable.tsx's internal exports.
export { STATUS_COLORS };
