/**
 * Tier 14 Phase 14.9 — Pools (resource grouping).
 */
import { useCallback, useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { api } from '../../lib/api';
import ProxmoxShell from './ProxmoxShell';
import ProxmoxHostSelector from './ProxmoxHostSelector';

interface Pool {
  poolid: string;
  comment?: string;
  members?: Array<{ id: string; node: string; type: 'qemu' | 'lxc' | 'storage' }>;
}

export default function ProxmoxPools() {
  const [hostId, setHostId] = useState('');
  const [pools, setPools] = useState<Pool[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [creating, setCreating] = useState(false);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!hostId) return;
    setLoading(true);
    setError('');
    try {
      const data = await api<{ pools?: Pool[] } | Pool[]>('GET', `/api/v1/proxmox/hosts/${hostId}/pools`);
      setPools(Array.isArray(data) ? data : data.pools ?? []);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to load pools.');
    } finally {
      setLoading(false);
    }
  }, [hostId]);

  useEffect(() => { void load(); }, [load]);

  async function doDelete() {
    if (!deleting) return;
    try {
      await api('DELETE', `/api/v1/proxmox/hosts/${hostId}/pools/${encodeURIComponent(deleting)}`);
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
          <h1 className="px-detail-name">Pools</h1>
          {error && <div className="px-error">⚠ {error}</div>}
          <div className="px-tab-toolbar">
            <button type="button" className="px-btn" onClick={() => setCreating(true)} disabled={loading}>
              + Create pool
            </button>
          </div>
          {loading ? (
            <p className="px-muted">Loading pools…</p>
          ) : pools.length === 0 ? (
            <div className="px-info-card">
              <h3>No pools</h3>
              <p>Pools group VMs and storage for resource management and permissions.</p>
            </div>
          ) : (
            <table className="px-network-table">
              <thead>
                <tr>
                  <th>Pool ID</th>
                  <th>Comment</th>
                  <th>Members</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {pools.map((p) => {
                  const isOpen = expanded === p.poolid;
                  return (
                    <>
                      <tr key={p.poolid}>
                        <td className="px-mono">
                          <button
                            type="button"
                            className="px-btn px-btn-quiet"
                            onClick={() => setExpanded(isOpen ? null : p.poolid)}
                            style={{ padding: 0, fontSize: 13 }}
                          >
                            {isOpen ? '▾' : '▸'} {p.poolid}
                          </button>
                        </td>
                        <td>{p.comment || '—'}</td>
                        <td className="px-mono">{p.members?.length ?? 0}</td>
                        <td>
                          <button type="button" className="px-btn px-btn-quiet px-btn-danger-text" onClick={() => setDeleting(p.poolid)}>
                            Delete
                          </button>
                        </td>
                      </tr>
                      {isOpen && (
                        <tr key={`${p.poolid}-members`}>
                          <td colSpan={4} style={{ background: 'var(--px-surface-2)', padding: '12px 16px' }}>
                            {p.members && p.members.length > 0 ? (
                              <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
                                {p.members.map((m) => (
                                  <li key={`${m.node}-${m.id}`} className="px-mono" style={{ padding: '4px 0' }}>
                                    [{m.type}] {m.node}/{m.id}
                                  </li>
                                ))}
                              </ul>
                            ) : (
                              <p className="px-muted">No members. Add members via the Proxmox web UI (Phase 14.10 adds it here).</p>
                            )}
                          </td>
                        </tr>
                      )}
                    </>
                  );
                })}
              </tbody>
            </table>
          )}

          {creating && (
            <CreatePoolDialog
              hostId={hostId}
              onClose={() => setCreating(false)}
              onCreated={() => { setCreating(false); void load(); }}
            />
          )}

          {deleting && (
            <motion.div className="px-confirm-overlay" initial={{ opacity: 0 }} animate={{ opacity: 1 }} onClick={() => setDeleting(null)}>
              <motion.div className="px-confirm-dialog" initial={{ scale: 0.95 }} animate={{ scale: 1 }} onClick={(e) => e.stopPropagation()}>
                <h3>Delete pool?</h3>
                <p>Delete pool <code>{deleting}</code>? Its members are NOT deleted — they just become unpooled.</p>
                <div className="px-confirm-actions">
                  <button type="button" className="px-btn px-btn-quiet" onClick={() => setDeleting(null)}>Cancel</button>
                  <button type="button" className="px-btn px-btn-danger" onClick={doDelete}>Yes, delete</button>
                </div>
              </motion.div>
            </motion.div>
          )}
        </>
      )}
    </motion.div>
  );
}

function CreatePoolDialog({ hostId, onClose, onCreated }: { hostId: string; onClose: () => void; onCreated: () => void }) {
  const [poolid, setPoolid] = useState('');
  const [comment, setComment] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  async function create() {
    if (!poolid) { setError('Pool ID required.'); return; }
    setBusy(true); setError('');
    try {
      await api('POST', `/api/v1/proxmox/hosts/${hostId}/pools`, { poolid, comment });
      onCreated();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Create failed.');
      setBusy(false);
    }
  }

  return (
    <ProxmoxShell title="Pools" subtitle="Resource pools for grouping">
      <motion.div className="px-confirm-overlay" initial={{ opacity: 0 }} animate={{ opacity: 1 }} onClick={onClose}>
      <motion.div className="px-confirm-dialog" initial={{ scale: 0.95 }} animate={{ scale: 1 }} onClick={(e) => e.stopPropagation()}>
        <h3>Create pool</h3>
        <div className="px-form-field">
          <label>Pool ID *</label>
          <input type="text" value={poolid} onChange={(e) => setPoolid(e.target.value)} placeholder="prod" className="px-confirm-input" autoFocus />
        </div>
        <div className="px-form-field">
          <label>Comment</label>
          <input type="text" value={comment} onChange={(e) => setComment(e.target.value)} placeholder="optional" className="px-confirm-input" />
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