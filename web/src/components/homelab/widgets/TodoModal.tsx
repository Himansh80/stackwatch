import type { TodoFormState, TodoPriority } from './types';
import Button from '../../shared/Button';
import Input from '../../shared/Input';
import Modal from '../../shared/Modal';
import Select from '../../shared/Select';
import Textarea from '../../shared/Textarea';

/**
 * TodoModal — controlled form for POST/PATCH /homelab/todos.
 *
 * Tags are entered as a comma-separated string and split on
 * submit. The due_date input is a datetime-local string
 * ("YYYY-MM-DDTHH:MM"); the modal converts it to RFC3339 before
 * sending so the server can accept it via parseFlexibleDate.
 *
 * Shared between create + edit modes — the parent passes an
 * optional `editingId` to switch between POST and PATCH.
 *
 * Tier 20 Phase E: refactored to use shared Modal/Input/Select/Textarea/
 * Button primitives. Custom sw-modal-backdrop / homelab-pin-form-row
 * markup and empty-state-cta buttons are gone.
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
    <Modal
      open
      onClose={onClose}
      title={editingId ? 'Edit todo' : 'New todo'}
      size="md"
    >
      <form
        onSubmit={(e) => {
          e.preventDefault();
          onSubmit();
        }}
        style={{ display: 'flex', flexDirection: 'column', gap: 12 }}
      >
        <Input
          id="todo-title"
          label="Title"
          type="text"
          value={form.title}
          onChange={(e) => setForm({ ...form, title: e.target.value })}
          placeholder="Buy new SSD for NAS"
          maxLength={200}
          required
          fullWidth
        />

        <Textarea
          id="todo-description"
          label="Description (optional)"
          value={form.description}
          onChange={(e) => setForm({ ...form, description: e.target.value })}
          placeholder="More context, links, or reminders."
          rows={4}
          fullWidth
        />

        <div style={{ display: 'flex', gap: 12 }}>
          <div style={{ flex: 1 }}>
            <Select
              id="todo-priority"
              label="Priority"
              value={form.priority}
              onChange={(e) => setForm({ ...form, priority: e.target.value as TodoPriority })}
              options={PRIORITIES.map((p) => ({ value: p.value, label: p.label }))}
              fullWidth
            />
          </div>
          <div style={{ flex: 1 }}>
            <Input
              id="todo-due"
              label="Due date (optional)"
              type="datetime-local"
              value={form.dueDate}
              onChange={(e) => setForm({ ...form, dueDate: e.target.value })}
              fullWidth
            />
          </div>
        </div>

        <Input
          id="todo-tags"
          label="Tags (comma-separated)"
          type="text"
          value={form.tagsRaw}
          onChange={(e) => setForm({ ...form, tagsRaw: e.target.value })}
          placeholder="homelab, urgent, todo"
          maxLength={200}
          description="Up to 16 tags, 32 chars each."
          fullWidth
        />

        {error ? <div className="dash-error" role="alert">{error}</div> : null}

        <div className="sw-form-actions" style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
          <Button type="button" variant="ghost" size="sm" onClick={onClose} disabled={busy}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" size="sm" disabled={busy || !form.title.trim()}>
            {busy ? 'Saving…' : editingId ? 'Save changes' : 'Create todo'}
          </Button>
        </div>
      </form>
    </Modal>
  );
}