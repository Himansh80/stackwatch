/**
 * Tier 14 Phase 14.2 — VM detail page.
 *
 * Route: /proxmox-vms/:hostId/:node/:vmid
 * Tab:   URL hash (#summary | #hardware | #network | #console | #snapshots | #firewall)
 *
 * Composes: header + actions + tab nav + tab content.
 * Auto-refreshes status every 5s. Network tab loads on first activation.
 */
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { motion } from 'framer-motion';
import { api } from '../../lib/api';
import ProxmoxDetailHeader from './ProxmoxDetailHeader';
import ProxmoxDetailTabs, { type DetailTab } from './ProxmoxDetailTabs';
import ProxmoxDetailSummary from './ProxmoxDetailSummary';
import ProxmoxDetailHardware, { type VMConfig } from './ProxmoxDetailHardware';
import ProxmoxDetailNetwork from './ProxmoxDetailNetwork';
import ProxmoxDetailConsole from './ProxmoxDetailConsole';
import ProxmoxDetailSnapshots from './ProxmoxDetailSnapshots';
import ProxmoxDetailFirewall from './ProxmoxDetailFirewall';
import type { ProxmoxResource } from '../../lib/proxmox';
import type { NetworkInterfacesResponse } from './lib/proxmox-vm-types';

const REFRESH_MS = 5000;
const VALID_TABS: DetailTab[] = ['summary', 'hardware', 'network', 'console', 'snapshots', 'firewall'];

function readTabFromHash(): DetailTab {
  const h = window.location.hash.replace('#', '');
  return VALID_TABS.includes(h as DetailTab) ? (h as DetailTab) : 'summary';
}

export default function ProxmoxVmDetailPage() {
  const { hostId = '', node = '', vmid = '' } = useParams();
  const navigate = useNavigate();
  const vmidNum = Number(vmid);

  const [tab, setTab] = useState<DetailTab>(readTabFromHash);
  const [vm, setVm] = useState<ProxmoxResource | null>(null);
  const [config, setConfig] = useState<VMConfig | null>(null);
  const [networkInfo, setNetworkInfo] = useState<NetworkInterfacesResponse | null>(null);
  const [networkLoading, setNetworkLoading] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState('');

  // Sync hash <-> tab
  useEffect(() => {
    window.history.replaceState(null, '', `#${tab}`);
  }, [tab]);

  const loadVM = useCallback(async () => {
    if (!hostId || !Number.isFinite(vmidNum)) return;
    setLoading(true);
    try {
      const data = await api<{ resources?: ProxmoxResource[] } | ProxmoxResource[]>(
        'GET',
        `/api/v1/proxmox/hosts/${hostId}/qemu`,
      );
      const list = Array.isArray(data) ? data : data.resources ?? [];
      const found = list.find((r) => Number(r.vmid) === vmidNum);
      setVm(found ?? null);
      setError(found ? '' : `VM ${vmid} not found on host ${hostId.slice(0, 8)}`);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to load VM.');
    } finally {
      setLoading(false);
    }
  }, [hostId, vmidNum, vmid]);

  const loadConfig = useCallback(async () => {
    if (!hostId || !node || !Number.isFinite(vmidNum)) return;
    try {
      const cfg = await api<VMConfig>(
        'GET',
        `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/qemu/${vmidNum}`,
      );
      setConfig(cfg);
    } catch {
      /* non-fatal */
    }
  }, [hostId, node, vmidNum]);

  const loadNetwork = useCallback(async () => {
    if (!hostId || !node || !Number.isFinite(vmidNum)) return;
    setNetworkLoading(true);
    try {
      const data = await api<NetworkInterfacesResponse>(
        'GET',
        `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/qemu/${vmidNum}/agent/network-get-interfaces`,
      );
      setNetworkInfo(data);
    } catch {
      setNetworkInfo({ interfaces: null, agent_running: false });
    } finally {
      setNetworkLoading(false);
    }
  }, [hostId, node, vmidNum]);

  // Initial load
  useEffect(() => {
    void loadVM();
    void loadConfig();
  }, [loadVM, loadConfig]);

  // Auto-refresh status every 5s
  useEffect(() => {
    if (!hostId) return;
    const tick = () => {
      if (document.visibilityState === 'visible') void loadVM();
    };
    const handle = window.setInterval(tick, REFRESH_MS);
    document.addEventListener('visibilitychange', tick);
    return () => {
      window.clearInterval(handle);
      document.removeEventListener('visibilitychange', tick);
    };
  }, [loadVM, hostId]);

  // Lazy-load network on first activation
  useEffect(() => {
    if (tab === 'network' && networkInfo === null) {
      void loadNetwork();
    }
  }, [tab, networkInfo, loadNetwork]);

  const onAction = useCallback(
    async (action: 'start' | 'shutdown' | 'stop' | 'reboot' | 'delete') => {
      if (!hostId || !node) return;
      setBusy(true);
      setError('');
      try {
        if (action === 'delete') {
          await api(
            'DELETE',
            `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/qemu/${vmidNum}`,
          );
          setNotice(`VM ${vmidNum} deleted.`);
          setTimeout(() => navigate('/proxmox-vms'), 800);
        } else {
          await api(
            'POST',
            `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/qemu/${vmidNum}/status/${action}`,
            { force: false },
          );
          setNotice(`${action} requested.`);
          setTimeout(() => void loadVM(), 800);
        }
      } catch (cause) {
        setError(cause instanceof Error ? cause.message : `Unable to ${action} VM.`);
      } finally {
        setBusy(false);
        setTimeout(() => setNotice(''), 4000);
      }
    },
    [hostId, node, vmidNum, loadVM, navigate],
  );

  const onMigrateClick = useCallback(() => {
    setNotice('Migrate ships in Phase 14.3.');
    setTimeout(() => setNotice(''), 3000);
  }, []);

  const tabContent = useMemo(() => {
    if (!vm) return null;
    switch (tab) {
      case 'summary':
        return <ProxmoxDetailSummary vm={vm} />;
      case 'hardware':
        return config ? (
          <ProxmoxDetailHardware config={config} />
        ) : (
          <div className="px-tab-pane">
            <p className="px-muted">Loading hardware config…</p>
          </div>
        );
      case 'network':
        return (
          <ProxmoxDetailNetwork
            interfaces={networkInfo?.interfaces ?? null}
            agentRunning={networkInfo?.agent_running ?? false}
            loading={networkLoading}
          />
        );
      case 'console':
        return <ProxmoxDetailConsole />;
      case 'snapshots':
        return <ProxmoxDetailSnapshots snapshots={[]} loading={false} />;
      case 'firewall':
        return <ProxmoxDetailFirewall rules={[]} loading={false} />;
      default:
        return null;
    }
  }, [tab, vm, config, networkInfo, networkLoading]);

  if (loading && !vm) {
    return (
      <div className="px-vm-page">
        <p className="px-muted">Loading VM…</p>
      </div>
    );
  }

  if (!vm) {
    return (
      <div className="px-vm-page">
        <div className="px-error">⚠ {error || `VM ${vmid} not found`}</div>
        <button
          type="button"
          className="px-btn px-btn-quiet"
          onClick={() => navigate('/proxmox-vms')}
        >
          ← Back to VM list
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
      <ProxmoxDetailHeader
        vm={vm}
        hostId={hostId}
        busy={busy}
        onAction={onAction}
        onMigrateClick={onMigrateClick}
      />
      <ProxmoxDetailTabs active={tab} onChange={setTab} />
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
      {tabContent}
    </motion.div>
  );
}