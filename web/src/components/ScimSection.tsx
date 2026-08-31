import { useCallback, useEffect, useMemo, useState } from 'react';
import KpiCard from './shared/KpiCard';
import CopyBox from './shared/CopyBox';
import ScimTokenTable, { ScimTokenRow } from './shared/ScimTokenTable';
import Button from './shared/Button';
import Input from './shared/Input';
import Modal from './shared/Modal';
import { ApiError, api } from '../lib/api';
import { motion, kpiStagger } from '../lib/motion';

/**
 * ScimSection — Tier 9.2 (Phase 2) UI surface for the SCIM
 * Provisioning page section. Mirrors the SsoSection pattern so the
 * future EnterprisePage (Phase 6) can compose this in without going
 * over the 400-LOC cap.
 *
 * Layout:
 *   - 3 KpiCards: active tokens / total users provisioned / recent
 *     sync errors (using kpiStagger for entrance + existing tokens
 *     for accent).
 *   - ScimTokenTable: Datadog-style cards for each SCIM bearer token.
 *   - "Create token" modal hosting a name + scopes multi-select.
 *     On success the plaintext token is shown ONCE in a copy-able
 *     warning box (via the shared CopyBox component).
 *   - Instructions card explaining how to wire the IdP (URL +
 *     "use the bearer token you generated above").
 *
 * Motion: reuses existing exports (kpiStagger) — no new variants.
 * Tokens: --surface, --border, --accent, --green, --amber, --red
 * via inline styles.
 *
 * Tier 20 Phase E: refactored to use shared Modal + Button + Input.
 */

const SCOPE_OPTIONS: { value: string; label: string; description: string }[] = [
  { value: 'users:read', label: 'users:read', description: 'Read users via GET /scim/v2/Users' },
  { value: 'users:write', label: 'users:write', description: 'Create / update / disable users' },
  { value: 'groups:read', label: 'groups:read', description: 'Read groups (Phase 2 stub)' },
  { value: 'groups:write', label: 'groups:write', description: 'Modify groups (Phase 2 stub)' },
];

interface ScimSectionProps {
  tokens: ScimTokenRow[];
  usersCount: number;
  recentErrors: number;
  busy?: boolean;
  onError: (msg: string) => void;
  onChanged: () => void;
}

interface CreateFormState {
  name: string;
  scopes: string[];
}

function initialForm(): CreateFormState {
  return { name: '', scopes: ['users:read', 'users:write'] };
}

