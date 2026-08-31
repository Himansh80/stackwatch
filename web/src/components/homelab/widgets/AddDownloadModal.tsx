import { kindLabel } from './downloadFormatters';
import type { DownloadClientFormState, DownloadKind } from './types';
import Button from '../../shared/Button';
import Input from '../../shared/Input';
import Modal from '../../shared/Modal';
import Select from '../../shared/Select';

/**
 * AddDownloadModal — controlled form for POST /homelab/downloads/clients.
 *
 * The kind dropdown shows all 6 supported kinds; the api_key field is
 * shown for every kind but labelled with the kind's expected auth
 * (api_key for *arr/SABnzbd; api_key OR username+password for
 * qBittorrent). The username/password fields are only rendered
 * when kind === 'qbittorrent' since the other kinds ignore them.
 *
 * Lives in its own file (split from DownloadStatsWidget.tsx) so
 * the parent stays under 400 LOC.
 *
 * Tier 20 Phase E: refactored to use shared Modal/Input/Select/Button
 * primitives. Custom sw-modal-backdrop / homelab-pin-form-row markup and
 * empty-state-cta buttons are gone.
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
    <Modal
      open
      onClose={onClose}
      title="Pin a download client"
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
          id="dl-name"
          label="Name"
          type="text"
          value={form.name}
          onChange={(e) => setForm({ ...form, name: e.target.value })}
          placeholder="Home Sonarr"
          autoFocus
          required
          fullWidth
        />

        <Select
          id="dl-kind"
          label="Kind"
          value={form.kind}
          onChange={(e) =>
            setForm({ ...form, kind: e.target.value as DownloadKind })
          }
          options={KINDS.map((k) => ({ value: k, label: kindLabel(k) }))}
          fullWidth
        />

        <Input
          id="dl-url"
          label="Base URL"
          type="url"
          value={form.base_url}
          onChange={(e) => setForm({ ...form, base_url: e.target.value })}
          placeholder="http://192.168.1.50:8989"
          required
          fullWidth
        />

        <Input
          id="dl-apikey"
          label={isQbittorrent ? 'API key (optional if username/password set)' : 'API key'}
          type="password"
          value={form.api_key}
          onChange={(e) => setForm({ ...form, api_key: e.target.value })}
          placeholder="Settings → General → API Key"
          autoComplete="off"
          fullWidth
        />

        {isQbittorrent ? (
          <>
            <Input
              id="dl-user"
              label="Username (optional)"
              type="text"
              value={form.username}
              onChange={(e) => setForm({ ...form, username: e.target.value })}
              placeholder="admin"
              autoComplete="off"
              fullWidth
            />
            <Input
              id="dl-pass"
              label="Password (optional)"
              type="password"
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
              placeholder="adminadmin"
              autoComplete="new-password"
              fullWidth
            />
          </>
        ) : null}

        {error ? (
          <div className="homelab-pin-form-error" role="alert">
            {error}
          </div>
        ) : null}

        <div className="homelab-pin-form-actions">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={onClose}
            disabled={busy}
          >
            Cancel
          </Button>
          <Button type="submit" variant="primary" size="sm" disabled={busy}>
            {busy ? 'Pinning…' : 'Pin client'}
          </Button>
        </div>
      </form>
    </Modal>
  );
}