import type { PinFormState, ServiceKind } from './types';

/**
 * PinModal — controlled form for POST /homelab/services.
 *
 * Uses the shared .sw-modal* classes from styles-tier2.css so it
 * inherits the platform modal look without re-styling. Lives in
 * its own file (split out of ServiceStatusWidget.tsx) so the
 * parent file stays under 400 LOC.
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
    <div
      className="sw-modal-backdrop"
      role="dialog"
      aria-modal="true"
      aria-label="Pin a new service"
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
          <h3 style={{ margin: 0, fontSize: 16 }}>Pin a service</h3>

          <div className="homelab-pin-form-row">
            <label htmlFor="pin-name">Name</label>
            <input
              id="pin-name"
              type="text"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="Proxmox"
              maxLength={64}
              required
            />
          </div>

          <div className="homelab-pin-form-row">
            <label htmlFor="pin-url">URL</label>
            <input
              id="pin-url"
              type="text"
              value={form.url}
              onChange={(e) => setForm({ ...form, url: e.target.value })}
              placeholder={KIND_HINT[form.kind]}
              required
            />
            <p className="homelab-pin-form-hint">{KIND_HINT[form.kind]}</p>
          </div>

          <div className="homelab-pin-form-row">
            <label htmlFor="pin-kind">Kind</label>
            <select
              id="pin-kind"
              value={form.kind}
              onChange={(e) => setForm({ ...form, kind: e.target.value as ServiceKind })}
            >
              <option value="http">http</option>
              <option value="https">https</option>
              <option value="tcp">tcp (host:port)</option>
              <option value="icmp">icmp (ping)</option>
            </select>
          </div>

          <div className="homelab-pin-form-row">
            <label htmlFor="pin-icon">Icon (emoji or short text)</label>
            <input
              id="pin-icon"
              type="text"
              value={form.icon}
              onChange={(e) => setForm({ ...form, icon: e.target.value })}
              placeholder="🛜"
              maxLength={8}
            />
          </div>

          {error ? <div className="dash-error" role="alert">{error}</div> : null}

          <div className="sw-form-actions" style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <button type="button" className="empty-state-cta" onClick={onClose} disabled={busy}>
              Cancel
            </button>
            <button type="submit" className="empty-state-cta" disabled={busy || !form.name.trim() || !form.url.trim()}>
              {busy ? 'Pinning…' : 'Pin service'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}