/**
 * Tier 14 Phase 14.6 — Single NIC editor row.
 * Reusable: passes index, nic, onChange, onRemove.
 */
import { motion } from 'framer-motion';
import type { Nic } from './lib/vm-types';

interface Props {
  index: number;
  nic: Nic;
  onChange: (patch: Partial<Nic>) => void;
  onRemove: () => void;
}

const MODELS: Nic['netModel'][] = ['virtio', 'e1000', 'vmxnet3'];

export default function ProxmoxNicRow({ index, nic, onChange, onRemove }: Props) {
  return (
    <motion.div
      className="px-nic-row"
      initial={{ opacity: 0, x: -8 }}
      animate={{ opacity: 1, x: 0 }}
      exit={{ opacity: 0, x: 8 }}
      transition={{ duration: 0.18 }}
    >
      <div className="px-nic-label">nic{index}</div>
      <div className="px-nic-fields">
        <input
          type="text"
          value={nic.bridge}
          onChange={(e) => onChange({ bridge: e.target.value })}
          placeholder="vmbr0"
          className="px-confirm-input px-nic-bridge"
          aria-label={`NIC ${index} bridge`}
        />
        <select
          value={nic.netModel}
          onChange={(e) => onChange({ netModel: e.target.value as Nic['netModel'] })}
          className="px-form-select px-nic-model"
          aria-label={`NIC ${index} model`}
        >
          {MODELS.map((m) => <option key={m} value={m}>{m}</option>)}
        </select>
        <input
          type="text"
          value={nic.vlanTag}
          onChange={(e) => onChange({ vlanTag: e.target.value.replace(/\D/g, '') })}
          placeholder="VLAN (optional)"
          className="px-confirm-input px-nic-vlan"
          aria-label={`NIC ${index} VLAN tag`}
        />
        <button
          type="button"
          className="px-btn px-btn-quiet px-btn-danger-text"
          onClick={onRemove}
          aria-label={`Remove NIC ${index}`}
        >
          ✕
        </button>
      </div>
    </motion.div>
  );
}