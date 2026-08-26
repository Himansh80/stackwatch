/**
 * Tier 14 Phase 14.1 — VM list page (Datadog-style dark theme).
 * Composes: host selector, KPI strip, filter bar, VM table, empty/error states.
 * Auto-refreshes every 5s (pauses when tab hidden).
 */
import { useCallback, useEffect, useMemo, useState } from 'react';
import { motion } from 'framer-motion';
import ProxmoxHostSelector from './ProxmoxHostSelector';
import ProxmoxKpiStrip from './ProxmoxKpiStrip';
import ProxmoxFilterBar, { type StatusFilter } from './ProxmoxFilterBar';
import ProxmoxVmTable from './ProxmoxVmTable';
import ProxmoxEmptyState from './ProxmoxEmptyState';
import { api } from '../../lib/api';
import type { ProxmoxResource } from '../../lib/proxmox';
import { readString } from '../../lib/proxmox';
import ProxmoxShell from './ProxmoxShell';

const REFRESH_MS = 5000;

interface Props {
  /** Initial host id (auto-select first available if omitted) */
  initialHostId?: string;
}

export default function ProxmoxVmListPage({ initialHostId }: Props) {
  const [hostId, setHostId] = useState(initialHostId ?? '');
  const [resources, setResources] = useState<ProxmoxResource[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState('');
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState<StatusFilter>('all');

  const load = useCallback(async () => {
      if (!hostId) return;
      setLoading(true);
      try {
        // Proxmox's /vms returns {total, vms:[...]}; tolerates /cluster/resources too
        // We call /vms (which Proxmox itself uses) and the field is `vms`.
        const data = await api<{ vms?: ProxmoxResource[] } | ProxmoxResource[]>(
          'GET',
          `/api/v1/proxmox/hosts/${hostId}/vms`,
        );
        const raw = Array.isArray(data) ? data : data.vms ?? [];
        // /vms also returns the cluster's node/storage/network rows + the
        // storage pool row (kind='pool'). Filter to just qemu + lxc
        // — that's what the dashboard cares about.
        const list = raw.filter((r) => {
          const kind = readString(r.kind).toLowerCase();
          return kind === 'qemu' || kind === 'lxc';
        });
        setResources(list);
        setError('');
      } catch (cause) {
        setError(cause instanceof Error ? cause.message : 'Unable to load VMs.');
        setResources([]);
      } finally {
        setLoading(false);
      }
    }, [hostId]);

  // Initial + host-id change
  useEffect(() => {
    void load();
  }, [load]);

  // Auto-refresh (paused when tab hidden)
  useEffect(() => {
    let alive = true;
    const tick = () => {
      if (document.visibilityState === 'visible' && alive) void load();
    };
    const handle = window.setInterval(tick, REFRESH_MS);
    document.addEventListener('visibilitychange', tick);
    return () => {
      alive = false;
      window.clearInterval(handle);
      document.removeEventListener('visibilitychange', tick);
    };
  }, [load]);

  // If no host was pre-selected, pick the first available one.
  useEffect(() => {
    if (hostId) return;
    let alive = true;
    api<{ hosts?: Array<{ id: string; name: string }> }>('GET', '/api/v1/proxmox/hosts')
      .then((data) => {
        if (!alive) return;
        const hosts = data.hosts ?? [];
        if (hosts.length > 0 && hosts[0]) setHostId(hosts[0].id);
      })
      .catch(() => {/* ignore */});
    return () => { alive = false; };
  }, [hostId]);

  // After loading the VM list, fetch IPs from the QEMU guest agent
  // for each running VM. /vms doesn't carry IPs; only the agent does.
  // IPs land in `ips` keyed by vmid; table reads from there.
  const [ips, setIps] = useState<Record<number, string>>({});
  const fetchIps = useCallback(async (rows: ProxmoxResource[]) => {
    if (!hostId) return;
    const candidates = rows.filter((r) =>
      readString(r.kind).toLowerCase() === 'qemu' &&
      readString(r.status).toLowerCase() === 'running',
    );
    const updates: Record<number, string> = {};
    await Promise.all(
      candidates.map(async (vm) => {
        if (!vm.node || !vm.vmid) return;
        try {
          const data = await api<{ result?: Array<{ ip?: string; name: string }> }>(
            'GET',
            `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(String(vm.node))}/qemu/${vm.vmid}/agent/network-get-interfaces`,
          );
          const ifaces = data.result ?? [];
          for (const iface of ifaces) {
            const ip = String(iface.ip ?? '');
            if (ip && !ip.startsWith('127.') && !ip.startsWith('::1')) {
              updates[Number(vm.vmid)] = ip;
              return;
            }
          }
        } catch {
          /* agent not installed; skip */
        }
      }),
    );
    if (Object.keys(updates).length > 0) {
      setIps((prev) => ({ ...prev, ...updates }));
    }
  }, [hostId]);

  // Re-fetch IPs after each successful load (keeps them fresh)
  useEffect(() => {
    if (resources.length > 0) void fetchIps(resources);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [resources, hostId]);

  // Filtered rows
  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    return resources.filter((row) => {
      // Status filter
      if (status !== 'all') {
        const s = readString(row.status).toLowerCase();
        if (s !== status) return false;
      }
      // Search filter
      if (q) {
        const hay = `${readString(row.name)} ${row.vmid} ${readString((row as unknown as Record<string, unknown>).ip)}`.toLowerCase();
        if (!hay.includes(q)) return false;
      }
      return true;
    });
  }, [resources, search, status]);

  const counts = useMemo(() => {
    const result: Record<StatusFilter, number> = {
      all: resources.length,
      running: 0,
      stopped: 0,
      paused: 0,
    };
    for (const row of resources) {
      const s = readString(row.status).toLowerCase();
      if (s in result) result[s as StatusFilter] += 1;
    }
    return result;
  }, [resources]);

  const onAction = useCallback(
    async (resource: ProxmoxResource, action: 'start' | 'shutdown' | 'stop' | 'reboot') => {
      if (!hostId) return;
      setBusy(true);
      setError('');
      try {
        const kind = readString(resource.type).toLowerCase().includes('lxc') ? 'lxc' : 'qemu';
        await api(
          'POST',
          `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(readString(resource.node))}/${kind}/${resource.vmid}/status/${action}`,
          { force: false },
        );
        setNotice(`${action} requested for ${readString(resource.name) || `VM ${resource.vmid}`}.`);
        await load();
      } catch (cause) {
        setError(cause instanceof Error ? cause.message : `Unable to ${action} resource.`);
      } finally {
        setBusy(false);
        setTimeout(() => setNotice(''), 4000);
      }
    },
    [hostId, load],
  );

  // Show empty states
    if (!hostId && !loading) {
      return (
        <ProxmoxShell title="VMs" subtitle="Pick a Proxmox host to see its VMs and LXC containers">
          <ProxmoxHostSelector selected="" onChange={setHostId} />
          <ProxmoxEmptyState kind="no-hosts" />
        </ProxmoxShell>
      );
    }

    if (error && resources.length === 0) {
      return (
        <ProxmoxShell title="VMs" subtitle="Pick a Proxmox host to see its VMs and LXC containers">
          <ProxmoxHostSelector selected={hostId} onChange={setHostId} />
          <ProxmoxEmptyState kind="error" message={error} />
          <button type="button" className="px-btn px-btn-quiet" onClick={() => void load()}>
            Retry
          </button>
        </ProxmoxShell>
      );
    }

    return (
      <ProxmoxShell
        title="VMs"
        subtitle="Browse all VMs and LXC containers on this Proxmox host"
        actions={
          <button
            type="button"
            className="px-btn px-btn-primary"
            onClick={() => (window.location.href = '/proxmox-vms/new')}
          >
            + Create VM
          </button>
        }
      >
        <motion.div
          className="px-vm-page"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.2 }}
        >
        <ProxmoxHostSelector selected={hostId} onChange={setHostId} />

        <ProxmoxKpiStrip resources={resources} loading={loading} />

        <ProxmoxFilterBar
          search={search}
          onSearchChange={setSearch}
          status={status}
          onStatusChange={setStatus}
          counts={counts}
        />

        {notice && (
          <motion.div
            className="px-notice"
            initial={{ opacity: 0, y: -8 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0 }}
          >
            ✓ {notice}
          </motion.div>
        )}

        {error && <div className="px-error">⚠ {error}</div>}

        <ProxmoxVmTable
          rows={filtered}
          busy={busy}
          onAction={onAction}
          hostId={hostId}
          ips={ips}
          emptyHint={
            resources.length === 0
              ? loading
                ? 'Loading live resources…'
                : 'No VMs or containers on this host yet.'
              : undefined
          }
        />
        </motion.div>
      </ProxmoxShell>
    );
  }