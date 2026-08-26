/**
 * Tier 14 Phase 14.2 — Network tab content.
 * Uses QEMU guest agent network-get-interfaces. If agent isn't running,
 * shows an info card telling the user how to install it.
 */
import { motion } from 'framer-motion';
import type { VMNetworkInterface } from './lib/proxmox-vm-types';

interface Props {
  interfaces: VMNetworkInterface[] | null;
  agentRunning: boolean;
  loading: boolean;
}

export default function ProxmoxDetailNetwork({ interfaces, agentRunning, loading }: Props) {
  if (loading) {
    return (
      <div className="px-tab-pane">
        <p className="px-muted">Loading network info…</p>
      </div>
    );
  }

  if (!agentRunning) {
    return (
      <motion.div
        className="px-tab-pane"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
      >
        <div className="px-info-card">
          <h3>QEMU guest agent is not running inside this VM</h3>
          <p>
            To see the guest's IP address, MAC, and bridge info, install the{' '}
            <code>qemu-guest-agent</code> package inside the VM and enable it:
          </p>
          <pre className="px-code-block">apt install qemu-guest-agent
systemctl enable qemu-guest-agent
systemctl start qemu-guest-agent</pre>
          <p className="px-muted">
            Then enable <em>Guest Agent</em> in the VM's <em>Options</em> tab in Proxmox
            and restart the VM.
          </p>
        </div>
      </motion.div>
    );
  }

  if (!interfaces || interfaces.length === 0) {
    return (
      <div className="px-tab-pane">
        <p className="px-muted">No network interfaces reported by the guest agent.</p>
      </div>
    );
  }

  return (
    <motion.div
      className="px-tab-pane"
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2 }}
    >
      <div className="px-info-card px-info-card-good">
        <strong>QEMU guest agent:</strong> running
      </div>
      <table className="px-network-table">
        <thead>
          <tr>
            <th>Interface</th>
            <th>MAC</th>
            <th>IP address(es)</th>
            <th>MTU</th>
          </tr>
        </thead>
        <tbody>
          {interfaces.map((iface, idx) => (
            <tr key={`${iface.name}-${idx}`}>
              <td className="px-mono">{iface.name}</td>
              <td className="px-mono">{iface.hardware_addr || '—'}</td>
              <td>
                {iface.ip_addresses && iface.ip_addresses.length > 0 ? (
                  <div className="px-ip-list">
                    {iface.ip_addresses.map((ip, i) => (
                      <span key={`${ip.ip_address}-${i}`} className="px-ip-chip">
                        {ip.ip_address}/{ip.prefix}
                        {ip.ip_address_type === 'ipv6' ? ' (v6)' : ''}
                      </span>
                    ))}
                  </div>
                ) : (
                  <span className="px-muted">—</span>
                )}
              </td>
              <td className="px-mono">{iface.mtu || '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </motion.div>
  );
}