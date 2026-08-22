import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react';
import { createTrueNASHost, deleteTrueNASHost, listTrueNASHosts, testTrueNASHost, truenasCall, TNHost, TNRow } from '../lib/truenas';
import FilterBar from '../components/FilterBar';

type Section = { id: string; label: string; path: string };
const sections: Section[] = [
  { id: 'overview', label: 'Overview', path: '/system/info' },
  { id: 'pools', label: 'ZFS Pools', path: '/pools/list' },
  { id: 'datasets', label: 'Datasets', path: '/datasets/list' },
  { id: 'nfs', label: 'NFS Shares', path: '/nfs/list' },
  { id: 'smb', label: 'SMB Shares', path: '/smb/list' },
  { id: 'iscsi', label: 'iSCSI', path: '/iscsi/extents/list' },
  { id: 'snapshots', label: 'Snapshots', path: '/snapshots/list' },
  { id: 'disks', label: 'Disk Health', path: '/disks/list' },
  { id: 'users', label: 'Users & Groups', path: '/users/list' },
  { id: 'system', label: 'System Services', path: '/system/services/list' },
  { id: 'cloud', label: 'Cloud Sync', path: '/cloud/sync/list' },
];

const actionPaths: Record<string, string> = {
  pools: '/pools/create', datasets: '/datasets/create', nfs: '/nfs/create', smb: '/smb/create',
  snapshots: '/snapshots/create', users: '/users/create', cloud: '/cloud/sync/create',
};

function value(row: TNRow, key: string): string {
  const raw = row[key];
  if (raw === null || raw === undefined) return '—';
  if (typeof raw === 'object') return JSON.stringify(raw);
  return String(raw);
}

function rowsFrom(payload: unknown): TNRow[] {
  if (Array.isArray(payload)) return payload as TNRow[];
  if (!payload || typeof payload !== 'object') return [];
  const record = payload as Record<string, unknown>;
  for (const key of ['data', 'rows', 'items', 'pools', 'datasets', 'shares', 'snapshots', 'disks', 'users', 'groups', 'services', 'tasks']) {
    if (Array.isArray(record[key])) return record[key] as TNRow[];
    if (record[key] && typeof record[key] === 'object') {
      const nested = rowsFrom(record[key]);
      if (nested.length) return nested;
    }
  }
  return [record];
}

