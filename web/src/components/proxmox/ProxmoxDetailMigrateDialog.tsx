/**
 * Tier 14 Phase 14.3 — Migrate dialog (node picker + online toggle).
 */
import { useEffect, useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { api } from '../../lib/api';

interface ProxmoxNode {
  node: string;
  status: string;
}

interface Props {
  open: boolean;
  hostId: string;
  vmid: number;
  currentNode: string;
  running: boolean;
  onClose: () => void;
  onMigrate: (target: string, online: boolean, withLocalStorage: boolean) => Promise<void>;
}

export default function ProxmoxDetailMigrateDialog({
  open, hostId, vmid, currentNode, running, onClose, onMigrate,
}: Props) {
  const [nodes, setNodes] = useState<ProxmoxNode[]>([]);
  const [target, setTarget] = useState('');
  const [online, setOnline] = useState(true);
  const [withLocalStorage, setWithLocalStorage] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!open) return;
    setError('');
    setTarget('');
    setOnline(true);
    setLoading(true);
    api<{ nodes?: ProxmoxNode[] } | ProxmoxNode[]>(
      'GET',
      `/api/v1/proxmox/hosts/${hostId}/nodes`,
    )
      .then((data) => {
        const list = Array.isArray(data) ? data : data.nodes ?? [];
        setNodes(list.filter((n) => n.node !== currentNode));
      })
      .catch((cause) => setError(cause instanceof Error ? cause.message : 'Unable to load nodes.'))
      .finally(() => setLoading(false));
  }, [open, hostId, currentNode]);

  const targets = nodes.filter((n) => n.node !== currentNode);

  async function handle() {
    if (!target) {
      setError('Pick a target node.');
      return;
    }
    if (online && !running) {
      setError('Online migration requires the VM to be running.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      await onMigrate(target, online, withLocalStorage);
      onClose();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Migration failed.');
      setBusy(false);
    }
  }

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          className="px-confirm-overlay"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          onClick={onClose}
        >
          <motion.div
            className="px-confirm-dialog"
            initial={{ scale: 0.95, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            exit={{ scale: 0.95, opacity: 0 }}
            onClick={(e) => e.stopPropagation()}
            role="dialog"
          >
            <h3>Migrate VM {vmid}</h3>
            <p>Move this VM to another node in the cluster.</p>

            {loading && <p className="px-muted">Loading nodes…</p>}
            {!loading && targets.length === 0 && (
              <p className="px-confirm-warn">No other nodes in this cluster.</p>
            )}

            {targets.length > 0 && (
              <div className="px-form-field">
                <label htmlFor="px-migrate-target">Target node</label>
                <select
                  id="px-migrate-target"
                  value={target}
                  onChange={(e) => setTarget(e.target.value)}
                  className="px-form-select"
                >
                  <option value="">— select —</option>
                  {targets.map((n) => (
                    <option key={n.node} value={n.node}>
                      {n.node} ({n.status})
                    </option>
                  ))}
                </select>
              </div>
            )}

            <div className="px-form-field px-form-toggle">
              <label htmlFor="px-migrate-online">
                <input
                  id="px-migrate-online"
                  type="checkbox"
                  checked={online}
                  onChange={(e) => setOnline(e.target.checked)}
                />
                Online (live migration, no downtime)
              </label>
              {!running && online && <small className="px-confirm-warn">VM must be running for online migration.</small>}
            </div>

            <div className="px-form-field px-form-toggle">
              <label htmlFor="px-migrate-storage">
                <input
                  id="px-migrate-storage"
                  type="checkbox"
                  checked={withLocalStorage}
                  onChange={(e) => setWithLocalStorage(e.target.checked)}
                />
                Move local storage (slower, copies disks)
              </label>
            </div>

            {error && <div className="px-confirm-warn">{error}</div>}

            <div className="px-confirm-actions">
              <button type="button" className="px-btn px-btn-quiet" onClick={onClose} disabled={busy}>
                Cancel
              </button>
              <button
                type="button"
                className="px-btn"
                onClick={handle}
                disabled={busy || targets.length === 0 || !target}
              >
                {busy ? 'Migrating…' : 'Migrate'}
              </button>
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}