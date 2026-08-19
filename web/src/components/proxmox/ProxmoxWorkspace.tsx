import { FormEvent, ReactNode, useEffect, useMemo, useState } from 'react';
import { api } from '../../lib/api';
import {
  ProxmoxHost,
  ProxmoxNode,
  ProxmoxResource,
  JsonObject,
  displayValue,
  formatBytes,
  formatDate,
  formatPercent,
  hostPath,
  isObject,
  listFrom,
  objectFrom,
  pxGet,
  pxPost,
} from '../../lib/proxmox';

type Section = 'overview' | 'compute' | 'storage' | 'network' | 'access' | 'operations' | 'security' | 'templates' | 'cluster' | 'monitoring';
type Row = JsonObject;
type TableColumn = { key: string; label: string; render?: (row: Row) => ReactNode };

const sections: Array<{ id: Section; label: string; icon: string }> = [
  { id: 'overview', label: 'Overview', icon: '⌂' },
  { id: 'compute', label: 'Compute', icon: '▣' },
  { id: 'storage', label: 'Storage', icon: '▤' },
  { id: 'network', label: 'Network & Firewall', icon: '⌁' },
  { id: 'access', label: 'Access & Tokens', icon: '◇' },
  { id: 'operations', label: 'Tasks, Pools & Backup', icon: '◴' },
  { id: 'security', label: 'TLS & ACME', icon: '◈' },
  { id: 'templates', label: 'Templates & Cloud-init', icon: '◇' },
  { id: 'cluster', label: 'HA & Cluster', icon: '◎' },
  { id: 'monitoring', label: 'Monitoring', icon: '⌁' },
];

const emptyHost = { name: '', base_url: 'https://192.168.0.107:8006', api_token: '', verify_tls: false };
const emptyVM = { vmid: '', name: '', memory_mb: '2048', cores: '2', disk_gb: '32', storage: 'local-lvm', bridge: 'vmbr0', iso: '', boot: 'order=scsi0' };

function readString(value: unknown): string {
  return value === undefined || value === null ? '' : String(value);
}

function statusClass(status: unknown): string {
  const value = readString(status).toLowerCase();
  if (['online', 'running', 'active', 'ok', 'up'].includes(value)) return 'status-good';
  if (['offline', 'stopped', 'error', 'failed', 'down'].includes(value)) return 'status-bad';
  return 'status-neutral';
}

function valueFor(row: Row, key: string): string {
  const value = row[key];
  if (key.includes('mem') || key.includes('disk') || key.includes('size') || key.includes('bytes')) {
    return typeof value === 'number' ? formatBytes(value) : displayValue(value);
  }
  if (key === 'cpu' && typeof value === 'number') return formatPercent(value);
  if (key.includes('time') || key.endsWith('_at') || key === 'starttime' || key === 'updatetime') {
    return formatDate(value);
  }
  return displayValue(value);
}

function Button({ children, onClick, tone = 'default', disabled = false, type = 'button' }: {
  children: ReactNode;
  onClick?: () => void;
  tone?: 'default' | 'primary' | 'danger' | 'quiet';
  disabled?: boolean;
  type?: 'button' | 'submit';
}) {
  return <button className={`sw-button sw-button-${tone}`} onClick={onClick} disabled={disabled} type={type}>{children}</button>;
}

function Panel({ title, eyebrow, actions, children, className = '' }: {
  title: string;
  eyebrow?: string;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return <section className={`sw-panel ${className}`}>
    <div className="sw-panel-head">
      <div><span className="sw-eyebrow">{eyebrow}</span><h2>{title}</h2></div>
      {actions && <div className="sw-panel-actions">{actions}</div>}
    </div>
    {children}
  </section>;
}

function Stat({ label, value, hint, accent = 'cyan' }: { label: string; value: ReactNode; hint?: string; accent?: string }) {
  return <div className={`sw-stat sw-stat-${accent}`}><span>{label}</span><strong>{value}</strong>{hint && <small>{hint}</small>}</div>;
}

function DataTable({ rows, columns, empty = 'No records returned from this host.', onRowClick }: {
  rows: Row[];
  columns?: TableColumn[];
  empty?: string;
  onRowClick?: (row: Row) => void;
}) {
  const detected: TableColumn[] = useMemo(() => {
    if (columns) return columns;
    const keys = new Set<string>();
    rows.slice(0, 20).forEach((row) => Object.keys(row).slice(0, 10).forEach((key) => keys.add(key)));
    return Array.from(keys).slice(0, 8).map((key) => ({ key, label: key.replaceAll('_', ' ') }));
  }, [columns, rows]);
  if (!rows.length) return <div className="sw-empty">{empty}</div>;
  return <div className="sw-table-wrap"><table className="sw-table"><thead><tr>{detected.map((column) => <th key={column.key}>{column.label}</th>)}</tr></thead><tbody>
    {rows.map((row, index) => <tr key={readString(row.id) || `${index}`} onClick={() => onRowClick?.(row)} className={onRowClick ? 'sw-row-clickable' : ''}>
      {detected.map((column) => <td key={column.key}>{column.render ? column.render(row) : valueFor(row, column.key)}</td>)}
    </tr>)}
  </tbody></table></div>;
}

function JsonBlock({ value }: { value: unknown }) {
  return <pre className="sw-json">{JSON.stringify(value, null, 2)}</pre>;
}

function FormField({ label, value, onChange, type = 'text', placeholder, required = false, min, step }: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
  placeholder?: string;
  required?: boolean;
  min?: string;
  step?: string;
}) {
  return <label className="sw-field"><span>{label}</span><input value={value} type={type} placeholder={placeholder} required={required} min={min} step={step} onChange={(event) => onChange(event.target.value)} /></label>;
}

