/**
 * Tier 14 Phase 14.2 — VM detail header.
 * Breadcrumb + VM name + status pill + ID/Type/Node/Uptime + actions.
 */
import { motion } from 'framer-motion';
import { Link } from 'react-router-dom';
import type { ProxmoxResource } from '../../lib/proxmox';
import { formatUptime, readString } from '../../lib/proxmox';
import ProxmoxDetailActions from './ProxmoxDetailActions';

interface Props {
  vm: ProxmoxResource;
  hostId: string;
  busy: boolean;
  onAction: (action: 'start' | 'shutdown' | 'stop' | 'reboot' | 'delete') => void;
  onMigrateClick: () => void;
}

function statusAccent(status: string): 'green' | 'slate' | 'amber' | 'red' {
  const s = status.toLowerCase();
  if (s === 'running') return 'green';
  if (s === 'stopped') return 'slate';
  if (s === 'paused') return 'amber';
  return 'red';
}

export default function ProxmoxDetailHeader({ vm, hostId, busy, onAction, onMigrateClick }: Props) {
  const status = readString(vm.status);
  const accent = statusAccent(status);
  const node = readString(vm.node);

  return (
    <motion.div
      className="px-detail-header"
      initial={{ opacity: 0, y: -8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.25 }}
    >
      <div className="px-detail-breadcrumb">
        <Link to="/proxmox-vms">Proxmox</Link>
        <span className="px-detail-sep">›</span>
        <Link to="/proxmox-vms">{node}</Link>
        <span className="px-detail-sep">›</span>
        <span>VM {vm.vmid}</span>
      </div>

      <div className="px-detail-title-row">
        <h1 className="px-detail-name">{readString(vm.name) || `VM ${vm.vmid}`}</h1>
        <span className={`px-status px-status-${accent}`}>{status || 'unknown'}</span>
      </div>

      <div className="px-detail-meta">
        <span><strong>ID</strong> {vm.vmid}</span>
        <span><strong>Type</strong> {readString(vm.type) || 'qemu'}</span>
        <span><strong>Node</strong> {node}</span>
        <span><strong>Uptime</strong> {formatUptime(vm.uptime)}</span>
        {hostId && <span className="px-mono"><strong>Host</strong> {hostId.slice(0, 8)}</span>}
      </div>

      <ProxmoxDetailActions
        status={status}
        busy={busy}
        onAction={onAction}
        onMigrateClick={onMigrateClick}
      />
    </motion.div>
  );
}