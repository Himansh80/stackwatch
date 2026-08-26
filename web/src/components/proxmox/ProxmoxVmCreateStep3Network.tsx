/**
 * Tier 14 Phase 14.5 — VM wizard step 3: Network (bridge + model + VLAN).
 */
import type { VmSpec } from './lib/vm-types';

interface Props {
  spec: VmSpec;
  onChange: (patch: Partial<VmSpec>) => void;
}

export default function ProxmoxVmCreateStep3Network({ spec, onChange }: Props) {
  return (
    <div>
      <h3 className="px-wizard-step-title">Network</h3>
      <p className="px-wizard-step-desc">
        Configure the primary network interface (Phase 14.6 will add multi-NIC).
      </p>

      <div className="px-form-grid">
        <div className="px-form-field">
          <label>Bridge *</label>
          <input
            type="text"
            value={spec.bridge}
            onChange={(e) => onChange({ bridge: e.target.value })}
            placeholder="vmbr0"
          />
        </div>

        <div className="px-form-field">
          <label>Network model</label>
          <select
            value={spec.netModel}
            onChange={(e) => onChange({ netModel: e.target.value as VmSpec['netModel'] })}
            className="px-form-select"
          >
            <option value="virtio">virtio (fast, modern)</option>
            <option value="e1000">e1000 (broadcom compat)</option>
            <option value="vmxnet3">vmxnet3 (VMware compat)</option>
          </select>
        </div>

        <div className="px-form-field px-form-full">
          <label>VLAN tag (optional)</label>
          <input
            type="text"
            value={spec.vlanTag}
            onChange={(e) => onChange({ vlanTag: e.target.value.replace(/\D/g, '') })}
            placeholder="10"
          />
        </div>
      </div>
    </div>
  );
}