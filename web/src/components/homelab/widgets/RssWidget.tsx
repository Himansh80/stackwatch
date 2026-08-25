import { useCallback, useEffect, useMemo, useState } from 'react';
import { ApiError, api, getToken } from '../../../lib/api';
import { motion, pageEnter } from '../../../lib/motion';
import AddRssFeedModal, { type RssFormState } from './AddRssFeedModal';
import RssFeedCard from './RssFeedCard';
import RssItemDetailModal, { type RssItemRowForModal } from './RssItemDetailModal';
import {
  RSS_TABS,
  type RssFeedRow,
  type RssItemRow,
  type RssFeedsResponse,
  type RssItemsResponse,
  asRssFeeds,
  asRssItems,
  rssCategoryColor,
  rssRelativeTime,
} from './RssWidget.helpers';

/**
 * RssWidget — Tier 10 Phase 8 (H9 — RSS / Activity Feed).
 *
 * The RSS tab content on the HomelabPage. Calls:
 *   GET    /api/v1/homelab/rss/feeds
 *   GET    /api/v1/homelab/rss/items
 *   POST   /api/v1/homelab/rss/feeds
 *   DELETE /api/v1/homelab/rss/feeds/:id
 *   POST   /api/v1/homelab/rss/items/:id/read
 *
 * Tab strip (All / Unread / per-category), feeds grid, items
 * list with read/unread indicator, click-through to a modal.
 * Modals + helpers + types live in their own files so this
 * widget stays under the 400-LOC cap.
 *
 * Motion: pageEnter on the wrapper (existing token, no new variants).
 */

const emptyForm: RssFormState = {
  name: '',
  feed_url: '',
  category: 'general',
};

