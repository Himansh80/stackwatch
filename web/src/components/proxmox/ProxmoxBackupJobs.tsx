/**
 * Tier 14 Phase 14.10 — Backup jobs CRUD + run-now.
 */
import { useCallback, useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { api } from '../../lib/api';
import ProxmoxShell from './ProxmoxShell';

interface BackupJob {
  id: string;
  schedule?: string;
  selection_mode?: string;
  storage?: string;
  'prune-backups'?: string;
  'prune-days'?: number;
  enabled?: number;
  next_run?: number;
  comment?: string;
}

export default function ProxmoxBackupJobs() {
  const [hostId, setHostId] = useState('');
  const [jobs, setJobs] = useState<BackupJob[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [creating, setCreating] = useState(false);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [running, setRunning] = useState<string | null>(null);

  useEffect(() => {
    api<{ hosts?: Array<{ id: string }> }>('GET', '/api/v1/proxmox/hosts')
      .then((d) => {
        const hosts = d.hosts ?? [];
        if (hosts.length > 0 && hosts[0]) setHostId(hosts[0].id);
      })
      .catch(() => {});
  }, []);

  const load = useCallback(async () => {
    if (!hostId) return;
    setLoading(true);
    setError('');
    try {
      const data = await api<BackupJob[] | { jobs?: BackupJob[] }>('GET', `/api/v1/proxmox/hosts/${hostId}/cluster/backup`);
      setJobs(Array.isArray(data) ? data : data.jobs ?? []);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to load backup jobs.');
    } finally {
      setLoading(false);
    }
  }, [hostId]);

  useEffect(() => { void load(); }, [load]);

  async function runNow(id: string) {
    setRunning(id);
    try {
      await api('POST', `/api/v1/proxmox/hosts/${hostId}/cluster/backup/${encodeURIComponent(id)}/run`);
      setError('');
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Run failed.');
    } finally {
      setRunning(null);
    }
  }

  async function doDelete() {
    if (!deleting) return;
    try {
      await api('DELETE', `/api/v1/proxmox/hosts/${hostId}/cluster/backup/${encodeURIComponent(deleting)}`);
      setDeleting(null);
      await load();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Delete failed.');
    }
  }

  return (
    <motion.div className="px-vm-page" initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
      <h1 className="px-detail-name">Backup jobs</h1>
      {error && <div className="px-error">⚠ {error}</div>}
      <div className="px-tab-toolbar">
        <button type="button" className="px-btn" onClick={() => setCreating(true)}>+ Create job</button>
      </div>
      {loading ? (
        <p className="px-muted">Loading…</p>
      ) : jobs.length === 0 ? (
        <div className="px-info-card">
          <h3>No backup jobs</h3>
          <p>Create a backup job to schedule automatic backups of all your VMs and containers.</p>
        </div>
      ) : (
        <table className="px-network-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Schedule</th>
              <th>Mode</th>
              <th>Storage</th>
              <th>Keep</th>
              <th>Enabled</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {jobs.map((j) => (
              <tr key={j.id}>
                <td className="px-mono">{j.id}</td>
                <td className="px-mono">{j.schedule || '—'}</td>
                <td>{j.selection_mode || '—'}</td>
                <td className="px-mono">{j.storage || '—'}</td>
                <td className="px-mono">{j['prune-backups'] || '—'}</td>
                <td>
                  <span className={`px-action-badge px-action-${j.enabled === 1 ? 'accept' : 'drop'}`}>
                    {j.enabled === 1 ? 'yes' : 'no'}
                  </span>
                </td>
                <td>
                  <button type="button" className="px-btn px-btn-quiet" onClick={() => runNow(j.id)} disabled={running === j.id}>
                    {running === j.id ? 'Running…' : '▶ Run now'}
                  </button>
                  <button type="button" className="px-btn px-btn-quiet px-btn-danger-text" onClick={() => setDeleting(j.id)}>Delete</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {creating && (
        <BackupJobDialog hostId={hostId} onClose={() => setCreating(false)} onCreated={() => { setCreating(false); void load(); }} />
      )}

      {deleting && (
        <motion.div className="px-confirm-overlay" initial={{ opacity: 0 }} animate={{ opacity: 1 }} onClick={() => setDeleting(null)}>
          <motion.div className="px-confirm-dialog" initial={{ scale: 0.95 }} animate={{ scale: 1 }} onClick={(e) => e.stopPropagation()}>
            <h3>Delete backup job?</h3>
            <p>Delete <code>{deleting}</code>? Past backups on the storage are NOT deleted.</p>
            <div className="px-confirm-actions">
              <button type="button" className="px-btn px-btn-quiet" onClick={() => setDeleting(null)}>Cancel</button>
              <button type="button" className="px-btn px-btn-danger" onClick={doDelete}>Yes, delete</button>
            </div>
          </motion.div>
        </motion.div>
      )}
    </motion.div>
  );
}

function BackupJobDialog({ hostId, onClose, onCreated }: { hostId: string; onClose: () => void; onCreated: () => void }) {
  const [form, setForm] = useState({ id: '', schedule: 'daily', storage: 'local', all: 1, keep_daily: 7, comment: '' });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  async function create() {
    if (!form.id) { setError('ID required.'); return; }
    setBusy(true); setError('');
    try {
      await api('POST', `/api/v1/proxmox/hosts/${hostId}/cluster/backup`, form);
      onCreated();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Create failed.');
      setBusy(false);
    }
  }

  return (
    <ProxmoxShell title="Backup jobs" subtitle="Schedule + run backups">
      <motion.div className="px-confirm-overlay" initial={{ opacity: 0 }} animate={{ opacity: 1 }} onClick={onClose}>
      <motion.div className="px-confirm-dialog px-form-wide" initial={{ scale: 0.95 }} animate={{ scale: 1 }} onClick={(e) => e.stopPropagation()}>
        <h3>Create backup job</h3>
        <div className="px-form-grid">
          <div className="px-form-field">
            <label>ID *</label>
            <input type="text" value={form.id} onChange={(e) => setForm({ ...form, id: e.target.value })} placeholder="daily-backup" className="px-confirm-input" autoFocus />
          </div>
          <div className="px-form-field">
            <label>Schedule</label>
            <select value={form.schedule} onChange={(e) => setForm({ ...form, schedule: e.target.value })} className="px-form-select">
              <option value="daily">Daily (02:00)</option>
              <option value="weekly">Weekly (Sun 02:00)</option>
              <option value="monthly">Monthly (1st 02:00)</option>
            </select>
          </div>
          <div className="px-form-field">
            <label>Storage</label>
            <input type="text" value={form.storage} onChange={(e) => setForm({ ...form, storage: e.target.value })} placeholder="local" className="px-confirm-input" />
          </div>
          <div className="px-form-field">
            <label>Keep (days)</label>
            <input type="number" value={form.keep_daily} onChange={(e) => setForm({ ...form, keep_daily: Number(e.target.value) })} className="px-confirm-input" />
          </div>
          <div className="px-form-field px-form-full">
            <label>Comment</label>
            <input type="text" value={form.comment} onChange={(e) => setForm({ ...form, comment: e.target.value })} placeholder="optional" className="px-confirm-input" />
          </div>
        </div>
        {error && <div className="px-confirm-warn">{error}</div>}
        <div className="px-confirm-actions">
          <button type="button" className="px-btn px-btn-quiet" onClick={onClose} disabled={busy}>Cancel</button>
          <button type="button" className="px-btn" onClick={create} disabled={busy}>{busy ? 'Creating…' : 'Create'}</button>
        </div>
      </motion.div>
    </motion.div>
    </ProxmoxShell>
  );
}