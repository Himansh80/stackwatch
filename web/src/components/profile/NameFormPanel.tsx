import { FormEvent } from 'react';
import Button from '../shared/Button';
import Input from '../shared/Input';

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
        <Input
          label="Display name"
          value={draft}
          onChange={(e) => onDraftChange(e.target.value)}
          placeholder="Your full name"
          maxLength={255}
          required
          autoFocus
          description="This is the name shown across the dashboard and in alerts."
        />
        {error && <div className="auth-error prof-form-msg">{error}</div>}
        {message && !error && <div className="dash-banner-ok prof-form-msg">{message}</div>}
        <div className="sw-form-actions">
          <Button type="button" variant="ghost" onClick={onDiscard} disabled={saving}>
            Discard
          </Button>
          <Button type="submit" variant="primary" loading={saving} disabled={saving || unchanged}>
            {saving ? 'Saving…' : 'Save name'}
          </Button>
        </div>
      </form>
    </article>
  );
}
