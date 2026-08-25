import {
  motion,
  buttonSpring,
  useReducedMotion,
} from '../../lib/motion';
import type {
  RateLimitBlockedTenant,
  RateLimitGlobalLimits,
} from './RateLimitSection';

// Tier 11 Phase 7 — Rate Limiting (PL7).
//
// RateLimitTables — the per-plan caps grid + the
// blocked-tenants table for the rate-limit dashboard.
//
// Extracted from RateLimitSection.tsx so the parent
// stays under the 400-LOC cap. The two tables are
// related (both are admin views of the same Limiter
// state) so they live together rather than getting
// split into one-file-per-table.
//
// Pure presentational: takes the live data + edit
// state from the parent and renders. All mutation
// (PATCH /global, etc.) happens in RateLimitSection.

export interface RateLimitTablesProps {
  /** Current per-plan cap snapshot from the server. */
  limits: RateLimitGlobalLimits;
  /** Editable string values keyed by plan name (the
   *  input value is a string so it can be empty
   *  mid-edit). */
  edit: Record<string, string>;
  /** Setter for edit — parent's `setEdit({...edit, [k]: v})`. */
  onChangeEdit: (key: string, value: string) => void;
  /** Whether the caller can edit values (super_admin). */
  canMutate: boolean;
  /** Currently-blocked tenants (super_admin only). */
  blocked: RateLimitBlockedTenant[];
}

const PLAN_KEYS: Array<keyof RateLimitGlobalLimits> = [
  'free',
  'starter',
  'pro',
  'enterprise',
];

const PLAN_LABELS: Record<keyof RateLimitGlobalLimits, string> = {
  free: 'Free',
  starter: 'Starter',
  pro: 'Pro',
  enterprise: 'Enterprise',
};

export default function RateLimitTables({
  limits,
  edit,
  onChangeEdit,
  canMutate,
  blocked,
}: RateLimitTablesProps) {
  const reduce = useReducedMotion();
  return (
    <>
      {/* Per-plan limits grid */}
      <section className="dash-table-wrap" aria-label="Per-plan limits">
        <table className="dash-table">
          <thead>
            <tr>
              <th>Plan</th>
              <th>Limit / min</th>
              <th>Refill rate</th>
              <th>Edit (super_admin)</th>
            </tr>
          </thead>
          <tbody>
            {PLAN_KEYS.map((k) => {
              const cap = limits[k];
              const refillSec = cap > 0 ? Math.ceil(60 / cap) : 60;
              return (
                <tr key={k}>
                  <td className="font-mono">{PLAN_LABELS[k]}</td>
                  <td>
                    <span className="dash-status dash-status-neutral">
                      {cap}
                    </span>
                  </td>
                  <td className="text-xs opacity-70">
                    1 token / {refillSec}s
                  </td>
                  <td>
                    {canMutate ? (
                      <input
                        type="number"
                        min={1}
                        max={100000}
                        className="input input-sm w-24"
                        value={edit[k] ?? ''}
                        onChange={(e) => onChangeEdit(k, e.target.value)}
                      />
                    ) : (
                      <span className="opacity-50">—</span>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </section>

      {/* Blocked tenants (super_admin only). Wrapped in
          a motion.section so the entrance is staggered
          with the rest of the page (matches the parent's
          pageEnter + buttonSpring pattern). */}
      {canMutate ? (
        <motion.section
          className="dash-table-wrap"
          aria-label="Blocked tenants"
          initial={reduce ? false : { opacity: 0, y: 4 }}
          animate={{ opacity: 1, y: 0 }}
          transition={buttonSpring.transition}
        >
          <h3 className="text-sm font-semibold opacity-80 mb-2">
            Blocked tenants ({blocked.length})
          </h3>
          {blocked.length === 0 ? (
            <p className="text-sm opacity-60 px-3 py-2">
              No tenants are currently blocked.
            </p>
          ) : (
            <table className="dash-table">
              <thead>
                <tr>
                  <th>Tenant ID</th>
                  <th>Plan</th>
                  <th>Current count</th>
                  <th>Limit / min</th>
                  <th>Blocked for (s)</th>
                </tr>
              </thead>
              <tbody>
                {blocked.map((b) => (
                  <tr key={b.tenant_id}>
                    <td className="font-mono text-xs">{b.tenant_id}</td>
                    <td>
                      <span className="dash-status dash-status-neutral">
                        {b.plan}
                      </span>
                    </td>
                    <td>{b.current_count}</td>
                    <td>{b.limit_per_minute}</td>
                    <td>{b.blocked_until_seconds}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </motion.section>
      ) : null}
    </>
  );
}
