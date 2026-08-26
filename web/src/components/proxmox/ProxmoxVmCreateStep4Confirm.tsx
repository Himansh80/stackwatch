/**
 * Tier 14 Phase 14.5 — VM wizard step 4: Confirm + submit.
 */
import type { VmSpec } from './lib/vm-types';

interface Props {
  spec: VmSpec;
  isoName: string;
  onToggleStart: () => void;
}

export default function ProxmoxVmCreateStep4Confirm({ spec, isoName, onToggleStart }: Props) {
  return (
    <div>
      <h3 className="px-wizard-step-title">Review &amp; create</h3>
      <p className="px-wizard-step-desc">
        Double-check everything below, then click Create.
      </p>

      <div className="px-confirm-grid">
        <Row label="Installation media" value={isoName || 'No media'} />
        <Row label="VMID" value={spec.vmid} />
        <Row label="Name" value={spec.name || '—'} />
        <Row label="Cores" value={spec.cores} />
        <Row label="Memory" value={`${spec.memory} MB`} />
        <Row label="Disk" value={`${spec.disk} GB`} />
        <Row label="Storage" value={spec.storage || '—'} />
        <Row label="BIOS" value={spec.bios} />
        <Row label="Machine" value={spec.machine} />
        <Row label="CPU type" value={spec.cpuType} />
        <Row label="Bridge" value={spec.bridge} />
        <Row label="Network model" value={spec.netModel} />
        <Row label="VLAN" value={spec.vlanTag || 'none'} />
      </div>

      <div className="px-form-field px-form-toggle">
        <label htmlFor="px-vm-start">
          <input
            id="px-vm-start"
            type="checkbox"
            checked={spec.startAfterCreate}
            onChange={onToggleStart}
          />
          Start after creation
        </label>
      </div>
    </div>
  );
}

function Row({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="px-summary-row">
      <span className="px-summary-label">{label}</span>
      <span className="px-summary-value px-mono">{value}</span>
    </div>
  );
}