/**
 * Tier 14 Phase 14.5 — VM wizard step 1: Choose ISO / media.
 */
import ProxmoxIsoPicker from './ProxmoxIsoPicker';
import type { Iso } from './ProxmoxIsoPicker';
export type { Iso };

interface Props {
  hostId: string;
  node: string;
  selectedVolid: string;
  onSelect: (iso: Iso) => void;
}

export default function ProxmoxVmCreateStep1Iso({ hostId, node, selectedVolid, onSelect }: Props) {
  return (
    <div>
      <h3 className="px-wizard-step-title">Choose installation media</h3>
      <p className="px-wizard-step-desc">
        Pick an ISO to install from, or choose &quot;No media&quot; if the VM boots from an existing disk.
      </p>
      <ProxmoxIsoPicker
        hostId={hostId}
        node={node}
        selected={selectedVolid}
        onSelect={(iso) => iso && onSelect(iso)}
      />
    </div>
  );
}