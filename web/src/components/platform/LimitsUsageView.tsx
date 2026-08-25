import { motion, kpiEnter, kpiStagger, useReducedMotion } from '../../lib/motion';
import KpiCard, { KpiAccent } from '../shared/KpiCard';

// Tier 11 Phase 4 — Tenant Limits (PL4).
//
// LimitsUsageView — the live KPI strip + per-resource progress
// bars extracted from LimitsSection.tsx to keep the parent
// under the 400-LOC cap.
//
// Renders:
//   - 5 KpiCards in the standard metric strip
//   - Per-resource progress bar rows (color-coded by status)
//   - The "Billing warnings" banner (only when warnings exist)
//
// Owns no state — all values come from the `usage` prop. The
// parent re-renders this whenever the GET /usage response
// changes (plan change, refetch, etc.).

interface UsageMetric {
  current: number;
  limit: number;
  percent: number;
  status: 'ok' | 'warn' | 'exceeded';
}

interface LimitsUsage {
  plan_name: string;
  plan_display_name: string;
  servers: UsageMetric;
  alerts: UsageMetric;
  dashboards: UsageMetric;
  team_members: UsageMetric;
  storage_gb: UsageMetric;
  api_calls_per_min: UsageMetric;
  warnings: string[];
}

interface LimitsUsageViewProps {
  usage: LimitsUsage;
}

const formatLimit = (m: UsageMetric): string =>
  m.limit >= 999999 ? `${m.current} / ∞` : `${m.current} / ${m.limit}`;

const KpiAccentMap: Record<string, KpiAccent> = {
  Servers: 'cyan',
  Alerts: 'amber',
  Dashboards: 'indigo',
  'Team members': 'violet',
  'Storage (GB)': 'cyan', // 'pink' isn't in KpiAccent — fall back to cyan
};

export default function LimitsUsageView({ usage }: LimitsUsageViewProps) {
  const reduce = useReducedMotion();
  return (
    <>
      <motion.div
        className="dash-metric-strip"
        initial="hidden"
        animate="show"
        variants={kpiStagger}
      >
        {[
          { label: 'Servers', m: usage.servers },
          { label: 'Alerts', m: usage.alerts },
          { label: 'Dashboards', m: usage.dashboards },
          { label: 'Team members', m: usage.team_members },
          { label: 'Storage (GB)', m: usage.storage_gb },
        ].map(({ label, m }) => (
          <motion.div variants={kpiEnter} key={label}>
            <KpiCard
              label={label}
              value={formatLimit(m)}
              status={m.status === 'ok' ? 'neutral' : 'down'}
              accent={KpiAccentMap[label]}
            />
          </motion.div>
        ))}
      </motion.div>

      {usage.warnings.length > 0 ? (
        <div className="limits-warnings" role="alert">
          <div className="limits-warnings-title">Billing warnings</div>
          <ul>
            {usage.warnings.map((w, i) => (
              <li key={i}>{w}</li>
            ))}
          </ul>
        </div>
      ) : null}

      <div className="limits-bars">
        {[
          { name: 'Servers', m: usage.servers },
          { name: 'Alerts', m: usage.alerts },
          { name: 'Dashboards', m: usage.dashboards },
          { name: 'Team members', m: usage.team_members },
          { name: 'Storage GB', m: usage.storage_gb },
        ].map(({ name, m }) => (
          <div className="limits-bar-row" key={name}>
            <div className="limits-bar-label">{name}</div>
            <div className="limits-bar-track">
              <motion.div
                className={`limits-bar-fill limits-bar-${m.status}`}
                initial={{ width: 0 }}
                animate={{ width: `${m.percent}%` }}
                transition={{
                  duration: reduce ? 0 : 0.5,
                  ease: [0.16, 1, 0.3, 1],
                }}
              />
            </div>
            <div className="limits-bar-val">
              {m.limit >= 999999 ? `${m.current} / ∞` : `${m.percent}%`}
            </div>
          </div>
        ))}
      </div>
    </>
  );
}