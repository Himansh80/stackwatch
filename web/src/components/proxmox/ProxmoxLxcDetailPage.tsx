/**
 * Tier 14 Phase 14.4 — LXC container detail page.
 * Mirrors VM detail but uses /nodes/:node/lxc/:vmid endpoints.
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
import ProxmoxDetailSnapshots from './ProxmoxDetailSnapshots';
import ProxmoxDetailFirewall from './ProxmoxDetailFirewall';
import ProxmoxDetailMigrateDialog from './ProxmoxDetailMigrateDialog';
import type { ProxmoxResource } from '../../lib/proxmox';
import { readString } from '../../lib/proxmox';
import type { NetworkInterfacesResponse } from './lib/proxmox-vm-types';

const REFRESH_MS = 5000;
const VALID_TABS: DetailTab[] = ['summary', 'resources', 'network', 'console', 'snapshots', 'firewall'];

function readTabFromHash(): DetailTab {
  const h = window.location.hash.replace('#', '');
  return VALID_TABS.includes(h as DetailTab) ? (h as DetailTab) : 'summary';
}

export default function ProxmoxLxcDetailPage() {
  const { hostId = '', node = '', vmid = '' } = useParams();
  const navigate = useNavigate();
  const vmidNum = Number(vmid);

  const [tab, setTab] = useState<DetailTab>(readTabFromHash);
  const [ct, setCt] = useState<ProxmoxResource | null>(null);
  const [config, setConfig] = useState<VMConfig | null>(null);
  const [networkInfo, setNetworkInfo] = useState<NetworkInterfacesResponse | null>(null);
  const [networkLoading, setNetworkLoading] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState('');
  const [migrateOpen, setMigrateOpen] = useState(false);

  useEffect(() => {
    window.history.replaceState(null, '', `#${tab}`);
  }, [tab]);

  const loadCT = useCallback(async () => {
    if (!hostId || !Number.isFinite(vmidNum)) return;
    setLoading(true);
    try {
      const data = await api<{ resources?: ProxmoxResource[] } | ProxmoxResource[]>(
        'GET',
        `/api/v1/proxmox/hosts/${hostId}/qemu`,
      );
      const list = Array.isArray(data) ? data : data.resources ?? [];
      const found = list.find(
        (r) => Number(r.vmid) === vmidNum && readString(r.type).toLowerCase().includes('lxc'),
      );
      setCt(found ?? null);
      setError(found ? '' : `CT ${vmid} not found on host ${hostId.slice(0, 8)}`);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to load container.');
    } finally {
      setLoading(false);
    }
  }, [hostId, vmidNum, vmid]);

  const loadConfig = useCallback(async () => {
    if (!hostId || !node || !Number.isFinite(vmidNum)) return;
    try {
      const cfg = await api<VMConfig>(
        'GET',
        `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/lxc/${vmidNum}`,
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

  useEffect(() => {
    void loadCT();
    void loadConfig();
  }, [loadCT, loadConfig]);

  useEffect(() => {
    if (!hostId) return;
    const tick = () => {
      if (document.visibilityState === 'visible') void loadCT();
    };
    const handle = window.setInterval(tick, REFRESH_MS);
    document.addEventListener('visibilitychange', tick);
    return () => {
      window.clearInterval(handle);
      document.removeEventListener('visibilitychange', tick);
    };
  }, [loadCT, hostId]);

  useEffect(() => {
    if (tab === 'network' && networkInfo === null) void loadNetwork();
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
            `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/lxc/${vmidNum}`,
          );
          setNotice(`CT ${vmidNum} deleted.`);
          setTimeout(() => navigate('/proxmox-lxc'), 800);
        } else {
          await api(
            'POST',
            `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/lxc/${vmidNum}/status/${action}`,
            { force: false },
          );
          setNotice(`${action} requested.`);
          setTimeout(() => void loadCT(), 800);
        }
      } catch (cause) {
        setError(cause instanceof Error ? cause.message : `Unable to ${action} container.`);
      } finally {
        setBusy(false);
        setTimeout(() => setNotice(''), 4000);
      }
    },
    [hostId, node, vmidNum, loadCT, navigate],
  );

  const onMigrate = useCallback(
    async (target: string, online: boolean, withLocalStorage: boolean) => {
      if (!hostId || !node) return;
      setBusy(true);
      setError('');
      try {
        await api(
          'POST',
          `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/lxc/${vmidNum}/migrate`,
          { target, online: online ? 1 : 0, with_local_disks: withLocalStorage ? 1 : 0 },
        );
        setNotice(`Migrate to ${target} started.`);
        setTimeout(() => setNotice(''), 4000);
      } catch (cause) {
        throw cause instanceof Error ? cause : new Error('Migrate failed.');
      } finally {
        setBusy(false);
      }
    },
    [hostId, node, vmidNum],
  );

  const tabContent = useMemo(() => {
    if (!ct) return null;
    switch (tab) {
      case 'summary':
        return <ProxmoxDetailSummary vm={ct} />;
      case 'resources':
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
        return (
          <div className="px-tab-pane">
            <div className="px-info-card">
              <h3>LXC serial console — ships in v2.7</h3>
              <p>xterm.js terminal with serial-over-pty will land here. LXC containers don&apos;t have graphical consoles.</p>
            </div>
          </div>
        );
      case 'snapshots':
        return <ProxmoxDetailSnapshots hostId={hostId} node={node} vmid={vmidNum} />;
      case 'firewall':
        return <ProxmoxDetailFirewall hostId={hostId} node={node} vmid={vmidNum} />;
      default:
        return null;
    }
  }, [tab, ct, config, networkInfo, networkLoading, hostId, node, vmidNum]);

  if (loading && !ct) {
    return <div className="px-vm-page"><p className="px-muted">Loading CT…</p></div>;
  }

  if (!ct) {
    return (
      <div className="px-vm-page">
        <div className="px-error">⚠ {error || `CT ${vmid} not found`}</div>
        <button type="button" className="px-btn px-btn-quiet" onClick={() => navigate('/proxmox-lxc')}>
          ← Back to LXC list
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
        vm={ct}
        hostId={hostId}
        busy={busy}
        onAction={onAction}
        onMigrateClick={() => setMigrateOpen(true)}
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

      {ct && (
        <ProxmoxDetailMigrateDialog
          open={migrateOpen}
          hostId={hostId}
          vmid={vmidNum}
          currentNode={node}
          running={readString(ct.status).toLowerCase() === 'running'}
          onClose={() => setMigrateOpen(false)}
          onMigrate={onMigrate}
        />
      )}
    </motion.div>
  );
}