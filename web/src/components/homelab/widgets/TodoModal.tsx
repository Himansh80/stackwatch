import type { TodoFormState, TodoPriority } from './types';

/**
 * TodoModal — controlled form for POST/PATCH /homelab/todos.
 *
 * Reuses the same .sw-modal* classes the PinModal uses. Tags are
 * entered as a comma-separated string and split on submit. The
 * due_date input is a datetime-local string ("YYYY-MM-DDTHH:MM");
 * the modal converts it to RFC3339 before sending so the server
 * can accept it via parseFlexibleDate.
 *
 * Shared between create + edit modes — the parent passes an
 * optional `editingId` to switch between POST and PATCH.
 */

const PRIORITIES: { value: TodoPriority; label: string }[] = [
  { value: 'low', label: 'Low' },
  { value: 'medium', label: 'Medium' },
  { value: 'high', label: 'High' },
  { value: 'urgent', label: 'Urgent' },
];

interface TodoModalProps {
  form: TodoFormState;
  setForm: (next: TodoFormState) => void;
  busy: boolean;
  error: string;
  editingId: string | null;
  onClose: () => void;
  onSubmit: () => void;
}

// Convert the datetime-local input value ("YYYY-MM-DDTHH:MM")
// into an RFC3339 string the server can parse. We treat the
// input as local time and emit a Z-suffixed UTC string — the
// server stores it as timestamptz and renders relative time
// (so timezone choice doesn't matter for the dashboard UX).
export function datetimeLocalToRFC3339(local: string): string | null {
  if (!local) return null;
  // The datetime-local format is "YYYY-MM-DDTHH:MM". Append
  // ":00Z" to make it a parseable RFC3339 timestamp. We don't
  // shift to UTC here — the server's timestamptz column handles
  // the timezone conversion when it reads the string.
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(local)) {
    return null;
  }
  return `${local}:00Z`;
}

export default function TodoModal({
  form, setForm, busy, error, editingId, onClose, onSubmit,
}: TodoModalProps) {
  return (
    <div
      className="sw-modal-backdrop"
      role="dialog"
      aria-modal="true"
      aria-label={editingId ? 'Edit todo' : 'New todo'}
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
          <h3 style={{ margin: 0, fontSize: 16 }}>
            {editingId ? 'Edit todo' : 'New todo'}
          </h3>

          <div className="homelab-pin-form-row">
            <label htmlFor="todo-title">Title</label>
            <input
              id="todo-title"
              type="text"
              value={form.title}
              onChange={(e) => setForm({ ...form, title: e.target.value })}
              placeholder="Buy new SSD for NAS"
              maxLength={200}
              required
            />
          </div>

          <div className="homelab-pin-form-row">
            <label htmlFor="todo-description">Description (optional)</label>
            <textarea
              id="todo-description"
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
              placeholder="More context, links, or reminders."
              rows={4}
              style={{
                background: 'var(--surface-2)',
                color: 'var(--text)',
                border: '1px solid var(--surface-3)',
                borderRadius: 8,
                padding: '10px 12px',
                font: 'inherit',
                resize: 'vertical',
              }}
            />
          </div>

          <div style={{ display: 'flex', gap: 12 }}>
            <div className="homelab-pin-form-row" style={{ flex: 1 }}>
              <label htmlFor="todo-priority">Priority</label>
              <select
                id="todo-priority"
                value={form.priority}
                onChange={(e) => setForm({ ...form, priority: e.target.value as TodoPriority })}
              >
                {PRIORITIES.map((p) => (
                  <option key={p.value} value={p.value}>{p.label}</option>
                ))}
              </select>
            </div>
            <div className="homelab-pin-form-row" style={{ flex: 1 }}>
              <label htmlFor="todo-due">Due date (optional)</label>
              <input
                id="todo-due"
                type="datetime-local"
                value={form.dueDate}
                onChange={(e) => setForm({ ...form, dueDate: e.target.value })}
              />
            </div>
          </div>

          <div className="homelab-pin-form-row">
            <label htmlFor="todo-tags">Tags (comma-separated)</label>
            <input
              id="todo-tags"
              type="text"
              value={form.tagsRaw}
              onChange={(e) => setForm({ ...form, tagsRaw: e.target.value })}
              placeholder="homelab, urgent, todo"
              maxLength={200}
            />
            <p className="homelab-pin-form-hint">Up to 16 tags, 32 chars each.</p>
          </div>

          {error ? <div className="dash-error" role="alert">{error}</div> : null}

          <div className="sw-form-actions" style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <button type="button" className="empty-state-cta" onClick={onClose} disabled={busy}>
              Cancel
            </button>
            <button type="submit" className="empty-state-cta" disabled={busy || !form.title.trim()}>
              {busy ? 'Saving…' : editingId ? 'Save changes' : 'Create todo'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
