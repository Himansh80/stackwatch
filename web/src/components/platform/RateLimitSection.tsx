import { useEffect, useState } from 'react';
import { ApiError, api, getToken } from '../../lib/api';
import {
  motion,
  pageEnter,
  kpiStagger,
  buttonSpring,
  useReducedMotion,
} from '../../lib/motion';
import EmptyState from '../shared/EmptyState';
import KpiCard from '../shared/KpiCard';
import RateLimitTables from './RateLimitTables';

// Tier 11 Phase 7 — Rate Limiting (PL7).
//
// RateLimitSection renders the in-memory token-bucket
// rate-limit surface inside the unified PlatformPage
// (Phase 8 lands the 6-tab shell; today Phase 7 exports
// this component so a quick integration can drop it into
// any tab container).
//
// Layout (top → bottom):
//   - Header       : "Rate Limits" + "Reset to defaults" /
//                    "Save limits" buttons (super_admin only).
//   - KPI strip    : 4 tiles — current usage / my limit /
//                    reset in (seconds) / blocked until
//                    (only rendered when > 0).
//   - Per-plan grid: 4 cards (free/starter/pro/enterprise)
//                    with editable limit input (super_admin
//                    only) and current value.
//   - Blocked list : table of currently-blocked tenants
//                    (super_admin only).
//
// Data sources:
//   GET   /api/v1/platform/ratelimit/me      — my bucket
//   PATCH /api/v1/platform/ratelimit/global  — adjust caps
//   GET   /api/v1/platform/ratelimit/blocked  — blocked list
//
// Auth: every load bails early when no JWT is present so
// we don't spam a 401 from the platform-admin login flow.
// Mutations (global caps) require Role == super_admin.
//
// Motion : pageEnter on the section wrapper; kpiStagger
// on the KPI strip; buttonSpring on the action buttons.
// No new variants — reuses the existing motion
// primitives from src/lib/motion.tsx.
//
// Tables (per-plan grid + blocked list) live in
// RateLimitTables.tsx to keep this file under the
// 400-LOC cap while keeping the related views grouped.

export interface RateLimitMe {
  tenant_id: string;
  plan: string;
  limit_per_minute: number;
  current_window_count: number;
  remaining: number;
  reset_at_seconds: number;
  blocked_until_seconds: number;
}

export interface RateLimitGlobalLimits {
  free: number;
  starter: number;
  pro: number;
  enterprise: number;
}

export interface RateLimitBlockedTenant {
  tenant_id: string;
  plan: string;
  current_count: number;
  limit_per_minute: number;
  blocked_until_seconds: number;
}

interface RateLimitGlobalResp {
  limits: RateLimitGlobalLimits;
  reset: boolean;
}

