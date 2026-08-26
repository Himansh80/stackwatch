/**
 * Tier 14 Phase 14.4 — LXC wizard step 1: Choose template.
 */
import ProxmoxLxcTemplatePicker from './ProxmoxLxcTemplatePicker';
import type { Template } from './ProxmoxLxcTemplatePicker';
export type { Template };

interface Props {
  hostId: string;
  node: string;
  selectedVolid: string;
  onSelect: (template: Template) => void;
}

export default function ProxmoxLxcCreateStep1Template({ hostId, node, selectedVolid, onSelect }: Props) {
  return (
    <div>
      <h3 className="px-wizard-step-title">Choose a template</h3>
      <p className="px-wizard-step-desc">
        Pick an LXC template (vztmpl) to use as the root filesystem.
      </p>
      <ProxmoxLxcTemplatePicker
        hostId={hostId}
        node={node}
        selected={selectedVolid}
        onChange={(t) => t && onSelect(t)}
      />
    </div>
  );
}