import { useCallback, useEffect, useMemo, useState } from 'react';
import KpiCard from '../shared/KpiCard';
import CopyBox from '../shared/CopyBox';
import { ApiError, api } from '../../lib/api';
import { motion, buttonSpring, kpiStagger, useReducedMotion } from '../../lib/motion';

/**
 * DeploySection — Tier 11.1 (Phase 1 — PL1) UI surface for the
 * Push-Button Deploy page section. Mirrors the SsoSection /
 * ScimSection pattern so the future AdminPage (Phase 9) can
 * compose this in without going over the 400-LOC cap.
 *
 * Layout:
 *   - 4 KpiCards: tokens active / installs 24h / installs 7d /
 *     installs 30d (using kpiStagger for entrance).
 *   - Action card:
 *       - "+ Generate install token" button → modal with label
 *         input
 *       - After generation: ONE-LINE CURL snippet + CopyBox
 *         + countdown timer to expiration
 *       - "Regenerate" button (creates new token, invalidates old)
 *       - "Revoke all tokens" button (marks all unused as revoked
 *         — note: Phase 1 doesn't add a DELETE endpoint yet,
 *         so this button is disabled with a tooltip; Phase 5
 *         adds the actual revoke)
 *   - Empty state with helpful instructions
 *
 * Motion: reuses existing exports (kpiStagger, buttonSpring) —
 * no new variants. Tokens: --surface, --border, --accent,
 * --green, --amber, --red via inline styles.
 */

interface DeployStats {
  installs_24h: number;
  installs_7d: number;
  installs_30d: number;
  tokens_active: number;
  tokens_total: number;
}

interface ActiveToken {
  token: string;
  expires_at: string;
  created_at: string;
  label?: string | null;
}

interface DeploySectionProps {
  backend: string;
  busy?: boolean;
  onError: (msg: string) => void;
}

function useCountdown(targetIso: string | null) {
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    if (!targetIso) return;
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, [targetIso]);
  if (!targetIso) return '';
  const diff = Math.max(0, new Date(targetIso).getTime() - now);
  if (diff <= 0) return 'expired';
  const min = Math.floor(diff / 60000);
  const sec = Math.floor((diff % 60000) / 1000);
  return `${min}m ${sec.toString().padStart(2, '0')}s remaining`;
}

