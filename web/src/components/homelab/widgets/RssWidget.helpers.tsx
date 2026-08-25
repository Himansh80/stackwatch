/**
 * RssWidget shared types + helpers.
 *
 * Extracted from RssWidget.tsx so the main widget component stays
 * under the 400-LOC cap. The shape mirrors
 * web/src/components/homelab/widgets/types.ts, but kept local
 * here because types.ts is already at its 300-LOC cap.
 */

export interface RssFeedRow {
  id: string;
  name: string;
  feed_url: string;
  category: string;
  enabled: boolean;
  last_polled_at?: string;
  last_poll_status?: string;
  last_poll_error?: string;
  unread_count: number;
  created_at: string;
}

export interface RssItemRow {
  id: string;
  feed_id: string;
  feed_name: string;
  title: string;
  link: string;
  summary?: string;
  author?: string;
  published_at?: string;
  read_at?: string;
  created_at: string;
}

export interface RssFeedsResponse {
  feeds: RssFeedRow[] | string;
  count: number;
}

export interface RssItemsResponse {
  items: RssItemRow[] | string;
  count: number;
  limit: number;
}

export interface RssTab {
  id: string;
  label: string;
}

export const RSS_TABS: RssTab[] = [
  { id: 'all', label: 'All' },
  { id: 'unread', label: 'Unread' },
];

// Defensive shape walkers — the API client returns `any` cast as T,
// so a future backend refactor that renames `feeds` → `results`
// would silently break the dashboard without these helpers.
export function asRssFeeds(raw: unknown): RssFeedRow[] {
  if (!raw) return [];
  if (Array.isArray(raw)) return raw as RssFeedRow[];
  if (typeof raw === 'object') {
    const obj = raw as Record<string, unknown>;
    if (Array.isArray(obj.feeds)) return obj.feeds as RssFeedRow[];
  }
  return [];
}

export function asRssItems(raw: unknown): RssItemRow[] {
  if (!raw) return [];
  if (Array.isArray(raw)) return raw as RssItemRow[];
  if (typeof raw === 'object') {
    const obj = raw as Record<string, unknown>;
    if (Array.isArray(obj.items)) return obj.items as RssItemRow[];
  }
  return [];
}

// Compact relative-time formatter. Used in feed cards and item
// rows. Mirrors calendarTimeHelpers.relativeTime.
export function rssRelativeTime(iso?: string): string {
  if (!iso) return 'never';
  const t = new Date(iso).getTime();
  if (!Number.isFinite(t)) return 'unknown';
  const diff = Date.now() - t;
  if (diff < 60_000) return 'just now';
  if (diff < 3600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86400_000) return `${Math.floor(diff / 3600_000)}h ago`;
  return `${Math.floor(diff / 86400_000)}d ago`;
}

// Stable color per category — matches the chip background on
// both the feed card border-left and the tab strip.
export function rssCategoryColor(cat: string): string {
  switch (cat) {
    case 'news':
      return '#3b82f6';
    case 'releases':
      return '#22c55e';
    case 'blogs':
      return '#a855f7';
    case 'podcasts':
      return '#f59e0b';
    case 'other':
      return '#6b7280';
    default:
      return '#06b6d4';
  }
}