interface RateLimitBlockedResp {
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

interface RateLimitSectionProps {
  /** True when the caller can edit global caps and
   *  view the blocked-tenants list (super_admin). */
  canMutate: boolean;
}

export default function RateLimitSection({ canMutate }: RateLimitSectionProps) {
  const [me, setMe] = useState<RateLimitMe | null>(null);
  const [limits, setLimits] = useState<RateLimitGlobalLimits | null>(null);
  const [blocked, setBlocked] = useState<RateLimitBlockedTenant[]>([]);
  const [busy, setBusy] = useState(false);
  const [saving, setSaving] = useState(false);
  const [resetting, setResetting] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  // Editable per-plan limit values (string so the input
  // can be empty mid-edit). Commit on blur or "Save".
  const [edit, setEdit] = useState<Record<string, string>>({});
  const reduce = useReducedMotion();

  const requireAuth = (): boolean => {
    if (!getToken()) {
      setError('Sign in to view rate limits.');
      return false;
    }
    return true;
  };

  const loadAll = async () => {
    if (!requireAuth()) return;
    setBusy(true);
    setError('');
    try {
      const myReq = api<RateLimitMe>('GET', '/api/v1/platform/ratelimit/me');
      // /global and /blocked are super_admin only — fetch
      // them in parallel but tolerate 403 for non-admins
      // (the rest of the page still renders for everyone).
      const globalReq = api<RateLimitGlobalResp>(
        'GET',
        '/api/v1/platform/ratelimit/global'
      );
      const blockedReq = api<RateLimitBlockedResp>(
        'GET',
        '/api/v1/platform/ratelimit/blocked'
      );
      const [my, global, blockedRows] = await Promise.all([
        myReq,
        globalReq,
        blockedReq,
      ]);
      setMe(my);
      setLimits(global.limits);
      setBlocked(blockedRows.blocked ?? []);
      // Initialise edit-state from server values.
      const init: Record<string, string> = {};
      for (const k of PLAN_KEYS) {
        init[k] = String(global.limits[k] ?? '');
      }
      setEdit(init);
    } catch (cause) {
      // If /global or /blocked returns 403 the page still
      // renders /me for non-admins. Detect "auth" errors
      // and only surface them when /me also failed.
      if (cause instanceof ApiError && cause.status === 401) {
        setError('Sign in to view rate limits.');
      } else {
        setError(
          cause instanceof ApiError
            ? cause.friendlyMessage
            : (cause as Error).message
        );
      }
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

  // Build the PATCH body with only the keys that changed.
  // Validates every value against the 1..100000 range
  // BEFORE sending so the dashboard surfaces an inline
  // error instead of a server-side 400.
  const buildPatchBody = (): Record<string, number> | null => {
    const body: Record<string, number> = {};
    for (const k of PLAN_KEYS) {
      const raw = edit[k];
      const parsed = parseInt(raw ?? '', 10);
      if (Number.isNaN(parsed)) {
        setError(`Invalid value for ${PLAN_LABELS[k]}: must be an integer.`);
        return null;
      }
      if (parsed < 1 || parsed > 100000) {
        setError(
          `${PLAN_LABELS[k]} must be between 1 and 100000 requests/minute.`
        );
        return null;
      }
      if (limits && parsed !== limits[k]) {
        body[k] = parsed;
      }
    }
    return body;
  };

  const handleSave = async () => {
    if (!requireAuth()) return;
    const body = buildPatchBody();
    if (!body) return;
    if (Object.keys(body).length === 0) {
      setSuccess('No changes to save.');
      return;
    }
    setSaving(true);
    setError('');
    setSuccess('');
    try {
      const resp = await api<RateLimitGlobalResp>(
        'PATCH',
        '/api/v1/platform/ratelimit/global',
        body
      );
      setLimits(resp.limits);
      syncEdit(resp.limits);
      setSuccess('Rate limit caps updated.');
    } catch (cause) {
      setError(
        cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message
      );
    } finally {
      setSaving(false);
    }
  };

  const handleReset = async () => {
    if (!requireAuth()) return;
    setResetting(true);
    setError('');
    setSuccess('');
    try {
      // Empty body = "reset to defaults" on the server.
      const resp = await api<RateLimitGlobalResp>(
        'PATCH',
        '/api/v1/platform/ratelimit/global',
        {}
      );
      setLimits(resp.limits);
      syncEdit(resp.limits);
      setSuccess('Reset to defaults.');
    } catch (cause) {
      setError(
        cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message
      );
    } finally {
      setResetting(false);
    }
  };

  // syncEdit re-keys the edit-state map from a server
  // response so a clamped value (rare) is visible
  // immediately. Pulled out so handleSave + handleReset
  // don't duplicate the loop.
  const syncEdit = (next: RateLimitGlobalLimits) => {
    const synced: Record<string, string> = {};
    for (const k of PLAN_KEYS) {
      synced[k] = String(next[k] ?? '');
    }
    setEdit(synced);
  };

  const showEmpty = !busy && !me && !limits && !error;
  const myBlockedSec = me?.blocked_until_seconds ?? 0;

  return (
    <motion.div
      initial="hidden"
      animate="show"
      variants={pageEnter}
      className="space-y-6 p-2"
    >
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold">Rate Limits</h2>
          <p className="text-sm opacity-70">
            In-memory token-bucket limits per tenant plan. Buckets refill
            at 1 token every (60 / limit) seconds.
          </p>
        </div>
        {canMutate ? (
          <div className="flex gap-2">
            <motion.button
              type="button"
              onClick={handleReset}
              disabled={resetting || busy}
              className="btn-secondary"
              whileHover={reduce ? undefined : buttonSpring.whileHover}
              whileTap={reduce ? undefined : buttonSpring.whileTap}
              transition={buttonSpring.transition}
            >
              {resetting ? 'Resetting…' : 'Reset to defaults'}
            </motion.button>
            <motion.button
              type="button"
              onClick={handleSave}
              disabled={saving || busy}
              className="btn-primary"
              whileHover={reduce ? undefined : buttonSpring.whileHover}
              whileTap={reduce ? undefined : buttonSpring.whileTap}
              transition={buttonSpring.transition}
            >
              {saving ? 'Saving…' : 'Save limits'}
            </motion.button>
          </div>
        ) : null}
      </header>

      {error ? (
        <div className="callout-error" role="alert">
          {error}
        </div>
      ) : null}
      {success ? (
        <div className="callout-success" role="status">
          {success}
        </div>
      ) : null}

      {/* KPI strip */}
      <motion.section
        variants={kpiStagger}
        initial="hidden"
        animate="show"
        className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3"
      >
        <KpiCard
          label="My usage (this window)"
          value={me?.current_window_count ?? 0}
          accent="indigo"
        />
        <KpiCard
          label="My limit / min"
          value={me?.limit_per_minute ?? 0}
          accent="cyan"
        />
        <KpiCard
          label="Reset in (s)"
          value={me?.reset_at_seconds ?? 60}
          accent="green"
        />
        <KpiCard
          label="Blocked until (s)"
          value={myBlockedSec}
          status={myBlockedSec > 0 ? 'stale' : 'neutral'}
          accent={myBlockedSec > 0 ? 'amber' : 'cyan'}
        />
      </motion.section>

      {limits ? (
        <RateLimitTables
          limits={limits}
          edit={edit}
          onChangeEdit={(k, v) => setEdit({ ...edit, [k]: v })}
          canMutate={canMutate}
          blocked={blocked}
        />
      ) : null}

      {showEmpty ? (
        <EmptyState
          illustration={<span aria-hidden="true">⏱️</span>}
          headline="No rate-limit data yet"
          subhead="Once you're authenticated, the live bucket state + per-plan caps will render here."
        />
      ) : null}
    </motion.div>
  );
}
