/**
 * Tier 14 Phase 14.9 — Users list (per Proxmox host).
 */
import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { api } from '../../lib/api';
import ProxmoxShell from './ProxmoxShell';
import ProxmoxHostSelector from './ProxmoxHostSelector';
import { readString } from '../../lib/proxmox';

interface User {
  userid: string;
  first_name?: string;
  last_name?: string;
  email?: string;
  comment?: string;
  enable?: number;
  expire?: number;
  tokens?: unknown[];
  groups?: string[];
  role?: string;
}

const ROLES = ['Administrator', 'PVEAdmin', 'PVEVMAdmin', 'PVEAuditor', 'PVEDatacenterAdmin', 'PVEVMUser'];

export default function ProxmoxUsers() {
  const [hostId, setHostId] = useState('');
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [creating, setCreating] = useState(false);
  const [editing, setEditing] = useState<User | null>(null);
  const [deleting, setDeleting] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!hostId) return;
    setLoading(true);
    setError('');
    try {
      const data = await api<{ users?: User[] } | User[]>('GET', `/api/v1/proxmox/hosts/${hostId}/access/users`);
      setUsers(Array.isArray(data) ? data : data.users ?? []);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to load users.');
    } finally {
      setLoading(false);
    }
  }, [hostId]);

  useEffect(() => { void load(); }, [load]);

  async function doDelete() {
    if (!deleting) return;
    try {
      await api('DELETE', `/api/v1/proxmox/hosts/${hostId}/access/users/${encodeURIComponent(deleting)}`);
      setDeleting(null);
      await load();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Delete failed.');
    }
  }

  return (
    <motion.div className="px-vm-page" initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
      <ProxmoxHostSelector selected={hostId} onChange={setHostId} />
      {!hostId ? null : (
        <>
          <h1 className="px-detail-name">Users</h1>
          {error && <div className="px-error">⚠ {error}</div>}
          <div className="px-tab-toolbar">
            <button type="button" className="px-btn" onClick={() => setCreating(true)} disabled={loading}>
              + Create user
            </button>
          </div>
          {loading ? (
            <p className="px-muted">Loading users…</p>
          ) : users.length === 0 ? (
            <div className="px-info-card">
              <h3>No users on this host</h3>
              <p>Add your first Proxmox user.</p>
            </div>
          ) : (
            <table className="px-network-table">
              <thead>
                <tr>
                  <th>User ID</th>
                  <th>Name</th>
                  <th>Email</th>
                  <th>Role</th>
                  <th>Enabled</th>
                  <th>Expires</th>
                  <th>Tokens</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {users.map((u) => (
                  <tr key={u.userid}>
                    <td className="px-mono">
                      <Link to={`/proxmox/users/${encodeURIComponent(u.userid)}/tokens`}>
                        {u.userid}
                      </Link>
                    </td>
                    <td>{[u.first_name, u.last_name].filter(Boolean).join(' ') || '—'}</td>
                    <td>{u.email || '—'}</td>
                    <td>{readString(u.role) || '—'}</td>
                    <td>
                      <span className={`px-action-badge px-action-${u.enable === 0 ? 'drop' : 'accept'}`}>
                        {u.enable === 0 ? 'disabled' : 'enabled'}
                      </span>
                    </td>
                    <td className="px-mono">{u.expire && u.expire > 0 ? new Date(u.expire * 1000).toLocaleDateString() : '—'}</td>
                    <td className="px-mono">{u.tokens?.length ?? 0}</td>
                    <td>
                      <button type="button" className="px-btn px-btn-quiet" onClick={() => setEditing(u)}>Edit</button>
                      <button type="button" className="px-btn px-btn-quiet px-btn-danger-text" onClick={() => setDeleting(u.userid)}>Delete</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}

          {(creating || editing) && (
            <UserDialog
              hostId={hostId}
              user={editing}
              onClose={() => { setCreating(false); setEditing(null); }}
              onSaved={() => { setCreating(false); setEditing(null); void load(); }}
            />
          )}

          {deleting && (
            <ConfirmDelete
              name={deleting}
              onClose={() => setDeleting(null)}
              onConfirm={doDelete}
            />
          )}
        </>
      )}
    </motion.div>
  );
}

function UserDialog({ hostId, user, onClose, onSaved }: { hostId: string; user: User | null; onClose: () => void; onSaved: () => void }) {
  const [form, setForm] = useState({
    userid: user?.userid ?? '',
    password: '',
    first_name: user?.first_name ?? '',
    last_name: user?.last_name ?? '',
    email: user?.email ?? '',
    enable: user?.enable !== 0 ? 1 : 0,
    role: readString(user?.role) || 'PVEVMAdmin',
  });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  async function save() {
    if (!form.userid) { setError('User ID required.'); return; }
    if (!user && !form.password) { setError('Password required for new user.'); return; }
    setBusy(true); setError('');
    try {
      if (user) {
        // Update (no password)
        await api('PUT', `/api/v1/proxmox/hosts/${hostId}/access/users/${encodeURIComponent(user.userid)}`, {
          first_name: form.first_name, last_name: form.last_name, email: form.email, enable: form.enable,
        });
      } else {
        await api('POST', `/api/v1/proxmox/hosts/${hostId}/access/users`, form);
      }
      onSaved();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Save failed.');
    } finally {
      setBusy(false);
    }
  }

  return (
    <motion.div className="px-confirm-overlay" initial={{ opacity: 0 }} animate={{ opacity: 1 }} onClick={onClose}>
      <motion.div className="px-confirm-dialog px-form-wide" initial={{ scale: 0.95 }} animate={{ scale: 1 }} onClick={(e) => e.stopPropagation()}>
        <h3>{user ? 'Edit user' : 'Create user'}</h3>
        <div className="px-form-grid">
          <div className="px-form-field">
            <label>User ID *</label>
            <input type="text" value={form.userid} onChange={(e) => setForm({ ...form, userid: e.target.value })} disabled={!!user} className="px-confirm-input" />
          </div>
          {!user && (
            <div className="px-form-field">
              <label>Password *</label>
              <input type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} className="px-confirm-input" />
            </div>
          )}
          <div className="px-form-field">
            <label>First name</label>
            <input type="text" value={form.first_name} onChange={(e) => setForm({ ...form, first_name: e.target.value })} className="px-confirm-input" />
          </div>
          <div className="px-form-field">
            <label>Last name</label>
            <input type="text" value={form.last_name} onChange={(e) => setForm({ ...form, last_name: e.target.value })} className="px-confirm-input" />
          </div>
          <div className="px-form-field px-form-full">
            <label>Email</label>
            <input type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} className="px-confirm-input" />
          </div>
          <div className="px-form-field">
            <label>Role</label>
            <select value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })} className="px-form-select">
              {ROLES.map((r) => <option key={r} value={r}>{r}</option>)}
            </select>
          </div>
          <div className="px-form-field px-form-toggle">
            <label>
              <input type="checkbox" checked={form.enable === 1} onChange={(e) => setForm({ ...form, enable: e.target.checked ? 1 : 0 })} />
              Enabled
            </label>
          </div>
        </div>
        {error && <div className="px-confirm-warn">{error}</div>}
        <div className="px-confirm-actions">
          <button type="button" className="px-btn px-btn-quiet" onClick={onClose} disabled={busy}>Cancel</button>
          <button type="button" className="px-btn" onClick={save} disabled={busy}>{busy ? 'Saving…' : 'Save'}</button>
        </div>
      </motion.div>
    </motion.div>
  );
}

function ConfirmDelete({ name, onClose, onConfirm }: { name: string; onClose: () => void; onConfirm: () => void }) {
  return (
    <ProxmoxShell title="Access & Tokens" subtitle="Manage Proxmox users and API tokens">
      <motion.div className="px-confirm-overlay" initial={{ opacity: 0 }} animate={{ opacity: 1 }} onClick={onClose}>
      <motion.div className="px-confirm-dialog" initial={{ scale: 0.95 }} animate={{ scale: 1 }} onClick={(e) => e.stopPropagation()}>
        <h3>Delete user?</h3>
        <p>This will permanently delete user <code>{name}</code> and revoke all their tokens.</p>
        <div className="px-confirm-actions">
          <button type="button" className="px-btn px-btn-quiet" onClick={onClose}>Cancel</button>
          <button type="button" className="px-btn px-btn-danger" onClick={onConfirm}>Yes, delete</button>
        </div>
      </motion.div>
    </motion.div>
    </ProxmoxShell>
  );
}