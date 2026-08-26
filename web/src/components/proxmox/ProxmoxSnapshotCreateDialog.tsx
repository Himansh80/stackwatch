/**
 * Tier 14 Phase 14.3 — Snapshot create dialog.
 */
import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';

interface Props {
  open: boolean;
  onClose: () => void;
  onCreate: (name: string, description: string, vmState: boolean) => Promise<void>;
}

export default function ProxmoxSnapshotCreateDialog({ open, onClose, onCreate }: Props) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [vmState, setVmState] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  function reset() {
    setName('');
    setDescription('');
    setVmState(false);
    setError('');
    setBusy(false);
  }

  async function handle() {
    if (!name.trim()) {
      setError('Name is required.');
      return;
    }
    if (!/^[a-zA-Z0-9_\-]+$/.test(name)) {
      setError('Only letters, digits, hyphens and underscores allowed.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      await onCreate(name.trim(), description.trim(), vmState);
      reset();
      onClose();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Failed to create snapshot.');
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
            aria-labelledby="px-snap-title"
          >
            <h3 id="px-snap-title">Create snapshot</h3>
            <p>Create a new snapshot for this VM. Includes current disk state.</p>
            <div className="px-form-field">
              <label htmlFor="px-snap-name">Name</label>
              <input
                id="px-snap-name"
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. pre-upgrade"
                className="px-confirm-input"
                autoFocus
              />
            </div>
            <div className="px-form-field">
              <label htmlFor="px-snap-desc">Description (optional)</label>
              <textarea
                id="px-snap-desc"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="What is this snapshot for?"
                className="px-confirm-textarea"
                rows={2}
              />
            </div>
            <div className="px-form-field px-form-toggle">
              <label htmlFor="px-snap-vmstate">
                <input
                  id="px-snap-vmstate"
                  type="checkbox"
                  checked={vmState}
                  onChange={(e) => setVmState(e.target.checked)}
                />
                Include RAM state (slower, larger snapshot)
              </label>
            </div>
            {error && <div className="px-confirm-warn">{error}</div>}
            <div className="px-confirm-actions">
              <button type="button" className="px-btn px-btn-quiet" onClick={onClose} disabled={busy}>
                Cancel
              </button>
              <button type="button" className="px-btn" onClick={handle} disabled={busy}>
                {busy ? 'Creating…' : 'Create snapshot'}
              </button>
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}