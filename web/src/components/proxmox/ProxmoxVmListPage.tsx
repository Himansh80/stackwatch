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
      const data = await api<{ resources?: ProxmoxResource[] } | ProxmoxResource[]>(
        'GET',
        `/api/v1/proxmox/hosts/${hostId}/qemu`,
      );
      const list = Array.isArray(data) ? data : data.resources ?? [];
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
      <div className="px-vm-page">
        <ProxmoxHostSelector selected="" onChange={setHostId} />
        <ProxmoxEmptyState kind="no-hosts" />
      </div>
    );
  }

  if (error && resources.length === 0) {
    return (
      <div className="px-vm-page">
        <ProxmoxHostSelector selected={hostId} onChange={setHostId} />
        <ProxmoxEmptyState kind="error" message={error} />
        <button type="button" className="px-btn px-btn-quiet" onClick={() => void load()}>
          Retry
        </button>
      </div>
    );
  }

  return (
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
        emptyHint={
          resources.length === 0
            ? loading
              ? 'Loading live resources…'
              : 'No VMs or containers on this host yet.'
            : undefined
        }
      />
    </motion.div>
  );
}