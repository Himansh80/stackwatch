import { ComplianceReportRow } from './shared/ComplianceReportCard';

/**
 * ComplianceSectionHelpers — Tier 9.5 (Phase 5) shared utilities
 * used by ComplianceSection.tsx.
 *
 * Why a sibling file:
 *   ComplianceSection.tsx has a lot of JSX (KPI strip + tab strip
 *   + Reports list + Schedules list + 2 modals) and would push
 *   past the 400-LOC cap if it also owned the per-row helpers
 *   (formatDateTime / relativeTime) + the browser-native download
 *   plumbing (downloadReport). Extracting them keeps the main
 *   component readable while staying strictly under the
 *   modular-discipline cap.
 *
 * Exposes:
 *   - formatDateTime / relativeTime — string formatters used by
 *     the schedule list.
 *   - downloadReport — pulls the artifact blob from the download
 *     endpoint and triggers a browser-native download via a
 *     temporary <a>.
 *
 * Tokens reuses --surface / --border / --text / --text-muted /
 * .threat-card-* etc. — no new CSS. Motion: reuses existing
 * exports (kpiStagger, buttonSpring) — no new variants.
 */

export function formatDateTime(iso: string): string {
  if (!iso) return '—';
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return iso;
  return new Date(t).toISOString().replace('T', ' ').replace(/\..+$/, ' UTC');
}

export function relativeTime(iso: string): string {
  if (!iso) return '—';
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return iso;
  const diff = Math.max(0, Date.now() - t);
  const sec = Math.floor(diff / 1000);
  if (sec < 60) return `${sec}s ago`;
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min}m ago`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr}h ago`;
  const day = Math.floor(hr / 24);
  return `${day}d ago`;
}

// downloadReport pulls the artifact blob from the download
// endpoint and triggers a browser-native download via a
// temporary <a>. Lives outside ComplianceSection so the section
// component doesn't have to thread sessionToken through
// ComplianceReportCard; the card just calls onDownload(id).
//
// Exported as a top-level helper (not a hook) because it has no
// React state of its own — it just orchestrates fetch + blob +
// <a download> plumbing.
export async function downloadReport(
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
      `/api/v1/enterprise/compliance/reports/${id}/download`,
      { headers: { Authorization: `Bearer ${sessionToken}` } },
    );
    if (!res.ok) {
      throw new Error(`Download failed (${res.status}): ${await res.text()}`);
    }
    const blob = await res.blob();
    const cd = res.headers.get('Content-Disposition') ?? '';
    const match = cd.match(/filename="?([^";]+)"?/);
    const filename = match ? match[1] : `compliance-report-${id}.txt`;
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

// Suppress unused-import warning when only types are referenced.
export type { ComplianceReportRow };
