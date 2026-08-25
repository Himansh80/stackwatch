import type { NoteFormState } from './types';

/**
 * NoteModal — controlled form for POST/PATCH /homelab/notes.
 *
 * Reuses the same .sw-modal* classes the PinModal uses. Tags are
 * entered as a comma-separated string and split on submit (the
 * server stores them as text[]). The form is shared between
 * create + edit modes — the parent passes an optional `editingId`
 * to switch between POST and PATCH.
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
    <div
      className="sw-modal-backdrop"
      role="dialog"
      aria-modal="true"
      aria-label={editingId ? 'Edit note' : 'New note'}
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
            {editingId ? 'Edit note' : 'New note'}
          </h3>

          <div className="homelab-pin-form-row">
            <label htmlFor="note-title">Title</label>
            <input
              id="note-title"
              type="text"
              value={form.title}
              onChange={(e) => setForm({ ...form, title: e.target.value })}
              placeholder="Buy new SSD for NAS"
              maxLength={200}
              required
            />
          </div>

          <div className="homelab-pin-form-row">
            <label htmlFor="note-body">Body (Markdown allowed)</label>
            <textarea
              id="note-body"
              value={form.body}
              onChange={(e) => setForm({ ...form, body: e.target.value })}
              placeholder="Notes go here. Plain text or Markdown."
              rows={6}
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

          <div className="homelab-pin-form-row">
            <label htmlFor="note-tags">Tags (comma-separated)</label>
            <input
              id="note-tags"
              type="text"
              value={form.tagsRaw}
              onChange={(e) => setForm({ ...form, tagsRaw: e.target.value })}
              placeholder="homelab, urgent, todo"
              maxLength={200}
            />
            <p className="homelab-pin-form-hint">Up to 16 tags, 32 chars each.</p>
          </div>

          <div className="homelab-pin-form-row">
            <label htmlFor="note-pinned" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <input
                id="note-pinned"
                type="checkbox"
                checked={form.pinned}
                onChange={(e) => setForm({ ...form, pinned: e.target.checked })}
              />
              Pin to top of list
            </label>
          </div>

          {error ? <div className="dash-error" role="alert">{error}</div> : null}

          <div className="sw-form-actions" style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <button type="button" className="empty-state-cta" onClick={onClose} disabled={busy}>
              Cancel
            </button>
            <button type="submit" className="empty-state-cta" disabled={busy || !form.title.trim()}>
              {busy ? 'Saving…' : editingId ? 'Save changes' : 'Create note'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
