/**
 * Tier 14 Phase 14.8 — Node firewall (CRUD on node-level rules).
 * Reuses ProxmoxFirewallRuleDialog from Phase 14.3.
 */
import { useCallback, useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { motion } from 'framer-motion';
import { api } from '../../lib/api';
import ProxmoxFirewallRuleDialog, { type FirewallRule } from './ProxmoxFirewallRuleDialog';

export default function ProxmoxNodeFirewall() {
  const { hostId = '', node = '' } = useParams();
  const [rules, setRules] = useState<FirewallRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [editing, setEditing] = useState<FirewallRule | null>(null);
  const [adding, setAdding] = useState(false);
  const [deleting, setDeleting] = useState<FirewallRule | null>(null);

  const load = useCallback(async () => {
    if (!hostId || !node) return;
    setLoading(true);
    setError('');
    try {
      const data = await api<{ rules?: FirewallRule[] }>(
        'GET',
        `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/firewall/rules`,
      );
      setRules(data.rules ?? []);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to load rules.');
    } finally {
      setLoading(false);
    }
  }, [hostId, node]);

  useEffect(() => { void load(); }, [load]);

  const save = useCallback(async (rule: FirewallRule) => {
    if (editing) {
      await api(
        'PUT',
        `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/firewall/rules/${rule.pos}`,
        rule,
      );
    } else {
      await api(
        'POST',
        `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/firewall/rules`,
        rule,
      );
    }
    setEditing(null);
    setAdding(false);
    await load();
  }, [editing, hostId, node, load]);

  const doDelete = useCallback(async () => {
    if (!deleting || deleting.pos === undefined) return;
    try {
      await api(
        'DELETE',
        `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/firewall/rules/${deleting.pos}`,
      );
      setDeleting(null);
      await load();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Delete failed.');
    }
  }, [deleting, hostId, node, load]);

  if (loading) {
    return <div className="px-vm-page"><p className="px-muted">Loading node firewall…</p></div>;
  }

  return (
    <motion.div className="px-vm-page" initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
      <div className="px-detail-breadcrumb">
        <Link to="/proxmox">Proxmox</Link>
        <span className="px-detail-sep">›</span>
        <Link to={`/proxmox/nodes/${hostId}/${encodeURIComponent(node)}`}>{node}</Link>
        <span className="px-detail-sep">›</span>
        <span>Firewall</span>
      </div>
      <h1 className="px-detail-name">Node firewall</h1>

      {error && <div className="px-error">⚠ {error}</div>}

      <div className="px-tab-toolbar">
        <button type="button" className="px-btn" onClick={() => setAdding(true)}>
          + Add rule
        </button>
      </div>

      {rules.length === 0 ? (
        <div className="px-info-card">
          <h3>No firewall rules</h3>
          <p>Node-level firewall rules from Proxmox will appear here.</p>
        </div>
      ) : (
        <table className="px-network-table">
          <thead>
            <tr>
              <th>Pos</th>
              <th>Action</th>
              <th>Type</th>
              <th>Source</th>
              <th>Dest</th>
              <th>Proto</th>
              <th>Dport</th>
              <th>Comment</th>
              <th>On</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {rules.map((r) => (
              <tr key={r.pos}>
                <td className="px-mono">{r.pos}</td>
                <td>
                  <span className={`px-action-badge px-action-${(r.action || 'accept').toLowerCase()}`}>
                    {r.action}
                  </span>
                </td>
                <td>{r.type || 'in'}</td>
                <td className="px-mono">{r.source || 'any'}</td>
                <td className="px-mono">{r.dest || 'any'}</td>
                <td>{r.proto || '—'}</td>
                <td className="px-mono">{r.dport || '—'}</td>
                <td>{r.comment || '—'}</td>
                <td>{r.enable === 0 ? 'off' : 'on'}</td>
                <td>
                  <button type="button" className="px-btn px-btn-quiet" onClick={() => setEditing(r)}>Edit</button>
                  <button type="button" className="px-btn px-btn-quiet px-btn-danger-text" onClick={() => setDeleting(r)}>Delete</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <ProxmoxFirewallRuleDialog
        open={adding || !!editing}
        initial={editing ?? undefined}
        onClose={() => { setAdding(false); setEditing(null); }}
        onSave={save}
      />

      {deleting && (
        <motion.div className="px-confirm-overlay" initial={{ opacity: 0 }} animate={{ opacity: 1 }} onClick={() => setDeleting(null)}>
          <motion.div className="px-confirm-dialog" initial={{ scale: 0.95 }} animate={{ scale: 1 }} onClick={(e) => e.stopPropagation()}>
            <h3>Delete rule #{deleting.pos}?</h3>
            <p>This will permanently remove firewall rule <code>#{deleting.pos}</code>.</p>
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