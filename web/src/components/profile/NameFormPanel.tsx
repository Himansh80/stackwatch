import { FormEvent } from 'react';

interface NameFormPanelProps {
  draft: string;
  onDraftChange: (next: string) => void;
  onSubmit: (event: FormEvent) => void | Promise<void>;
  onDiscard: () => void;
  saving: boolean;
  unchanged: boolean;
  error: string | null;
  message: string | null;
}

/**
 * NameFormPanel — the "Display name" form. Lets the user rename
 * themselves. Save is disabled until the draft differs from the
 * saved value, so accidental clicks do nothing.
 */
export default function NameFormPanel({
  draft,
  onDraftChange,
  onSubmit,
  onDiscard,
  saving,
  unchanged,
  error,
  message,
}: NameFormPanelProps) {
  return (
    <article className="dash-panel prof-panel">
      <div className="dash-panel-head">
        <div>
          <span className="dash-eyebrow">Profile</span>
          <h3>Display name</h3>
        </div>
      </div>
      <form onSubmit={onSubmit} className="sw-form-grid prof-form">
        <label className="sw-field">
          <span>Display name</span>
          <input
            type="text"
            value={draft}
            onChange={(e) => onDraftChange(e.target.value)}
            placeholder="Your full name"
            maxLength={255}
            required
            autoFocus
          />
          <small>This is the name shown across the dashboard and in alerts.</small>
        </label>
        {error && <div className="auth-error prof-form-msg">{error}</div>}
        {message && !error && <div className="dash-banner-ok prof-form-msg">{message}</div>}
        <div className="sw-form-actions">
          <button type="button" className="sw-button sw-button-quiet" onClick={onDiscard} disabled={saving}>
            Discard
          </button>
          <button type="submit" className="sw-button sw-button-primary" disabled={saving || unchanged}>
            {saving ? 'Saving…' : 'Save name'}
          </button>
        </div>
      </form>
    </article>
  );
}
