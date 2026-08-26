/**
 * Tier 14 Phase 14.2 — Firewall tab.
 * Phase 14.2: view-only (lists rules from /cluster/firewall/rules).
 * Phase 14.3: add/edit/delete rules + aliases + IPSets.
 */
import { motion } from 'framer-motion';

interface FirewallRule {
  pos: number;
  action: 'ACCEPT' | 'REJECT' | 'DROP';
  source?: string;
  dest?: string;
  proto?: string;
  dport?: string;
  comment?: string;
  enable?: number;
}

interface Props {
  rules?: FirewallRule[];
  loading: boolean;
}

export default function ProxmoxDetailFirewall({ rules = [], loading }: Props) {
  if (loading) {
    return <div className="px-tab-pane"><p className="px-muted">Loading firewall rules…</p></div>;
  }

  if (rules.length === 0) {
    return (
      <motion.div
        className="px-tab-pane"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
      >
        <div className="px-info-card">
          <h3>No firewall rules</h3>
          <p>VM-level firewall rules from Proxmox will appear here.</p>
          <button type="button" className="px-btn px-btn-quiet" disabled title="Ships in Phase 14.3">
            + Add rule (coming in v2.3)
          </button>
        </div>
      </motion.div>
    );
  }

  return (
    <motion.div
      className="px-tab-pane"
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2 }}
    >
      <table className="px-network-table">
        <thead>
          <tr>
            <th>Pos</th>
            <th>Action</th>
            <th>Source</th>
            <th>Dest</th>
            <th>Proto</th>
            <th>Dport</th>
            <th>Comment</th>
          </tr>
        </thead>
        <tbody>
          {rules.map((rule) => (
            <tr key={rule.pos}>
              <td className="px-mono">{rule.pos}</td>
              <td>
                <span className={`px-action-badge px-action-${rule.action.toLowerCase()}`}>
                  {rule.action}
                </span>
              </td>
              <td className="px-mono">{rule.source || 'any'}</td>
              <td className="px-mono">{rule.dest || 'any'}</td>
              <td>{rule.proto || '—'}</td>
              <td className="px-mono">{rule.dport || '—'}</td>
              <td>{rule.comment || '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </motion.div>
  );
}