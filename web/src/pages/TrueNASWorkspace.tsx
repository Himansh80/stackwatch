import { FormEvent, Key, useCallback, useEffect, useMemo, useState } from 'react';
import {
  createTrueNASHost,
  deleteTrueNASHost,
  listTrueNASHosts,
  testTrueNASHost,
  truenasCall,
  TNHost,
  TNRow,
} from '../lib/truenas';
import FilterBar from '../components/FilterBar';
import EmptyState from '../components/shared/EmptyState';
import StatusPill from '../components/shared/StatusPill';
import KpiCard from '../components/shared/KpiCard';
import Button from '../components/shared/Button';
import Input from '../components/shared/Input';
import Select from '../components/shared/Select';
import Textarea from '../components/shared/Textarea';
import Modal from '../components/shared/Modal';
import DataTable from '../components/shared/DataTable';
import { motion, kpiStagger } from '../lib/motion';
import { DiskTempCard, PoolHealthCard, SnapshotGroup } from './TrueNASWorkspace.cards';
import { columnsFor } from './TrueNASWorkspace.columns';
import { filterRows, poolHealthTone, readNumber, rowsFrom } from './TrueNASWorkspace.helpers';
import { actionPaths, sections, type Section } from './TrueNASWorkspace.types';

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

// Keep these helper imports alive so the file still re-exports them in
// the unlikely event any downstream module imports them from here.
// The component below is the only public export — these are not.
export { poolHealthTone, readNumber };
export type { Section };