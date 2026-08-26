/**
 * Tier 14 Phase 14.8 — Alias manager (datacenter-scoped).
 * Reusable network aliases (name → CIDR).
 */
import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { api } from '../../lib/api';

interface Alias {
  name: string;
  cidr: string;
  comment?: string;
}

export default function ProxmoxAliasManager() {
  const [aliases, setAliases] = useState<Alias[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState({ name: '', cidr: '', comment: '' });
  const [deleting, setDeleting] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await api<{ aliases?: Alias[] } | Alias[]>('GET', '/api/v1/proxmox/aliases');
      const list = Array.isArray(data) ? data : data.aliases ?? [];
      setAliases(list);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to load aliases.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  async function create() {
    if (!form.name || !form.cidr) return;
    try {
      await api('POST', '/api/v1/proxmox/aliases', form);
      setCreating(false);
      setForm({ name: '', cidr: '', comment: '' });
      await load();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Create failed.');
    }
  }

  async function doDelete() {
    if (!deleting) return;
    try {
      await api('DELETE', `/api/v1/proxmox/aliases/${encodeURIComponent(deleting)}`);
      setDeleting(null);
      await load();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Delete failed.');
    }
  }

  if (loading) {
    return <div className="px-vm-page"><p className="px-muted">Loading aliases…</p></div>;
  }

  return (
    <motion.div className="px-vm-page" initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
      <div className="px-detail-breadcrumb">
        <Link to="/proxmox">Proxmox</Link>
        <span className="px-detail-sep">›</span>
        <span>Aliases</span>
      </div>
      <h1 className="px-detail-name">Network aliases</h1>

      {error && <div className="px-error">⚠ {error}</div>}

      <div className="px-tab-toolbar">
        <button type="button" className="px-btn" onClick={() => setCreating(true)}>
          + Create alias
        </button>
      </div>

      {creating && (
        <motion.div className="px-confirm-overlay" initial={{ opacity: 0 }} animate={{ opacity: 1 }} onClick={() => setCreating(false)}>
          <motion.div className="px-confirm-dialog" initial={{ scale: 0.95 }} animate={{ scale: 1 }} onClick={(e) => e.stopPropagation()}>
            <h3>Create alias</h3>
            <div className="px-form-field">
              <label>Name *</label>
              <input type="text" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="office-network" className="px-confirm-input" autoFocus />
            </div>
            <div className="px-form-field">
              <label>CIDR *</label>
              <input type="text" value={form.cidr} onChange={(e) => setForm({ ...form, cidr: e.target.value })} placeholder="192.168.0.0/24" className="px-confirm-input" />
            </div>
            <div className="px-form-field">
              <label>Comment</label>
              <input type="text" value={form.comment} onChange={(e) => setForm({ ...form, comment: e.target.value })} placeholder="optional" className="px-confirm-input" />
            </div>
            <div className="px-confirm-actions">
              <button type="button" className="px-btn px-btn-quiet" onClick={() => setCreating(false)}>Cancel</button>
              <button type="button" className="px-btn" onClick={create}>Create</button>
            </div>
          </motion.div>
        </motion.div>
      )}

      {aliases.length === 0 ? (
        <div className="px-info-card">
          <h3>No aliases</h3>
          <p>Create aliases to give friendly names to frequently-used networks.</p>
        </div>
      ) : (
        <table className="px-network-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>CIDR</th>
              <th>Comment</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {aliases.map((a) => (
              <tr key={a.name}>
                <td className="px-mono">{a.name}</td>
                <td className="px-mono">{a.cidr}</td>
                <td>{a.comment || '—'}</td>
                <td>
                  <button type="button" className="px-btn px-btn-quiet px-btn-danger-text" onClick={() => setDeleting(a.name)}>
                    Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {deleting && (
        <motion.div className="px-confirm-overlay" initial={{ opacity: 0 }} animate={{ opacity: 1 }} onClick={() => setDeleting(null)}>
          <motion.div className="px-confirm-dialog" initial={{ scale: 0.95 }} animate={{ scale: 1 }} onClick={(e) => e.stopPropagation()}>
            <h3>Delete alias?</h3>
            <p>Delete alias <code>{deleting}</code>? Rules that reference it will need updating.</p>
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