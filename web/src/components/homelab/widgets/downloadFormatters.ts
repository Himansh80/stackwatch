import type { DownloadKind } from './types';

/**
 * Formatting helpers for DownloadStatsWidget — split out so the
 * widget itself stays under the 400-LOC cap.
 *
 * fmtBytes / fmtSpeed render byte counts in human-readable units
 * (KB/MB/GB). statusPillClass / statusPillDataAttr map the backend's
 * last_poll_status to the CSS classes used by the homelab-service-
 * pill component (defined in styles-tier2.css — Phase 2).
 * kindLabel renders the human-readable name for the <select>
 * options. relativeTime renders "Ns ago" / "Nm ago" for the
 * per-client "polled N seconds ago" timestamp.
 */

export function fmtBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let n = bytes;
  let i = 0;
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024;
    i++;
  }
  return `${n.toFixed(n < 10 && i > 0 ? 1 : 0)} ${units[i]}`;
}

export function fmtSpeed(bps: number): string {
  if (!Number.isFinite(bps) || bps <= 0) return '0 B/s';
  return `${fmtBytes(bps)}/s`;
}

// statusPillClass returns the CSS class for the per-client status
// pill. The class is a stable hook ("homelab-service-pill" —
// defined in styles-tier2.css); the data-status attribute carries
// the actual status color (see statusPillDataAttr below).
//
// The status parameter is intentionally unused at the moment —
// every status value renders with the same base class. Kept in
// the signature so a future Phase 6 polish pass can wire
// per-status classes (e.g. "homelab-service-pill-success") without
// changing every call site.
export function statusPillClass(status: string | undefined): string {
  void status;
  return 'homelab-service-pill';
}

// statusPillDataAttr maps the backend's last_poll_status string
// to one of 'up' | 'down' | 'unknown' — the data-status attribute
// the homelab-service-pill CSS uses for color (Phase 2 styling).
export function statusPillDataAttr(status: string | undefined): string {
  switch (status) {
    case 'success':
      return 'up';
    case 'error':
      return 'down';
    default:
      return 'unknown';
  }
}

// kindLabel renders the human-readable name for the <select>
// dropdown + per-client pill. qBittorrent and SABnzbd are CamelCase
// in the homelab convention; the rest are Title-cased.
export function kindLabel(kind: DownloadKind): string {
  switch (kind) {
    case 'qbittorrent':
      return 'qBittorrent';
    case 'sabnzbd':
      return 'SABnzbd';
    default:
      return kind.charAt(0).toUpperCase() + kind.slice(1);
  }
}

// relativeTime renders the "polled N seconds ago" timestamp for
// the per-client footer. Inlined here (not imported from
// calendarTimeHelpers) because the calendar helper module is
// calendar-specific and would balloon the import surface for a
// single string format. Matches the relativeTime in
// CalendarWidget.tsx so the two surfaces render identically.
export function relativeTime(iso: string): string {
  const t = new Date(iso).getTime();
  if (!Number.isFinite(t)) return 'unknown';
  const dt = Date.now() - t;
  if (dt < 0) return 'just now';
  const s = Math.floor(dt / 1000);
  if (s < 60) return `${s}s ago`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  return `${d}d ago`;
}