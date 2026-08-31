import Button from '../../shared/Button';
import Input from '../../shared/Input';
import Modal from '../../shared/Modal';
import type { CalendarFormState } from './types';

/**
 * AddCalendarModal — controlled form for POST /homelab/calendars.
 *
 * The color picker is a row of swatches bound to the form's `color`
 * field. Lives in its own file (split from CalendarWidget.tsx) so
 * the parent stays under 400 LOC.
 *
 * Tier 20 Phase G: refactored custom sw-modal-backdrop +
 * homelab-pin-form-row + raw inputs to shared Modal + Input + Button.
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
    <Modal open onClose={onClose} title="Add calendar" size="md">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          onSubmit();
        }}
        style={{ display: 'flex', flexDirection: 'column', gap: 12 }}
      >
        <Input
          id="cal-name"
          label="Name"
          type="text"
          value={form.name}
          onChange={(e) => setForm({ ...form, name: e.target.value })}
          placeholder="Work"
          maxLength={200}
          required
          fullWidth
          autoFocus
        />

        <Input
          id="cal-url"
          label="iCal URL"
          type="text"
          value={form.ical_url}
          onChange={(e) => setForm({ ...form, ical_url: e.target.value })}
          placeholder="https://calendar.google.com/calendar/ical/.../basic.ics"
          description="Paste the read-only iCal URL from Google / Outlook / Apple Calendar."
          required
          fullWidth
        />

        <div>
          <label
            htmlFor="cal-color"
            style={{
              display: 'block',
              fontSize: 14,
              fontWeight: 500,
              marginBottom: 4,
              color: 'var(--color-text)',
            }}
          >
            Color
          </label>
          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
            {PALETTE.map((p) => {
              const selected = form.color === p.hex;
              return (
                <button
                  key={p.hex}
                  type="button"
                  onClick={() => setForm({ ...form, color: p.hex })}
                  title={p.label}
                  aria-label={p.label}
                  aria-pressed={selected}
                  style={{
                    width: 24,
                    height: 24,
                    borderRadius: 12,
                    background: p.hex,
                    border: selected
                      ? '3px solid var(--color-text)'
                      : '2px solid var(--color-border)',
                    cursor: 'pointer',
                    padding: 0,
                  }}
                />
              );
            })}
          </div>
        </div>

        {error ? <div className="dash-error" role="alert">{error}</div> : null}

        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={onClose}
            disabled={busy}
          >
            Cancel
          </Button>
          <Button
            type="submit"
            variant="primary"
            size="sm"
            disabled={busy || !form.name.trim() || !form.ical_url.trim()}
            loading={busy}
          >
            {busy ? 'Adding…' : 'Add calendar'}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
