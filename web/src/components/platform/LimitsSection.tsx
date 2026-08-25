import { useEffect, useState } from 'react';
import { ApiError, api, getToken } from '../../lib/api';
import { motion, pageEnter, useReducedMotion } from '../../lib/motion';
import EmptyState from '../shared/EmptyState';
import LimitsChangeModal from './LimitsChangeModal';
import LimitsCheckTool from './LimitsCheckTool';
import LimitsUsageView from './LimitsUsageView';

// Tier 11 Phase 4 — Tenant Limits (PL4).
//
// LimitsSection renders the per-tenant plan + cap surface
// inside the unified PlatformPage (Phase 8 lands the 6-tab
// shell; today Phase 4 exports this component so a quick
// integration can drop it into any tab container).
//
// Layout (top → bottom):
//   - Header    : "Plan & Limits" + "Change plan" button (opens modal)
//   - Plan strip: current plan card (display name, price, feature list)
//   - KPI strip : 5 tiles — servers / alerts / dashboards / team /
//                 storage — each with progress bar + percent
//   - Warnings  : banner list of "you're at 80% of your X limit"
//   - Check-tool: dry-run form ("would creating 1 more server work?")
//
// Data sources:
//   GET   /api/v1/platform/limits/definitions  — catalog (public)
//   GET   /api/v1/platform/limits/me           — my plan
//   PATCH /api/v1/platform/limits/me           — change plan
//   POST  /api/v1/platform/limits/check        — dry-run probe
//   GET   /api/v1/platform/limits/usage        — usage vs limits (KPI strip)
//
// Auth: every load bails early when no JWT is present so we
// don't spam a 401 from the platform-admin login flow.
//
// Motion : pageEnter on the section wrapper; kpiStagger on the
// KPI strip; kpiEnter on each card. No new variants — reuses the
// existing motion primitives from src/lib/motion.tsx.

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

interface MyLimits {
  tenant_id: string;
  plan_name: string;
  plan_display_name: string;
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
  suspended: boolean;
  suspend_reason?: string;
  trial_ends_at?: string;
  custom_overrides: Record<string, number>;
}

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

interface LimitsSectionProps {
  /** True when the caller can change plans (super_admin or platform_admin). */
  canChangePlan: boolean;
}

const formatUSD = (cents: number): string =>
  cents === 0 ? 'Custom' : `$${(cents / 100).toFixed(2)}`;

