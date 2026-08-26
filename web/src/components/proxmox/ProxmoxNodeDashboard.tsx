/**
 * Tier 14 Phase 14.7 — Node dashboard.
 * Sections: Disks, ZFS pools, Network, Services. Auto-refresh 30s.
 */
import { useCallback, useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { motion } from 'framer-motion';
import { api } from '../../lib/api';
import ProxmoxNodeKpiStrip from './ProxmoxNodeKpiStrip';
import { formatBytes, readString } from '../../lib/proxmox';

interface DiskRow {
  devpath?: string;
  vendor?: string;
  model?: string;
  size?: number;
  type?: string;
  used?: string;
  wwn?: string;
}

interface ZfsRow {
  name?: string;
  size?: number;
  alloc?: number;
  free?: number;
  frag?: string;
}

interface NetRow {
  iface?: string;
  mac?: string;
  cidr?: string;
  mtu?: string;
  speed?: string;
  type?: string;
}

interface ServiceRow {
  name?: string;
  state?: string;
  autostart?: boolean;
}

interface NodeStatus {
  cpu?: number;
  maxcpu?: number;
  mem?: number;
  maxmem?: number;
  disk?: number;
  maxdisk?: number;
  uptime?: number;
  loadavg?: [string, string, string];
  status?: string;
}

const REFRESH_MS = 30000;

export default function ProxmoxNodeDashboard() {
  const { hostId = '', node = '' } = useParams();
  const [status, setStatus] = useState<NodeStatus | null>(null);
  const [disks, setDisks] = useState<DiskRow[]>([]);
  const [zfs, setZfs] = useState<ZfsRow[]>([]);
  const [network, setNetwork] = useState<NetRow[]>([]);
  const [services, setServices] = useState<ServiceRow[]>([]);
  const [error, setError] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    if (!hostId || !node) return;
    setLoading(true);
    setError({});
    const result: { [k: string]: unknown } = {};
    try {
      const statusData = await api<NodeStatus>('GET', `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/status`);
      setStatus(statusData);
    } catch (cause) {
      setError((e) => ({ ...e, status: cause instanceof Error ? cause.message : 'Status load failed' }));
    }
    try {
      const diskData = await api<{ disks?: DiskRow[] } | DiskRow[]>(
        'GET',
        `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/disks/list`,
      );
      setDisks(Array.isArray(diskData) ? diskData : diskData.disks ?? []);
    } catch (cause) {
      setError((e) => ({ ...e, disks: cause instanceof Error ? cause.message : 'Disks load failed' }));
    }
    try {
      const zfsData = await api<ZfsRow[]>(
        'GET',
        `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/disks/zfs`,
      );
      setZfs(Array.isArray(zfsData) ? zfsData : []);
    } catch {
      setZfs([]); // No ZFS is OK
    }
    try {
      const netData = await api<{ network?: NetRow[] } | NetRow[]>(
        'GET',
        `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/network`,
      );
      setNetwork(Array.isArray(netData) ? netData : netData.network ?? []);
    } catch (cause) {
      setError((e) => ({ ...e, network: cause instanceof Error ? cause.message : 'Network load failed' }));
    }
    try {
      const svcData = await api<{ services?: ServiceRow[] } | ServiceRow[]>(
        'GET',
        `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/services`,
      );
      setServices(Array.isArray(svcData) ? svcData : svcData.services ?? []);
    } catch (cause) {
      setError((e) => ({ ...e, services: cause instanceof Error ? cause.message : 'Services load failed' }));
    }
    setLoading(false);
    void result;
  }, [hostId, node]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!hostId) return;
    const tick = () => { if (document.visibilityState === 'visible') void load(); };
    const handle = window.setInterval(tick, REFRESH_MS);
    document.addEventListener('visibilitychange', tick);
    return () => {
      window.clearInterval(handle);
      document.removeEventListener('visibilitychange', tick);
    };
  }, [load, hostId]);

  return (
    <motion.div className="px-vm-page" initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
      <div className="px-detail-breadcrumb">
        <Link to="/proxmox">Proxmox</Link>
        <span className="px-detail-sep">›</span>
        <span>{node}</span>
      </div>
      <h1 className="px-detail-name">{node}</h1>
      <div className="px-detail-meta">
        <span><strong>Status</strong> {readString(status?.status) || '—'}</span>
        <span><strong>Host</strong> {hostId.slice(0, 8)}</span>
      </div>

      {error.status && <div className="px-error">⚠ {error.status}</div>}

      <ProxmoxNodeKpiStrip status={status ?? {}} loading={loading} />

      <div className="px-node-grid">
        <Section title="Disks" error={error.disks}>
          {disks.length === 0 ? (
            <p className="px-muted">No disks reported.</p>
          ) : (
            <table className="px-network-table">
              <thead>
                <tr><th>Device</th><th>Model</th><th>Size</th><th>Type</th></tr>
              </thead>
              <tbody>
                {disks.map((d, i) => (
                  <tr key={d.devpath || i}>
                    <td className="px-mono">{d.devpath || '—'}</td>
                    <td>{[d.vendor, d.model].filter(Boolean).join(' ') || '—'}</td>
                    <td className="px-mono">{formatBytes(d.size ?? 0)}</td>
                    <td>{d.type || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </Section>

        <Section title="ZFS pools">
          {zfs.length === 0 ? (
            <p className="px-muted">No ZFS pools on this host.</p>
          ) : (
            <table className="px-network-table">
              <thead>
                <tr><th>Name</th><th>Size</th><th>Used</th><th>Free</th><th>Frag%</th></tr>
              </thead>
              <tbody>
                {zfs.map((z, i) => (
                  <tr key={z.name || i}>
                    <td className="px-mono">{z.name || '—'}</td>
                    <td className="px-mono">{formatBytes(z.size ?? 0)}</td>
                    <td className="px-mono">{formatBytes(z.alloc ?? 0)}</td>
                    <td className="px-mono">{formatBytes(z.free ?? 0)}</td>
                    <td className="px-mono">{z.frag || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </Section>

        <Section title="Network interfaces" error={error.network}>
          {network.length === 0 ? (
            <p className="px-muted">No network interfaces reported.</p>
          ) : (
            <table className="px-network-table">
              <thead>
                <tr><th>Iface</th><th>MAC</th><th>CIDR</th><th>MTU</th><th>Speed</th></tr>
              </thead>
              <tbody>
                {network.map((n, i) => (
                  <tr key={n.iface || i}>
                    <td className="px-mono">{n.iface || '—'}</td>
                    <td className="px-mono">{n.mac || '—'}</td>
                    <td className="px-mono">{n.cidr || '—'}</td>
                    <td className="px-mono">{n.mtu || '—'}</td>
                    <td className="px-mono">{n.speed || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </Section>

        <Section title="Services" error={error.services}>
          {services.length === 0 ? (
            <p className="px-muted">No services reported.</p>
          ) : (
            <table className="px-network-table">
              <thead>
                <tr><th>Name</th><th>State</th><th>Autostart</th></tr>
              </thead>
              <tbody>
                {services.map((s, i) => (
                  <tr key={s.name || i}>
                    <td className="px-mono">{s.name || '—'}</td>
                    <td>
                      <span className={`px-action-badge px-action-${s.state === 'running' ? 'accept' : 'drop'}`}>
                        {s.state || 'unknown'}
                      </span>
                    </td>
                    <td>{s.autostart ? 'yes' : 'no'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </Section>
      </div>
    </motion.div>
  );
}

function Section({ title, children, error }: { title: string; children: React.ReactNode; error?: string }) {
  return (
    <section className="px-summary-card">
      <h3>{title}</h3>
      {error ? <div className="px-confirm-warn">⚠ {error}</div> : children}
    </section>
  );
}