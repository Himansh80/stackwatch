import { fmtBytes, fmtSpeed, kindLabel, relativeTime, statusPillDataAttr } from './downloadFormatters';
import type { DownloadClientRow } from './types';

/**
 * DownloadClientCard — the per-client article element rendered
 * inside DownloadStatsWidget. Split out so the widget stays
 * under the 400-LOC cap.
 *
 * Renders one pinned client as a card:
 *   - header: name + kind pill
 *   - subheader: base_url (word-break: break-all so long URLs
 *     don't blow out the card)
 *   - 2x2 stat grid: queue count / queue bytes / down / up speeds
 *   - footer: "polled N seconds ago" + ⚠ tooltip on poll error
 *   - action row: × delete button (stopPropagation so card-level
 *     click handlers don't fire)
 *
 * The card itself is non-interactive — clicks fall through to
 * the underlying parent. Only the × button has a handler.
 */
interface DownloadClientCardProps {
  client: DownloadClientRow;
  onDelete: (c: DownloadClientRow) => void;
}

export default function DownloadClientCard({ client, onDelete }: DownloadClientCardProps) {
  const s = client.latest_snapshot;
  const status = client.last_poll_status;
  return (
    <article
      className="homelab-service-card"
      data-status={statusPillDataAttr(status)}
      style={{ cursor: 'default' }}
    >
      <div className="homelab-service-card-header">
        <span className="homelab-service-card-name">{client.name}</span>
        <span
          className="homelab-service-pill"
          data-status={statusPillDataAttr(status)}
          style={{ fontSize: 10, padding: '2px 6px' }}
        >
          {kindLabel(client.kind)}
        </span>
      </div>
      <span className="homelab-service-card-url" style={{ wordBreak: 'break-all' }}>
        {client.base_url}
      </span>
      {s ? (
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
            gap: 6,
            marginTop: 8,
            fontSize: 12,
          }}
        >
          <div>
            <span style={{ opacity: 0.6 }}>Queue: </span>
            <strong>{s.queue_count}</strong>
          </div>
          <div>
            <span style={{ opacity: 0.6 }}>Remaining: </span>
            <strong>{fmtBytes(s.queue_size_bytes)}</strong>
          </div>
          <div>
            <span style={{ opacity: 0.6 }}>↓ </span>
            <strong>{fmtSpeed(s.download_speed_bytes_per_sec)}</strong>
          </div>
          <div>
            <span style={{ opacity: 0.6 }}>↑ </span>
            <strong>{fmtSpeed(s.upload_speed_bytes_per_sec)}</strong>
          </div>
        </div>
      ) : (
        <div
          style={{
            fontSize: 11,
            opacity: 0.6,
            marginTop: 8,
            fontStyle: 'italic',
          }}
        >
          no snapshot yet — wait for the next 60s tick
        </div>
      )}
      <div className="homelab-service-card-footer">
        <span className="homelab-service-latency">
          {client.last_polled_at
            ? `polled ${relativeTime(client.last_polled_at)}`
            : 'never polled'}
        </span>
        {status === 'error' && client.last_poll_error ? (
          <span title={client.last_poll_error} style={{ color: 'var(--red, #ef4444)' }}>
            ⚠
          </span>
        ) : null}
      </div>
      <div className="homelab-service-card-actions" onClick={(e) => e.stopPropagation()}>
        <button
          type="button"
          className="homelab-service-action-btn"
          onClick={() => void onDelete(client)}
          title="Remove"
        >
          ×
        </button>
      </div>
    </article>
  );
}