export default function LimitsSection({ canChangePlan }: LimitsSectionProps) {
  const reduce = useReducedMotion();
  const [definitions, setDefinitions] = useState<PlanDefinition[]>([]);
  const [me, setMe] = useState<MyLimits | null>(null);
  const [usage, setUsage] = useState<LimitsUsage | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  // Modal state
  const [showChangeModal, setShowChangeModal] = useState(false);
  const [pendingPlan, setPendingPlan] = useState<string>('');
  const [patching, setPatching] = useState(false);

  // Check-tool state — the result lives here so we can clear
  // it on plan change (a PATCH may flip the answer). The
  // action + quantity inputs are owned by LimitsCheckTool.
  const [checkResult, setCheckResult] = useState<{
    allowed: boolean;
    message: string;
    current: number;
    limit: number;
    remaining: number;
  } | null>(null);
  const [checking, setChecking] = useState(false);

  const requireAuth = (): boolean => {
    if (!getToken()) {
      setError('Sign in to view plan & limits.');
      return false;
    }
    return true;
  };

  const loadAll = async () => {
    if (!requireAuth()) return;
    setBusy(true);
    setError('');
    try {
      const [defs, my, use] = await Promise.all([
        api<{ plans: PlanDefinition[] }>(
          'GET',
          '/api/v1/platform/limits/definitions'
        ),
        api<MyLimits>('GET', '/api/v1/platform/limits/me'),
        api<LimitsUsage>('GET', '/api/v1/platform/limits/usage'),
      ]);
      setDefinitions(defs.plans);
      setMe(my);
      setUsage(use);
    } catch (cause) {
      setError(
        cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message
      );
    } finally {
      setBusy(false);
    }
  };

  useEffect(() => {
    let cancelled = false;
    const run = async () => {
      await loadAll();
      if (cancelled) return;
    };
    void run();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Sort plans by price so the change-modal renders ASC.
  // (LimitsChangeModal re-sorts internally; nothing to do here.)

  const handlePatch = async (planName: string) => {
    if (!requireAuth()) return;
    setPatching(true);
    setError('');
    try {
      const updated = await api<MyLimits>('PATCH', '/api/v1/platform/limits/me', {
        plan_name: planName,
      });
      setMe(updated);
      setShowChangeModal(false);
      setPendingPlan('');
      await loadAll();
    } catch (cause) {
      setError(
        cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message
      );
    } finally {
      setPatching(false);
    }
  };

  const handleCheck = async (action: string, quantity: number) => {
    if (!requireAuth()) return;
    setChecking(true);
    setCheckResult(null);
    setError('');
    try {
      const result = await api<{
        allowed: boolean;
        message: string;
        current: number;
        limit: number;
        remaining: number;
      }>('POST', '/api/v1/platform/limits/check', {
        action,
        quantity,
      });
      setCheckResult(result);
    } catch (cause) {
      setError(
        cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message
      );
    } finally {
      setChecking(false);
    }
  };

  const showEmpty = !busy && !error && !me && !usage;

  return (
    <motion.section
      className="limits-section"
      initial="hidden"
      animate="show"
      variants={pageEnter}
    >
      <header className="limits-header">
        <div>
          <h2 className="limits-title">Plan &amp; Limits</h2>
          <p className="limits-sub">
            Your tenant&apos;s plan tier, caps, and current usage against them.
          </p>
        </div>
        {canChangePlan && me ? (
          <motion.button
            type="button"
            className="limits-change"
            onClick={() => {
              setPendingPlan(me.plan_name);
              setShowChangeModal(true);
            }}
            disabled={busy}
            whileHover={reduce ? undefined : { y: -1 }}
            whileTap={reduce ? undefined : { scale: 0.98 }}
          >
            Change plan
          </motion.button>
        ) : null}
      </header>

      {error ? (
        <div className="limits-error" role="alert">{error}</div>
      ) : null}

      {me ? (
        <div className="limits-plan-card">
          <div className="limits-plan-card-name">{me.plan_display_name}</div>
          <div className="limits-plan-card-price">
            {formatUSD(me.monthly_price_cents)} / month
          </div>
          {me.suspended ? (
            <div className="limits-plan-suspended" role="alert">
              ⚠ Suspended{me.suspend_reason ? `: ${me.suspend_reason}` : ''}
            </div>
          ) : null}
          {me.trial_ends_at ? (
            <div className="limits-plan-trial">
              Trial ends {new Date(me.trial_ends_at).toLocaleDateString()}
            </div>
          ) : null}
          {me.features && me.features.length > 0 ? (
            <ul className="limits-plan-features">
              {me.features.map((f) => (
                <li key={f}>{f.replace(/_/g, ' ')}</li>
              ))}
            </ul>
          ) : null}
          {Object.keys(me.custom_overrides || {}).length > 0 ? (
            <div className="limits-overrides">
              <strong>Custom overrides:</strong>{' '}
              {Object.entries(me.custom_overrides)
                .map(([k, v]) => `${k}=${v}`)
                .join(', ')}
            </div>
          ) : null}
        </div>
      ) : null}

      {usage ? <LimitsUsageView usage={usage} /> : null}

      <LimitsCheckTool result={checkResult} onCheck={handleCheck} checking={checking} />

      {showEmpty ? (
        <EmptyState
          illustration={<span aria-hidden="true">📋</span>}
          headline="No limits data yet"
          subhead="Once your tenant has plan + usage data, the KPI strip and progress bars will render here."
        />
      ) : null}

      <LimitsChangeModal
        open={showChangeModal && !!me}
        onClose={() => setShowChangeModal(false)}
        onSubmit={handlePatch}
        patching={patching}
        pendingPlan={pendingPlan}
        setPendingPlan={setPendingPlan}
        definitions={definitions}
        current={{
          plan_name: me?.plan_name ?? '',
          plan_display_name: me?.plan_display_name ?? '',
        }}
      />
    </motion.section>
  );
}