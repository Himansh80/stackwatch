/**
 * Tier 14 Phase 14.1 — VM table with mobile card layout.
 * Columns: Status / VMID / Type / Name / Node / IP / CPU / RAM / Uptime / Actions.
 * On <768px viewport, renders as a card list instead of a table.
 */
import { motion } from 'framer-motion';
import { useNavigate } from 'react-router-dom';
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
  /** Host id to construct detail URL when clicking a row */
  hostId?: string;
  /** Map of vmid -> IP from the QEMU guest agent. The table reads
   *  this first, falls back to row.ip, then to '—'. */
  ips?: Record<number, string>;
}

function statusAccent(status: string): 'green' | 'slate' | 'amber' | 'red' {
  const s = status.toLowerCase();
  if (s === 'running') return 'green';
  if (s === 'stopped') return 'slate';
  if (s === 'paused') return 'amber';
  return 'red';
}

function deriveIp(row: ProxmoxResource, ips?: Record<number, string>): string {
  // 1. Guest-agent-discovered IP (most accurate)
  if (ips && row.vmid !== undefined) {
    const agentIp = ips[Number(row.vmid)];
    if (agentIp) return agentIp;
  }
  // 2. Row-provided IP (if any) — /vms doesn't have this but /cluster/resources might
  const ip = readString((row as JsonObject).ip);
  if (ip) return ip;
  // 3. IPv6 (if present, strip CIDR)
  const ip6 = readString((row as JsonObject).ip6);
  if (ip6) return ip6.split('/')[0];
  return '—';
}

export default function ProxmoxVmTable({ rows, busy, onAction, onRowClick, emptyHint, hostId, ips }: Props) {
  const navigate = useNavigate();

  const handleRowClick = (row: ProxmoxResource) => {
    if (onRowClick) {
      onRowClick(row);
      return;
    }
    if (hostId && row.node && row.vmid !== undefined) {
      navigate(`/proxmox-vms/${hostId}/${encodeURIComponent(String(row.node))}/${row.vmid}`);
    }
  };
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
              // Proxmox /vms returns: cpu_usage (fraction), mem_used/mem_total
              // (bytes), uptime_seconds. Falls back to legacy keys so old code paths
              // that read /cluster/resources still work.
              const cpu = typeof row.cpu_usage === 'number' ? row.cpu_usage
                : typeof row.cpu === 'number' ? row.cpu : NaN;
              const memUsed = typeof row.mem_used === 'number' ? row.mem_used
                : typeof row.mem === 'number' ? row.mem : 0;
              const memTotal = typeof row.mem_total === 'number' ? row.mem_total
                : typeof row.maxmem === 'number' ? row.maxmem : 0;
              const uptime = typeof row.uptime_seconds === 'number' ? row.uptime_seconds
                : typeof row.uptime === 'number' ? row.uptime : 0;
              // Type badge: prefer `kind` (qemu/lxc) over `type` (usually null)
              const typeLabel = (() => {
                const kind = readString(row.kind).toLowerCase();
                if (kind) return kind.toUpperCase();
                const t = readString(row.type).toLowerCase();
                return t ? t.toUpperCase() : 'QEMU';
              })();
              return (
                <motion.tr
                  key={`${row.node}-${row.vmid}-${idx}`}
                  initial={{ opacity: 0, y: 4 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.18, delay: idx * 0.015 }}
                  whileHover={{ backgroundColor: 'rgba(148, 163, 184, 0.06)' }}
                  onClick={() => handleRowClick(row)}
                  className={onRowClick ? 'px-vm-row-clickable' : ''}
                >
                  <td>
                    <span className={`px-status px-status-${accent}`}>{status || 'unknown'}</span>
                  </td>
                  <td className="px-mono">{row.vmid}</td>
                  <td>
                    <span className="px-type-badge">{typeLabel}</span>
                  </td>
                  <td className="px-vm-name">{readString(row.name) || '—'}</td>
                  <td>{readString(row.node) || '—'}</td>
                  <td className="px-mono">{deriveIp(row, ips)}</td>
                  <td>{formatPercent(cpu)}</td>
                  <td>
                    {memTotal ? (
                      <>
                        {formatBytes(memUsed)} / <span className="px-muted">{formatBytes(memTotal)}</span>
                      </>
                    ) : (
                      <span className="px-muted">—</span>
                    )}
                  </td>
                  <td>{formatUptime(uptime)}</td>
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
          const cpu = typeof row.cpu_usage === 'number' ? row.cpu_usage
            : typeof row.cpu === 'number' ? row.cpu : NaN;
          const memUsed = typeof row.mem_used === 'number' ? row.mem_used
            : typeof row.mem === 'number' ? row.mem : 0;
          const memTotal = typeof row.mem_total === 'number' ? row.mem_total
            : typeof row.maxmem === 'number' ? row.maxmem : 0;
          const uptime = typeof row.uptime_seconds === 'number' ? row.uptime_seconds
            : typeof row.uptime === 'number' ? row.uptime : 0;
          const typeLabel = (() => {
            const kind = readString(row.kind).toLowerCase();
            if (kind) return kind.toUpperCase();
            const t = readString(row.type).toLowerCase();
            return t ? t.toUpperCase() : 'QEMU';
          })();
          return (
            <motion.div
              key={`m-${row.node}-${row.vmid}-${idx}`}
              initial={{ opacity: 0, y: 4 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.18, delay: idx * 0.015 }}
              className={`px-vm-card px-vm-card-${accent}`}
              onClick={() => handleRowClick(row)}
            >
              <div className="px-vm-card-head">
                <span className={`px-status px-status-${accent}`}>{status || 'unknown'}</span>
                <span className="px-mono px-muted">#{row.vmid}</span>
              </div>
              <div className="px-vm-card-name">{readString(row.name) || '—'}</div>
              <div className="px-vm-card-meta">
                <span>{typeLabel}</span>
                <span>·</span>
                <span>{readString(row.node) || '—'}</span>
              </div>
              <div className="px-vm-card-stats">
                <span>IP {deriveIp(row, ips)}</span>
                <span>CPU {formatPercent(cpu)}</span>
                <span>
                  RAM {memTotal ? `${formatBytes(memUsed)}/${formatBytes(memTotal)}` : '—'}
                </span>
                <span>↑ {formatUptime(uptime)}</span>
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