/**
 * Tier 14 Phase 14.1 — Per-VM action menu (Start / Stop / Reboot / Shutdown).
 * Destructive actions (stop/reboot/shutdown) require a confirm step.
 */
import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import type { ProxmoxResource } from '../../lib/proxmox';
import { readString } from '../../lib/proxmox';

type ActionKind = 'start' | 'shutdown' | 'stop' | 'reboot';

const DESTRUCTIVE: ActionKind[] = ['stop', 'shutdown', 'reboot'];

interface Props {
  resource: ProxmoxResource;
  busy: boolean;
  onAction: (resource: ProxmoxResource, action: ActionKind) => void;
}

export default function ProxmoxVmActions({ resource, busy, onAction }: Props) {
  const [confirmAction, setConfirmAction] = useState<ActionKind | null>(null);
  const status = readString(resource.status).toLowerCase();
  const isRunning = status === 'running';
  const isStopped = status === 'stopped';

  function handle(action: ActionKind) {
    if (DESTRUCTIVE.includes(action)) {
      setConfirmAction(action);
    } else {
      onAction(resource, action);
    }
  }

  return (
    <>
      <div className="px-vm-actions">
        <button
          type="button"
          className="px-vm-action px-vm-action-start"
          onClick={() => handle('start')}
          disabled={busy || !isStopped}
          title={isStopped ? 'Start VM' : 'VM already running'}
        >
          ▶ Start
        </button>
        <button
          type="button"
          className="px-vm-action px-vm-action-stop"
          onClick={() => handle('shutdown')}
          disabled={busy || !isRunning}
          title={isRunning ? 'Shutdown VM' : 'VM is not running'}
        >
          ⏹ Shutdown
        </button>
        <button
          type="button"
          className="px-vm-action px-vm-action-reboot"
          onClick={() => handle('reboot')}
          disabled={busy || !isRunning}
          title={isRunning ? 'Reboot VM' : 'VM is not running'}
        >
          ↻ Reboot
        </button>
      </div>

      <AnimatePresence>
        {confirmAction && (
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
              aria-labelledby="px-confirm-title"
            >
              <h3 id="px-confirm-title">Confirm {confirmAction}</h3>
              <p>
                Are you sure you want to <strong>{confirmAction}</strong>{' '}
                <code>{readString(resource.name) || `VM ${resource.vmid}`}</code>?
              </p>
              {confirmAction !== 'reboot' && (
                <p className="px-confirm-warn">
                  Data loss may occur if unsaved work is in progress.
                </p>
              )}
              <div className="px-confirm-actions">
                <button type="button" className="px-btn px-btn-quiet" onClick={() => setConfirmAction(null)}>
                  Cancel
                </button>
                <button
                  type="button"
                  className="px-btn px-btn-danger"
                  onClick={() => {
                    onAction(resource, confirmAction);
                    setConfirmAction(null);
                  }}
                >
                  Yes, {confirmAction}
                </button>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </>
  );
}