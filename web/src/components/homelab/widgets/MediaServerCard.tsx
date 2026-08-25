import { relativeTime, statusPillDataAttr } from './downloadFormatters';
import { kindLabel } from './mediaFormatters';
import type { MediaServerRow } from './types';

/**
 * MediaServerCard — the per-server article element rendered
 * inside MediaWidget. Split out so the widget stays under the
 * 400-LOC cap.
 *
 * Renders one pinned server as a card:
 *   - header: name + kind pill (with last_poll_status color)
 *   - subheader: base_url (word-break: break-all so long URLs
 *     don't blow out the card)
 *   - footer: "polled N seconds ago" + ⚠ tooltip on poll error
 *   - action row: × delete button (stopPropagation so card-level
 *     click handlers don't fire)
 *
 * The card itself is non-interactive — clicks fall through to
 * the underlying parent. Only the × button has a handler.
 */
interface MediaServerCardProps {
  server: MediaServerRow;
  onDelete: (s: MediaServerRow) => void;
}

export default function MediaServerCard({ server, onDelete }: MediaServerCardProps) {
  const status = server.last_poll_status;
  return (
    <article
      className="homelab-service-card"
      data-status={statusPillDataAttr(status)}
      style={{ cursor: 'default' }}
    >
      <div className="homelab-service-card-header">
        <span className="homelab-service-card-name">{server.name}</span>
        <span
          className="homelab-service-pill"
          data-status={statusPillDataAttr(status)}
          style={{ fontSize: 10, padding: '2px 6px' }}
        >
          {kindLabel(server.kind)}
        </span>
      </div>
      <span
        className="homelab-service-card-url"
        style={{ wordBreak: 'break-all' }}
      >
        {server.base_url}
      </span>
      <div className="homelab-service-card-footer">
        <span className="homelab-service-latency">
          {server.last_polled_at
            ? `polled ${relativeTime(server.last_polled_at)}`
            : 'never polled'}
        </span>
        {status === 'error' && server.last_poll_error ? (
          <span
            title={server.last_poll_error}
            style={{ color: 'var(--red, #ef4444)' }}
          >
            ⚠
          </span>
        ) : null}
      </div>
      <div
        className="homelab-service-card-actions"
        onClick={(e) => e.stopPropagation()}
      >
        <button
          type="button"
          className="homelab-service-action-btn"
          onClick={() => void onDelete(server)}
          title="Remove"
        >
          ×
        </button>
      </div>
    </article>
  );
}
