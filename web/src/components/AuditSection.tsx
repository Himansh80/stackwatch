import {
  FormEvent,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from 'react';
import Button from './shared/Button';
import KpiCard from './shared/KpiCard';
import AuditArchiveCard, { AuditArchiveRow } from './shared/AuditArchiveCard';
import { motion, kpiStagger } from '../lib/motion';
import {
  ArchiveForm,
  ExportForm,
  Modal,
  handleCreateArchiveSubmit,
  handleExportSubmit,
  initialArchiveForm,
  initialExportForm,
} from './AuditSectionModals';
import { ArchiveFormView, ExportFormView } from './AuditSectionFormViews';

/**
 * AuditSection — Tier 9.4 (Phase 4) UI surface for the Audit Log
 * Retention + Export page section. Mirrors the SsoSection /
 * ScimSection / RbacSection pattern so the future EnterprisePage
 * (Phase 6) can compose this in without going over the 400-LOC cap.
 *
 * Layout:
 *   - 3 KpiCards: total archives / total events archived / total
 *     size (sum of size_bytes), driven by kpiStagger for entrance.
 *   - "Create archive" + "Export logs" buttons (top-right of
 *     section) that open modal forms.
 *   - Archive list rendered as a stack of AuditArchiveCards.
 *   - The "Create archive" + "Export logs" modal bodies live in
 *     AuditSectionModals.tsx (kept in a sibling file to keep this
 *     section under the 400-LOC cap). Both submissions issue a
 *     browser-native download via a temporary <a download>.
 *
 * Motion: reuses existing exports (kpiStagger, buttonSpring) — no
 * new variants.
 */

interface AuditSectionProps {
  archives: AuditArchiveRow[];
  busy?: boolean;
  onError: (msg: string) => void;
  onChanged: () => void;
  // The session token to use for the browser download. We can't
  // use the api() helper because Content-Disposition bodies don't
  // deserialize as JSON. Provided by the parent (EnterprisePage /
  // dev console) which owns the auth header construction.
  sessionToken?: string | null;
}

function formatBytes(bytes: number): string {
  if (!bytes || bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.min(units.length - 1, Math.floor(Math.log10(bytes) / 3));
  const scaled = bytes / Math.pow(1000, i);
  return `${i === 0 ? Math.round(scaled) : scaled.toFixed(1)} ${units[i]}`;
}

function totalSize(archives: AuditArchiveRow[]): number {
  return archives.reduce(
    (sum, a) => sum + (a.status === 'completed' ? a.size_bytes : 0),
    0,
  );
}

function totalEvents(archives: AuditArchiveRow[]): number {
  return archives.reduce(
    (sum, a) => sum + (a.status === 'completed' ? a.event_count : 0),
    0,
  );
}

export default function AuditSection({
  archives,
  busy = false,
  onError,
  onChanged,
  sessionToken,
}: AuditSectionProps) {
  const [sectionBusy, setSectionBusy] = useState(false);
  const isBusy = busy || sectionBusy;
  const [showArchive, setShowArchive] = useState(false);
  const [showExport, setShowExport] = useState(false);
  const [archiveForm, setArchiveForm] = useState<ArchiveForm>(initialArchiveForm);
  const [exportForm, setExportForm] = useState<ExportForm>(initialExportForm);

  // Sorted: running first (user wants to see them) then completed,
  // then failed; within each bucket by created_at DESC.
  const sorted = useMemo(() => {
    const bucketOrder: Record<string, number> = { running: 0, failed: 1, completed: 2 };
    return [...archives].sort((a, b) => {
      const ba = bucketOrder[a.status] ?? 9;
      const bb = bucketOrder[b.status] ?? 9;
      if (ba !== bb) return ba - bb;
      return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
    });
  }, [archives]);

  const handleCreateArchive = useCallback(
    async (e: FormEvent) => {
      await handleCreateArchiveSubmit({
        e,
        archiveForm,
        setSectionBusy,
        onError,
        onComplete: () => {
          setShowArchive(false);
          setArchiveForm(initialArchiveForm());
          onChanged();
        },
      });
    },
    [archiveForm, onError, onChanged],
  );

  const handleExport = useCallback(
    async (e: FormEvent) => {
      await handleExportSubmit({
        e,
        exportForm,
        setSectionBusy,
        sessionToken: sessionToken ?? null,
        onError,
        onComplete: () => {
          setShowExport(false);
          setExportForm(initialExportForm());
        },
      });
    },
    [exportForm, onError, sessionToken],
  );

  // Close modals on Escape.
  useEffect(() => {
    if (!showArchive && !showExport) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setShowArchive(false);
        setShowExport(false);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [showArchive, showExport]);

  return (
    <>
      <motion.div
        className="dash-metric-strip"
        initial="hidden"
        animate="show"
        variants={kpiStagger}
        style={{ marginTop: 16 }}
      >
        <KpiCard
          label="Total archives"
          value={archives.length}
          status={archives.length > 0 ? 'up' : 'neutral'}
          accent="indigo"
        />
        <KpiCard
          label="Events archived"
          value={totalEvents(archives)}
          status={totalEvents(archives) > 0 ? 'up' : 'neutral'}
          accent="cyan"
        />
        <KpiCard
          label="Total size"
          value={formatBytes(totalSize(archives))}
          status={totalSize(archives) > 0 ? 'up' : 'neutral'}
          accent="green"
        />
      </motion.div>

      <section className="dash-section">
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'baseline',
            gap: 8,
          }}
        >
          <div>
            <span className="dash-eyebrow">Audit Log Retention</span>
            <h2 className="dash-section-title">Archives</h2>
          </div>
          <div style={{ display: 'flex', gap: 8 }}>
            <Button
              variant="primary"
              size="sm"
              onClick={() => setShowArchive(true)}
              disabled={isBusy}
            >
              + Create archive
            </Button>
            <Button
              variant="secondary"
              size="sm"
              onClick={() => setShowExport(true)}
              disabled={isBusy || !sessionToken}
              title={!sessionToken ? 'Sign in to enable downloads' : 'Export filtered audit events'}
            >
              Export logs
            </Button>
          </div>
        </div>

        <p
          className="dash-section-sub"
          style={{ color: 'var(--text-muted)', fontSize: 12, marginTop: 0 }}
        >
          Archives compress a slice of audit_log events into a single
          gzip blob for long-term retention (SOC2 / ISO27001 / HIPAA).
          The &quot;Export logs&quot; action streams filtered events as
          CSV or NDJSON.
        </p>

        <div className="threat-card-list">
          {sorted.length === 0 ? (
            <div
              className="threat-card"
              style={{ textAlign: 'center', padding: 24, color: 'var(--text-muted)' }}
            >
              <strong style={{ color: 'var(--text)' }}>No archives yet</strong>
              <p style={{ margin: '8px 0 16px', fontSize: 12 }}>
                Click <strong>+ Create archive</strong> to compress a period of audit
                events into a single gzip file you can hand to auditors.
              </p>
            </div>
          ) : (
            sorted.map((a) => (
              <AuditArchiveCard
                key={a.id}
                archive={a}
                busy={isBusy}
                onDownload={(id) =>
                  void downloadArchive(id, sessionToken, onError)
                }
              />
            ))
          )}
        </div>
      </section>

      {showArchive ? (
        <Modal title="Create audit archive" onClose={() => setShowArchive(false)}>
          <ArchiveFormView
            archiveForm={archiveForm}
            setArchiveForm={setArchiveForm}
            busy={isBusy}
            onSubmit={handleCreateArchive}
            onClose={() => setShowArchive(false)}
          />
        </Modal>
      ) : null}

      {showExport ? (
        <Modal title="Export audit events" onClose={() => setShowExport(false)}>
          <ExportFormView
            exportForm={exportForm}
            setExportForm={setExportForm}
            busy={isBusy}
            onSubmit={handleExport}
            onClose={() => setShowExport(false)}
          />
        </Modal>
      ) : null}
    </>
  );
}

// downloadArchive pulls a gzip blob from the download endpoint
// and triggers a browser download. Lives outside the component so
// the AuditSection component doesn't have to thread `sessionToken`
// through AuditArchiveCard; the card just calls onDownload(id).
async function downloadArchive(
  id: string,
  sessionToken: string | null | undefined,
  onError: (msg: string) => void,
): Promise<void> {
  if (!sessionToken) {
    onError('Session token missing — cannot initiate download.');
    return;
  }
  try {
    const res = await fetch(
      `/api/v1/enterprise/audit/archives/${id}/download`,
      { headers: { Authorization: `Bearer ${sessionToken}` } },
    );
    if (!res.ok) {
      throw new Error(`Download failed (${res.status}): ${await res.text()}`);
    }
    const blob = await res.blob();
    const cd = res.headers.get('Content-Disposition') ?? '';
    const match = cd.match(/filename="?([^";]+)"?/);
    const filename = match ? match[1] : `audit-archive-${id}.json.gz`;
    const dlUrl = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = dlUrl;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(dlUrl);
  } catch (cause) {
    onError((cause as Error).message ?? String(cause));
  }
}
