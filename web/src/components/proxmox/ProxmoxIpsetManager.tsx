/**
 * Tier 14 Phase 14.8 — IPset manager (datacenter-scoped).
 * Lists IPSets + their CIDR entries. Create/delete IPset + entries.
 */
import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { api } from '../../lib/api';

interface IPset {
  name: string;
  comment?: string;
  entries?: Array<{ cidr: string; nomatch?: boolean; comment?: string }>;
}

export default function ProxmoxIpsetManager() {
  const [ipsets, setIpsets] = useState<IPset[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [creating, setCreating] = useState(false);
  const [newName, setNewName] = useState('');
  const [newComment, setNewComment] = useState('');
  const [expanded, setExpanded] = useState<string | null>(null);
  const [addingCidrTo, setAddingCidrTo] = useState<string | null>(null);
  const [newCidr, setNewCidr] = useState('');
  const [deleting, setDeleting] = useState<{ name: string; cidr?: string } | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await api<{ ipsets?: IPset[] } | IPset[]>('GET', '/api/v1/proxmox/ipsets');
      const list = Array.isArray(data) ? data : data.ipsets ?? [];
      setIpsets(list);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to load IPSets.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  async function create() {
    if (!newName) return;
    try {
      await api('POST', '/api/v1/proxmox/ipsets', { name: newName, comment: newComment });
      setCreating(false);
      setNewName('');
      setNewComment('');
      await load();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Create failed.');
    }
  }

  async function doDelete() {
    if (!deleting) return;
    try {
      if (deleting.cidr) {
        await api('DELETE', `/api/v1/proxmox/ipsets/${encodeURIComponent(deleting.name)}/${encodeURIComponent(deleting.cidr)}`);
      } else {
        await api('DELETE', `/api/v1/proxmox/ipsets/${encodeURIComponent(deleting.name)}`);
      }
      setDeleting(null);
      await load();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Delete failed.');
    }
  }

  async function addCidr(name: string) {
    if (!newCidr) return;
    try {
      await api('POST', `/api/v1/proxmox/ipsets/${encodeURIComponent(name)}`, { cidr: newCidr });
      setAddingCidrTo(null);
      setNewCidr('');
      await load();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Add CIDR failed.');
    }
  }

  if (loading) {
    return <div className="px-vm-page"><p className="px-muted">Loading IPSets…</p></div>;
  }

  return (
    <motion.div className="px-vm-page" initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
      <div className="px-detail-breadcrumb">
        <Link to="/proxmox">Proxmox</Link>
        <span className="px-detail-sep">›</span>
        <span>IPSets</span>
      </div>
      <h1 className="px-detail-name">IPSets</h1>
      <div className="px-detail-meta">
        <span><strong>Total</strong> {ipsets.length}</span>
      </div>

      {error && <div className="px-error">⚠ {error}</div>}

      <div className="px-tab-toolbar">
        <button type="button" className="px-btn" onClick={() => setCreating(true)}>
          + Create IPset
        </button>
      </div>

      {creating && (
        <motion.div className="px-confirm-overlay" initial={{ opacity: 0 }} animate={{ opacity: 1 }} onClick={() => setCreating(false)}>
          <motion.div className="px-confirm-dialog" initial={{ scale: 0.95 }} animate={{ scale: 1 }} onClick={(e) => e.stopPropagation()}>
            <h3>Create IPset</h3>
            <div className="px-form-field">
              <label>Name *</label>
              <input type="text" value={newName} onChange={(e) => setNewName(e.target.value)} placeholder="office-net" className="px-confirm-input" autoFocus />
            </div>
            <div className="px-form-field">
              <label>Comment</label>
              <input type="text" value={newComment} onChange={(e) => setNewComment(e.target.value)} placeholder="optional" className="px-confirm-input" />
            </div>
            <div className="px-confirm-actions">
              <button type="button" className="px-btn px-btn-quiet" onClick={() => setCreating(false)}>Cancel</button>
              <button type="button" className="px-btn" onClick={create}>Create</button>
            </div>
          </motion.div>
        </motion.div>
      )}

      {ipsets.length === 0 ? (
        <div className="px-info-card">
          <h3>No IPSets</h3>
          <p>Create an IPset to reuse groups of CIDRs across multiple firewall rules.</p>
        </div>
      ) : (
        <table className="px-network-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Comment</th>
              <th># CIDRs</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {ipsets.map((s) => {
              const isOpen = expanded === s.name;
              return (
                <>
                  <tr key={s.name}>
                    <td className="px-mono">
                      <button
                        type="button"
                        className="px-btn px-btn-quiet"
                        onClick={() => setExpanded(isOpen ? null : s.name)}
                        style={{ padding: 0, fontSize: 13 }}
                      >
                        {isOpen ? '▾' : '▸'} {s.name}
                      </button>
                    </td>
                    <td>{s.comment || '—'}</td>
                    <td className="px-mono">{s.entries?.length ?? 0}</td>
                    <td>
                      <button type="button" className="px-btn px-btn-quiet" onClick={() => setAddingCidrTo(s.name)}>
                        + CIDR
                      </button>
                      <button type="button" className="px-btn px-btn-quiet px-btn-danger-text" onClick={() => setDeleting({ name: s.name })}>
                        Delete
                      </button>
                    </td>
                  </tr>
                  {isOpen && (
                    <tr key={`${s.name}-entries`}>
                      <td colSpan={4} style={{ background: 'var(--px-surface-2)', padding: '12px 16px' }}>
                        {addingCidrTo === s.name && (
                          <div style={{ display: 'flex', gap: 8, marginBottom: 8 }}>
                            <input
                              type="text"
                              value={newCidr}
                              onChange={(e) => setNewCidr(e.target.value)}
                              placeholder="192.168.0.0/24"
                              className="px-confirm-input"
                              style={{ flex: 1 }}
                              autoFocus
                            />
                            <button type="button" className="px-btn" onClick={() => addCidr(s.name)}>Add</button>
                            <button type="button" className="px-btn px-btn-quiet" onClick={() => setAddingCidrTo(null)}>Cancel</button>
                          </div>
                        )}
                        {s.entries && s.entries.length > 0 ? (
                          <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
                            {s.entries.map((e) => (
                              <li key={`${s.name}-${e.cidr}`} style={{ display: 'flex', justifyContent: 'space-between', padding: '4px 0' }}>
                                <span className="px-mono">{e.cidr}{e.nomatch ? ' (nomatch)' : ''}</span>
                                <button
                                  type="button"
                                  className="px-btn px-btn-quiet px-btn-danger-text"
                                  onClick={() => setDeleting({ name: s.name, cidr: e.cidr })}
                                >
                                  ✕
                                </button>
                              </li>
                            ))}
                          </ul>
                        ) : (
                          <p className="px-muted">No entries.</p>
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

      {deleting && (
        <motion.div className="px-confirm-overlay" initial={{ opacity: 0 }} animate={{ opacity: 1 }} onClick={() => setDeleting(null)}>
          <motion.div className="px-confirm-dialog" initial={{ scale: 0.95 }} animate={{ scale: 1 }} onClick={(e) => e.stopPropagation()}>
            <h3>Delete {deleting.cidr ? 'CIDR' : 'IPset'}?</h3>
            <p>
              {deleting.cidr
                ? <>Remove <code>{deleting.cidr}</code> from <code>{deleting.name}</code>?</>
                : <>Delete IPset <code>{deleting.name}</code> and all its entries?</>}
            </p>
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