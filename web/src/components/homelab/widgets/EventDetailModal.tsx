import type { EventRow } from './types';

/**
 * EventDetailModal — read-only view of one event's full details.
 *
 * Shows summary, source calendar, time range, location, and the
 * raw description (if any). The header is colored with the
 * parent calendar's color so the user can tell at a glance which
 * feed the event came from.
 */

interface EventDetailModalProps {
  event: EventRow;
  calendarName: string;
  calendarColor: string;
  onClose: () => void;
}

export default function EventDetailModal({
  event, calendarName, calendarColor, onClose,
}: EventDetailModalProps) {
  return (
    <div
      className="sw-modal-backdrop"
      role="dialog"
      aria-modal="true"
      aria-label="Event details"
      onClick={onClose}
    >
      <div className="sw-modal" onClick={(e) => e.stopPropagation()}>
        <h3
          style={{
            margin: 0,
            fontSize: 16,
            color: calendarColor || '#3b82f6',
          }}
        >
          {event.summary}
        </h3>
        <p style={{ margin: '8px 0', fontSize: 13, opacity: 0.7 }}>
          from <strong>{calendarName || 'calendar'}</strong>
        </p>
        <p style={{ margin: '4px 0', fontSize: 13 }}>
          <strong>Starts:</strong> {new Date(event.starts_at).toLocaleString()}
        </p>
        {event.ends_at ? (
          <p style={{ margin: '4px 0', fontSize: 13 }}>
            <strong>Ends:</strong> {new Date(event.ends_at).toLocaleString()}
          </p>
        ) : null}
        {event.all_day ? (
          <p style={{ margin: '4px 0', fontSize: 13, opacity: 0.7 }}>
            All-day event
          </p>
        ) : null}
        {event.location ? (
          <p style={{ margin: '8px 0', fontSize: 13 }}>
            <strong>Location:</strong> {event.location}
          </p>
        ) : null}
        {event.description ? (
          <p
            style={{
              margin: '8px 0',
              fontSize: 13,
              whiteSpace: 'pre-wrap',
              background: 'var(--surface-2)',
              padding: 8,
              borderRadius: 6,
            }}
          >
            {event.description}
          </p>
        ) : null}
        <div
          className="sw-form-actions"
          style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}
        >
          <button
            type="button"
            className="empty-state-cta"
            onClick={onClose}
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