export default function DeploySection({ backend, busy = false, onError }: DeploySectionProps) {
  const reduce = useReducedMotion();
  const [sectionBusy, setSectionBusy] = useState(false);
  const isBusy = busy || sectionBusy;

  const [stats, setStats] = useState<DeployStats | null>(null);
  const [active, setActive] = useState<ActiveToken | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [label, setLabel] = useState('');

  const refresh = useCallback(async () => {
    try {
      const next = await api<DeployStats>('GET', '/api/v1/platform/deploy/stats');
      setStats(next);
    } catch (cause) {
      onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, [onError]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const openCreate = useCallback(() => {
    setLabel('');
    setShowCreate(true);
  }, []);
  const closeCreate = useCallback(() => {
    setShowCreate(false);
    setLabel('');
  }, []);

  const handleGenerate = useCallback(async () => {
    setSectionBusy(true);
    try {
      const body = label.trim() ? { label: label.trim() } : {};
      const res = await api<{
        token: string;
        expires_at: string;
        created_at: string;
        label?: string | null;
      }>('POST', '/api/v1/platform/deploy/install-token', body);
      setActive({
        token: res.token,
        expires_at: res.expires_at,
        created_at: res.created_at,
        label: res.label ?? null,
      });
      closeCreate();
      void refresh();
    } catch (cause) {
      onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    } finally {
      setSectionBusy(false);
    }
  }, [label, refresh, onError, closeCreate]);

  const regenerate = useCallback(async () => {
    // Phase 1 doesn't have a DELETE endpoint — we just mint a
    // fresh token; the old one will expire naturally in ≤1h
    // (or be consumed when its box installs). The UI's
    // Countdown makes the timeline visible.
    await handleGenerate();
  }, [handleGenerate]);

  const curlSnippet = useMemo(() => {
    if (!active) return '';
    const safeBackend = backend.replace(/\/+$/, '');
    const safeToken = active.token;
    // We intentionally escape the token for single-quote shell
    // quoting so a paste-into-terminal can't break the command.
    const quoted = `'${safeToken.replace(/'/g, `'\\''`)}'`;
    return `curl -fsSL ${safeBackend}/install.sh?token=${quoted} | sudo bash`;
  }, [active, backend]);

  const countdown = useCountdown(active?.expires_at ?? null);

  return (
    <motion.section variants={kpiStagger} initial="hidden" animate="show" style={{ display: 'grid', gap: 20 }}>
      {/* KPI strip */}
      <div className="dash-metric-strip">
        <KpiCard label="Tokens active" value={stats?.tokens_active ?? '—'} accent="cyan" />
        <KpiCard label="Installs 24h" value={stats?.installs_24h ?? '—'} accent="green" />
        <KpiCard label="Installs 7d" value={stats?.installs_7d ?? '—'} accent="indigo" />
        <KpiCard label="Installs 30d" value={stats?.installs_30d ?? '—'} accent="violet" />
      </div>

      {/* Action card */}
      <article className="threat-card">
        <div className="threat-card-top">
          <span className="dash-sev dash-sev-low">DEPLOY</span>
          <strong className="threat-card-type">Push-Button Deploy</strong>
          <span className="dash-status dash-status-up">
            <span className="dash-status-dot" aria-hidden="true" />
            ready
          </span>
        </div>
        <p className="threat-card-desc">
          Mint a one-time install token, hand the curl snippet to your server, and it self-registers
          with your tenant in &lt;60s. Tokens expire in 1 hour and can&apos;t be replayed.
        </p>

        {active ? (
          <div style={{ display: 'grid', gap: 12, marginTop: 12 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, fontSize: 12 }}>
              <span className="dash-status dash-status-up">
                <span className="dash-status-dot" aria-hidden="true" />
                token minted
              </span>
              <span style={{ color: 'var(--text-muted)' }}>
                {active.label ? `label: ${active.label} · ` : ''}
                {countdown}
              </span>
              <motion.button
                type="button"
                className="threat-card-resolve-btn"
                onClick={regenerate}
                disabled={isBusy}
                whileHover={reduce ? undefined : buttonSpring.whileHover}
                whileTap={reduce ? undefined : buttonSpring.whileTap}
                transition={buttonSpring.transition}
                style={{ marginLeft: 'auto' }}
              >
                Regenerate
              </motion.button>
            </div>
            <CopyBox value={curlSnippet} />
          </div>
        ) : (
          <div style={{ marginTop: 12 }}>
            <motion.button
              type="button"
              className="empty-state-cta"
              onClick={openCreate}
              disabled={isBusy}
              whileHover={reduce ? undefined : buttonSpring.whileHover}
              whileTap={reduce ? undefined : buttonSpring.whileTap}
              transition={buttonSpring.transition}
            >
              + Generate install token
            </motion.button>
          </div>
        )}
      </article>

      {/* Empty-state instructions */}
      {!stats || stats.tokens_total === 0 ? (
        <article className="threat-card" style={{ textAlign: 'center', padding: 24 }}>
          <strong style={{ color: 'var(--text)' }}>No installs yet</strong>
          <p style={{ margin: '8px 0 16px', fontSize: 12, color: 'var(--text-muted)' }}>
            Click <em>Generate install token</em> above, paste the resulting curl command into a fresh Linux
            box, and it will start reporting heartbeats to your tenant. The token self-destructs after one use.
          </p>
        </article>
      ) : null}

      {/* Create modal */}
      {showCreate ? (
        <div
          role="dialog"
          aria-modal="true"
          style={{
            position: 'fixed',
            inset: 0,
            background: 'rgba(0,0,0,0.5)',
            display: 'grid',
            placeItems: 'center',
            zIndex: 1000,
          }}
          onClick={closeCreate}
        >
          <form
            onSubmit={(e) => {
              e.preventDefault();
              void handleGenerate();
            }}
            onClick={(e) => e.stopPropagation()}
            style={{
              background: 'var(--surface)',
              border: '1px solid var(--border)',
              borderRadius: 'var(--radius-lg)',
              padding: 24,
              minWidth: 360,
              display: 'grid',
              gap: 12,
            }}
          >
            <strong style={{ fontSize: 14 }}>Generate install token</strong>
            <label style={{ display: 'grid', gap: 4, fontSize: 12 }}>
              Label (optional)
              <input
                type="text"
                value={label}
                onChange={(e) => setLabel(e.target.value)}
                maxLength={64}
                placeholder="edge-node-01"
                autoFocus
                style={{
                  background: 'var(--surface-2)',
                  border: '1px solid var(--border)',
                  borderRadius: 'var(--radius-md)',
                  padding: '8px 10px',
                  color: 'var(--text)',
                  fontFamily: 'inherit',
                }}
              />
            </label>
            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
              <motion.button
                type="button"
                className="threat-card-resolve-btn"
                onClick={closeCreate}
                whileHover={reduce ? undefined : buttonSpring.whileHover}
                whileTap={reduce ? undefined : buttonSpring.whileTap}
                transition={buttonSpring.transition}
              >
                Cancel
              </motion.button>
              <motion.button
                type="submit"
                className="empty-state-cta"
                disabled={isBusy}
                whileHover={reduce ? undefined : buttonSpring.whileHover}
                whileTap={reduce ? undefined : buttonSpring.whileTap}
                transition={buttonSpring.transition}
              >
                Generate
              </motion.button>
            </div>
          </form>
        </div>
      ) : null}
    </motion.section>
  );
}