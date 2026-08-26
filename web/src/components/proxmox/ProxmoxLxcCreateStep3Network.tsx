/**
 * Tier 14 Phase 14.4 — LXC wizard step 3: Network (bridge + IPv4 + IPv6 + VLAN).
 */
import type { LxcSpec } from './lib/lxc-types';

interface Props {
  spec: LxcSpec;
  onChange: (patch: Partial<LxcSpec>) => void;
}

export default function ProxmoxLxcCreateStep3Network({ spec, onChange }: Props) {
  return (
    <div>
      <h3 className="px-wizard-step-title">Network</h3>
      <p className="px-wizard-step-desc">
        Configure network interface, IP addressing, and optional VLAN tag.
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
          <label>VLAN tag (optional)</label>
          <input
            type="text"
            value={spec.vlanTag}
            onChange={(e) => onChange({ vlanTag: e.target.value.replace(/\D/g, '') })}
            placeholder="10"
          />
        </div>

        <div className="px-form-field">
          <label>IPv4 mode</label>
          <select
            value={spec.ipv4Mode}
            onChange={(e) => onChange({ ipv4Mode: e.target.value as LxcSpec['ipv4Mode'] })}
            className="px-form-select"
          >
            <option value="dhcp">DHCP</option>
            <option value="static">Static</option>
            <option value="none">None</option>
          </select>
        </div>

        {spec.ipv4Mode === 'static' && (
          <>
            <div className="px-form-field">
              <label>IPv4 address (CIDR)</label>
              <input
                type="text"
                value={spec.ipv4Addr}
                onChange={(e) => onChange({ ipv4Addr: e.target.value })}
                placeholder="192.168.0.50/24"
              />
            </div>
            <div className="px-form-field px-form-full">
              <label>IPv4 gateway</label>
              <input
                type="text"
                value={spec.ipv4Gw}
                onChange={(e) => onChange({ ipv4Gw: e.target.value })}
                placeholder="192.168.0.1"
              />
            </div>
          </>
        )}

        <div className="px-form-field">
          <label>IPv6 mode</label>
          <select
            value={spec.ipv6Mode}
            onChange={(e) => onChange({ ipv6Mode: e.target.value as LxcSpec['ipv6Mode'] })}
            className="px-form-select"
          >
            <option value="none">None</option>
            <option value="dhcp">DHCP</option>
            <option value="slaac">SLAAC</option>
            <option value="static">Static</option>
          </select>
        </div>

        {spec.ipv6Mode === 'static' && (
          <>
            <div className="px-form-field">
              <label>IPv6 address (CIDR)</label>
              <input
                type="text"
                value={spec.ipv6Addr}
                onChange={(e) => onChange({ ipv6Addr: e.target.value })}
                placeholder="2001:db8::50/64"
              />
            </div>
            <div className="px-form-field">
              <label>IPv6 gateway</label>
              <input
                type="text"
                value={spec.ipv6Gw}
                onChange={(e) => onChange({ ipv6Gw: e.target.value })}
                placeholder="2001:db8::1"
              />
            </div>
          </>
        )}
      </div>
    </div>
  );
}