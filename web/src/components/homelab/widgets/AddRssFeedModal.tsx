import { ApiError } from '../../../lib/api';

/**
 * AddRssFeedModal — Tier 10 Phase 8 (H9 — RSS).
 *
 * The "+ Add feed" modal extracted from RssWidget so the main
 * widget stays under the 400-LOC cap. Owns the name / feed_url /
 * category form state and the POST /api/v1/homelab/rss/feeds
 * call. Validation runs server-side; this component only reflects
 * the API error in a friendly message.
 *
 * Props:
 *   form      — controlled form state
 *   setForm   — form state setter
 *   busy      — disables inputs while the POST is in flight
 *   error     — server-side validation error (already-friendly)
 *   onClose   — close without submitting
 *   onSubmit  — fire the POST + close + reload (parent owns the
 *               refresh logic)
 */

export interface RssFormState {
  name: string;
  feed_url: string;
  category: string;
}

const CATEGORY_OPTIONS = [
  'general',
  'news',
  'releases',
  'blogs',
  'podcasts',
  'other',
];

interface AddRssFeedModalProps {
  form: RssFormState;
  setForm: (next: RssFormState) => void;
  busy: boolean;
  error: string;
  onClose: () => void;
  onSubmit: () => void;
}

export default function AddRssFeedModal({
  form,
  setForm,
  busy,
  error,
  onClose,
  onSubmit,
}: AddRssFeedModalProps) {
  // Re-export the helper so RssWidget doesn't have to.
  void ApiError;
  return (
    <div
      className="homelab-modal-backdrop"
      role="dialog"
      aria-modal="true"
      onClick={onClose}
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.6)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 1000,
      }}
    >
      <div
        className="homelab-modal-panel"
        onClick={(e) => e.stopPropagation()}
        style={{
          background: 'var(--surface-2)',
          borderRadius: 8,
          padding: 24,
          minWidth: 360,
          maxWidth: 480,
          width: '90%',
        }}
      >
        <h3 style={{ marginTop: 0 }}>Add RSS feed</h3>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <label style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <span style={{ fontSize: 12, fontWeight: 600 }}>Name</span>
            <input
              type="text"
              className="homelab-search-input"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="r/selfhosted"
              maxLength={200}
            />
          </label>
          <label style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <span style={{ fontSize: 12, fontWeight: 600 }}>Feed URL</span>
            <input
              type="url"
              className="homelab-search-input"
              value={form.feed_url}
              onChange={(e) => setForm({ ...form, feed_url: e.target.value })}
              placeholder="https://example.com/feed.xml"
            />
          </label>
          <label style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <span style={{ fontSize: 12, fontWeight: 600 }}>Category</span>
            <select
              className="homelab-search-input"
              value={form.category}
              onChange={(e) => setForm({ ...form, category: e.target.value })}
            >
              {CATEGORY_OPTIONS.map((c) => (
                <option key={c} value={c}>
                  {c.charAt(0).toUpperCase() + c.slice(1)}
                </option>
              ))}
            </select>
          </label>
          {error ? (
            <div className="dash-error" role="alert">
              {error}
            </div>
          ) : null}
          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
            <button type="button" className="empty-state-cta" onClick={onClose} disabled={busy}>
              Cancel
            </button>
            <button
              type="button"
              className="empty-state-cta"
              onClick={onSubmit}
              disabled={busy || !form.name.trim() || !form.feed_url.trim()}
            >
              {busy ? 'Adding…' : 'Add feed'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
