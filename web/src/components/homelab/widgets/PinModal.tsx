import Button from '../../shared/Button';
import Input from '../../shared/Input';
import Modal from '../../shared/Modal';
import Select from '../../shared/Select';
import type { PinFormState, ServiceKind } from './types';

/**
 * PinModal — controlled form for POST /homelab/services.
 *
 * Tier 20 Phase G: refactored custom sw-modal-backdrop +
 * sw-modal + homelab-pin-form-row + raw inputs to shared
 * Modal + Input + Select + Button.
 */

const KIND_HINT: Record<ServiceKind, string> = {
  http: 'http://host[:port]/path',
  https: 'https://host/path',
  tcp: 'host:port (e.g. 10.0.0.5:22)',
  icmp: 'hostname or IP (no scheme)',
};

interface PinModalProps {
  form: PinFormState;
  setForm: (next: PinFormState) => void;
  busy: boolean;
  error: string;
  onClose: () => void;
  onSubmit: () => void;
}

export default function PinModal({
  form, setForm, busy, error, onClose, onSubmit,
}: PinModalProps) {
  return (
    <Modal open onClose={onClose} title="Pin a service" size="md">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          onSubmit();
        }}
        style={{ display: 'flex', flexDirection: 'column', gap: 12 }}
      >
        <Input
          id="pin-name"
          label="Name"
          type="text"
          value={form.name}
          onChange={(e) => setForm({ ...form, name: e.target.value })}
          placeholder="Proxmox"
          maxLength={64}
          required
          fullWidth
          autoFocus
        />

        <Input
          id="pin-url"
          label="URL"
          type="text"
          value={form.url}
          onChange={(e) => setForm({ ...form, url: e.target.value })}
          placeholder={KIND_HINT[form.kind]}
          description={KIND_HINT[form.kind]}
          required
          fullWidth
        />

        <Select
          id="pin-kind"
          label="Kind"
          value={form.kind}
          onChange={(e) => setForm({ ...form, kind: e.target.value as ServiceKind })}
          fullWidth
          options={[
            { value: 'http', label: 'http' },
            { value: 'https', label: 'https' },
            { value: 'tcp', label: 'tcp (host:port)' },
            { value: 'icmp', label: 'icmp (ping)' },
          ]}
        />

        <Input
          id="pin-icon"
          label="Icon (emoji or short text)"
          type="text"
          value={form.icon}
          onChange={(e) => setForm({ ...form, icon: e.target.value })}
          placeholder="🛜"
          maxLength={8}
          fullWidth
        />

        {error ? <div className="dash-error" role="alert">{error}</div> : null}

        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
          <Button type="button" variant="ghost" size="sm" onClick={onClose} disabled={busy}>
            Cancel
          </Button>
          <Button
            type="submit"
            variant="primary"
            size="sm"
            disabled={busy || !form.name.trim() || !form.url.trim()}
            loading={busy}
          >
            {busy ? 'Pinning…' : 'Pin service'}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
