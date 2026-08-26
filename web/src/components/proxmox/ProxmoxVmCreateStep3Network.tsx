/**
 * Tier 14 Phase 14.6 — VM wizard step 3: Network (multi-NIC with add/remove).
 */
import { AnimatePresence } from 'framer-motion';
import ProxmoxNicRow from './ProxmoxNicRow';
import { MAX_NICS, type Nic, type VmSpec } from './lib/vm-types';

interface Props {
  spec: VmSpec;
  onChange: (patch: Partial<VmSpec>) => void;
}

export default function ProxmoxVmCreateStep3Network({ spec, onChange }: Props) {
  function addNic() {
    if (spec.nics.length >= MAX_NICS) return;
    onChange({ nics: [...spec.nics, { bridge: 'vmbr0', netModel: 'virtio', vlanTag: '' }] });
  }
  function updateNic(idx: number, patch: Partial<Nic>) {
    onChange({ nics: spec.nics.map((n, i) => (i === idx ? { ...n, ...patch } : n)) });
  }
  function removeNic(idx: number) {
    onChange({ nics: spec.nics.filter((_, i) => i !== idx) });
  }

  return (
    <div>
      <h3 className="px-wizard-step-title">Network</h3>
      <p className="px-wizard-step-desc">
        Configure up to {MAX_NICS} network interfaces. Each can be on a different bridge / VLAN.
      </p>

      <div className="px-nic-toolbar">
        <span className="px-muted">{spec.nics.length} of {MAX_NICS} NICs</span>
        <button
          type="button"
          className="px-btn px-btn-quiet"
          onClick={addNic}
          disabled={spec.nics.length >= MAX_NICS}
        >
          + Add NIC
        </button>
      </div>

      <div className="px-nic-list">
        <AnimatePresence>
          {spec.nics.map((nic, idx) => (
            <ProxmoxNicRow
              key={idx}
              index={idx}
              nic={nic}
              onChange={(patch) => updateNic(idx, patch)}
              onRemove={() => removeNic(idx)}
            />
          ))}
        </AnimatePresence>
      </div>

      {spec.nics.length === 0 && (
        <p className="px-muted" style={{ marginTop: 12 }}>No NICs configured. Click "+ Add NIC" to add one.</p>
      )}
    </div>
  );
}