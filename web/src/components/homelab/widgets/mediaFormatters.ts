import type { MediaServerKind } from './types';

/**
 * Formatting helpers for MediaWidget — split out so the widget
 * itself stays under the 400-LOC cap.
 *
 * kindLabel renders the human-readable name for the <select>
 * options + per-server chip. Matches the convention in
 * downloadFormatters.ts (downloadFormatters.kindLabel uses
 * Title-case for most kinds but CamelCase for qBittorrent +
 * SABnzbd; media labels are simpler — Plex/Jellyfin/Emby all
 * CamelCase).
 *
 * progressPercent / fmtProgressMs render the playback-progress
 * block for the Now Playing section. progressPercent returns
 * 0-100; fmtProgressMs renders a "1h 23m" / "23m 04s" string
 * from a millisecond count. Both are defensive: 0 duration
 * means "we don't know" — return 0 / "0m 00s" so the progress
 * bar renders empty rather than blowing up.
 */

export function kindLabel(kind: MediaServerKind): string {
  switch (kind) {
    case 'plex':
      return 'Plex';
    case 'jellyfin':
      return 'Jellyfin';
    case 'emby':
      return 'Emby';
  }
}

export function progressPercent(progressMs: number, durationMs: number): number {
  if (!Number.isFinite(progressMs) || !Number.isFinite(durationMs)) return 0;
  if (durationMs <= 0) return 0;
  const pct = (progressMs / durationMs) * 100;
  if (pct < 0) return 0;
  if (pct > 100) return 100;
  return Math.round(pct);
}

export function fmtProgressMs(ms: number): string {
  if (!Number.isFinite(ms) || ms <= 0) return '0m 00s';
  const total = Math.floor(ms / 1000);
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  if (h > 0) return `${h}h ${m.toString().padStart(2, '0')}m`;
  return `${m}m ${s.toString().padStart(2, '0')}s`;
}