export default function RssWidget({ config: _config }: { config?: { feed_id?: string } }) {
  void _config;
  const [feeds, setFeeds] = useState<RssFeedRow[]>([]);
  const [items, setItems] = useState<RssItemRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tab, setTab] = useState<string>('all');
  const [modalOpen, setModalOpen] = useState(false);
  const [form, setForm] = useState<RssFormState>(emptyForm);
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState('');
  const [detailItem, setDetailItem] = useState<RssItemRowForModal | null>(null);
  const [busyItemId, setBusyItemId] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!getToken()) {
      setError('Sign in to view your RSS feeds.');
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const qs = tab === 'unread' ? '?unread=true&limit=200' : '?limit=200';
      const [f, it] = await Promise.all([
        api<RssFeedsResponse>('GET', '/api/v1/homelab/rss/feeds'),
        api<RssItemsResponse>('GET', `/api/v1/homelab/rss/items${qs}`),
      ]);
      setFeeds(asRssFeeds(f));
      setItems(asRssItems(it));
      setError('');
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, [tab]);

  useEffect(() => {
    void load();
  }, [load]);

  const dynamicCategoryTabs = useMemo(() => {
    const set = new Set<string>();
    for (const f of feeds) set.add(f.category);
    return Array.from(set).sort();
  }, [feeds]);

  const filteredItems = useMemo(() => {
    if (tab === 'unread' || tab === 'all') return items;
    const catByFeed: Record<string, string> = {};
    for (const f of feeds) catByFeed[f.id] = f.category;
    return items.filter((it) => catByFeed[it.feed_id] === tab);
  }, [items, tab, feeds]);

  const totalUnread = useMemo(
    () => feeds.reduce((sum, f) => sum + (f.unread_count || 0), 0),
    [feeds],
  );

  const openForm = useCallback(() => {
    setForm(emptyForm);
    setFormError('');
    setModalOpen(true);
  }, []);

  const closeForm = useCallback(() => setModalOpen(false), []);

  const submitForm = useCallback(async () => {
    setBusy(true);
    setFormError('');
    try {
      await api('POST', '/api/v1/homelab/rss/feeds', {
        name: form.name.trim(),
        feed_url: form.feed_url.trim(),
        category: form.category,
      });
      setModalOpen(false);
      // POST fires immediate poll; wait 1.5s for upsert.
      window.setTimeout(() => { void load(); }, 1500);
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setFormError(msg);
    } finally {
      setBusy(false);
    }
  }, [form, load]);

  const deleteFeed = useCallback(async (f: RssFeedRow) => {
    if (!window.confirm(`Remove feed "${f.name}" and all its cached items?`)) return;
    try {
      await api('DELETE', `/api/v1/homelab/rss/feeds/${f.id}`);
      await load();
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
    }
  }, [load]);

  const refreshAll = useCallback(async () => {
    await load();
  }, [load]);

  const markRead = useCallback(async (id: string) => {
    setBusyItemId(id);
    try {
      await api('POST', `/api/v1/homelab/rss/items/${id}/read`);
      setItems((prev) =>
        prev.map((it) =>
          it.id === id ? { ...it, read_at: new Date().toISOString() } : it,
        ),
      );
      const targetFeedId = items.find((i) => i.id === id)?.feed_id;
      if (targetFeedId) {
        setFeeds((prev) =>
          prev.map((f) =>
            f.id === targetFeedId
              ? { ...f, unread_count: Math.max(0, f.unread_count - 1) }
              : f,
          ),
        );
      }
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
    } finally {
      setBusyItemId(null);
    }
  }, [items]);

  if (loading) {
    return (
      <div className="homelab-grid-loading" role="status" aria-live="polite">
        Loading your RSS feeds…
      </div>
    );
  }

  return (
    <motion.div
      className="homelab-services-widget"
      initial="hidden"
      animate="show"
      variants={pageEnter}
    >
      {error ? <div className="dash-error" role="alert">{error}</div> : null}

      <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
        <button type="button" className="empty-state-cta" onClick={openForm}>
          + Add feed
        </button>
        <button type="button" className="empty-state-cta" onClick={() => void refreshAll()}>
          Refresh
        </button>
        {totalUnread > 0 ? (
          <span
            className="homelab-service-pill"
            data-status="unknown"
            style={{ fontSize: 11, padding: '2px 8px' }}
            title={`${totalUnread} unread item${totalUnread === 1 ? '' : 's'}`}
          >
            {totalUnread} unread
          </span>
        ) : null}
      </div>

      {/* Tab strip */}
      <div
        style={{
          display: 'flex',
          gap: 6,
          flexWrap: 'wrap',
          marginTop: 12,
          borderBottom: '1px solid var(--surface-3)',
          paddingBottom: 8,
        }}
      >
        {RSS_TABS.map((t) => (
          <button
            key={t.id}
            type="button"
            onClick={() => setTab(t.id)}
            className={`homelab-search-item${tab === t.id ? ' homelab-search-item-active' : ''}`}
            style={{
              padding: '4px 12px',
              border: 'none',
              borderRadius: 6,
              cursor: 'pointer',
              background: tab === t.id ? 'var(--surface-3)' : 'transparent',
              color: 'inherit',
              fontSize: 12,
              fontWeight: 600,
            }}
          >
            {t.label}
            {t.id === 'unread' && totalUnread > 0 ? ` (${totalUnread})` : ''}
          </button>
        ))}
        {dynamicCategoryTabs.map((cat) => (
          <button
            key={cat}
            type="button"
            onClick={() => setTab(cat)}
            className={`homelab-search-item${tab === cat ? ' homelab-search-item-active' : ''}`}
            style={{
              padding: '4px 12px',
              border: 'none',
              borderRadius: 6,
              cursor: 'pointer',
              background: tab === cat ? rssCategoryColor(cat) : 'transparent',
              color: tab === cat ? '#fff' : 'inherit',
              fontSize: 12,
              fontWeight: 600,
              textTransform: 'capitalize',
            }}
          >
            {cat}
          </button>
        ))}
      </div>

      {/* Feeds list (only when not filtering by category) */}
      {tab !== 'unread' && feeds.length > 0 ? (
        <div
          className="homelab-services-grid"
          style={{
            gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))',
            marginTop: 12,
          }}
        >
          {feeds.map((f) => (
            <RssFeedCard key={f.id} feed={f} onDelete={(feed) => void deleteFeed(feed)} />
          ))}
        </div>
      ) : null}

      {/* Items list */}
      {filteredItems.length === 0 ? (
        <div className="homelab-services-empty" style={{ marginTop: 16 }}>
          <p className="homelab-services-empty-title">
            {feeds.length === 0 ? 'No RSS feeds yet' : 'No items'}
          </p>
          <p className="homelab-services-empty-hint">
            {feeds.length === 0
              ? 'Click "+ Add feed" and paste an RSS / Atom / JSON feed URL. The worker polls every feed every 5 minutes.'
              : 'New items appear here as the worker polls each feed.'}
          </p>
        </div>
      ) : (
        <ul style={{ listStyle: 'none', padding: 0, margin: '16px 0' }}>
          {filteredItems.map((it) => (
            <li
              key={it.id}
              style={{
                padding: '10px 12px',
                marginBottom: 6,
                border: '1px solid var(--surface-3)',
                borderRadius: 6,
                background: it.read_at ? 'var(--surface-1)' : 'var(--surface-2)',
                cursor: 'pointer',
                display: 'flex',
                gap: 12,
                alignItems: 'flex-start',
              }}
              onClick={() => setDetailItem(it)}
            >
              <span
                aria-hidden="true"
                style={{
                  width: 8,
                  height: 8,
                  borderRadius: '50%',
                  marginTop: 6,
                  background: it.read_at ? 'transparent' : 'var(--blue, #3b82f6)',
                  flexShrink: 0,
                }}
              />
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 13, fontWeight: it.read_at ? 400 : 600 }}>
                  {it.title}
                </div>
                <div style={{ fontSize: 11, opacity: 0.7, marginTop: 2 }}>
                  {it.feed_name}
                  {it.published_at ? ` · ${rssRelativeTime(it.published_at)}` : ''}
                  {it.author ? ` · ${it.author}` : ''}
                </div>
              </div>
            </li>
          ))}
        </ul>
      )}

      {modalOpen ? (
        <AddRssFeedModal
          form={form}
          setForm={setForm}
          busy={busy}
          error={formError}
          onClose={closeForm}
          onSubmit={() => void submitForm()}
        />
      ) : null}

      {detailItem ? (
        <RssItemDetailModal
          item={detailItem}
          busy={busyItemId === detailItem.id}
          onClose={() => setDetailItem(null)}
          onMarkRead={() => {
            void markRead(detailItem.id);
            setDetailItem(null);
          }}
        />
      ) : null}
    </motion.div>
  );
}
