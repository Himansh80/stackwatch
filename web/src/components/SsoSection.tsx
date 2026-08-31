import { useCallback, useEffect, useMemo, useState } from 'react';
import Button from './shared/Button';
import EmptyState from './shared/EmptyState';
import KpiCard from './shared/KpiCard';
import SsoProviderCard, { SsoProviderRow } from './shared/SsoProviderCard';
import SsoProviderForm, {
  ProviderType,
  SsoProviderDraft,
} from './shared/SsoProviderForm';
import { ApiError, api } from '../lib/api';
import { motion, kpiStagger } from '../lib/motion';

/**
 * SsoSection — Tier 9.1 (Phase 1) UI surface for the Single
 * Sign-On page section. Mirrors the NoiseReductionSection pattern
 * so the future EnterprisePage (Phase 6) can compose this in
 * without going over the 400-LOC cap.
 *
 * Layout:
 *   - 2 KpiCards: active providers / total connections (using
 *     kpiStagger for entrance + existing tokens for accent)
 *   - Provider list (SsoProviderCards, one per configured IdP)
 *   - "+ Add OIDC provider" / "+ Add SAML provider" buttons →
 *     modal hosting the extracted SsoProviderForm sub-component
 *
 * Motion: reuses existing exports (kpiStagger, buttonSpring) — no
 * new variants. Tokens: --surface, --border, --accent, --green,
 * --amber, --red via inline styles.
 */

interface SsoConnectionRow {
  id: string;
  provider_id: string;
  provider_name?: string;
  subject: string;
  created_at: string;
  last_used_at?: string | null;
}

interface SsoSectionProps {
  providers: SsoProviderRow[];
  connections: SsoConnectionRow[];
  busy: boolean;
  onError: (msg: string) => void;
  onChanged: () => void;
}

export default function SsoSection({
  providers,
  connections,
  busy,
  onError,
  onChanged,
}: SsoSectionProps) {
  const [sectionBusy, setSectionBusy] = useState(false);
  const isBusy = busy || sectionBusy;

  const [modalType, setModalType] = useState<ProviderType | null>(null);

  // KPIs derived locally.
  const counts = useMemo(() => {
    const active = providers.filter((p) => p.enabled).length;
    return { activeProviders: active, totalConnections: connections.length };
  }, [providers, connections]);

  const testProvider = useCallback(
    async (id: string) => {
      setSectionBusy(true);
      try {
        await api('POST', '/api/v1/enterprise/sso/test', { provider_id: id });
        onChanged();
      } catch (cause) {
        onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
      } finally {
        setSectionBusy(false);
      }
    },
    [onChanged, onError],
  );

  const deleteProvider = useCallback(
    async (id: string) => {
      const p = providers.find((x) => x.id === id);
      if (!p) return;
      if (!window.confirm(`Disable SSO provider "${p.name}"? (soft delete)`)) return;
      setSectionBusy(true);
      try {
        await api('DELETE', `/api/v1/enterprise/sso/providers/${id}`);
        onChanged();
      } catch (cause) {
        onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
      } finally {
        setSectionBusy(false);
      }
    },
    [providers, onChanged, onError],
  );

  // Edit is a placeholder — Phase 6 (EnterprisePage) wires the full
  // edit flow with PATCH + modal pre-fill. For Phase 1 we just
  // surface the button so the layout is complete.
  const editProvider = useCallback((id: string) => {
    window.alert(
      'Edit provider modal arrives in Phase 6 (EnterprisePage). Use the API directly for now.',
    );
    void id;
  }, []);

  const openModal = (typ: ProviderType) => setModalType(typ);
  const closeModal = useCallback(() => setModalType(null), []);

  const handleFormSubmit = useCallback(
    async (result: { draft: SsoProviderDraft; body: Record<string, unknown> }) => {
      setSectionBusy(true);
      try {
        // Forward the pre-built body (with config jsonb shape) to
        // the API. The form already validated + trimmed everything.
        await api('POST', '/api/v1/enterprise/sso/providers', result.body);
        closeModal();
        onChanged();
      } catch (cause) {
        onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
      } finally {
        setSectionBusy(false);
      }
    },
    [closeModal, onChanged, onError],
  );

  // Close modal on Escape.
  useEffect(() => {
    if (!modalType) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') closeModal();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [modalType, closeModal]);

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
          label="Active providers"
          value={counts.activeProviders}
          status={counts.activeProviders > 0 ? 'up' : 'neutral'}
          accent={counts.activeProviders > 0 ? 'green' : 'cyan'}
        />
        <KpiCard
          label="SSO connections"
          value={counts.totalConnections}
          status={counts.totalConnections > 0 ? 'up' : 'neutral'}
          accent="indigo"
        />
      </motion.div>

      <section className="dash-section">
        <span className="dash-eyebrow">Single Sign-On</span>
        <h2 className="dash-section-title">Identity providers</h2>
        <p
          className="dash-section-sub"
          style={{ color: 'var(--text-muted)', fontSize: 12, marginTop: 0 }}
        >
          Configure OIDC (Okta, Azure AD, Google Workspace) or SAML IdPs
          per tenant. Secrets are encrypted at rest — only the
          non-secret preview fields (client_id, discovery_url,
          entity_id) are returned by the list endpoint.
        </p>
        {providers.length === 0 ? (
          <EmptyState
            illustration={<span style={{ fontSize: 36 }}>🔐</span>}
            headline="No SSO providers configured"
            subhead="Add an OIDC or SAML IdP so your team can sign in with corporate credentials instead of separate passwords."
          />
        ) : (
          <div className="threat-card-list">
            {providers.map((p) => (
              <SsoProviderCard
                key={p.id}
                provider={p}
                onTest={(id) => void testProvider(id)}
                onEdit={editProvider}
                onDelete={(id) => void deleteProvider(id)}
              />
            ))}
          </div>
        )}

        <div style={{ marginTop: 12, display: 'flex', gap: 8 }}>
          <Button
            variant="primary"
            size="sm"
            onClick={() => openModal('oidc')}
            disabled={isBusy}
          >
            + Add OIDC provider
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => openModal('saml')}
            disabled={isBusy}
          >
            + Add SAML provider
          </Button>
        </div>
      </section>

      {modalType ? (
        <div className="slow-query-explain-modal" role="dialog" aria-modal="true">
          <div className="slow-query-explain-modal-head">
            <strong>New {modalType.toUpperCase()} provider</strong>
            <button
              type="button"
              className="slow-query-explain-close"
              onClick={closeModal}
              aria-label="Close"
            >
              ✕
            </button>
          </div>
          <SsoProviderForm
            type={modalType}
            onCancel={closeModal}
            onSubmit={handleFormSubmit}
            busy={isBusy}
          />
        </div>
      ) : null}
    </>
  );
}
