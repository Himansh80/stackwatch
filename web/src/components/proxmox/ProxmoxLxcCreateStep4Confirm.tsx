/**
 * Tier 14 Phase 14.4 — LXC wizard step 4: Confirm + submit.
 */
import type { LxcSpec } from './lib/lxc-types';

interface Props {
  spec: LxcSpec;
  templateName: string;
  onToggleStart: () => void;
}

export default function ProxmoxLxcCreateStep4Confirm({ spec, templateName, onToggleStart }: Props) {
  const ipLine = () => {
    const parts: string[] = [];
    if (spec.ipv4Mode === 'dhcp') parts.push('IPv4 DHCP');
    if (spec.ipv4Mode === 'static' && spec.ipv4Addr) parts.push(`IPv4 ${spec.ipv4Addr}${spec.ipv4Gw ? ` (gw ${spec.ipv4Gw})` : ''}`);
    if (spec.ipv6Mode !== 'none') parts.push(`IPv6 ${spec.ipv6Mode.toUpperCase()}${spec.ipv6Addr ? ` ${spec.ipv6Addr}` : ''}`);
    return parts.length ? parts.join(', ') : 'No networking';
  };

  return (
    <div>
      <h3 className="px-wizard-step-title">Review &amp; create</h3>
      <p className="px-wizard-step-desc">
        Double-check everything below, then click Create.
      </p>

      <div className="px-confirm-grid">
        <Row label="Template" value={templateName || '—'} />
        <Row label="VMID" value={spec.vmid} />
        <Row label="Hostname" value={spec.hostname || '—'} />
        <Row label="Cores" value={spec.cores} />
        <Row label="Memory" value={`${spec.memory} MB`} />
        <Row label="Swap" value={spec.swap ? `${spec.swap} MB` : 'none'} />
        <Row label="Disk" value={`${spec.disk} GB`} />
        <Row label="Root storage" value={spec.storage || '—'} />
        <Row label="Network" value={ipLine()} />
        <Row label="Bridge" value={spec.bridge} />
        <Row label="VLAN" value={spec.vlanTag || 'none'} />
      </div>

      <div className="px-form-field px-form-toggle">
        <label htmlFor="px-lxc-start">
          <input
            id="px-lxc-start"
            type="checkbox"
            checked={spec.startOnBoot}
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