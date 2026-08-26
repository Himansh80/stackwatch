/**
 * Tier 14 Phase 14.3 — Snapshots tab content (real CRUD).
 */
import { useCallback, useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { api } from '../../lib/api';
import ProxmoxSnapshotCreateDialog from './ProxmoxSnapshotCreateDialog';

export interface Snapshot {
  name: string;
  date?: string;
  vmstate?: boolean;
  size?: number;
  description?: string;
}

interface Props {
  hostId: string;
  node: string;
  vmid: number;
}

export default function ProxmoxDetailSnapshots({ hostId, node, vmid }: Props) {
  const [snapshots, setSnapshots] = useState<Snapshot[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showCreate, setShowCreate] = useState(false);
  const [confirm, setConfirm] = useState<{ kind: 'delete' | 'rollback'; name: string } | null>(null);

  const load = useCallback(async () => {
    if (!hostId || !node || !vmid) return;
    setLoading(true);
    setError('');
    try {
      const data = await api<{ snapshots?: Snapshot[] }>(
        'GET',
        `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/qemu/${vmid}/snapshot`,
      );
      setSnapshots(data.snapshots ?? []);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to load snapshots.');
    } finally {
      setLoading(false);
    }
  }, [hostId, node, vmid]);

  useEffect(() => {
    void load();
  }, [load]);

  const create = useCallback(async (name: string, description: string, vmState: boolean) => {
    await api(
      'POST',
      `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/qemu/${vmid}/snapshot`,
      { snapname: name, description, vmstate: vmState },
    );
    await load();
  }, [hostId, node, vmid, load]);

  const execute = useCallback(async () => {
    if (!confirm) return;
    const { kind, name } = confirm;
    setError('');
    try {
      if (kind === 'delete') {
        await api(
          'DELETE',
          `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/qemu/${vmid}/snapshot?snapname=${encodeURIComponent(name)}`,
        );
      } else {
        await api(
          'POST',
          `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/qemu/${vmid}/snapshot/${encodeURIComponent(name)}/rollback`,
        );
      }
      setConfirm(null);
      await load();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : `${kind} failed.`);
    }
  }, [confirm, hostId, node, vmid, load]);

  if (loading) {
    return <div className="px-tab-pane"><p className="px-muted">Loading snapshots…</p></div>;
  }

  return (
    <motion.div
      className="px-tab-pane"
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
    >
      {error && <div className="px-error">⚠ {error}</div>}

      <div className="px-tab-toolbar">
        <button type="button" className="px-btn" onClick={() => setShowCreate(true)}>
          + Create snapshot
        </button>
      </div>

      {snapshots.length === 0 ? (
        <div className="px-info-card">
          <h3>No snapshots yet</h3>
          <p>Create a snapshot to capture the current VM state (RAM + disk) and restore later.</p>
        </div>
      ) : (
        <table className="px-network-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Date</th>
              <th>VM State</th>
              <th>Size</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {snapshots.map((s) => (
              <tr key={s.name}>
                <td className="px-mono">{s.name}</td>
                <td className="px-mono">{s.date || '—'}</td>
                <td>{s.vmstate ? '✓ included' : '—'}</td>
                <td className="px-mono">{s.size ? `${(s.size / 1024 ** 3).toFixed(1)} GB` : '—'}</td>
                <td>
                  <button type="button" className="px-btn px-btn-quiet" onClick={() => setConfirm({ kind: 'rollback', name: s.name })}>
                    Rollback
                  </button>
                  <button type="button" className="px-btn px-btn-quiet px-btn-danger-text" onClick={() => setConfirm({ kind: 'delete', name: s.name })}>
                    Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <ProxmoxSnapshotCreateDialog
        open={showCreate}
        onClose={() => setShowCreate(false)}
        onCreate={create}
      />

      {confirm && (
        <motion.div className="px-confirm-overlay" initial={{ opacity: 0 }} animate={{ opacity: 1 }} onClick={() => setConfirm(null)}>
          <motion.div className="px-confirm-dialog" initial={{ scale: 0.95 }} animate={{ scale: 1 }} onClick={(e) => e.stopPropagation()}>
            <h3>{confirm.kind === 'delete' ? 'Delete snapshot?' : 'Rollback to snapshot?'}</h3>
            <p>
              {confirm.kind === 'delete'
                ? <>This will permanently delete snapshot <code>{confirm.name}</code>.</>
                : <>This will revert the VM to <code>{confirm.name}</code>. Current state will be lost.</>}
            </p>
            <div className="px-confirm-actions">
              <button type="button" className="px-btn px-btn-quiet" onClick={() => setConfirm(null)}>Cancel</button>
              <button type="button" className={`px-btn ${confirm.kind === 'delete' ? 'px-btn-danger' : ''}`} onClick={execute}>
                Yes, {confirm.kind}
              </button>
            </div>
          </motion.div>
        </motion.div>
      )}
    </motion.div>
  );
}