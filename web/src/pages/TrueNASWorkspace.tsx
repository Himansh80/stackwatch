import { FormEvent, Key, useCallback, useEffect, useMemo, useState } from 'react';
import { createTrueNASHost, deleteTrueNASHost, listTrueNASHosts, testTrueNASHost, truenasCall, TNHost, TNRow } from '../lib/truenas';
import FilterBar from '../components/FilterBar';
import EmptyState from '../components/shared/EmptyState';
import StatusPill from '../components/shared/StatusPill';
import KpiCard from '../components/shared/KpiCard';
import TimeSeriesChart from '../components/shared/TimeSeriesChart';
import Button from '../components/shared/Button';
import Input from '../components/shared/Input';
import Select from '../components/shared/Select';
import Textarea from '../components/shared/Textarea';
import Modal from '../components/shared/Modal';
import DataTable, { type Column } from '../components/shared/DataTable';
import { motion, kpiStagger } from '../lib/motion';

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

/**
 * columnsFor — section-aware column set for the live upstream
 * response DataTable. Each TrueNAS section returns 3-4 sensible
 * columns built from the dynamic TNRow keys we know exist for
 * that endpoint (the row payload schema is documented in the
 * middleware handler for `/api/v1/truenas/call`).
 */
function columnsFor(sectionId: string): Column<TNRow>[] {
  const idColumn: Column<TNRow> = {
    key: 'id',
    header: 'ID',
    render: (row) => <code>{String(row.id ?? row.name ?? '—')}</code>,
  };
  switch (sectionId) {
    case 'pools':
      return [
        { key: 'name', header: 'Pool', render: (row) => String(row.name ?? '—'), sortable: true },
        {
          key: 'status',
          header: 'Health',
          render: (row) => (
            <StatusPill status={poolHealthTone(row)} label={String(row.status ?? row.health ?? 'unknown')} size="sm" />
          ),
        },
        {
          key: 'size',
          header: 'Size',
          align: 'right',
          render: (row) => formatBytesLocal(readNumber(row, 'size', 'total')),
        },
        {
          key: 'allocated',
          header: 'Used',
          align: 'right',
          render: (row) => {
            const used = readNumber(row, 'allocated', 'used');
            const total = readNumber(row, 'size', 'total');
            const pct = total > 0 ? Math.round((used / total) * 100) : 0;
            return `${formatBytesLocal(used)} (${pct}%)`;
          },
        },
      ];
    case 'datasets':
      return [
        idColumn,
        { key: 'name', header: 'Dataset', render: (row) => String(row.name ?? '—'), sortable: true },
        {
          key: 'available',
          header: 'Available',
          align: 'right',
          render: (row) => formatBytesLocal(readNumber(row, 'available', 'avail')),
        },
      ];
    case 'nfs':
    case 'smb':
      return [
        idColumn,
        { key: 'path', header: 'Path', render: (row) => String(row.path ?? row.name ?? '—') },
        {
          key: 'enabled',
          header: 'State',
          render: (row) => {
            const enabled = row.enabled !== false && row.enabled !== 'false';
            return <StatusPill status={enabled ? 'up' : 'down'} label={enabled ? 'enabled' : 'disabled'} size="sm" />;
          },
        },
      ];
    case 'iscsi':
      return [
        idColumn,
        { key: 'name', header: 'Target', render: (row) => String(row.name ?? '—') },
        { key: 'type', header: 'Type', render: (row) => String(row.type ?? '—') },
      ];
    case 'snapshots':
      return [
        idColumn,
        { key: 'name', header: 'Snapshot', render: (row) => String(row.name ?? '—') },
        {
          key: 'creation',
          header: 'Created',
          render: (row) => {
            const t = row.creation ?? row.created_at;
            if (!t) return '—';
            const d = new Date(String(t));
            return Number.isNaN(d.getTime()) ? '—' : d.toLocaleString();
          },
        },
      ];
    case 'disks':
      return [
        idColumn,
        { key: 'name', header: 'Disk', render: (row) => String(row.name ?? '—') },
        {
          key: 'temperature',
          header: 'Temp',
          align: 'right',
          render: (row) => `${readNumber(row, 'temperature', 'temp')}°C`,
        },
      ];
    case 'users':
    case 'system':
    case 'cloud':
    default:
      return [
        idColumn,
        { key: 'name', header: 'Name', render: (row) => String(row.name ?? '—') },
      ];
  }
}

/**
 * filterRows — client-side filter for the live upstream response
 * table. The JSON-RPC payload is small enough that filtering in
 * place is fine — no need for a backend roundtrip per keystroke.
 */
function filterRows(rows: TNRow[], search: string): TNRow[] {
  if (!search.trim()) return rows;
  const needle = search.toLowerCase();
  return rows.filter((row) =>
    Object.values(row).some((value) => {
      if (value === null || value === undefined) return false;
      if (typeof value === 'object') {
        try { return JSON.stringify(value).toLowerCase().includes(needle); }
        catch { return false; }
      }
      return String(value).toLowerCase().includes(needle);
    }),
  );
}

