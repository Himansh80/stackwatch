/**
 * Tier 14 Phase 14.1 — Empty + error states for the VM list page.
 */
import { motion } from 'framer-motion';

interface EmptyProps {
  kind: 'no-hosts' | 'no-vms' | 'no-filter-results' | 'error';
  message?: string;
}

export default function ProxmoxEmptyState({ kind, message }: EmptyProps) {
  const config = {
    'no-hosts': {
      icon: '⌬',
      title: 'No Proxmox hosts registered',
      hint: message ?? 'Add your first Proxmox host to start managing VMs and containers.',
    },
    'no-vms': {
      icon: '▣',
      title: 'No VMs on this host',
      hint: message ?? 'Click "+ Create VM" above to provision your first virtual machine.',
    },
    'no-filter-results': {
      icon: '◯',
      title: 'No matches',
      hint: message ?? 'Try clearing the search or changing the status filter.',
    },
    error: {
      icon: '⚠',
      title: 'Unable to load VMs',
      hint: message ?? 'Check that the host is reachable and try again.',
    },
  }[kind];

  return (
    <motion.div
      className={`px-empty px-empty-${kind}`}
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.25 }}
    >
      <div className="px-empty-icon">{config.icon}</div>
      <h3 className="px-empty-title">{config.title}</h3>
      <p className="px-empty-hint">{config.hint}</p>
    </motion.div>
  );
}