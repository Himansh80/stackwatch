/**
 * Tier 14 Phase 14.2 — Snapshots tab.
 * Phase 14.2: list-only. Create/delete/rollback ships in Phase 14.3.
 */
import { motion } from 'framer-motion';

interface Snapshot {
  name: string;
  date: string;
  vmstate?: boolean;
  size?: number;
}

interface Props {
  snapshots?: Snapshot[];
  loading: boolean;
}

export default function ProxmoxDetailSnapshots({ snapshots = [], loading }: Props) {
  if (loading) {
    return <div className="px-tab-pane"><p className="px-muted">Loading snapshots…</p></div>;
  }

  if (snapshots.length === 0) {
    return (
      <motion.div
        className="px-tab-pane"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
      >
        <div className="px-info-card">
          <h3>No snapshots yet</h3>
          <p>Create a snapshot to capture the current VM state (RAM + disk) and restore later.</p>
          <button type="button" className="px-btn px-btn-quiet" disabled title="Ships in Phase 14.3">
            + Create snapshot (coming in v2.3)
          </button>
        </div>
      </motion.div>
    );
  }

  return (
    <motion.div
      className="px-tab-pane"
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2 }}
    >
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
          {snapshots.map((snap) => (
            <tr key={snap.name}>
              <td className="px-mono">{snap.name}</td>
              <td className="px-mono">{snap.date}</td>
              <td>{snap.vmstate ? '✓ included' : '—'}</td>
              <td className="px-mono">{snap.size ? `${(snap.size / 1024 ** 3).toFixed(1)} GB` : '—'}</td>
              <td>
                <button type="button" className="px-btn px-btn-quiet" disabled>
                  Rollback
                </button>
                <button type="button" className="px-btn px-btn-quiet" disabled>
                  Delete
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </motion.div>
  );
}