import Button from '../../shared/Button';
import Input from '../../shared/Input';
import Modal from '../../shared/Modal';
import Select from '../../shared/Select';

/**
 * AddRssFeedModal — Tier 10 Phase 8 (H9 — RSS).
 *
 * The "+ Add feed" modal extracted from RssWidget so the main
 * widget stays under the 400-LOC cap. Owns the name / feed_url /
 * category form state and the POST /api/v1/homelab/rss/feeds
 * call. Validation runs server-side; this component only reflects
 * the API error in a friendly message.
 *
 * Tier 20 Phase G: refactored custom homelab-modal-backdrop +
 * homelab-modal-panel + homelab-search-input to shared Modal +
 * Input + Select + Button.
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

function categoryLabel(c: string): string {
  return c.charAt(0).toUpperCase() + c.slice(1);
}

export default function AddRssFeedModal({
  form,
  setForm,
  busy,
  error,
  onClose,
  onSubmit,
}: AddRssFeedModalProps) {
  return (
    <Modal open onClose={onClose} title="Add RSS feed" size="md">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          onSubmit();
        }}
        style={{ display: 'flex', flexDirection: 'column', gap: 12 }}
      >
        <Input
          id="rss-name"
          label="Name"
          type="text"
          value={form.name}
          onChange={(e) => setForm({ ...form, name: e.target.value })}
          placeholder="r/selfhosted"
          maxLength={200}
          required
          fullWidth
          autoFocus
        />

        <Input
          id="rss-url"
          label="Feed URL"
          type="url"
          value={form.feed_url}
          onChange={(e) => setForm({ ...form, feed_url: e.target.value })}
          placeholder="https://example.com/feed.xml"
          required
          fullWidth
        />

        <Select
          id="rss-category"
          label="Category"
          value={form.category}
          onChange={(e) => setForm({ ...form, category: e.target.value })}
          fullWidth
          options={CATEGORY_OPTIONS.map((c) => ({
            value: c,
            label: categoryLabel(c),
          }))}
        />

        {error ? <div className="dash-error" role="alert">{error}</div> : null}

        <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
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
            disabled={busy || !form.name.trim() || !form.feed_url.trim()}
            loading={busy}
          >
            {busy ? 'Adding…' : 'Add feed'}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
