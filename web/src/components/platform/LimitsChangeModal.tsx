import { motion, useReducedMotion } from '../../lib/motion';

// Tier 11 Phase 4 — Tenant Limits (PL4).
//
// LimitsChangeModal — the "switch to a different plan" dialog.
//
// Extracted from LimitsSection.tsx so the main section file
// stays under the 400-LOC cap (with all 6 sections + the modal
// the parent would have been ~530 LOC). The modal renders a
// 4-card radio grid (one per catalog plan) and a single submit
// button. The parent owns the open/closed state, the
// in-flight patching flag, and the catalog sort.
//
// Why a separate component (not a sub-component file):
//   Reuse across pages — Phase 8 PL8 may embed the same modal
//   in the PlatformHealth "you're on free; upgrade?" banner.
//
// Props mirror the parent's view-state so the modal has zero
// of its own fetch / mutate logic.

interface PlanDefinition {
  name: string;
  display_name: string;
  monthly_price_cents: number;
  currency: string;
  max_servers: number;
  max_alerts: number;
  max_dashboards: number;
  max_team_members: number;
  data_retention_days: number;
  metrics_retention_days: number;
  storage_gb_limit: number;
  api_calls_per_minute: number;
  features: string[];
}

interface CurrentPlan {
  plan_name: string;
  plan_display_name: string;
}

interface LimitsChangeModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (planName: string) => void;
  patching: boolean;
  pendingPlan: string;
  setPendingPlan: (name: string) => void;
  definitions: PlanDefinition[];
  current: CurrentPlan;
}

const formatUSD = (cents: number): string =>
  cents === 0 ? 'Custom' : `$${(cents / 100).toFixed(2)}`;

export default function LimitsChangeModal({
  open,
  onClose,
  onSubmit,
  patching,
  pendingPlan,
  setPendingPlan,
  definitions,
  current,
}: LimitsChangeModalProps) {
  const reduce = useReducedMotion();
  if (!open) return null;

  // Render plans ASC by price so the cheapest option is
  // first (matches the marketing-site ordering convention).
  const sortedDefs = [...definitions].sort(
    (a, b) => a.monthly_price_cents - b.monthly_price_cents
  );

  return (
    <div
      className="sw-modal-backdrop"
      role="dialog"
      aria-modal="true"
      aria-label="Change plan"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="sw-modal">
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (pendingPlan && pendingPlan !== current.plan_name) {
              onSubmit(pendingPlan);
            }
          }}
          noValidate
        >
          <h3 className="limits-modal-title">Change plan</h3>
          <p className="limits-modal-sub">
            Current plan: <strong>{current.plan_display_name}</strong>
          </p>
          <div className="limits-modal-grid">
            {sortedDefs.map((p) => (
              <label
                key={p.name}
                className={`limits-modal-option ${
                  pendingPlan === p.name ? 'limits-modal-option-active' : ''
                }`}
              >
                <input
                  type="radio"
                  name="plan"
                  value={p.name}
                  checked={pendingPlan === p.name}
                  onChange={() => setPendingPlan(p.name)}
                />
                <div className="limits-modal-option-name">{p.display_name}</div>
                <div className="limits-modal-option-price">
                  {formatUSD(p.monthly_price_cents)} / mo
                </div>
                <ul className="limits-modal-option-features">
                  <li>
                    {p.max_servers >= 999999 ? '∞' : p.max_servers} servers
                  </li>
                  <li>{p.max_alerts >= 999999 ? '∞' : p.max_alerts} alerts</li>
                  <li>{p.data_retention_days}d retention</li>
                  <li>
                    {p.storage_gb_limit >= 99999 ? '∞' : p.storage_gb_limit}GB
                    storage
                  </li>
                </ul>
              </label>
            ))}
          </div>
          <div className="sw-form-actions">
            <motion.button
              type="submit"
              className="signup-submit"
              disabled={
                patching || !pendingPlan || pendingPlan === current.plan_name
              }
              whileHover={reduce ? undefined : { y: -1 }}
              whileTap={reduce ? undefined : { scale: 0.98 }}
            >
              {patching ? 'Switching…' : `Switch to ${pendingPlan || '?'}`}
            </motion.button>
            <button
              type="button"
              className="signup-back"
              onClick={onClose}
              disabled={patching}
            >
              Cancel
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}