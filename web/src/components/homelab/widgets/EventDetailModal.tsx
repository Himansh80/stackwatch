import Button from '../../shared/Button';
import Modal from '../../shared/Modal';
import type { EventRow } from './types';

/**
 * EventDetailModal — read-only view of one event's full details.
 *
 * Shows summary, source calendar, time range, location, and the
 * raw description (if any). The header is colored with the
 * parent calendar's color so the user can tell at a glance which
 * feed the event came from.
 *
 * Tier 20 Phase G: refactored custom sw-modal-backdrop +
 * empty-state-cta close button to shared Modal + Button.
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
    <Modal open onClose={onClose} title={event.summary} size="md">
      <div
        style={{
          height: 4,
          background: calendarColor || 'var(--color-primary)',
          borderRadius: 2,
          marginBottom: 12,
          marginTop: -8,
        }}
      />
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
            background: 'var(--color-surface)',
            padding: 8,
            borderRadius: 6,
          }}
        >
          {event.description}
        </p>
      ) : null}
      <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
        <Button variant="secondary" size="sm" onClick={onClose}>
          Close
        </Button>
      </div>
    </Modal>
  );
}
