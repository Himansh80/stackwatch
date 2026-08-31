import Button from '../../shared/Button';
import Input from '../../shared/Input';
import Modal from '../../shared/Modal';
import Select from '../../shared/Select';
import type { MediaServerKind } from './types';

/**
 * AddMediaModal — controlled form for POST /homelab/media/servers.
 *
 * The kind dropdown shows all 3 supported kinds (Plex/Jellyfin/Emby);
 * the api_key field is shown for every kind but labelled with a
 * link to where to find the token.
 *
 * Tier 20 Phase G: refactored custom sw-modal-backdrop +
 * homelab-pin-form-row + raw inputs/select to shared Modal + Input
 * + Select + Button.
 */

const KINDS: MediaServerKind[] = ['plex', 'jellyfin', 'emby'];

// kindHelp returns the per-kind "where do I find my token?"
// hint rendered as the api_key input's placeholder. The string
// mirrors the docs links for each upstream project.
function kindHelp(kind: MediaServerKind): string {
  switch (kind) {
    case 'plex':
      return 'Plex Web → Settings → Account → Show XML token';
    case 'jellyfin':
      return 'Jellyfin Dashboard → Administration → API Keys → +';
    case 'emby':
      return 'Emby Dashboard → Advanced → API Keys → + New API Key';
  }
}

function kindLabel(kind: MediaServerKind): string {
  switch (kind) {
    case 'plex':
      return 'Plex';
    case 'jellyfin':
      return 'Jellyfin';
    case 'emby':
      return 'Emby';
  }
}

interface AddMediaModalProps {
  form: {
    name: string;
    kind: MediaServerKind;
    base_url: string;
    api_key: string;
  };
  setForm: (next: AddMediaModalProps['form']) => void;
  busy: boolean;
  error: string;
  onClose: () => void;
  onSubmit: () => void;
}

export default function AddMediaModal({
  form,
  setForm,
  busy,
  error,
  onClose,
  onSubmit,
}: AddMediaModalProps) {
  return (
    <Modal open onClose={onClose} title="Pin a media server" size="md">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          onSubmit();
        }}
        style={{ display: 'flex', flexDirection: 'column', gap: 12 }}
      >
        <Input
          id="media-name"
          label="Name"
          type="text"
          value={form.name}
          onChange={(e) => setForm({ ...form, name: e.target.value })}
          placeholder="Living Room Plex"
          autoFocus
          required
          fullWidth
        />

        <Select
          id="media-kind"
          label="Kind"
          value={form.kind}
          onChange={(e) =>
            setForm({ ...form, kind: e.target.value as MediaServerKind })
          }
          fullWidth
          options={KINDS.map((k) => ({ value: k, label: kindLabel(k) }))}
        />

        <Input
          id="media-url"
          label="Base URL"
          type="url"
          value={form.base_url}
          onChange={(e) => setForm({ ...form, base_url: e.target.value })}
          placeholder="http://192.168.1.20:32400"
          required
          fullWidth
        />

        <Input
          id="media-apikey"
          label="API key / token"
          type="password"
          value={form.api_key}
          onChange={(e) => setForm({ ...form, api_key: e.target.value })}
          placeholder={kindHelp(form.kind)}
          autoComplete="off"
          required
          fullWidth
        />

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
          <Button type="submit" variant="primary" size="sm" disabled={busy} loading={busy}>
            {busy ? 'Pinning…' : 'Pin server'}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
