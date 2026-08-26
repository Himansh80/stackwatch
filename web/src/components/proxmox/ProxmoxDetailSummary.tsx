/**
 * Tier 14 Phase 14.2 — Summary tab content.
 * Shows VM metadata + live stats from cluster/resources.
 */
import { motion } from 'framer-motion';
import type { ProxmoxResource } from '../../lib/proxmox';
import { formatBytes, formatPercent, formatUptime, readString } from '../../lib/proxmox';

interface Props {
  vm: ProxmoxResource;
}

function Row({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="px-summary-row">
      <span className="px-summary-label">{label}</span>
      <span className="px-summary-value">{value}</span>
    </div>
  );
}

function MemoryBar({ used, total }: { used: number; total: number }) {
  const pct = total > 0 ? Math.round((used / total) * 100) : 0;
  const accent = pct > 85 ? 'red' : pct > 65 ? 'amber' : 'green';
  return (
    <div className="px-meter">
      <div className={`px-meter-fill px-meter-${accent}`} style={{ width: `${pct}%` }} />
      <span className="px-meter-text">{formatBytes(used)} / {formatBytes(total)} ({pct}%)</span>
    </div>
  );
}

export default function ProxmoxDetailSummary({ vm }: Props) {
  const memUsed = typeof vm.mem === 'number' ? vm.mem : 0;
  const memTotal = typeof vm.maxmem === 'number' ? vm.maxmem : 0;
  const diskUsed = typeof vm.disk === 'number' ? vm.disk : 0;
  const diskTotal = typeof vm.maxdisk === 'number' ? vm.maxdisk : 0;
  const cpu = typeof vm.cpu === 'number' ? vm.cpu : 0;

  return (
    <motion.div
      className="px-tab-pane"
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2 }}
    >
      <div className="px-summary-grid">
        <section className="px-summary-card">
          <h3>Identity</h3>
          <Row label="VMID" value={<span className="px-mono">{vm.vmid}</span>} />
          <Row label="Name" value={readString(vm.name) || '—'} />
          <Row label="Type" value={readString(vm.type) || 'qemu'} />
          <Row label="Node" value={readString(vm.node) || '—'} />
          <Row label="Status" value={readString(vm.status) || '—'} />
          <Row label="Uptime" value={formatUptime(vm.uptime)} />
          {readString(vm.pool) && <Row label="Pool" value={readString(vm.pool)} />}
        </section>

        <section className="px-summary-card">
          <h3>Resource usage (live)</h3>
          <Row label="CPU" value={formatPercent(cpu)} />
          <Row label="Memory" value={<MemoryBar used={memUsed} total={memTotal} />} />
          <Row label="Disk" value={<MemoryBar used={diskUsed} total={diskTotal} />} />
        </section>

        <section className="px-summary-card">
          <h3>Tags</h3>
          <div className="px-tag-list">
            {readString(vm.tags)
              ? readString(vm.tags)
                  .split(';')
                  .filter(Boolean)
                  .map((tag) => <span key={tag} className="px-tag-chip">{tag.trim()}</span>)
              : <span className="px-muted">No tags</span>}
          </div>
        </section>
      </div>
    </motion.div>
  );
}