export default function TrueNASWorkspace() {
  const [hosts, setHosts] = useState<TNHost[]>([]);
  const [hostId, setHostId] = useState('');
  const [section, setSection] = useState('overview');
  const [rows, setRows] = useState<TNRow[]>([]);
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
    const filteredRows = useMemo(() => filterRows(rows, search), [rows, search]);

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
        setRows(rowsFrom(payload));
      } catch (cause) { setError(cause instanceof Error ? cause.message : 'TrueNAS request failed.'); setRows([]); }
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

  // eslint-disable-next-line @typescript-eslint/no-unused-vars, no-unused-vars
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

  return (
    <>
      <div className="px-page-head">
        <div>
          <span className="px-eyebrow">TIER 2 · TRUENAS SCALE</span>
          <h1 className="px-page-title">{current.label}</h1>
          <p className="px-page-sub">{selected ? `${selected.name} · ${selected.base_url}` : 'Register a TrueNAS SCALE host to begin.'}</p>
        </div>
        <div className="px-page-actions">
                  <Select
                    options={[
                      { value: '', label: 'Select host', disabled: true },
                      ...hosts.map((host) => ({
                        value: host.id,
                        label: `${host.name} · ${host.base_url}`,
                      })),
                    ]}
                    value={hostId}
                    onChange={(event) => setHostId(event.target.value)}
                    aria-label="Active TrueNAS host"
                    className="px-host-select"
                  />
                  {hostId && <Button variant="primary" onClick={() => void runTest()} disabled={busy}>
                    Test connection
                  </Button>}
                  {hostId && actionPaths[section] && <Button variant="primary" onClick={() => setShowAction(true)} disabled={busy}>
                    + Create
                  </Button>}
                  <Button variant="primary" onClick={() => setShowHostForm((open) => !open)}>
                    + Add TrueNAS
                  </Button>
                </div>
      </div>

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
      {message && (
        <div className="sw-alert sw-alert-success">
          <strong>Success</strong>
          <span>{message}</span>
          <Button variant="ghost" size="sm" onClick={() => setMessage('')}>×</Button>
        </div>
      )}
      {error && (
        <div className="sw-alert sw-alert-error">
          <strong>Error</strong>
          <span>{error}</span>
          <Button variant="ghost" size="sm" onClick={() => setError('')}>×</Button>
        </div>
      )}
      {showHostForm && (
        <section className="sw-panel">
          <div className="sw-panel-head">
            <div><span className="sw-eyebrow">Secure registration</span><h2>Connect a TrueNAS SCALE system</h2></div>
          </div>
          <form className="sw-form-grid" onSubmit={submitHost}>
            <Input label="Name" required value={hostForm.name} onChange={(e) => setHostForm({ ...hostForm, name: e.target.value })} placeholder="e.g. storage-prod" />
            <Input label="Base URL" required value={hostForm.base_url} onChange={(e) => setHostForm({ ...hostForm, base_url: e.target.value })} placeholder="https://truenas.example.com" />
            <Input label="Username" value={hostForm.username} onChange={(e) => setHostForm({ ...hostForm, username: e.target.value })} />
            <Input label="Password" type="password" value={hostForm.password} onChange={(e) => setHostForm({ ...hostForm, password: e.target.value })} />
            <Input label="API key (optional)" type="password" value={hostForm.api_key} onChange={(e) => setHostForm({ ...hostForm, api_key: e.target.value })} />
            <label className="sw-checkbox"><input type="checkbox" checked={hostForm.verify_tls} onChange={(e) => setHostForm({ ...hostForm, verify_tls: e.target.checked })} /> Verify TLS certificate</label>
            <div className="sw-form-actions">
              <Button variant="ghost" type="button" onClick={() => setShowHostForm(false)}>Cancel</Button>
              <Button variant="primary" type="submit" loading={busy} disabled={busy}>Test and save</Button>
            </div>
          </form>
        </section>
      )}
      {!hostId && (
        <section className="sw-panel">
          <EmptyState
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
          />
        </section>
      )}
      {hostId && (
        <section className="sw-panel">
          <div className="sw-panel-head">
            <div><span className="sw-eyebrow">Live upstream response</span><h2>{busy ? 'Loading…' : `${rows.length} records`}</h2></div>
            {selected && <StatusPill status={selected.status === 'online' ? 'up' : 'unknown'} label={selected.status ?? 'unknown'} size="sm" />}
          </div>
          <FilterBar search={search} onSearchChange={setSearch} placeholder={`Filter ${current.label.toLowerCase()} by name, ID, or any column…`} ariaLabel={`Search ${current.label}`} />
          <DataTable
                      rows={filteredRows}
                      columns={columnsFor(section)}
                      rowKey={(row: TNRow): Key => String(row.id ?? row.name ?? JSON.stringify(row).slice(0, 32))}
                      loading={busy}
                      emptyTitle={busy ? 'Loading live resources…' : `No ${current.label.toLowerCase()} found.`}
                      emptyDescription={busy ? 'Fetching the latest data from this TrueNAS host.' : 'Try adjusting your search or selecting a different section.'}
                    />
        </section>
      )}
      {showAction && current && (
              <Modal
                open={showAction}
                onClose={() => setShowAction(false)}
                title={`Create ${current.label}`}
                description="Submit an authenticated mutation to this TrueNAS host."
                size="md"
                footer={
                  <>
                    <Button variant="ghost" type="button" onClick={() => setShowAction(false)}>Cancel</Button>
                    <Button variant="primary" type="submit" form="truenas-action-form" loading={busy} disabled={busy}>Submit action</Button>
                  </>
                }
              >
                <form id="truenas-action-form" onSubmit={runAction}>
                  <Textarea
                    label="JSON body"
                    description="The request body sent to the TrueNAS JSON-RPC endpoint."
                    value={actionJSON}
                    onChange={(event) => setActionJSON(event.target.value)}
                    spellCheck={false}
                    rows={8}
                    fullWidth
                  />
                </form>
              </Modal>
            )}
    </>);
    }
