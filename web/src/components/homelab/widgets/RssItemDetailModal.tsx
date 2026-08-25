/**
 * RssItemDetailModal — Tier 10 Phase 8 (H9 — RSS).
 *
 * The item detail modal extracted from RssWidget so the main
 * widget stays under the 400-LOC cap. Shows the full summary +
 * an "Open in new tab" link + a "Mark as read" button (only when
 * the item isn't already read).
 *
 * Props:
 *   item        — the RssItemRow to display
 *   busy        — disables the "Mark as read" button while the
 *                 POST is in flight
 *   onClose     — close the modal
 *   onMarkRead  — fire the POST + close (parent owns the reload)
 */

export interface RssItemRowForModal {
  id: string;
  title: string;
  link: string;
  summary?: string;
  author?: string;
  published_at?: string;
  read_at?: string;
  feed_name: string;
}

interface RssItemDetailModalProps {
  item: RssItemRowForModal;
  busy: boolean;
  onClose: () => void;
  onMarkRead: () => void;
}

function relativeTime(iso?: string): string {
  if (!iso) return '';
  const t = new Date(iso).getTime();
  if (!Number.isFinite(t)) return '';
  const diff = Date.now() - t;
  if (diff < 60_000) return 'just now';
  if (diff < 3600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86400_000) return `${Math.floor(diff / 3600_000)}h ago`;
  return `${Math.floor(diff / 86400_000)}d ago`;
}

export default function RssItemDetailModal({
  item,
  busy,
  onClose,
  onMarkRead,
}: RssItemDetailModalProps) {
  return (
    <div
      className="homelab-modal-backdrop"
      role="dialog"
      aria-modal="true"
      onClick={onClose}
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.6)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 1000,
      }}
    >
      <div
        className="homelab-modal-panel"
        onClick={(e) => e.stopPropagation()}
        style={{
          background: 'var(--surface-2)',
          borderRadius: 8,
          padding: 24,
          minWidth: 360,
          maxWidth: 560,
          width: '90%',
        }}
      >
        <h3 style={{ marginTop: 0, marginBottom: 4 }}>{item.title}</h3>
        <div style={{ fontSize: 11, opacity: 0.7, marginBottom: 12 }}>
          {item.feed_name}
          {item.published_at ? ` · ${relativeTime(item.published_at)}` : ''}
          {item.author ? ` · ${item.author}` : ''}
        </div>
        {item.summary ? (
          <div style={{ fontSize: 13, marginBottom: 12, whiteSpace: 'pre-wrap' }}>
            {item.summary}
          </div>
        ) : null}
        <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', flexWrap: 'wrap' }}>
          <a
            href={item.link}
            target="_blank"
            rel="noreferrer noopener"
            className="empty-state-cta"
            style={{ textDecoration: 'none' }}
          >
            Open in new tab ↗
          </a>
          {!item.read_at ? (
            <button
              type="button"
              className="empty-state-cta"
              onClick={onMarkRead}
              disabled={busy}
            >
              {busy ? 'Marking…' : 'Mark as read'}
            </button>
          ) : null}
          <button type="button" className="empty-state-cta" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
