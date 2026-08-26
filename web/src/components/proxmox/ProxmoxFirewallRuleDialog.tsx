/**
 * Tier 14 Phase 14.3 — Firewall rule dialog (used for both add + edit).
 */
import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';

export interface FirewallRule {
  pos?: number;
  type?: string;
  action: string;
  enable?: number;
  source?: string;
  dest?: string;
  proto?: string;
  dport?: string;
  sport?: string;
  iface?: string;
  comment?: string;
  macro?: string;
}

interface Props {
  open: boolean;
  initial?: FirewallRule;
  onClose: () => void;
  onSave: (rule: FirewallRule) => Promise<void>;
}

const ACTIONS = ['ACCEPT', 'REJECT', 'DROP'];
const TYPES = ['in', 'out', 'forward', 'group'];
const PROTOS = ['', 'tcp', 'udp', 'icmp', 'all'];

export default function ProxmoxFirewallRuleDialog({ open, initial, onClose, onSave }: Props) {
  const [rule, setRule] = useState<FirewallRule>(initial ?? { action: 'ACCEPT' });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  async function handle() {
    if (!rule.action) {
      setError('Action is required.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      await onSave(rule);
      onClose();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Failed to save rule.');
      setBusy(false);
    }
  }

  function field<K extends keyof FirewallRule>(key: K, value: FirewallRule[K]) {
    setRule((r) => ({ ...r, [key]: value }));
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
            className="px-confirm-dialog px-form-wide"
            initial={{ scale: 0.95, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            exit={{ scale: 0.95, opacity: 0 }}
            onClick={(e) => e.stopPropagation()}
            role="dialog"
          >
            <h3>{initial ? 'Edit firewall rule' : 'Add firewall rule'}</h3>
            <div className="px-form-grid">
              <div className="px-form-field">
                <label>Type</label>
                <select value={rule.type ?? ''} onChange={(e) => field('type', e.target.value)} className="px-form-select">
                  {TYPES.map((t) => <option key={t} value={t}>{t || '(none)'}</option>)}
                </select>
              </div>
              <div className="px-form-field">
                <label>Action *</label>
                <select value={rule.action} onChange={(e) => field('action', e.target.value)} className="px-form-select">
                  {ACTIONS.map((a) => <option key={a} value={a}>{a}</option>)}
                </select>
              </div>
              <div className="px-form-field">
                <label>Source</label>
                <input type="text" value={rule.source ?? ''} onChange={(e) => field('source', e.target.value)} placeholder="192.168.0.0/24 or empty" className="px-confirm-input" />
              </div>
              <div className="px-form-field">
                <label>Destination</label>
                <input type="text" value={rule.dest ?? ''} onChange={(e) => field('dest', e.target.value)} placeholder="e.g. 10.0.0.0/8" className="px-confirm-input" />
              </div>
              <div className="px-form-field">
                <label>Proto</label>
                <select value={rule.proto ?? ''} onChange={(e) => field('proto', e.target.value)} className="px-form-select">
                  {PROTOS.map((p) => <option key={p} value={p}>{p || 'any'}</option>)}
                </select>
              </div>
              <div className="px-form-field">
                <label>Dest port</label>
                <input type="text" value={rule.dport ?? ''} onChange={(e) => field('dport', e.target.value)} placeholder="e.g. 22" className="px-confirm-input" />
              </div>
              <div className="px-form-field">
                <label>Source port</label>
                <input type="text" value={rule.sport ?? ''} onChange={(e) => field('sport', e.target.value)} className="px-confirm-input" />
              </div>
              <div className="px-form-field">
                <label>Interface</label>
                <input type="text" value={rule.iface ?? ''} onChange={(e) => field('iface', e.target.value)} placeholder="net0" className="px-confirm-input" />
              </div>
              <div className="px-form-field px-form-full">
                <label>Comment</label>
                <input type="text" value={rule.comment ?? ''} onChange={(e) => field('comment', e.target.value)} placeholder="Why this rule?" className="px-confirm-input" />
              </div>
            </div>
            {error && <div className="px-confirm-warn">{error}</div>}
            <div className="px-confirm-actions">
              <button type="button" className="px-btn px-btn-quiet" onClick={onClose} disabled={busy}>
                Cancel
              </button>
              <button type="button" className="px-btn" onClick={handle} disabled={busy}>
                {busy ? 'Saving…' : (initial ? 'Save changes' : 'Add rule')}
              </button>
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}