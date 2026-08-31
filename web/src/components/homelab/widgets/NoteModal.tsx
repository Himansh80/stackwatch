import Button from '../../shared/Button';
import Input from '../../shared/Input';
import Modal from '../../shared/Modal';
import Textarea from '../../shared/Textarea';
import type { NoteFormState } from './types';

/**
 * NoteModal — controlled form for POST/PATCH /homelab/notes.
 *
 * Tags are entered as a comma-separated string and split on submit
 * (the server stores them as text[]). The form is shared between
 * create + edit modes — the parent passes an optional `editingId`
 * to switch between POST and PATCH.
 *
 * Tier 20 Phase G: refactored custom sw-modal-backdrop +
 * sw-modal + homelab-pin-form-row + raw inputs/textarea to shared
 * Modal + Input + Textarea + Button.
 */

interface NoteModalProps {
  form: NoteFormState;
  setForm: (next: NoteFormState) => void;
  busy: boolean;
  error: string;
  editingId: string | null;
  onClose: () => void;
  onSubmit: () => void;
}

export default function NoteModal({
  form, setForm, busy, error, editingId, onClose, onSubmit,
}: NoteModalProps) {
  return (
    <Modal
      open
      onClose={onClose}
      title={editingId ? 'Edit note' : 'New note'}
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
          id="note-title"
          label="Title"
          type="text"
          value={form.title}
          onChange={(e) => setForm({ ...form, title: e.target.value })}
          placeholder="Buy new SSD for NAS"
          maxLength={200}
          required
          autoFocus
          fullWidth
        />

        <Textarea
          id="note-body"
          label="Body (Markdown allowed)"
          value={form.body}
          onChange={(e) => setForm({ ...form, body: e.target.value })}
          placeholder="Notes go here. Plain text or Markdown."
          rows={6}
          fullWidth
        />

        <Input
          id="note-tags"
          label="Tags (comma-separated)"
          type="text"
          value={form.tagsRaw}
          onChange={(e) => setForm({ ...form, tagsRaw: e.target.value })}
          placeholder="homelab, urgent, todo"
          maxLength={200}
          description="Up to 16 tags, 32 chars each."
          fullWidth
        />

        <label
          htmlFor="note-pinned"
          style={{ display: 'flex', alignItems: 'center', gap: 8 }}
        >
          <input
            id="note-pinned"
            type="checkbox"
            checked={form.pinned}
            onChange={(e) => setForm({ ...form, pinned: e.target.checked })}
          />
          Pin to top of list
        </label>

        {error ? <div className="dash-error" role="alert">{error}</div> : null}

        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
          <Button type="button" variant="ghost" size="sm" onClick={onClose} disabled={busy}>
            Cancel
          </Button>
          <Button
            type="submit"
            variant="primary"
            size="sm"
            disabled={busy || !form.title.trim()}
            loading={busy}
          >
            {busy ? 'Saving…' : editingId ? 'Save changes' : 'Create note'}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
