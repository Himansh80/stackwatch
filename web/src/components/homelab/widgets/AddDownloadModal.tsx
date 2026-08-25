import { kindLabel } from './downloadFormatters';
import type { DownloadClientFormState, DownloadKind } from './types';

/**
 * AddDownloadModal — controlled form for POST /homelab/downloads/clients.
 *
 * Reuses the shared .sw-modal* classes from styles-tier2.css. The
 * kind dropdown shows all 6 supported kinds; the api_key field is
 * shown for every kind but labelled with the kind's expected auth
 * (api_key for *arr/SABnzbd; api_key OR username+password for
 * qBittorrent). The username/password fields are only rendered
 * when kind === 'qbittorrent' since the other kinds ignore them.
 *
 * Lives in its own file (split from DownloadStatsWidget.tsx) so
 * the parent stays under 400 LOC.
 */

const KINDS: DownloadKind[] = [
  'sonarr',
  'radarr',
  'qbittorrent',
  'sabnzbd',
  'lidarr',
  'readarr',
];

interface AddDownloadModalProps {
  form: DownloadClientFormState;
  setForm: (next: DownloadClientFormState) => void;
  busy: boolean;
  error: string;
  onClose: () => void;
  onSubmit: () => void;
}

export default function AddDownloadModal({
  form,
  setForm,
  busy,
  error,
  onClose,
  onSubmit,
}: AddDownloadModalProps) {
  const isQbittorrent = form.kind === 'qbittorrent';
  return (
    <div
      className="sw-modal-backdrop"
      role="dialog"
      aria-modal="true"
      aria-label="Pin a new download client"
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
          <h3 style={{ margin: 0, fontSize: 16 }}>Pin a download client</h3>

          <div className="homelab-pin-form-row">
            <label htmlFor="dl-name">Name</label>
            <input
              id="dl-name"
              type="text"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="Home Sonarr"
              autoFocus
              required
            />
          </div>

          <div className="homelab-pin-form-row">
            <label htmlFor="dl-kind">Kind</label>
            <select
              id="dl-kind"
              value={form.kind}
              onChange={(e) =>
                setForm({ ...form, kind: e.target.value as DownloadKind })
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
            <label htmlFor="dl-url">Base URL</label>
            <input
              id="dl-url"
              type="url"
              value={form.base_url}
              onChange={(e) => setForm({ ...form, base_url: e.target.value })}
              placeholder="http://192.168.1.50:8989"
              required
            />
          </div>

          <div className="homelab-pin-form-row">
            <label htmlFor="dl-apikey">
              {isQbittorrent ? 'API key (optional if username/password set)' : 'API key'}
            </label>
            <input
              id="dl-apikey"
              type="password"
              value={form.api_key}
              onChange={(e) => setForm({ ...form, api_key: e.target.value })}
              placeholder="Settings → General → API Key"
              autoComplete="off"
            />
          </div>

          {isQbittorrent ? (
            <>
              <div className="homelab-pin-form-row">
                <label htmlFor="dl-user">Username (optional)</label>
                <input
                  id="dl-user"
                  type="text"
                  value={form.username}
                  onChange={(e) => setForm({ ...form, username: e.target.value })}
                  placeholder="admin"
                  autoComplete="off"
                />
              </div>
              <div className="homelab-pin-form-row">
                <label htmlFor="dl-pass">Password (optional)</label>
                <input
                  id="dl-pass"
                  type="password"
                  value={form.password}
                  onChange={(e) => setForm({ ...form, password: e.target.value })}
                  placeholder="adminadmin"
                  autoComplete="new-password"
                />
              </div>
            </>
          ) : null}

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
              {busy ? 'Pinning…' : 'Pin client'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}