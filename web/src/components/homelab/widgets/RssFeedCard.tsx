import { motion } from '../../../lib/motion';
import {
  type RssFeedRow,
  rssCategoryColor,
  rssRelativeTime,
} from './RssWidget.helpers';

/**
 * RssFeedCard — single feed row extracted from RssWidget so the
 * main widget stays under the 400-LOC cap. Datadog-style card
 * with category border-left, unread badge, last-polled time,
 * status pill, open + remove actions.
 */

interface RssFeedCardProps {
  feed: RssFeedRow;
  onDelete: (feed: RssFeedRow) => void;
}

export default function RssFeedCard({ feed, onDelete }: RssFeedCardProps) {
  return (
    <motion.article
      className="homelab-service-card"
      data-status="unknown"
      style={{
        borderLeftColor: rssCategoryColor(feed.category),
        cursor: 'default',
      }}
    >
      <div className="homelab-service-card-header">
        <span className="homelab-service-card-name">{feed.name}</span>
        {feed.unread_count > 0 ? (
          <span
            className="homelab-service-pill"
            data-status="unknown"
            style={{ fontSize: 10, padding: '2px 6px' }}
          >
            {feed.unread_count}
          </span>
        ) : null}
      </div>
      <span className="homelab-service-card-url" style={{ wordBreak: 'break-all' }}>
        {feed.feed_url}
      </span>
      <div className="homelab-service-card-footer">
        <span className="homelab-service-latency">
          polled {rssRelativeTime(feed.last_polled_at)}
        </span>
        {feed.last_poll_status === 'error' && feed.last_poll_error ? (
          <span title={feed.last_poll_error} style={{ color: 'var(--red, #ef4444)' }}>
            ⚠
          </span>
        ) : null}
      </div>
      <div className="homelab-service-card-actions" onClick={(e) => e.stopPropagation()}>
        <a
          href={feed.feed_url}
          target="_blank"
          rel="noreferrer noopener"
          className="homelab-service-action-btn"
          title="Open feed"
        >
          ↗
        </a>
        <button
          type="button"
          className="homelab-service-action-btn"
          onClick={() => onDelete(feed)}
          title="Remove"
        >
          ×
        </button>
      </div>
    </motion.article>
  );
}
