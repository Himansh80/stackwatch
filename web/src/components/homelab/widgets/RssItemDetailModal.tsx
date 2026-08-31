import Button from '../../shared/Button';
import Modal from '../../shared/Modal';

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
 *
 * Tier 20 Phase G: refactored custom homelab-modal-backdrop +
 * empty-state-cta buttons to shared Modal + Button.
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
    <Modal open onClose={onClose} title={item.title} size="md">
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
      <div
        style={{
          display: 'flex',
          gap: 8,
          justifyContent: 'flex-end',
          flexWrap: 'wrap',
        }}
      >
        <Button
          variant="ghost"
          size="sm"
          onClick={() => window.open(item.link, '_blank', 'noopener,noreferrer')}
        >
          Open in new tab ↗
        </Button>
        {!item.read_at ? (
          <Button
            variant="primary"
            size="sm"
            onClick={onMarkRead}
            disabled={busy}
            loading={busy}
          >
            {busy ? 'Marking…' : 'Mark as read'}
          </Button>
        ) : null}
        <Button variant="secondary" size="sm" onClick={onClose}>
          Close
        </Button>
      </div>
    </Modal>
  );
}
