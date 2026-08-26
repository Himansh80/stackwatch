/**
 * Tier 14 Phase 14.9 — API tokens for a specific user.
 */
import { useCallback, useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { motion } from 'framer-motion';
import { api } from '../../lib/api';
import ProxmoxShell from './ProxmoxShell';
import ProxmoxHostSelector from './ProxmoxHostSelector';
import { readString } from '../../lib/proxmox';

interface Token {
  tokenid: string;
  comment?: string;
  expire?: number;
  privsep?: number;
}

export default function ProxmoxUserTokens() {
  const { userid = '' } = useParams();
  const [hostId, setHostId] = useState('');
  const [tokens, setTokens] = useState<Token[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [creating, setCreating] = useState(false);
  const [issuedToken, setIssuedToken] = useState<{ tokenid: string; value: string } | null>(null);
  const [deleting, setDeleting] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!hostId || !userid) return;
    setLoading(true);
    setError('');
    try {
      const data = await api<Token[] | { tokens?: Token[] }>(
        'GET',
        `/api/v1/proxmox/hosts/${hostId}/access/users/${encodeURIComponent(userid)}/token`,
      );
      setTokens(Array.isArray(data) ? data : data.tokens ?? []);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to load tokens.');
    } finally {
      setLoading(false);
    }
  }, [hostId, userid]);

  useEffect(() => { void load(); }, [load]);

  async function doDelete() {
    if (!deleting) return;
    try {
      await api('DELETE', `/api/v1/proxmox/hosts/${hostId}/access/users/${encodeURIComponent(userid)}/token/${encodeURIComponent(deleting)}`);
      setDeleting(null);
      await load();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Delete failed.');
    }
  }

  return (
    <motion.div className="px-vm-page" initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
      <div className="px-detail-breadcrumb">
        <Link to="/proxmox">Proxmox</Link>
        <span className="px-detail-sep">›</span>
        <Link to="/proxmox/users">Users</Link>
        <span className="px-detail-sep">›</span>
        <span>{userid}</span>
      </div>
      <h1 className="px-detail-name">API tokens for {userid}</h1>

      <ProxmoxHostSelector selected={hostId} onChange={setHostId} />

      {!hostId ? null : (
        <>
          {error && <div className="px-error">⚠ {error}</div>}

          <div className="px-tab-toolbar">
            <button type="button" className="px-btn" onClick={() => setCreating(true)} disabled={loading}>
              + Create token
            </button>
          </div>

          {loading ? (
            <p className="px-muted">Loading tokens…</p>
          ) : tokens.length === 0 ? (
            <div className="px-info-card">
              <h3>No API tokens</h3>
              <p>Create a token to allow API access without sharing the password.</p>
            </div>
          ) : (
            <table className="px-network-table">
              <thead>
                <tr>
                  <th>Token ID</th>
                  <th>Comment</th>
                  <th>Expires</th>
                  <th>Privilege separation</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {tokens.map((t) => (
                  <tr key={t.tokenid}>
                    <td className="px-mono">{t.tokenid}</td>
                    <td>{t.comment || '—'}</td>
                    <td className="px-mono">{t.expire && t.expire > 0 ? new Date(t.expire * 1000).toLocaleDateString() : 'never'}</td>
                    <td>{t.privsep === 1 ? 'yes' : 'no'}</td>
                    <td>
                      <button type="button" className="px-btn px-btn-quiet px-btn-danger-text" onClick={() => setDeleting(t.tokenid)}>
                        Revoke
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}

          {creating && (
            <CreateTokenDialog
              hostId={hostId}
              userid={userid}
              onClose={() => setCreating(false)}
              onIssued={(tk) => {
                setCreating(false);
                setIssuedToken(tk);
                void load();
              }}
            />
          )}

          {issuedToken && (
            <motion.div className="px-confirm-overlay" initial={{ opacity: 0 }} animate={{ opacity: 1 }} onClick={() => setIssuedToken(null)}>
              <motion.div className="px-confirm-dialog" initial={{ scale: 0.95 }} animate={{ scale: 1 }} onClick={(e) => e.stopPropagation()}>
                <h3>Token created</h3>
                <p>Copy this value now. <strong>You will not see it again.</strong></p>
                <code style={{ display: 'block', padding: 12, background: 'var(--px-bg)', borderRadius: 4, fontSize: 12, wordBreak: 'break-all' }}>
                  {issuedToken.value}
                </code>
                <div className="px-confirm-actions">
                  <button type="button" className="px-btn" onClick={() => navigator.clipboard?.writeText(issuedToken.value)}>Copy</button>
                  <button type="button" className="px-btn px-btn-quiet" onClick={() => setIssuedToken(null)}>Done</button>
                </div>
              </motion.div>
            </motion.div>
          )}

          {deleting && (
            <motion.div className="px-confirm-overlay" initial={{ opacity: 0 }} animate={{ opacity: 1 }} onClick={() => setDeleting(null)}>
              <motion.div className="px-confirm-dialog" initial={{ scale: 0.95 }} animate={{ scale: 1 }} onClick={(e) => e.stopPropagation()}>
                <h3>Revoke token?</h3>
                <p>Revoke <code>{deleting}</code>? Any service using this token will stop working.</p>
                <div className="px-confirm-actions">
                  <button type="button" className="px-btn px-btn-quiet" onClick={() => setDeleting(null)}>Cancel</button>
                  <button type="button" className="px-btn px-btn-danger" onClick={doDelete}>Yes, revoke</button>
                </div>
              </motion.div>
            </motion.div>
          )}
        </>
      )}
    </motion.div>
  );
}

function CreateTokenDialog({ hostId, userid, onClose, onIssued }: { hostId: string; userid: string; onClose: () => void; onIssued: (tk: { tokenid: string; value: string }) => void }) {
  const [form, setForm] = useState({ tokenid: '', comment: '', expire: 0, privsep: 0 });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  async function create() {
    if (!form.tokenid) { setError('Token ID required.'); return; }
    setBusy(true); setError('');
    try {
      const resp = await api<{ value?: string } | { tokenid: string; value: string }>(
        'POST',
        `/api/v1/proxmox/hosts/${hostId}/access/users/${encodeURIComponent(userid)}/token`,
        { tokenid: form.tokenid, comment: form.comment, expire: form.expire, privsep: form.privsep },
      );
      onIssued({ tokenid: form.tokenid, value: readString((resp as { value?: string }).value) || '(no value returned)' });
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Create failed.');
      setBusy(false);
    }
  }

  return (
    <ProxmoxShell title="User tokens" subtitle="API tokens for this user">
      <motion.div className="px-confirm-overlay" initial={{ opacity: 0 }} animate={{ opacity: 1 }} onClick={onClose}>
      <motion.div className="px-confirm-dialog" initial={{ scale: 0.95 }} animate={{ scale: 1 }} onClick={(e) => e.stopPropagation()}>
        <h3>Create API token</h3>
        <div className="px-form-field">
          <label>Token ID *</label>
          <input type="text" value={form.tokenid} onChange={(e) => setForm({ ...form, tokenid: e.target.value })} placeholder="mytoken" className="px-confirm-input" autoFocus />
        </div>
        <div className="px-form-field">
          <label>Comment</label>
          <input type="text" value={form.comment} onChange={(e) => setForm({ ...form, comment: e.target.value })} placeholder="optional" className="px-confirm-input" />
        </div>
        <div className="px-form-field">
          <label>Expire (epoch seconds, 0 = never)</label>
          <input type="number" min="0" value={form.expire} onChange={(e) => setForm({ ...form, expire: Number(e.target.value) })} className="px-confirm-input" />
        </div>
        <div className="px-form-field px-form-toggle">
          <label>
            <input type="checkbox" checked={form.privsep === 1} onChange={(e) => setForm({ ...form, privsep: e.target.checked ? 1 : 0 })} />
            Privilege separation (no permissions)
          </label>
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