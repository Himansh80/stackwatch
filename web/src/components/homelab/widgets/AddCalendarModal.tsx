import type { CalendarFormState } from './types';

/**
 * AddCalendarModal — controlled form for POST /homelab/calendars.
 *
 * Reuses the shared .sw-modal* classes. The color picker is a row
 * of swatches bound to the form's `color` field. Lives in its own
 * file (split from CalendarWidget.tsx) so the parent stays under
 * 400 LOC.
 */

const PALETTE: Array<{ hex: string; label: string }> = [
  { hex: '#3b82f6', label: 'Blue' },
  { hex: '#22c55e', label: 'Green' },
  { hex: '#ef4444', label: 'Red' },
  { hex: '#f59e0b', label: 'Amber' },
  { hex: '#a855f7', label: 'Purple' },
  { hex: '#06b6d4', label: 'Cyan' },
  { hex: '#6b7280', label: 'Gray' },
];

interface AddCalendarModalProps {
  form: CalendarFormState;
  setForm: (next: CalendarFormState) => void;
  busy: boolean;
  error: string;
  onClose: () => void;
  onSubmit: () => void;
}

export default function AddCalendarModal({
  form, setForm, busy, error, onClose, onSubmit,
}: AddCalendarModalProps) {
  return (
    <div
      className="sw-modal-backdrop"
      role="dialog"
      aria-modal="true"
      aria-label="Add calendar"
      onClick={onClose}
    >
      <div className="sw-modal" onClick={(e) => e.stopPropagation()}>
        <form
          className="homelab-pin-form"
          onSubmit={(e) => {
            e.preventDefault();
            onSubmit();
          }}
        >
          <h3 style={{ margin: 0, fontSize: 16 }}>Add calendar</h3>

          <div className="homelab-pin-form-row">
            <label htmlFor="cal-name">Name</label>
            <input
              id="cal-name"
              type="text"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="Work"
              maxLength={200}
              required
            />
          </div>

          <div className="homelab-pin-form-row">
            <label htmlFor="cal-url">iCal URL</label>
            <input
              id="cal-url"
              type="text"
              value={form.ical_url}
              onChange={(e) => setForm({ ...form, ical_url: e.target.value })}
              placeholder="https://calendar.google.com/calendar/ical/.../basic.ics"
              required
            />
            <p className="homelab-pin-form-hint">
              Paste the read-only iCal URL from Google / Outlook / Apple Calendar.
            </p>
          </div>

          <div className="homelab-pin-form-row">
            <label htmlFor="cal-color">Color</label>
            <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
              {PALETTE.map((p) => (
                <button
                  key={p.hex}
                  type="button"
                  onClick={() => setForm({ ...form, color: p.hex })}
                  title={p.label}
                  aria-label={p.label}
                  style={{
                    width: 24,
                    height: 24,
                    borderRadius: 12,
                    background: p.hex,
                    border: form.color === p.hex
                      ? '3px solid var(--text)'
                      : '2px solid var(--surface-3)',
                    cursor: 'pointer',
                    padding: 0,
                  }}
                />
              ))}
            </div>
          </div>

          {error ? <div className="dash-error" role="alert">{error}</div> : null}

          <div
            className="sw-form-actions"
            style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}
          >
            <button
              type="button"
              className="empty-state-cta"
              onClick={onClose}
              disabled={busy}
            >
              Cancel
            </button>
            <button
              type="submit"
              className="empty-state-cta"
              disabled={busy || !form.name.trim() || !form.ical_url.trim()}
            >
              {busy ? 'Adding…' : 'Add calendar'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
