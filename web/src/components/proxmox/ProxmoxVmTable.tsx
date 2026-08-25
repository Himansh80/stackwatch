/**
 * Tier 14 Phase 14.1 — VM table with mobile card layout.
 * Columns: Status / VMID / Type / Name / Node / IP / CPU / RAM / Uptime / Actions.
 * On <768px viewport, renders as a card list instead of a table.
 */
import { motion } from 'framer-motion';
import type { ProxmoxResource } from '../../lib/proxmox';
import {
  formatBytes,
  formatPercent,
  formatUptime,
  readString,
  type JsonObject,
} from '../../lib/proxmox';
import ProxmoxVmActions from './ProxmoxVmActions';

interface Props {
  rows: ProxmoxResource[];
  busy: boolean;
  onAction: (resource: ProxmoxResource, action: 'start' | 'shutdown' | 'stop' | 'reboot') => void;
  onRowClick?: (resource: ProxmoxResource) => void;
  emptyHint?: string;
}

function statusAccent(status: string): 'green' | 'slate' | 'amber' | 'red' {
  const s = status.toLowerCase();
  if (s === 'running') return 'green';
  if (s === 'stopped') return 'slate';
  if (s === 'paused') return 'amber';
  return 'red';
}

function deriveIp(row: ProxmoxResource): string {
  // Proxmox /cluster/resources can include `ip` or `ip6`; otherwise derive
  // from net config (often absent in /resources). Fall back to '—'.
  const ip = readString((row as JsonObject).ip);
  if (ip) return ip;
  const ip6 = readString((row as JsonObject).ip6);
  if (ip6) return ip6.split('/')[0];
  return '—';
}

export default function ProxmoxVmTable({ rows, busy, onAction, onRowClick, emptyHint }: Props) {
  if (rows.length === 0) {
    return (
      <div className="px-vm-empty">
        <p>{emptyHint ?? 'No VMs or containers match the current filters.'}</p>
      </div>
    );
  }

  return (
    <>
      {/* Desktop table */}
      <div className="px-vm-table-wrap px-hide-mobile">
        <table className="px-vm-table">
          <thead>
            <tr>
              <th>Status</th>
              <th>VMID</th>
              <th>Type</th>
              <th>Name</th>
              <th>Node</th>
              <th>IP</th>
              <th>CPU</th>
              <th>RAM</th>
              <th>Uptime</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row, idx) => {
              const status = readString(row.status);
              const accent = statusAccent(status);
              const memUsed = typeof row.mem === 'number' ? row.mem : 0;
              const memTotal = typeof row.maxmem === 'number' ? row.maxmem : 0;
              return (
                <motion.tr
                  key={`${row.node}-${row.vmid}-${idx}`}
                  initial={{ opacity: 0, y: 4 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.18, delay: idx * 0.015 }}
                  whileHover={{ backgroundColor: 'rgba(148, 163, 184, 0.06)' }}
                  onClick={() => onRowClick?.(row)}
                  className={onRowClick ? 'px-vm-row-clickable' : ''}
                >
                  <td>
                    <span className={`px-status px-status-${accent}`}>{status || 'unknown'}</span>
                  </td>
                  <td className="px-mono">{row.vmid}</td>
                  <td>
                    <span className="px-type-badge">{readString(row.type) || 'qemu'}</span>
                  </td>
                  <td className="px-vm-name">{readString(row.name) || '—'}</td>
                  <td>{readString(row.node) || '—'}</td>
                  <td className="px-mono">{deriveIp(row)}</td>
                  <td>{formatPercent(row.cpu)}</td>
                  <td>
                    {memTotal ? (
                      <>
                        {formatBytes(memUsed)} / <span className="px-muted">{formatBytes(memTotal)}</span>
                      </>
                    ) : (
                      <span className="px-muted">—</span>
                    )}
                  </td>
                  <td>{formatUptime(row.uptime)}</td>
                  <td onClick={(e) => e.stopPropagation()}>
                    <ProxmoxVmActions resource={row} busy={busy} onAction={onAction} />
                  </td>
                </motion.tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {/* Mobile card list */}
      <div className="px-vm-cards px-show-mobile">
        {rows.map((row, idx) => {
          const status = readString(row.status);
          const accent = statusAccent(status);
          const memUsed = typeof row.mem === 'number' ? row.mem : 0;
          const memTotal = typeof row.maxmem === 'number' ? row.maxmem : 0;
          return (
            <motion.div
              key={`m-${row.node}-${row.vmid}-${idx}`}
              initial={{ opacity: 0, y: 4 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.18, delay: idx * 0.015 }}
              className={`px-vm-card px-vm-card-${accent}`}
              onClick={() => onRowClick?.(row)}
            >
              <div className="px-vm-card-head">
                <span className={`px-status px-status-${accent}`}>{status || 'unknown'}</span>
                <span className="px-mono px-muted">#{row.vmid}</span>
              </div>
              <div className="px-vm-card-name">{readString(row.name) || '—'}</div>
              <div className="px-vm-card-meta">
                <span>{readString(row.type) || 'qemu'}</span>
                <span>·</span>
                <span>{readString(row.node) || '—'}</span>
              </div>
              <div className="px-vm-card-stats">
                <span>IP {deriveIp(row)}</span>
                <span>CPU {formatPercent(row.cpu)}</span>
                <span>
                  RAM {memTotal ? `${formatBytes(memUsed)}/${formatBytes(memTotal)}` : '—'}
                </span>
                <span>↑ {formatUptime(row.uptime)}</span>
              </div>
              <div onClick={(e) => e.stopPropagation()}>
                <ProxmoxVmActions resource={row} busy={busy} onAction={onAction} />
              </div>
            </motion.div>
          );
        })}
      </div>
    </>
  );
}