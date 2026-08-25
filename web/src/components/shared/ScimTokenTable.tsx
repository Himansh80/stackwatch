import { useMemo } from 'react';
import { motion, buttonSpring, useReducedMotion } from '../../lib/motion';

/**
 * ScimTokenTable — Tier 9.2 (Phase 2) Datadog-style table for SCIM
 * bearer tokens returned by GET /api/v1/enterprise/scim/tokens.
 *
 * Mirrors the scimTokenRow JSON shape (id, name, scopes, last_used_at,
 * expires_at, created_at). Layout:
 *   - Columns: name | scopes (badges) | last_used_at (relative time)
 *     | expires_at (relative time + red "expired" badge) | actions (Revoke)
 *   - "+ Create token" button → caller-provided onCreate callback
 *     (the actual modal lives in ScimSection so the file stays under
 *     the 400-LOC cap)
 *
 * This component is PURELY DISPLAY — it doesn't own the API calls.
 * The parent ScimSection owns the fetch + create + revoke lifecycle
 * and passes results down. That separation keeps the table reusable
 * elsewhere if Phase 6 ever exposes a per-org token editor.
 *
 * Tokens: --surface, --border, --accent, --green, --amber, --red,
 * --text-muted. No new CSS — reuses the .threat-card-* / .dash-status
 * classes that the existing Tier 6/7 surfaces already use.
 */

export interface ScimTokenRow {
  id: string;
  tenant_id: string;
  name: string;
  scopes: string[];
  expires_at?: string | null;
  last_used_at?: string | null;
  created_at: string;
}

interface ScimTokenTableProps {
  tokens: ScimTokenRow[];
  busy?: boolean;
  onCreate?: () => void;
  onRevoke?: (id: string) => void;
}

function relativeTime(iso: string | null | undefined): string {
  if (!iso) return '—';
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return iso;
  const diff = Math.max(0, Date.now() - t);
  const sec = Math.floor(diff / 1000);
  if (sec < 60) return `${sec}s ago`;
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min}m ago`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr}h ago`;
  const day = Math.floor(hr / 24);
  return `${day}d ago`;
}

function isExpired(iso: string | null | undefined): boolean {
  if (!iso) return false;
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return false;
  return t < Date.now();
}

const SCOPE_ACCENT: Record<string, 'green' | 'cyan' | 'indigo' | 'amber'> = {
  'users:read': 'cyan',
  'users:write': 'green',
  'groups:read': 'indigo',
  'groups:write': 'amber',
};

export default function ScimTokenTable({
  tokens,
  busy = false,
  onCreate,
  onRevoke,
}: ScimTokenTableProps) {
  const reduce = useReducedMotion();

  // Sort by created_at desc — newest first.
  const sorted = useMemo(
    () =>
      [...tokens].sort((a, b) => {
        const ta = new Date(a.created_at).getTime();
        const tb = new Date(b.created_at).getTime();
        return tb - ta;
      }),
    [tokens],
  );

  return (
    <div className="threat-card-list">
      {sorted.length === 0 ? (
        <div
          className="threat-card"
          style={{ textAlign: 'center', padding: 24, color: 'var(--text-muted)' }}
        >
          <strong style={{ color: 'var(--text)' }}>No SCIM tokens yet</strong>
          <p style={{ margin: '8px 0 16px', fontSize: 12 }}>
            Create a bearer token to let your IdP (Okta, Azure AD, Google Workspace) provision
            users via <code>POST /scim/v2/Users</code>.
          </p>
          {onCreate ? (
            <motion.button
              type="button"
              className="empty-state-cta"
              onClick={onCreate}
              whileHover={reduce ? undefined : buttonSpring.whileHover}
              whileTap={reduce ? undefined : buttonSpring.whileTap}
              transition={buttonSpring.transition}
              disabled={busy}
            >
              + Create token
            </motion.button>
          ) : null}
        </div>
      ) : (
        <>
          {sorted.map((t) => {
            const expired = isExpired(t.expires_at);
            return (
              <article key={t.id} className="threat-card">
                <div className="threat-card-top">
                  <span className="dash-sev dash-sev-medium">SCIM</span>
                  <strong className="threat-card-type">{t.name}</strong>
                  <span
                    className={`dash-status ${
                      expired ? 'dash-status-stale' : 'dash-status-up'
                    }`}
                  >
                    <span className="dash-status-dot" aria-hidden="true" />
                    {expired ? 'expired' : 'active'}
                  </span>
                </div>

                <p className="threat-card-desc">
                  scopes{' '}
                  {t.scopes.map((s) => (
                    <span
                      key={s}
                      className="dash-sev"
                      style={{
                        marginRight: 4,
                        fontSize: 10,
                        background:
                          SCOPE_ACCENT[s] === 'green'
                            ? 'var(--green)'
                            : SCOPE_ACCENT[s] === 'amber'
                              ? 'var(--amber)'
                              : SCOPE_ACCENT[s] === 'indigo'
                                ? 'var(--accent)'
                                : 'var(--accent-soft)',
                      }}
                    >
                      {s}
                    </span>
                  ))}
                </p>

                <div className="threat-card-meta">
                  <span title={t.last_used_at ?? ''}>
                    last used{' '}
                    <strong style={{ color: 'var(--text)' }}>
                      {relativeTime(t.last_used_at)}
                    </strong>
                  </span>
                  <span style={{ marginLeft: 12 }} title={t.expires_at ?? ''}>
                    expires{' '}
                    <strong style={{ color: expired ? 'var(--red)' : 'var(--text)' }}>
                      {relativeTime(t.expires_at)}
                    </strong>
                  </span>
                  {onRevoke ? (
                    <button
                      type="button"
                      className="threat-card-resolve-btn"
                      onClick={() => onRevoke(t.id)}
                      aria-label={`Revoke SCIM token ${t.name}`}
                      style={{ color: 'var(--red)', marginLeft: 'auto' }}
                      disabled={busy}
                    >
                      Revoke
                    </button>
                  ) : null}
                </div>
              </article>
            );
          })}
          {onCreate ? (
            <div style={{ marginTop: 12 }}>
              <motion.button
                type="button"
                className="empty-state-cta"
                onClick={onCreate}
                whileHover={reduce ? undefined : buttonSpring.whileHover}
                whileTap={reduce ? undefined : buttonSpring.whileTap}
                transition={buttonSpring.transition}
                disabled={busy}
              >
                + Create token
              </motion.button>
            </div>
          ) : null}
        </>
      )}
    </div>
  );
}