export default function TrueNASWorkspace() {
  const [hosts, setHosts] = useState<TNHost[]>([]);
  const [hostId, setHostId] = useState('');
  const [section, setSection] = useState('overview');
  const [rows, setRows] = useState<TNRow[]>([]);
  const [raw, setRaw] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [showHostForm, setShowHostForm] = useState(false);
  const [showAction, setShowAction] = useState(false);
  const [actionJSON, setActionJSON] = useState('{\n  "name": "example"\n}');
  const [hostForm, setHostForm] = useState({ name: '', base_url: '', username: '', password: '', api_key: '', verify_tls: false });
  // Client-side filter for the live upstream response table. The
  // JSON-RPC payload is small enough that filtering in-place is
  // fine — no need for a backend roundtrip per keystroke.
  const [search, setSearch] = useState('');
  const selected = useMemo(() => hosts.find((host) => host.id === hostId), [hosts, hostId]);
  const current = sections.find((item) => item.id === section) ?? sections[0];

  const loadHosts = useCallback(async () => {
    const next = await listTrueNASHosts();
    setHosts(next);
    if (!hostId && next[0]) setHostId(next[0].id);
  }, [hostId]);

  const loadData = useCallback(async (nextSection: string, nextHost: string) => {
    if (!nextHost) return;
    const descriptor = sections.find((item) => item.id === nextSection) ?? sections[0];
    setBusy(true); setError('');
    try {
      const payload = await truenasCall(descriptor.path, { host_id: nextHost });
      setRaw(payload); setRows(rowsFrom(payload));
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'TrueNAS request failed.'); setRows([]); setRaw(null); }
    finally { setBusy(false); }
  }, []);

  useEffect(() => { void loadHosts().catch((cause) => setError(cause instanceof Error ? cause.message : 'Unable to load TrueNAS hosts.')); }, [loadHosts]);
  useEffect(() => { if (hostId) void loadData(section, hostId); }, [hostId, section, loadData]);

  async function submitHost(event: FormEvent) {
    event.preventDefault(); setBusy(true); setError('');
    try {
      const created = await createTrueNASHost({ ...hostForm, api_key: hostForm.api_key || undefined, password: hostForm.password || undefined });
      setMessage('TrueNAS host registered.'); setShowHostForm(false); await loadHosts(); setHostId(created.id);
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Unable to register TrueNAS host.'); }
    finally { setBusy(false); }
  }

  async function runTest() {
    if (!hostId) return; setBusy(true); setError('');
    try { await testTrueNASHost(hostId); setMessage('TrueNAS connection is healthy.'); await loadHosts(); }
    catch (cause) { setError(cause instanceof Error ? cause.message : 'TrueNAS connection test failed.'); }
    finally { setBusy(false); }
  }

  async function removeHost() {
    if (!hostId || !window.confirm('Remove this saved TrueNAS connection?')) return;
    setBusy(true); setError('');
    try { await deleteTrueNASHost(hostId); setHostId(''); setMessage('TrueNAS host removed.'); await loadHosts(); }
    catch (cause) { setError(cause instanceof Error ? cause.message : 'Unable to remove host.'); }
    finally { setBusy(false); }
  }

  async function runAction(event: FormEvent) {
    event.preventDefault(); if (!hostId || !actionPaths[section]) return;
    let body: Record<string, unknown>;
    try { body = JSON.parse(actionJSON) as Record<string, unknown>; }
    catch { setError('Action body must be valid JSON.'); return; }
    setBusy(true); setError('');
    try { await truenasCall(actionPaths[section], { host_id: hostId, ...body }); setMessage(`${current.label} action completed.`); setShowAction(false); await loadData(section, hostId); }
    catch (cause) { setError(cause instanceof Error ? cause.message : 'TrueNAS action failed.'); }
    finally { setBusy(false); }
  }

  return <div className="sw-shell">
    <header className="sw-topbar"><div className="sw-brand"><div className="sw-brand-mark">S</div><div>StackWatch<span className="sw-product-label">TrueNAS control plane</span></div></div><div className="sw-host-picker"><span>TrueNAS host</span><select value={hostId} onChange={(event) => setHostId(event.target.value)}><option value="">Select host</option>{hosts.map((host) => <option key={host.id} value={host.id}>{host.name} · {host.base_url}</option>)}</select></div><div className="sw-top-actions"><button className="sw-button sw-button-primary" onClick={() => setShowHostForm((open) => !open)}>+ Add TrueNAS</button><a className="sw-button" href="/">Proxmox</a></div></header>
    <div className="sw-layout"><aside className="sw-sidebar"><div className="sw-side-title">TrueNAS control</div>{sections.map((item) => <button key={item.id} className={`sw-nav-item ${section === item.id ? 'active' : ''}`} onClick={() => setSection(item.id)}><span>◆</span>{item.label}</button>)}<div className="sw-side-note"><strong>TIER 2</strong><span>TrueNAS SCALE</span><small>JSON-RPC over WebSocket. Destructive actions remain explicit and confirmation-backed.</small></div></aside><main className="sw-main">
      <div className="sw-page-head"><div><span className="sw-eyebrow">TIER 2 · TRUENAS SCALE</span><h1>{current.label}</h1><p>{selected ? `${selected.name} · ${selected.base_url}` : 'Register a TrueNAS SCALE host to begin.'}</p></div><div className="sw-page-actions">{hostId && <button className="sw-button" onClick={() => void runTest()} disabled={busy}>Test connection</button>}{hostId && actionPaths[section] && <button className="sw-button sw-button-primary" onClick={() => setShowAction(true)} disabled={busy}>+ Create</button>}</div></div>
      {message && <div className="sw-alert sw-alert-success"><strong>Success</strong><span>{message}</span><button onClick={() => setMessage('')}>×</button></div>}{error && <div className="sw-alert sw-alert-error"><strong>Error</strong><span>{error}</span><button onClick={() => setError('')}>×</button></div>}
      {showHostForm && <section className="sw-panel"><div className="sw-panel-head"><div><span className="sw-eyebrow">Secure registration</span><h2>Connect a TrueNAS SCALE system</h2></div></div><form className="sw-form-grid" onSubmit={submitHost}><label className="sw-field"><span>Name</span><input required value={hostForm.name} onChange={(e) => setHostForm({ ...hostForm, name: e.target.value })} placeholder="e.g. storage-prod" /></label><label className="sw-field"><span>Base URL</span><input required value={hostForm.base_url} onChange={(e) => setHostForm({ ...hostForm, base_url: e.target.value })} placeholder="https://truenas.example.com" /></label><label className="sw-field"><span>Username</span><input value={hostForm.username} onChange={(e) => setHostForm({ ...hostForm, username: e.target.value })} /></label><label className="sw-field"><span>Password</span><input type="password" value={hostForm.password} onChange={(e) => setHostForm({ ...hostForm, password: e.target.value })} /></label><label className="sw-field"><span>API key (optional)</span><input type="password" value={hostForm.api_key} onChange={(e) => setHostForm({ ...hostForm, api_key: e.target.value })} /></label><label className="sw-checkbox"><input type="checkbox" checked={hostForm.verify_tls} onChange={(e) => setHostForm({ ...hostForm, verify_tls: e.target.checked })} /> Verify TLS certificate</label><div className="sw-form-actions"><button type="button" className="sw-button" onClick={() => setShowHostForm(false)}>Cancel</button><button type="submit" className="sw-button sw-button-primary" disabled={busy}>Test and save</button></div></form></section>}
      {!hostId && <section className="sw-panel"><div className="sw-empty sw-empty-large"><div className="sw-empty-icon">◇</div><h3>No TrueNAS host registered</h3><p>Add a TrueNAS SCALE host to manage pools, datasets, shares, iSCSI, snapshots, disks, users, services, boot environments, and cloud sync from StackWatch.</p><button className="sw-button sw-button-primary" onClick={() => setShowHostForm(true)}>Register first host</button></div></section>}
      {hostId && <section className="sw-panel"><div className="sw-panel-head"><div><span className="sw-eyebrow">Live upstream response</span><h2>{busy ? 'Loading…' : `${rows.length} records`}</h2></div>{selected && <span className={`sw-status ${selected.status === 'online' ? 'status-good' : 'status-neutral'}`}>{selected.status ?? 'unknown'}</span>}</div>
        <FilterBar search={search} onSearchChange={setSearch} placeholder={`Filter ${current.label.toLowerCase()} by name, ID, or any column…`} ariaLabel={`Search ${current.label}`} />
        {(() => {
          const q = search.trim().toLowerCase();
          const filtered = q ? rows.filter((row) => {
            for (const key in row) {
              const v = row[key];
              if (v === null || v === undefined) continue;
              if (String(v).toLowerCase().includes(q)) return true;
            }
            return false;
          }) : rows;
          if (!rows.length) return <div className="sw-empty">{busy ? 'Loading live data…' : 'No records returned by TrueNAS.'}</div>;
          if (!filtered.length) return <div className="sw-empty"><strong>No matches for &ldquo;{search}&rdquo;</strong><span>Try a different search term, or clear the field to see all {rows.length} row{rows.length === 1 ? '' : 's'}.</span></div>;
          return <div className="sw-table-wrap"><table className="sw-table"><thead><tr>{Object.keys(rows[0]).slice(0, 8).map((key) => <th key={key}>{key}</th>)}</tr></thead><tbody>{filtered.map((row, index) => <tr key={String(row.id ?? row.name ?? index)}>{Object.keys(rows[0]).slice(0, 8).map((key) => <td key={key}>{value(row, key)}</td>)}</tr>)}</tbody></table></div>;
        })()}
        <pre className="sw-json">{JSON.stringify(raw, null, 2)}</pre>
      </section>}
      {hostId && <div className="sw-danger-zone"><div><strong>Remove saved connection</strong><span>This removes StackWatch credentials only; it does not delete anything on TrueNAS.</span></div><button className="sw-button sw-button-danger" onClick={() => void removeHost()} disabled={busy}>Remove host</button></div>}
      {showAction && <div className="sw-modal-backdrop"><div className="sw-modal"><div className="sw-panel-head"><div><span className="sw-eyebrow">Authenticated mutation</span><h2>Create {current.label}</h2></div></div><form onSubmit={runAction}><textarea className="sw-json-editor" value={actionJSON} onChange={(e) => setActionJSON(e.target.value)} spellCheck={false} /><div className="sw-form-actions"><button type="button" className="sw-button" onClick={() => setShowAction(false)}>Cancel</button><button className="sw-button sw-button-primary" disabled={busy}>Submit action</button></div></form></div></div>}
    </main></div></div>;
}