export default function ScimSection({
  tokens,
  usersCount,
  recentErrors,
  busy = false,
  onError,
  onChanged,
}: ScimSectionProps) {
  const [sectionBusy, setSectionBusy] = useState(false);
  const isBusy = busy || sectionBusy;
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState<CreateFormState>(initialForm);
  const [justCreatedToken, setJustCreatedToken] = useState<string | null>(null);

  // KPIs derived locally.
  const activeTokens = useMemo(() => {
    const now = Date.now();
    return tokens.filter((t) => {
      if (!t.expires_at) return true;
      return new Date(t.expires_at).getTime() > now;
    }).length;
  }, [tokens]);

  const revokeToken = useCallback(
    async (id: string) => {
      const t = tokens.find((x) => x.id === id);
      if (!t) return;
      if (!window.confirm(`Revoke SCIM token "${t.name}"? (cannot be undone)`)) return;
      setSectionBusy(true);
      try {
        await api('DELETE', `/api/v1/enterprise/scim/tokens/${id}`);
        onChanged();
      } catch (cause) {
        onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
      } finally {
        setSectionBusy(false);
      }
    },
    [tokens, onChanged, onError],
  );

  const openCreate = useCallback(() => {
    setForm(initialForm());
    setJustCreatedToken(null);
    setShowCreate(true);
  }, []);
  const closeCreate = useCallback(() => {
    setShowCreate(false);
    setJustCreatedToken(null);
    setForm(initialForm());
  }, []);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (!form.name.trim()) {
        onError('Token name is required.');
        return;
      }
      setSectionBusy(true);
      try {
        const res = await api<{ plaintext_token: string }>(
          'POST',
          '/api/v1/enterprise/scim/tokens',
          { name: form.name.trim(), scopes: form.scopes },
        );
        setJustCreatedToken(res.plaintext_token);
        onChanged();
      } catch (cause) {
        onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
      } finally {
        setSectionBusy(false);
      }
    },
    [form, onChanged, onError],
  );

  const toggleScope = useCallback((scope: string) => {
    setForm((f) => ({
      ...f,
      scopes: f.scopes.includes(scope)
        ? f.scopes.filter((s) => s !== scope)
        : [...f.scopes, scope],
    }));
  }, []);

  // Close modal on Escape (only when no fresh token is showing).
  useEffect(() => {
    if (!showCreate) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !justCreatedToken) closeCreate();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [showCreate, justCreatedToken, closeCreate]);

  return (
    <>
      <motion.div
        className="dash-metric-strip"
        initial="hidden"
        animate="show"
        variants={kpiStagger}
        style={{ marginTop: 16 }}
      >
        <KpiCard
          label="Active tokens"
          value={activeTokens}
          status={activeTokens > 0 ? 'up' : 'neutral'}
          accent={activeTokens > 0 ? 'green' : 'cyan'}
        />
        <KpiCard
          label="Users provisioned"
          value={usersCount}
          status={usersCount > 0 ? 'up' : 'neutral'}
          accent="indigo"
        />
        <KpiCard
          label="Recent sync errors"
          value={recentErrors}
          status={recentErrors > 0 ? 'crit' : 'up'}
          accent={recentErrors > 0 ? 'red' : 'green'}
        />
      </motion.div>

      <section className="dash-section">
        <span className="dash-eyebrow">SCIM Provisioning</span>
        <h2 className="dash-section-title">Bearer tokens</h2>
        <p
          className="dash-section-sub"
          style={{ color: 'var(--text-muted)', fontSize: 12, marginTop: 0 }}
        >
          SCIM 2.0 lets your IdP (Okta, Azure AD, Google Workspace)
          provision users automatically. Create a bearer token below,
          paste it into your IdP&apos;s SCIM configuration, and we&apos;ll
          receive <code>POST /scim/v2/Users</code> requests whenever
          someone is added or removed from your directory.
        </p>

        <ScimTokenTable
          tokens={tokens}
          busy={isBusy}
          onCreate={openCreate}
          onRevoke={(id) => void revokeToken(id)}
        />
      </section>

      <section className="dash-section">
        <span className="dash-eyebrow">IdP configuration</span>
        <h2 className="dash-section-title">How to wire your IdP</h2>
        <article className="threat-card">
          <p className="threat-card-desc">
            Use this URL in your IdP&apos;s SCIM connector:
          </p>
          <div
            style={{
              fontFamily: 'var(--font-mono, monospace)',
              fontSize: 12,
              background: 'var(--surface-2)',
              border: '1px solid var(--border)',
              borderRadius: 'var(--radius-md)',
              padding: 12,
              color: 'var(--accent)',
            }}
          >
            {window.location.protocol}//{window.location.host}/scim/v2/
          </div>
          <p className="threat-card-desc" style={{ marginTop: 12 }}>
            Authenticate with the bearer token you generated above.
            Map the SCIM <code>userName</code> / <code>emails</code> to
            your IdP&apos;s email field. Every successful or failed
            operation is recorded in <code>scim_sync_log</code>.
          </p>
        </article>
      </section>

      <Modal
        open={showCreate}
        onClose={closeCreate}
        title="New SCIM token"
        closeOnBackdrop={justCreatedToken === null}
        closeOnEscape={justCreatedToken === null}
      >
        {justCreatedToken ? (
          <>
            <p
              style={{
                color: 'var(--amber)',
                fontSize: 13,
                margin: '0 0 var(--space-3)',
              }}
            >
              Copy this token now — we&apos;ll never show it again.
            </p>
            <CopyBox value={justCreatedToken} />
            <div
              style={{
                marginTop: 'var(--space-3)',
                display: 'flex',
                justifyContent: 'flex-end',
              }}
            >
              <Button type="button" variant="primary" onClick={closeCreate}>
                Done
              </Button>
            </div>
          </>
        ) : (
          <form onSubmit={handleSubmit}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
              <Input
                label="Token name"
                value={form.name}
                onChange={(e) =>
                  setForm((f) => ({ ...f, name: e.target.value }))
                }
                placeholder="Okta SCIM, Azure AD SCIM, ..."
                maxLength={64}
                required
                fullWidth
              />
              <div>
                <div
                  style={{
                    fontSize: 14,
                    fontWeight: 500,
                    color: 'var(--text)',
                    marginBottom: 'var(--space-1)',
                  }}
                >
                  Scopes
                </div>
                <div
                  style={{
                    display: 'flex',
                    flexDirection: 'column',
                    gap: 'var(--space-2)',
                  }}
                >
                  {SCOPE_OPTIONS.map((opt) => (
                    <label
                      key={opt.value}
                      style={{
                        display: 'flex',
                        gap: 'var(--space-2)',
                        alignItems: 'flex-start',
                        fontSize: 13,
                        cursor: 'pointer',
                      }}
                    >
                      <input
                        type="checkbox"
                        checked={form.scopes.includes(opt.value)}
                        onChange={() => toggleScope(opt.value)}
                        style={{ marginTop: 3 }}
                      />
                      <span>
                        <strong style={{ color: 'var(--text)' }}>{opt.label}</strong>
                        <br />
                        <span style={{ color: 'var(--text-muted)' }}>
                          {opt.description}
                        </span>
                      </span>
                    </label>
                  ))}
                </div>
              </div>
              <div
                style={{
                  display: 'flex',
                  gap: 'var(--space-2)',
                  justifyContent: 'flex-end',
                  marginTop: 'var(--space-2)',
                }}
              >
                <Button
                  type="button"
                  variant="secondary"
                  onClick={closeCreate}
                  disabled={isBusy}
                >
                  Cancel
                </Button>
                <Button
                  type="submit"
                  variant="primary"
                  disabled={isBusy}
                  loading={isBusy}
                >
                  {isBusy ? 'Creating…' : 'Create token'}
                </Button>
              </div>
            </div>
          </form>
        )}
      </Modal>
    </>
  );
}
