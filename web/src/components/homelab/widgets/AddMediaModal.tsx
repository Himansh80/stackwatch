import type { MediaServerKind } from './types';

/**
 * AddMediaModal — controlled form for POST /homelab/media/servers.
 *
 * Reuses the shared .sw-modal* classes from styles-tier2.css. The
 * kind dropdown shows all 3 supported kinds (Plex/Jellyfin/Emby);
 * the api_key field is shown for every kind but labelled with a
 * link to where to find the token.
 *
 * Lives in its own file (split from MediaWidget.tsx) so the
 * parent stays under 400 LOC.
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
    <div
      className="sw-modal-backdrop"
      role="dialog"
      aria-modal="true"
      aria-label="Pin a new media server"
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
          <h3 style={{ margin: 0, fontSize: 16 }}>Pin a media server</h3>

          <div className="homelab-pin-form-row">
            <label htmlFor="media-name">Name</label>
            <input
              id="media-name"
              type="text"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="Living Room Plex"
              autoFocus
              required
            />
          </div>

          <div className="homelab-pin-form-row">
            <label htmlFor="media-kind">Kind</label>
            <select
              id="media-kind"
              value={form.kind}
              onChange={(e) =>
                setForm({ ...form, kind: e.target.value as MediaServerKind })
              }
            >
              {KINDS.map((k) => (
                <option key={k} value={k}>
                  {kindLabel(k)}
                </option>
              ))}
            </select>
          </div>

          <div className="homelab-pin-form-row">
            <label htmlFor="media-url">Base URL</label>
            <input
              id="media-url"
              type="url"
              value={form.base_url}
              onChange={(e) => setForm({ ...form, base_url: e.target.value })}
              placeholder="http://192.168.1.20:32400"
              required
            />
          </div>

          <div className="homelab-pin-form-row">
            <label htmlFor="media-apikey">API key / token</label>
            <input
              id="media-apikey"
              type="password"
              value={form.api_key}
              onChange={(e) => setForm({ ...form, api_key: e.target.value })}
              placeholder={kindHelp(form.kind)}
              autoComplete="off"
              required
            />
          </div>

          {error ? (
            <div className="homelab-pin-form-error" role="alert">
              {error}
            </div>
          ) : null}

          <div className="homelab-pin-form-actions">
            <button
              type="button"
              className="empty-state-cta"
              onClick={onClose}
              disabled={busy}
            >
              Cancel
            </button>
            <button
              type="submit"
              className="empty-state-cta"
              disabled={busy}
            >
              {busy ? 'Pinning…' : 'Pin server'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
