/**
 * Tier 14 Phase 14.2 — VM action bar (Start / Shutdown / Reboot / Migrate / Delete).
 * Migrate is deferred (shows toast). Delete has type-DELETE confirmation.
 */
import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { readString } from '../../lib/proxmox';

interface Props {
  status: string;
  busy: boolean;
  onAction: (action: 'start' | 'shutdown' | 'stop' | 'reboot' | 'delete') => void;
  onMigrateClick: () => void;
}

const DESTRUCTIVE = ['shutdown', 'stop', 'reboot', 'delete'] as const;

export default function ProxmoxDetailActions({ status, busy, onAction, onMigrateClick }: Props) {
  const [confirmAction, setConfirmAction] = useState<typeof DESTRUCTIVE[number] | null>(null);
  const [deleteConfirmText, setDeleteConfirmText] = useState('');

  const isRunning = status === 'running';
  const isStopped = status === 'stopped';

  function handle(action: typeof DESTRUCTIVE[number]) {
    if (action === 'delete') {
      setConfirmAction('delete');
    } else if (DESTRUCTIVE.includes(action as typeof DESTRUCTIVE[number])) {
      setConfirmAction(action);
    } else {
      onAction(action);
    }
  }

  return (
    <>
      <div className="px-detail-actions">
        <button
          type="button"
          className="px-action px-action-start"
          onClick={() => onAction('start')}
          disabled={busy || !isStopped}
          title={isStopped ? 'Start VM' : 'Already running'}
        >
          ▶ Start
        </button>
        <button
          type="button"
          className="px-action px-action-shutdown"
          onClick={() => handle('shutdown')}
          disabled={busy || !isRunning}
          title={isRunning ? 'Shutdown VM' : 'Not running'}
        >
          ⏹ Shutdown
        </button>
        <button
          type="button"
          className="px-action px-action-reboot"
          onClick={() => handle('reboot')}
          disabled={busy || !isRunning}
          title={isRunning ? 'Reboot VM' : 'Not running'}
        >
          ↻ Reboot
        </button>
        <button
          type="button"
          className="px-action px-action-migrate"
          onClick={onMigrateClick}
          disabled={busy}
          title="Migrate VM to another node (opens picker)"
        >
          ⇄ Migrate
        </button>
        <button
          type="button"
          className="px-action px-action-delete"
          onClick={() => handle('delete')}
          disabled={busy}
          title="Delete VM (destructive)"
        >
          🗑 Delete
        </button>
      </div>

      <AnimatePresence>
        {confirmAction && confirmAction !== 'delete' && (
          <motion.div
            className="px-confirm-overlay"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={() => setConfirmAction(null)}
          >
            <motion.div
              className="px-confirm-dialog"
              initial={{ scale: 0.95, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.95, opacity: 0 }}
              onClick={(e) => e.stopPropagation()}
              role="dialog"
            >
              <h3>Confirm {confirmAction}</h3>
              <p>
                Are you sure you want to <strong>{confirmAction}</strong> this VM?
              </p>
              {confirmAction !== 'reboot' && (
                <p className="px-confirm-warn">Data loss may occur if unsaved work is in progress.</p>
              )}
              <div className="px-confirm-actions">
                <button type="button" className="px-btn px-btn-quiet" onClick={() => setConfirmAction(null)}>
                  Cancel
                </button>
                <button
                  type="button"
                  className="px-btn px-btn-danger"
                  onClick={() => {
                    onAction(confirmAction);
                    setConfirmAction(null);
                  }}
                >
                  Yes, {confirmAction}
                </button>
              </div>
            </motion.div>
          </motion.div>
        )}

        {confirmAction === 'delete' && (
          <motion.div
            className="px-confirm-overlay"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={() => {
              setConfirmAction(null);
              setDeleteConfirmText('');
            }}
          >
            <motion.div
              className="px-confirm-dialog"
              initial={{ scale: 0.95, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.95, opacity: 0 }}
              onClick={(e) => e.stopPropagation()}
              role="dialog"
            >
              <h3>Delete this VM permanently?</h3>
              <p>
                This will delete <code>VM {readString('')}</code> and all its data.
                This action cannot be undone.
              </p>
              <p className="px-confirm-warn">
                Type <strong>DELETE</strong> below to confirm:
              </p>
              <input
                type="text"
                value={deleteConfirmText}
                onChange={(e) => setDeleteConfirmText(e.target.value)}
                placeholder="Type DELETE"
                className="px-confirm-input"
                autoFocus
              />
              <div className="px-confirm-actions">
                <button
                  type="button"
                  className="px-btn px-btn-quiet"
                  onClick={() => {
                    setConfirmAction(null);
                    setDeleteConfirmText('');
                  }}
                >
                  Cancel
                </button>
                <button
                  type="button"
                  className="px-btn px-btn-danger"
                  disabled={deleteConfirmText !== 'DELETE'}
                  onClick={() => {
                    onAction('delete');
                    setConfirmAction(null);
                    setDeleteConfirmText('');
                  }}
                >
                  Yes, delete permanently
                </button>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </>
  );
}