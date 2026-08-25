import { useState } from 'react';
import { motion, useReducedMotion } from '../../lib/motion';

// Tier 11 Phase 4 — Tenant Limits (PL4).
//
// LimitsCheckTool — the dry-run probe widget embedded in
// LimitsSection ("would creating N more servers work?").
//
// Extracted from LimitsSection.tsx so the parent stays under
// the 400-LOC cap. The component owns no fetch / mutate
// logic — the parent passes in the result so the form can
// be re-used later (e.g. on a future public pricing page
// where the visitor is not logged in and the result is just
// the cap, not the "would exceed?" answer).
//
// Why a sub-component file (not inlined):
//   The check form + result is a self-contained UI surface
//   with its own local state (action + quantity), so the
//   parent doesn't need to manage those inputs. Keeping it
//   here means the parent stays focused on KPI rendering.

interface CheckResult {
  allowed: boolean;
  message: string;
  current: number;
  limit: number;
  remaining: number;
}

interface LimitsCheckToolProps {
  result: CheckResult | null;
  onCheck: (action: string, quantity: number) => void;
  checking: boolean;
}

const ACTIONS = [
  { value: 'create_server', label: 'Create server' },
  { value: 'create_alert', label: 'Create alert' },
  { value: 'create_dashboard', label: 'Create dashboard' },
  { value: 'invite_member', label: 'Invite team member' },
];

export default function LimitsCheckTool({
  result,
  onCheck,
  checking,
}: LimitsCheckToolProps) {
  const reduce = useReducedMotion();
  // Local UI state. Kept inside the component because the
  // form is self-contained — the parent only needs the
  // final (action, quantity) pair at submit time.
  const [action, setAction] = useState<string>('create_server');
  const [quantity, setQuantity] = useState<number>(1);

  return (
    <div className="limits-check">
      <div className="limits-check-title">Check limit</div>
      <p className="limits-check-sub">
        Dry-run a future operation — would it exceed your current plan?
      </p>
      <div className="limits-check-form">
        <label>
          Action
          <select
            value={action}
            onChange={(e) => setAction(e.target.value)}
            disabled={checking}
          >
            {ACTIONS.map((a) => (
              <option key={a.value} value={a.value}>
                {a.label}
              </option>
            ))}
          </select>
        </label>
        <label>
          Quantity
          <input
            type="number"
            min={1}
            max={10000}
            value={quantity}
            onChange={(e) => setQuantity(parseInt(e.target.value, 10) || 1)}
            disabled={checking}
          />
        </label>
        <motion.button
          type="button"
          className="limits-check-btn"
          onClick={() => onCheck(action, quantity)}
          disabled={checking}
          whileHover={reduce ? undefined : { y: -1 }}
          whileTap={reduce ? undefined : { scale: 0.98 }}
        >
          {checking ? 'Checking…' : 'Check'}
        </motion.button>
      </div>
      {result ? (
        <div
          className={`limits-check-result limits-check-${result.allowed ? 'ok' : 'fail'}`}
          role="status"
        >
          <strong>{result.allowed ? '✓ Allowed' : '✗ Would exceed'}</strong>
          <span>{result.message}</span>
          <span className="limits-check-meta">
            current={result.current}, limit={result.limit}, remaining=
            {result.remaining}
          </span>
        </div>
      ) : null}
    </div>
  );
}