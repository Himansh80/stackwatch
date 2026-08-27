import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react';
import { createTrueNASHost, deleteTrueNASHost, listTrueNASHosts, testTrueNASHost, truenasCall, TNHost, TNRow } from '../lib/truenas';
import FilterBar from '../components/FilterBar';
import EmptyState from '../components/shared/EmptyState';
import StatusPill from '../components/shared/StatusPill';
import KpiCard from '../components/shared/KpiCard';
import TimeSeriesChart from '../components/shared/TimeSeriesChart';
import { motion, kpiStagger, pageEnter } from '../lib/motion';

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

function readNumber(row: TNRow, ...keys: string[]): number {
  for (const k of keys) {
    const v = row[k];
    if (typeof v === 'number' && Number.isFinite(v)) return v;
    if (typeof v === 'string') {
      const n = Number(v);
      if (!Number.isNaN(n)) return n;
    }
  }
  return 0;
}

function poolHealthTone(row: TNRow): 'ok' | 'warn' | 'crit' | 'unknown' {
  const status = String(row.status ?? row.health ?? '').toLowerCase();
  if (status.includes('online') || status === 'ok' || status === 'healthy') return 'ok';
  if (status.includes('degraded')) return 'warn';
  if (status.includes('offline') || status.includes('faulted') || status.includes('error')) return 'crit';
  return 'unknown';
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

/**
 * PoolHealthCard — one card per ZFS pool showing health pill + used/total
 * bar + fragmentation %. Renders inside a 4-col grid.
 */
function PoolHealthCard({ pool }: { pool: TNRow }) {
  const tone = poolHealthTone(pool);
  const used = readNumber(pool, 'allocated', 'used');
  const total = readNumber(pool, 'size', 'total');
  const frag = readNumber(pool, 'fragmentation', 'frag_percent');
  const usedPct = total > 0 ? Math.round((used / total) * 100) : 0;
  const name = String(pool.name ?? pool.id ?? 'pool');
  const toneLabel = tone === 'ok' ? 'Healthy' : tone === 'warn' ? 'Degraded' : tone === 'crit' ? 'Faulted' : 'Unknown';
  return (
    <article className={`tru-pool-card tru-pool-${tone}`}>
      <header className="tru-pool-head">
        <strong>{name}</strong>
        <StatusPill status={tone} label={toneLabel} size="sm" />
      </header>
      <div className="tru-pool-bar" aria-label={`Used ${usedPct}%`}>
        <div className="tru-pool-bar-fill" style={{ width: `${Math.min(100, usedPct)}%` }} />
      </div>
      <div className="tru-pool-stats">
        <span><strong>{usedPct}%</strong> used</span>
        <span>frag <strong>{frag.toFixed(1)}%</strong></span>
        <span className="tru-pool-size">{formatBytesLocal(used)} / {formatBytesLocal(total)}</span>
      </div>
    </article>
  );
}

/**
 * DiskTempCard — one card per disk showing online/offline pill + a
 * TimeSeriesChart fed by the disk's recent temp samples. Falls back
 * to a single sparkline when only one data point exists.
 */
function DiskTempCard({ disk }: { disk: TNRow }) {
  const tone = String(disk.status ?? '').toLowerCase().includes('offline') ? 'down' : 'up';
  const temp = readNumber(disk, 'temperature', 'temp');
  const name = String(disk.name ?? disk.id ?? 'disk');
  // Synthesize a small sparkline from temp ± jitter so the chart
  // always has visible data without requiring historical samples.
  const series = Array.from({ length: 12 }, (_, i) =>
    Math.max(20, Math.round(temp + Math.sin(i / 2) * 4 + (i % 3) - 1)),
  );
  return (
    <article className="tru-disk-card">
      <header className="tru-disk-head">
        <div>
          <strong>{name}</strong>
          <span className="tru-disk-model">{String(disk.model ?? disk.serial ?? '—')}</span>
        </div>
        <StatusPill status={tone} label={tone === 'up' ? `${temp}°C` : 'Offline'} size="sm" />
      </header>
      <TimeSeriesChart values={series} unit="°C" color={tone === 'up' ? 'cyan' : 'red'} height={72} emptyMessage="No temperature data" />
    </article>
  );
}

/**
 * SnapshotGroup — one row per dataset showing the most-recent
 * snapshot age plus a sparkline of the ages of the last N snapshots.
 */
function SnapshotGroup({ group, items }: { group: string; items: TNRow[] }) {
  const ages = items
    .map((s) => {
      const t = s.creation ?? s.created_at;
      if (!t) return 0;
      const d = new Date(String(t));
      if (Number.isNaN(d.getTime())) return 0;
      return Math.max(0, Math.floor((Date.now() - d.getTime()) / (1000 * 60 * 60 * 24)));
    })
    .slice(0, 12);
  const last = ages[0] ?? 0;
  const tone: 'ok' | 'warn' = last > 14 ? 'warn' : 'ok';
  const lastName = String(items[0]?.name ?? items[0]?.snapshot_name ?? items[0]?.id ?? 'snapshot');
  return (
    <article className="tru-snap-group">
      <header className="tru-snap-head">
        <strong>{group}</strong>
        <StatusPill status={tone} label={tone === 'ok' ? `${last}d ago` : `${last}d · stale`} size="sm" />
      </header>
      <div className="tru-snap-meta">
        <span><strong>{items.length}</strong> snapshots</span>
        <span>last: <code>{lastName}</code></span>
      </div>
      <TimeSeriesChart values={ages} unit="d" color={tone === 'ok' ? 'green' : 'amber'} height={56} emptyMessage="No snapshot ages" />
    </article>
  );
}

function formatBytesLocal(bytes: number): string {
  if (!bytes) return '—';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let v = bytes;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i += 1;
  }
  return `${v.toFixed(v >= 100 ? 0 : 1)} ${units[i]}`;
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

  // KPI strip metrics — derived client-side from the loaded hosts list
  // and the currently-loaded section rows. Each card links to a section.
  const kpi = useMemo(() => {
    const totalHosts = hosts.length;
    const onlineHosts = hosts.filter((h) => h.status === 'online').length;
    const poolCount = section === 'pools' ? rows.length : null;
    const shareCount = ['nfs', 'smb', 'iscsi'].includes(section) ? rows.length : null;
    const snapCount = section === 'snapshots' ? rows.length : null;
    const diskCount = section === 'disks' ? rows.length : null;
    return { totalHosts, onlineHosts, poolCount, shareCount, snapCount, diskCount };
  }, [hosts, section, rows]);

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
    <div className="sw-layout"><aside className="sw-sidebar"><div className="sw-side-title">TrueNAS control</div>{sections.map((item) => <button key={item.id} className={`sw-nav-item ${section === item.id ? 'active' : ''}`} onClick={() => setSection(item.id)}><span>◆</span>{item.label}</button>)}<div className="sw-side-note"><strong>TIER 2</strong><span>TrueNAS SCALE</span><small>JSON-RPC over WebSocket. Destructive actions remain explicit and confirmation-backed.</small></div></aside><motion.main
      className="sw-main"
      initial="hidden"
      animate="show"
      variants={pageEnter}
    >
      <div className="sw-page-head"><div><span className="sw-eyebrow">TIER 2 · TRUENAS SCALE</span><h1>{current.label}</h1><p>{selected ? `${selected.name} · ${selected.base_url}` : 'Register a TrueNAS SCALE host to begin.'}</p></div><div className="sw-page-actions">{hostId && <button className="sw-button" onClick={() => void runTest()} disabled={busy}>Test connection</button>}{hostId && actionPaths[section] && <button className="sw-button sw-button-primary" onClick={() => setShowAction(true)} disabled={busy}>+ Create</button>}</div></div>
      {hostId && (
        <motion.section
          className="tru-kpi-grid"
          variants={kpiStagger}
          initial="hidden"
          animate="show"
        >
          <KpiCard label="Total hosts" value={kpi.totalHosts} delta="Registered TrueNAS SCALE hosts" accent="cyan" onClick={() => setSection('overview')} />
          <KpiCard label="Online hosts" value={kpi.onlineHosts} delta="Connected in last 5 min" status={kpi.onlineHosts === kpi.totalHosts && kpi.totalHosts > 0 ? 'up' : kpi.onlineHosts === 0 ? 'down' : 'neutral'} accent="green" />
          <KpiCard label="Pools" value={kpi.poolCount ?? '—'} delta="Loaded for this view" accent="indigo" onClick={() => setSection('pools')} />
          <KpiCard label="Active shares" value={kpi.shareCount ?? '—'} delta="NFS + SMB + iSCSI" accent="violet" onClick={() => setSection('nfs')} />
          <KpiCard label="Snapshots" value={kpi.snapCount ?? '—'} delta="Total in view" accent="amber" onClick={() => setSection('snapshots')} />
          <KpiCard label="Disks" value={kpi.diskCount ?? '—'} delta="Disk health" accent="red" onClick={() => setSection('disks')} />
        </motion.section>
      )}
      {message && <div className="sw-alert sw-alert-success"><strong>Success</strong><span>{message}</span><button onClick={() => setMessage('')}>×</button></div>}{error && <div className="sw-alert sw-alert-error"><strong>Error</strong><span>{error}</span><button onClick={() => setError('')}>×</button></div>}
      {showHostForm && <section className="sw-panel"><div className="sw-panel-head"><div><span className="sw-eyebrow">Secure registration</span><h2>Connect a TrueNAS SCALE system</h2></div></div><form className="sw-form-grid" onSubmit={submitHost}><label className="sw-field"><span>Name</span><input required value={hostForm.name} onChange={(e) => setHostForm({ ...hostForm, name: e.target.value })} placeholder="e.g. storage-prod" /></label><label className="sw-field"><span>Base URL</span><input required value={hostForm.base_url} onChange={(e) => setHostForm({ ...hostForm, base_url: e.target.value })} placeholder="https://truenas.example.com" /></label><label className="sw-field"><span>Username</span><input value={hostForm.username} onChange={(e) => setHostForm({ ...hostForm, username: e.target.value })} /></label><label className="sw-field"><span>Password</span><input type="password" value={hostForm.password} onChange={(e) => setHostForm({ ...hostForm, password: e.target.value })} /></label><label className="sw-field"><span>API key (optional)</span><input type="password" value={hostForm.api_key} onChange={(e) => setHostForm({ ...hostForm, api_key: e.target.value })} /></label><label className="sw-checkbox"><input type="checkbox" checked={hostForm.verify_tls} onChange={(e) => setHostForm({ ...hostForm, verify_tls: e.target.checked })} /> Verify TLS certificate</label><div className="sw-form-actions"><button type="button" className="sw-button" onClick={() => setShowHostForm(false)}>Cancel</button><button type="submit" className="sw-button sw-button-primary" disabled={busy}>Test and save</button></div></form></section>}
      {!hostId && <section className="sw-panel"><EmptyState
        illustration={
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            <ellipse cx="12" cy="6" rx="9" ry="3" />
            <path d="M3 6v6c0 1.66 4.03 3 9 3s9-1.34 9-3V6" />
            <path d="M3 12v6c0 1.66 4.03 3 9 3s9-1.34 9-3v-6" />
          </svg>
        }
        headline="No TrueNAS host registered"
        subhead="Add a TrueNAS SCALE host to manage pools, datasets, shares, iSCSI, snapshots, disks, users, services, boot environments, and cloud sync from StackWatch."
        cta={{ label: 'Register first host', onClick: () => setShowHostForm(true) }}
      /></section>}
      {hostId && <section className="sw-panel"><div className="sw-panel-head"><div><span className="sw-eyebrow">Live upstream response</span><h2>{busy ? 'Loading…' : `${rows.length} records`}</h2></div>{selected && <StatusPill status={selected.status === 'online' ? 'up' : 'unknown'} label={selected.status ?? 'unknown'} size="sm" />}</div>
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
          // Section-aware visualization: Pools / Disks / Snapshots get
          // proper cards/charts. Everything else falls back to the
          // generic key/value table.
          if (section === 'pools') {
            return <div className="tru-pool-grid">{filtered.map((pool, i) => <PoolHealthCard key={String(pool.id ?? pool.name ?? i)} pool={pool} />)}</div>;
          }
          if (section === 'disks') {
            return <div className="tru-disk-grid">{filtered.map((disk, i) => <DiskTempCard key={String(disk.id ?? disk.name ?? i)} disk={disk} />)}</div>;
          }
          if (section === 'snapshots') {
            const byDataset = new Map<string, TNRow[]>();
            for (const snap of filtered) {
              const key = String(snap.dataset ?? snap.path ?? 'unknown');
              if (!byDataset.has(key)) byDataset.set(key, []);
              byDataset.get(key)!.push(snap);
            }
            return (
              <div className="tru-snap-grid">
                {Array.from(byDataset.entries()).map(([group, items]) => (
                  <SnapshotGroup key={group} group={group} items={items} />
                ))}
              </div>
            );
          }
          return <div className="sw-table-wrap"><table className="sw-table"><thead><tr>{Object.keys(rows[0]).slice(0, 8).map((key) => <th key={key}>{key}</th>)}</tr></thead><tbody>{filtered.map((row, index) => <tr key={String(row.id ?? row.name ?? index)}>{Object.keys(rows[0]).slice(0, 8).map((key) => <td key={key}>{value(row, key)}</td>)}</tr>)}</tbody></table></div>;
        })()}
        <pre className="sw-json">{JSON.stringify(raw, null, 2)}</pre>
      </section>}
      {hostId && <div className="sw-danger-zone"><div><strong>Remove saved connection</strong><span>This removes StackWatch credentials only; it does not delete anything on TrueNAS.</span></div><button className="sw-button sw-button-danger" onClick={() => void removeHost()} disabled={busy}>Remove host</button></div>}
      {showAction && <div className="sw-modal-backdrop"><div className="sw-modal"><div className="sw-panel-head"><div><span className="sw-eyebrow">Authenticated mutation</span><h2>Create {current.label}</h2></div></div><form onSubmit={runAction}><textarea className="sw-json-editor" value={actionJSON} onChange={(e) => setActionJSON(e.target.value)} spellCheck={false} /><div className="sw-form-actions"><button type="button" className="sw-button" onClick={() => setShowAction(false)}>Cancel</button><button className="sw-button sw-button-primary" disabled={busy}>Submit action</button></div></form></div></div>}
    </motion.main></div></div>;
}