function Unsupported({ value }: { value: unknown }) {
  if (!isObject(value) || value.supported !== false) return null;
  return <div className="sw-unsupported">This capability is not supported by the connected Proxmox version.</div>;
}

export default function ProxmoxWorkspace() {
  const [section, setSection] = useState<Section>('overview');
  const [hosts, setHosts] = useState<ProxmoxHost[]>([]);
  const [hostId, setHostId] = useState('');
  const [nodes, setNodes] = useState<ProxmoxNode[]>([]);
  const [resources, setResources] = useState<ProxmoxResource[]>([]);
  const [hostState, setHostState] = useState<Row>({});
  const [storage, setStorage] = useState<Row[]>([]);
  const [storageContent, setStorageContent] = useState<Row[]>([]);
  const [network, setNetwork] = useState<Row[]>([]);
  const [firewall, setFirewall] = useState<Row[]>([]);
  const [ipsets, setIPSets] = useState<Row[]>([]);
  const [disks, setDisks] = useState<Row[]>([]);
  const [zfs, setZFS] = useState<Row[]>([]);
  const [users, setUsers] = useState<Row[]>([]);
  const [tasks, setTasks] = useState<Row[]>([]);
  const [pools, setPools] = useState<Row[]>([]);
  const [backups, setBackups] = useState<Row[]>([]);
  const [certificates, setCertificates] = useState<Row[]>([]);
  const [acme, setACME] = useState<Record<string, unknown>>({});
  const [templates, setTemplates] = useState<Row[]>([]);
  const [haStatus, setHAStatus] = useState<Row>({});
  const [haResources, setHAResources] = useState<Row[]>([]);
  const [monitoring, setMonitoring] = useState<Row[]>([]);
  const [selectedNode, setSelectedNode] = useState('');
  const [selectedStorage, setSelectedStorage] = useState('');
  const [selectedUser, setSelectedUser] = useState<Row | null>(null);
  const [showHostForm, setShowHostForm] = useState(false);
  const [showCreate, setShowCreate] = useState<'vm' | 'lxc' | null>(null);
  const [hostForm, setHostForm] = useState(emptyHost);
  const [vmForm, setVMForm] = useState(emptyVM);
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');

  const host = hosts.find((item) => item.id === hostId);
  const selectedNodeName = selectedNode || readString(nodes[0]?.node);

  async function loadHosts() {
    try {
      const response = await api<{ hosts?: ProxmoxHost[] }>('GET', '/api/v1/proxmox/hosts');
      const next = response.hosts || [];
      setHosts(next);
      if (!hostId && next[0]) setHostId(next[0].id);
      if (hostId && !next.some((item) => item.id === hostId)) setHostId(next[0]?.id || '');
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to load Proxmox hosts.');
    }
  }

  async function read<T>(request: Promise<T>, fallback: T): Promise<T> {
    try { return await request; } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'The Proxmox request failed.');
      return fallback;
    }
  }

  async function loadWorkspace() {
    if (!hostId) return;
    setLoading(true);
    setError('');
    try {
      const nodePayload = await read(pxGet(hostId, '/nodes'), {});
      const nextNodes = listFrom(nodePayload, 'nodes');
      setNodes(nextNodes as ProxmoxNode[]);
      if (!selectedNode && nextNodes[0]) setSelectedNode(readString(nextNodes[0].node));

      if (section === 'overview') {
        const [resourcePayload, statusPayload, infoPayload, testPayload] = await Promise.all([
          read(pxGet(hostId, '/cluster/resources'), {}),
          read(pxGet(hostId, '/cluster/status'), {}),
          read(pxGet(hostId, '/cluster/info'), {}),
          read(pxGet(hostId, '/test'), {}),
        ]);
        setResources(listFrom(resourcePayload, 'resources', 'vms') as ProxmoxResource[]);
        setHostState({ status: host?.status, ...objectFrom(statusPayload), cluster: objectFrom(infoPayload), connectivity: objectFrom(testPayload) });
      }
      if (section === 'compute') {
        const resourcePayload = await read(pxGet(hostId, '/vms'), {});
        setResources(listFrom(resourcePayload, 'vms', 'resources') as ProxmoxResource[]);
      }
      if (section === 'storage') {
        const [storagePayload, diskPayload, zfsPayload] = await Promise.all([
          read(pxGet(hostId, '/storage'), {}),
          selectedNodeName ? read(pxGet(hostId, `/nodes/${encodeURIComponent(selectedNodeName)}/disks/list`), {}) : Promise.resolve({}),
          selectedNodeName ? read(pxGet(hostId, `/nodes/${encodeURIComponent(selectedNodeName)}/disks/zfs`), {}) : Promise.resolve({}),
        ]);
        setStorage(listFrom(storagePayload, 'storage', 'storages'));
        setDisks(listFrom(diskPayload, 'disks'));
        setZFS(listFrom(zfsPayload, 'pools', 'zfs'));
      }
      if (section === 'network') {
        const [networkPayload, firewallPayload, ipsetPayload] = await Promise.all([
          selectedNodeName ? read(pxGet(hostId, `/nodes/${encodeURIComponent(selectedNodeName)}/network`), {}) : Promise.resolve({}),
          selectedNodeName ? read(pxGet(hostId, `/nodes/${encodeURIComponent(selectedNodeName)}/firewall/rules`), {}) : Promise.resolve({}),
          read(pxGet(hostId, '/firewall/ipsets'), {}),
        ]);
        setNetwork(listFrom(networkPayload, 'network', 'interfaces'));
        setFirewall(listFrom(firewallPayload, 'rules', 'firewall_rules'));
        setIPSets(listFrom(ipsetPayload, 'ipsets'));
      }
      if (section === 'access') {
        const payload = await read(pxGet(hostId, '/access/users'), {});
        setUsers(listFrom(payload, 'users'));
      }
      if (section === 'operations') {
        const [taskPayload, poolPayload, backupPayload] = await Promise.all([
          read(pxGet(hostId, '/cluster/tasks'), {}),
          read(pxGet(hostId, '/pools'), {}),
          read(pxGet(hostId, '/cluster/backup'), {}),
        ]);
        setTasks(listFrom(taskPayload, 'tasks'));
        setPools(listFrom(poolPayload, 'pools'));
        setBackups(listFrom(backupPayload, 'backup', 'backups', 'jobs'));
      }
      if (section === 'security') {
        const [certificatePayload, accountPayload, pluginPayload, challengePayload, directoryPayload, infoPayload] = await Promise.all([
          selectedNodeName ? read(pxGet(hostId, `/nodes/${encodeURIComponent(selectedNodeName)}/certificates`), {}) : Promise.resolve({}),
          read(pxGet(hostId, '/cluster/acme/account'), {}),
          read(pxGet(hostId, '/cluster/acme/plugins'), {}),
          read(pxGet(hostId, '/cluster/acme/challenge-schema'), {}),
          read(pxGet(hostId, '/cluster/acme/directories'), {}),
          read(pxGet(hostId, '/cluster/acme/info'), {}),
        ]);
        setCertificates(listFrom(certificatePayload, 'certificates'));
        setACME({ accounts: accountPayload, plugins: pluginPayload, challenges: challengePayload, directories: directoryPayload, info: infoPayload });
      }
      if (section === 'templates' && selectedNodeName) {
        const storageName = selectedStorage || readString(storage[0]?.storage || storage[0]?.name);
        if (storageName) {
          const payload = await read(pxGet(hostId, `/nodes/${encodeURIComponent(selectedNodeName)}/templates?storage=${encodeURIComponent(storageName)}`), {});
          setTemplates(listFrom(payload, 'templates'));
        }
      }
      if (section === 'cluster') {
        const [statusPayload, resourcePayload] = await Promise.all([
          read(pxGet(hostId, '/cluster/ha/status'), {}),
          read(pxGet(hostId, '/cluster/ha/resources'), {}),
        ]);
        setHAStatus(objectFrom(statusPayload));
        setHAResources(listFrom(resourcePayload, 'resources'));
      }
      if (section === 'monitoring' && selectedNodeName) {
        const payload = await read(pxGet(hostId, `/nodes/${encodeURIComponent(selectedNodeName)}/monitoring?timeframe=hour&cf=AVERAGE`), {});
        setMonitoring(listFrom(payload, 'points'));
      }
    } finally {
      setLoading(false);
    }
  }

  // These loaders are event-style functions; host/section are the deliberate triggers.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { void loadHosts(); }, []);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { void loadWorkspace(); }, [hostId, section, selectedNodeName]);

  async function registerHost(event: FormEvent) {
    event.preventDefault();
    setBusy(true); setError(''); setNotice('');
    try {
      const created = await api<ProxmoxHost>('POST', '/api/v1/proxmox/hosts', hostForm);
      setNotice(`Connected to ${created.name}.`);
      setHostForm(emptyHost); setShowHostForm(false);
      await loadHosts();
      if (created.id) setHostId(created.id);
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Host registration failed.'); }
    finally { setBusy(false); }
  }

  async function testHost() {
    if (!hostId) return;
    setBusy(true); setError('');
    try { const result = await api<Row>('GET', hostPath(hostId, '/test')); setNotice(result.ok === false ? `Connection failed: ${readString(result.error)}` : 'Proxmox connection is healthy.'); await loadHosts(); }
    catch (cause) { setError(cause instanceof Error ? cause.message : 'Host test failed.'); }
    finally { setBusy(false); }
  }

  async function deleteHost() {
    if (!hostId || !window.confirm('Delete this registered Proxmox host from StackWatch?')) return;
    setBusy(true);
    try { await api('DELETE', hostPath(hostId)); setNotice('Host removed.'); setHostId(''); await loadHosts(); }
    catch (cause) { setError(cause instanceof Error ? cause.message : 'Host deletion failed.'); }
    finally { setBusy(false); }
  }

  async function lifecycle(resource: ProxmoxResource, action: string) {
    if (!hostId || !resource.node || resource.vmid === undefined) return;
    setBusy(true); setError('');
    try {
      const kind = readString(resource.type).toLowerCase().includes('lxc') ? 'lxc' : 'qemu';
      await pxPost(hostId, `/nodes/${encodeURIComponent(readString(resource.node))}/${kind}/${resource.vmid}/status/${action}`, { force: false });
      setNotice(`${action} requested for ${resource.name || `VM ${resource.vmid}`}.`);
      await loadWorkspace();
    } catch (cause) { setError(cause instanceof Error ? cause.message : `Unable to ${action} resource.`); }
    finally { setBusy(false); }
  }

  async function createResource(event: FormEvent) {
    event.preventDefault();
    if (!hostId || !selectedNodeName || !showCreate) return;
    setBusy(true); setError('');
    const isVM = showCreate === 'vm';
    const payload = isVM ? {
      vmid: Number(vmForm.vmid), name: vmForm.name, memory_mb: Number(vmForm.memory_mb), cores: Number(vmForm.cores),
      disk_gb: Number(vmForm.disk_gb), storage: vmForm.storage, bridge: vmForm.bridge, iso: vmForm.iso, boot: vmForm.boot,
    } : {
      vmid: Number(vmForm.vmid), hostname: vmForm.name, memory_mb: Number(vmForm.memory_mb), cores: Number(vmForm.cores),
      disk_gb: Number(vmForm.disk_gb), storage: vmForm.storage, bridge: vmForm.bridge, ostemplate: vmForm.iso,
    };
    try {
      const result = await pxPost(hostId, `/nodes/${encodeURIComponent(selectedNodeName)}/${isVM ? 'qemu' : 'lxc'}`, payload);
      setNotice(`${isVM ? 'VM' : 'LXC'} creation queued. ${readString(objectFrom(result).task)}`);
      setShowCreate(null); setVMForm(emptyVM); await loadWorkspace();
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Resource creation failed.'); }
    finally { setBusy(false); }
  }

  async function loadContent(storageName: string) {
    if (!hostId || !selectedNodeName) return;
    setSelectedStorage(storageName);
    const payload = await read(pxGet(hostId, `/nodes/${encodeURIComponent(selectedNodeName)}/storage/${encodeURIComponent(storageName)}/content`), {});
    setStorageContent(listFrom(payload, 'content', 'entries'));
  }

  async function loadTokens(row: Row) {
    if (!hostId || !row.userid && !row.id) return;
    setSelectedUser({ ...row, tokens: listFrom(await read(pxGet(hostId, `/access/users/${encodeURIComponent(readString(row.userid || row.id))}/token`), {}), 'tokens') });
  }

  async function markTemplate(row: Row) {
    if (!hostId || !window.confirm(`Convert ${readString(row.name || row.vmid)} into a template?`)) return;
    setBusy(true); setError('');
    try { await pxPost(hostId, `/nodes/${encodeURIComponent(readString(row.node))}/${readString(row.type).toLowerCase().includes('lxc') ? 'lxc' : 'qemu'}/${row.vmid}/template`, {}); setNotice('Template conversion queued.'); await loadWorkspace(); }
    catch (cause) { setError(cause instanceof Error ? cause.message : 'Template conversion failed.'); }
    finally { setBusy(false); }
  }

  async function applyCloudInit(row: Row) {
    if (!hostId) return;
    const username = window.prompt('Cloud-init username (optional):', 'stackwatch');
    if (username === null) return;
    const sshkeys = window.prompt('SSH public key (optional):', '') || '';
    setBusy(true); setError('');
    try { await pxPost(hostId, `/nodes/${encodeURIComponent(readString(row.node))}/${readString(row.type).toLowerCase().includes('lxc') ? 'lxc' : 'qemu'}/${row.vmid}/cloud-init`, { ciuser: username, sshkeys }); setNotice('Cloud-init configuration applied.'); }
    catch (cause) { setError(cause instanceof Error ? cause.message : 'Cloud-init update failed.'); }
    finally { setBusy(false); }
  }

  async function migrateResource(row: Row) {
    if (!hostId) return;
    const target = window.prompt(`Target node for ${readString(row.name || row.vmid)}:`);
    if (!target || !window.confirm(`Migrate ${readString(row.name || row.vmid)} to ${target}?`)) return;
    setBusy(true); setError('');
    try { await pxPost(hostId, `/nodes/${encodeURIComponent(readString(row.node))}/${readString(row.type).toLowerCase().includes('lxc') ? 'lxc' : 'qemu'}/${row.vmid}/migrate`, { target, online: true }); setNotice('Migration task queued.'); }
    catch (cause) { setError(cause instanceof Error ? cause.message : 'Migration failed.'); }
    finally { setBusy(false); }
  }

  const hostActions = <>
    <Button tone="quiet" onClick={() => void loadWorkspace()} disabled={loading || !hostId}>↻ Refresh</Button>
    <Button tone="quiet" onClick={() => void testHost()} disabled={busy || !hostId}>Test connection</Button>
    <Button tone="primary" onClick={() => setShowHostForm((value) => !value)}>+ Add Proxmox host</Button>
  </>;

  return <div className="sw-shell">
    <header className="sw-topbar">
      <div><div className="sw-brand"><span className="sw-brand-mark">S</span><span>StackWatch</span></div><span className="sw-product-label">Infrastructure control plane</span></div>
      <div className="sw-host-picker"><span>PROXMOX HOST</span><select value={hostId} onChange={(event) => setHostId(event.target.value)}><option value="">Select a host</option>{hosts.map((item) => <option key={item.id} value={item.id}>{item.name} · {item.base_url}</option>)}</select></div>
      <div className="sw-top-actions">{host && <span className={`sw-status ${statusClass(host.status)}`}>{host.status || 'unknown'}</span>}<Button tone="quiet" onClick={() => { localStorage.removeItem('stackwatch.token'); window.location.href = '/login'; }}>Sign out</Button></div>
    </header>
    <div className="sw-layout">
      <aside className="sw-sidebar">
        <div className="sw-side-title">CONTROL PLANE</div>
        {sections.map((item) => <button className={`sw-nav-item ${section === item.id ? 'active' : ''}`} key={item.id} onClick={() => setSection(item.id)}><span>{item.icon}</span>{item.label}</button>)}
        <div className="sw-side-note"><strong>Tier 1</strong><span>Proxmox VE management</span><small>Every action uses the authenticated API and reports upstream support status.</small></div>
      </aside>
      <main className="sw-main">
        <div className="sw-page-head"><div><span className="sw-eyebrow">TIER 1 · PROXMOX VE</span><h1>{sections.find((item) => item.id === section)?.label}</h1><p>{host ? `${host.name} · ${host.base_url}` : 'Register a Proxmox VE host to begin.'}</p></div><div className="sw-page-actions">{hostActions}</div></div>
        {showHostForm && <Panel title="Register Proxmox VE" eyebrow="Secure connection">
          <form className="sw-form-grid" onSubmit={registerHost}><FormField label="Display name" value={hostForm.name} required onChange={(value) => setHostForm({ ...hostForm, name: value })} placeholder="router" /><FormField label="Base URL" value={hostForm.base_url} required onChange={(value) => setHostForm({ ...hostForm, base_url: value })} placeholder="https://192.168.0.107:8006" /><FormField label="API token" value={hostForm.api_token} required type="password" onChange={(value) => setHostForm({ ...hostForm, api_token: value })} placeholder="user@pam!tokenid=uuid" /><label className="sw-checkbox"><input type="checkbox" checked={hostForm.verify_tls} onChange={(event) => setHostForm({ ...hostForm, verify_tls: event.target.checked })} /><span>Verify TLS certificate</span></label><div className="sw-form-actions"><Button tone="quiet" onClick={() => setShowHostForm(false)}>Cancel</Button><Button type="submit" tone="primary" disabled={busy}>Connect and save</Button></div></form>
        </Panel>}
        {error && <div className="sw-alert sw-alert-error"><strong>Request failed</strong><span>{error}</span><button onClick={() => setError('')}>×</button></div>}
        {notice && <div className="sw-alert sw-alert-success"><strong>Done</strong><span>{notice}</span><button onClick={() => setNotice('')}>×</button></div>}
        {!hostId && <Panel title="No Proxmox host registered" eyebrow="Start here"><div className="sw-empty sw-empty-large"><div className="sw-empty-icon">◈</div><h3>Connect your Proxmox cluster</h3><p>Register a Proxmox VE API token above. StackWatch tests the connection before saving credentials and never returns the token to the browser after registration.</p><Button tone="primary" onClick={() => setShowHostForm(true)}>Register first host</Button></div></Panel>}
        {hostId && section === 'overview' && <Overview nodes={nodes} resources={resources} hostState={hostState} onResource={(row) => { setSection('compute'); setSelectedNode(readString(row.node)); }} />}
        {hostId && section === 'compute' && <Compute nodes={nodes} resources={resources} selectedNode={selectedNodeName} setSelectedNode={setSelectedNode} showCreate={showCreate} setShowCreate={setShowCreate} form={vmForm} setForm={setVMForm} onCreate={createResource} onLifecycle={lifecycle} loading={loading} />}
        {hostId && section === 'storage' && <StorageView nodes={nodes} selectedNode={selectedNodeName} setSelectedNode={setSelectedNode} storage={storage} disks={disks} zfs={zfs} content={storageContent} selectedStorage={selectedStorage} onContent={loadContent} />}
        {hostId && section === 'network' && <NetworkView nodes={nodes} selectedNode={selectedNodeName} setSelectedNode={setSelectedNode} network={network} firewall={firewall} ipsets={ipsets} />}
        {hostId && section === 'access' && <AccessView users={users} selectedUser={selectedUser} onTokens={loadTokens} />}
        {hostId && section === 'operations' && <Operations tasks={tasks} pools={pools} backups={backups} />}
        {hostId && section === 'security' && <Security certificates={certificates} acme={acme} />}
        {hostId && section === 'templates' && <TemplatesView selectedNode={selectedNodeName} templates={templates} storage={storage} resources={resources} onSelectStorage={(value) => setSelectedStorage(value)} onMarkTemplate={markTemplate} onCloudInit={applyCloudInit} />}
        {hostId && section === 'cluster' && <ClusterView status={haStatus} resources={haResources} allResources={resources} onMigrate={migrateResource} />}
        {hostId && section === 'monitoring' && <MonitoringView nodes={nodes} selectedNode={selectedNodeName} setSelectedNode={setSelectedNode} points={monitoring} />}
        {hostId && <div className="sw-danger-zone"><div><strong>Remove registered host</strong><span>This removes StackWatch’s saved connection only. It does not delete anything in Proxmox.</span></div><Button tone="danger" onClick={() => void deleteHost()} disabled={busy}>Remove host</Button></div>}
      </main>
    </div>
  </div>;
}

function Overview({ nodes, resources, hostState, onResource }: { nodes: ProxmoxNode[]; resources: ProxmoxResource[]; hostState: Row; onResource: (row: Row) => void }) {
  const running = resources.filter((row) => readString(row.status).toLowerCase() === 'running').length;
  const stopped = resources.filter((row) => readString(row.status).toLowerCase() === 'stopped').length;
  return <>
    <div className="sw-stat-grid"><Stat label="Cluster nodes" value={nodes.length} hint="Live from Proxmox" /><Stat label="Resources" value={resources.length} hint={`${running} running · ${stopped} stopped`} accent="blue" /><Stat label="Connection" value={readString(objectFrom(hostState.connectivity).ok) === 'false' ? 'Degraded' : 'Healthy'} hint="Last live test" accent="green" /><Stat label="Cluster mode" value={readString(objectFrom(hostState.cluster).type || hostState.status || 'standalone')} hint="Reported by host" accent="purple" /></div>
    <Panel title="Nodes" eyebrow="Compute fabric"><DataTable rows={nodes} columns={[{ key: 'node', label: 'Node' }, { key: 'status', label: 'Status', render: (row) => <span className={`sw-status ${statusClass(row.status)}`}>{displayValue(row.status)}</span> }, { key: 'maxcpu', label: 'CPU capacity' }, { key: 'maxmem', label: 'Memory', render: (row) => formatBytes(row.maxmem) }, { key: 'uptime', label: 'Uptime', render: (row) => row.uptime ? `${Math.floor(Number(row.uptime) / 86400)}d` : '—' }]} /></Panel>
    <Panel title="Cluster resources" eyebrow="VMs and containers"><DataTable rows={resources} onRowClick={onResource} columns={[{ key: 'type', label: 'Type' }, { key: 'vmid', label: 'ID' }, { key: 'name', label: 'Name' }, { key: 'node', label: 'Node' }, { key: 'status', label: 'Status', render: (row) => <span className={`sw-status ${statusClass(row.status)}`}>{displayValue(row.status)}</span> }, { key: 'cpu', label: 'CPU', render: (row) => formatPercent(row.cpu) }, { key: 'mem', label: 'Memory', render: (row) => formatBytes(row.mem) }]} /></Panel>
  </>;
}

function Compute({ nodes, resources, selectedNode, setSelectedNode, showCreate, setShowCreate, form, setForm, onCreate, onLifecycle, loading }: { nodes: ProxmoxNode[]; resources: ProxmoxResource[]; selectedNode: string; setSelectedNode: (value: string) => void; showCreate: 'vm' | 'lxc' | null; setShowCreate: (value: 'vm' | 'lxc' | null) => void; form: typeof emptyVM; setForm: (value: typeof emptyVM) => void; onCreate: (event: FormEvent) => void; onLifecycle: (resource: ProxmoxResource, action: string) => void; loading: boolean }) {
  const nodeResources = resources.filter((row) => !selectedNode || readString(row.node) === selectedNode);
  return <>
    <Panel title="Compute fleet" eyebrow="QEMU + LXC" actions={<><select className="sw-inline-select" value={selectedNode} onChange={(event) => setSelectedNode(event.target.value)}><option value="">All nodes</option>{nodes.map((node) => <option key={readString(node.node)} value={readString(node.node)}>{readString(node.node)}</option>)}</select><Button tone="primary" onClick={() => setShowCreate('vm')}>+ Create VM</Button><Button tone="quiet" onClick={() => setShowCreate('lxc')}>+ Create LXC</Button></>}>
      <DataTable rows={nodeResources} empty={loading ? 'Loading live resources…' : 'No VMs or containers on this node.'} columns={[{ key: 'type', label: 'Type' }, { key: 'vmid', label: 'VMID' }, { key: 'name', label: 'Name' }, { key: 'node', label: 'Node' }, { key: 'status', label: 'Status', render: (row) => <span className={`sw-status ${statusClass(row.status)}`}>{displayValue(row.status)}</span> }, { key: 'cpu', label: 'CPU', render: (row) => formatPercent(row.cpu) }, { key: 'mem', label: 'Memory', render: (row) => formatBytes(row.mem) }, { key: 'actions', label: 'Actions', render: (row) => <div className="sw-row-actions"><Button tone="quiet" onClick={() => void onLifecycle(row as ProxmoxResource, 'start')}>Start</Button><Button tone="quiet" onClick={() => void onLifecycle(row as ProxmoxResource, 'shutdown')}>Stop</Button><Button tone="quiet" onClick={() => void onLifecycle(row as ProxmoxResource, 'reboot')}>Reboot</Button></div> }]} />
    </Panel>
    {showCreate && <Panel title={`Create ${showCreate === 'vm' ? 'QEMU virtual machine' : 'LXC container'}`} eyebrow={`On node ${selectedNode || 'select a node'}`}><form className="sw-form-grid" onSubmit={onCreate}><FormField label="VMID" value={form.vmid} required type="number" min="100" onChange={(value) => setForm({ ...form, vmid: value })} /><FormField label={showCreate === 'vm' ? 'Name' : 'Hostname'} value={form.name} required onChange={(value) => setForm({ ...form, name: value })} /><FormField label="Memory MB" value={form.memory_mb} required type="number" min="128" onChange={(value) => setForm({ ...form, memory_mb: value })} /><FormField label="CPU cores" value={form.cores} required type="number" min="1" onChange={(value) => setForm({ ...form, cores: value })} /><FormField label="Disk GB" value={form.disk_gb} required type="number" min="1" onChange={(value) => setForm({ ...form, disk_gb: value })} /><FormField label="Storage" value={form.storage} onChange={(value) => setForm({ ...form, storage: value })} /><FormField label="Bridge" value={form.bridge} onChange={(value) => setForm({ ...form, bridge: value })} /><FormField label={showCreate === 'vm' ? 'ISO path (optional)' : 'OS template (optional)'} value={form.iso} onChange={(value) => setForm({ ...form, iso: value })} /><div className="sw-form-actions"><Button tone="quiet" onClick={() => setShowCreate(null)}>Cancel</Button><Button type="submit" tone="primary">Queue creation</Button></div></form></Panel>}
  </>;
}

function StorageView({ nodes, selectedNode, setSelectedNode, storage, disks, zfs, content, selectedStorage, onContent }: { nodes: ProxmoxNode[]; selectedNode: string; setSelectedNode: (value: string) => void; storage: Row[]; disks: Row[]; zfs: Row[]; content: Row[]; selectedStorage: string; onContent: (storage: string) => void }) {
  return <><Panel title="Storage pools" eyebrow="Datacenter storage" actions={<select className="sw-inline-select" value={selectedNode} onChange={(event) => setSelectedNode(event.target.value)}><option value="">Select node</option>{nodes.map((node) => <option key={readString(node.node)} value={readString(node.node)}>{readString(node.node)}</option>)}</select>}><DataTable rows={storage} columns={[{ key: 'storage', label: 'Storage' }, { key: 'type', label: 'Type' }, { key: 'status', label: 'Status', render: (row) => <span className={`sw-status ${statusClass(row.status)}`}>{displayValue(row.status)}</span> }, { key: 'content', label: 'Content' }, { key: 'total', label: 'Capacity', render: (row) => formatBytes(row.total) }, { key: 'used', label: 'Used', render: (row) => formatBytes(row.used) }, { key: 'actions', label: 'Browse', render: (row) => <Button tone="quiet" onClick={() => onContent(readString(row.storage || row.name))}>View content</Button> }]} /></Panel><div className="sw-two-col"><Panel title={selectedStorage ? `Content · ${selectedStorage}` : 'Storage content'} eyebrow="Images, templates and volumes"><DataTable rows={content} empty="Choose View content on a storage pool." /></Panel><Panel title="Physical disks" eyebrow="Node inventory"><DataTable rows={disks} empty="Select a node to inspect disks." /></Panel></div><Panel title="ZFS pools" eyebrow="Node inventory"><DataTable rows={zfs} empty="No ZFS pools reported by this node." /></Panel></>;
}

function NetworkView({ nodes, selectedNode, setSelectedNode, network, firewall, ipsets }: { nodes: ProxmoxNode[]; selectedNode: string; setSelectedNode: (value: string) => void; network: Row[]; firewall: Row[]; ipsets: Row[] }) {
  return <><Panel title="Network interfaces" eyebrow="Node networking" actions={<select className="sw-inline-select" value={selectedNode} onChange={(event) => setSelectedNode(event.target.value)}><option value="">Select node</option>{nodes.map((node) => <option key={readString(node.node)} value={readString(node.node)}>{readString(node.node)}</option>)}</select>}><DataTable rows={network} empty="Select a node to load interfaces." /></Panel><div className="sw-two-col"><Panel title="Firewall rules" eyebrow="Node firewall"><DataTable rows={firewall} /></Panel><Panel title="IP sets" eyebrow="Cluster firewall"><DataTable rows={ipsets} /></Panel></div></>;
}

function AccessView({ users, selectedUser, onTokens }: { users: Row[]; selectedUser: Row | null; onTokens: (row: Row) => void }) {
  return <div className="sw-two-col"><Panel title="Proxmox users" eyebrow="Access control"><DataTable rows={users} onRowClick={onTokens} columns={[{ key: 'userid', label: 'User' }, { key: 'comment', label: 'Comment' }, { key: 'enable', label: 'Enabled', render: (row) => displayValue(row.enable ?? row.enabled) }, { key: 'expire', label: 'Expires' }, { key: 'actions', label: 'Tokens', render: (row) => <Button tone="quiet" onClick={() => onTokens(row)}>View tokens</Button> }]} /></Panel><Panel title={selectedUser ? `API tokens · ${readString(selectedUser.userid || selectedUser.id)}` : 'API tokens'} eyebrow="Credentials"><DataTable rows={listFrom(selectedUser?.tokens, 'tokens')} empty="Select a user to load API tokens." /></Panel></div>;
}

function Operations({ tasks, pools, backups }: { tasks: Row[]; pools: Row[]; backups: Row[] }) {
  return <><Panel title="Cluster tasks" eyebrow="Recent operations"><DataTable rows={tasks} columns={[{ key: 'upid', label: 'UPID' }, { key: 'type', label: 'Type' }, { key: 'node', label: 'Node' }, { key: 'status', label: 'Status', render: (row) => <span className={`sw-status ${statusClass(row.status)}`}>{displayValue(row.status)}</span> }, { key: 'starttime', label: 'Started' }, { key: 'user', label: 'User' }]} /></Panel><div className="sw-two-col"><Panel title="Pools" eyebrow="Resource pools"><DataTable rows={pools} /></Panel><Panel title="Backup jobs" eyebrow="Scheduled backups"><DataTable rows={backups} /></Panel></div></>;
}

function Security({ certificates, acme }: { certificates: Row[]; acme: Record<string, unknown> }) {
  return <><Panel title="Node certificates" eyebrow="TLS inventory"><DataTable rows={certificates} /></Panel><div className="sw-two-col"><Panel title="ACME accounts" eyebrow="Certificate authorities"><Unsupported value={acme.accounts} /><DataTable rows={listFrom(acme.accounts, 'accounts')} /></Panel><Panel title="ACME plugins" eyebrow="DNS and standalone challenges"><Unsupported value={acme.plugins} /><DataTable rows={listFrom(acme.plugins, 'plugins')} /></Panel></div><div className="sw-two-col"><Panel title="Challenge schema" eyebrow="Supported challenge types"><DataTable rows={listFrom(acme.challenges, 'challenges')} /></Panel><Panel title="ACME directories" eyebrow="Certificate authorities"><DataTable rows={listFrom(acme.directories, 'directories')} /></Panel></div><Panel title="ACME cluster info" eyebrow="Capability status"><Unsupported value={acme.info} /><JsonBlock value={acme.info} /></Panel></>;
}

function TemplatesView({ selectedNode, templates, storage, resources, onSelectStorage, onMarkTemplate, onCloudInit }: { selectedNode: string; templates: Row[]; storage: Row[]; resources: Row[]; onSelectStorage: (value: string) => void; onMarkTemplate: (row: Row) => void; onCloudInit: (row: Row) => void }) {
  return <><Panel title="Template library" eyebrow="ISO and LXC templates" actions={<><select className="sw-inline-select" value={selectedNode} disabled><option>{selectedNode || 'Select node'}</option></select><select className="sw-inline-select" onChange={(event) => onSelectStorage(event.target.value)}><option value="">Select storage</option>{storage.map((item) => <option key={readString(item.storage || item.name)} value={readString(item.storage || item.name)}>{readString(item.storage || item.name)}</option>)}</select></>}><DataTable rows={templates} empty="Select a storage backend containing vztmpl content." columns={[{ key: 'volid', label: 'Template' }, { key: 'format', label: 'Format' }, { key: 'size', label: 'Size', render: (row) => formatBytes(row.size) }, { key: 'notes', label: 'Notes' }]} /></Panel><Panel title="Guest template actions" eyebrow="Convert and provision"><DataTable rows={resources.filter((row) => readString(row.node) === selectedNode)} empty="No guests on this node." columns={[{ key: 'vmid', label: 'VMID' }, { key: 'name', label: 'Name' }, { key: 'type', label: 'Type' }, { key: 'status', label: 'Status' }, { key: 'actions', label: 'Actions', render: (row) => <div className="sw-row-actions"><Button tone="quiet" onClick={() => void onCloudInit(row)}>Cloud-init</Button><Button tone="quiet" onClick={() => void onMarkTemplate(row)}>Convert to template</Button></div> }]} /></Panel></>;
}

function ClusterView({ status, resources, allResources, onMigrate }: { status: Row; resources: Row[]; allResources: Row[]; onMigrate: (row: Row) => void }) {
  return <><div className="sw-stat-grid"><Stat label="HA status" value={readString(status.state || status.type || status.quorate || 'reported')} hint="Live cluster response" accent="green" /><Stat label="HA resources" value={resources.length} hint="Configured resources" accent="blue" /></div><Panel title="HA resources" eyebrow="High availability"><DataTable rows={resources} empty="No HA resources configured on this cluster." /></Panel><Panel title="Migration actions" eyebrow="Workload movement"><DataTable rows={allResources} empty="No workloads available." columns={[{ key: 'vmid', label: 'VMID' }, { key: 'name', label: 'Name' }, { key: 'node', label: 'Source node' }, { key: 'status', label: 'Status' }, { key: 'actions', label: 'Action', render: (row) => <Button tone="quiet" onClick={() => void onMigrate(row)}>Migrate</Button> }]} /></Panel></>;
}

function MonitoringView({ nodes, selectedNode, setSelectedNode, points }: { nodes: Row[]; selectedNode: string; setSelectedNode: (value: string) => void; points: Row[] }) {
  return <Panel title="Host performance" eyebrow="Proxmox RRD data" actions={<select className="sw-inline-select" value={selectedNode} onChange={(event) => setSelectedNode(event.target.value)}><option value="">Select node</option>{nodes.map((node) => <option key={readString(node.node)} value={readString(node.node)}>{readString(node.node)}</option>)}</select>}><DataTable rows={points} empty="Select a node to load the last-hour performance series." columns={[{ key: 'time', label: 'Timestamp', render: (row) => formatDate(row.time) }, { key: 'cpu', label: 'CPU' }, { key: 'memused', label: 'Memory used', render: (row) => formatBytes(row.memused) }, { key: 'netin', label: 'Network in', render: (row) => formatBytes(row.netin) }, { key: 'netout', label: 'Network out', render: (row) => formatBytes(row.netout) }]} /></Panel>;
}
