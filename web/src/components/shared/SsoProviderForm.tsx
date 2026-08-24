import { useState, type FormEvent } from 'react';
import { motion, buttonSpring, useReducedMotion } from '../../lib/motion';

/**
 * SsoProviderForm — modal form body for creating a new SSO provider.
 * Extracted from SsoSection so that file stays under the 400-LOC cap.
 *
 * Renders either OIDC fields (name, client_id, client_secret,
 * discovery_url, redirect_uri, scopes) OR SAML fields (name,
 * entity_id, sso_url, metadata_url, metadata_xml, x509_cert) based
 * on the `type` prop. Owns local form state + submit handler;
 * delegates the actual API call to the parent's onSubmit so the
 * parent can refresh its providers list on success.
 */

export type ProviderType = 'oidc' | 'saml';

export interface SsoProviderDraft {
  type: ProviderType;
  name: string;
  // OIDC fields
  client_id?: string;
  client_secret?: string;
  discovery_url?: string;
  redirect_uri?: string;
  scopes?: string;
  // SAML fields
  entity_id?: string;
  sso_url?: string;
  metadata_url?: string;
  metadata_xml?: string;
  x509_cert?: string;
}

interface SsoProviderFormProps {
  type: ProviderType;
  onSubmit: (result: { draft: SsoProviderDraft; body: Record<string, unknown> }) => void | Promise<void>;
  onCancel: () => void;
  busy?: boolean;
}

const initialDraft = (type: ProviderType): SsoProviderDraft => {
  if (type === 'oidc') {
    return {
      type: 'oidc',
      name: '',
      client_id: '',
      client_secret: '',
      discovery_url: '',
      redirect_uri: '',
      scopes: 'openid,email,profile',
    };
  }
  return {
    type: 'saml',
    name: '',
    entity_id: '',
    sso_url: '',
    metadata_url: '',
    metadata_xml: '',
    x509_cert: '',
  };
};

function buildBody(d: SsoProviderDraft): Record<string, unknown> {
  const trimmedName = d.name.trim();
  if (d.type === 'oidc') {
    return {
      type: 'oidc',
      name: trimmedName,
      config: {
        client_id: (d.client_id || '').trim(),
        client_secret: (d.client_secret || '').trim(),
        discovery_url: (d.discovery_url || '').trim(),
        redirect_uri: (d.redirect_uri || '').trim(),
        scopes: (d.scopes || '')
          .split(',')
          .map((s) => s.trim())
          .filter(Boolean),
      },
    };
  }
  return {
    type: 'saml',
    name: trimmedName,
    config: {
      entity_id: (d.entity_id || '').trim(),
      sso_url: (d.sso_url || '').trim(),
      metadata_url: (d.metadata_url || '').trim(),
      metadata_xml: (d.metadata_xml || '').trim(),
      x509_cert: (d.x509_cert || '').trim(),
    },
  };
}

function validate(d: SsoProviderDraft): string {
  if (!d.name.trim()) return 'Display name is required.';
  if (d.type === 'oidc') {
    if (!d.client_id?.trim()) return 'client_id is required.';
    if (!d.client_secret?.trim()) return 'client_secret is required.';
    if (!d.discovery_url?.trim()) return 'discovery_url is required.';
  } else {
    if (!d.entity_id?.trim()) return 'entity_id is required.';
    if (!d.sso_url?.trim()) return 'sso_url is required.';
    if (!d.x509_cert?.trim()) return 'x509_cert is required.';
  }
  return '';
}

export default function SsoProviderForm({
  type,
  onSubmit,
  onCancel,
  busy,
}: SsoProviderFormProps) {
  const reduce = useReducedMotion();
  const [draft, setDraft] = useState<SsoProviderDraft>(() => initialDraft(type));
  const [error, setError] = useState('');

  const update = <K extends keyof SsoProviderDraft>(key: K, value: SsoProviderDraft[K]) => {
    setDraft((prev) => ({ ...prev, [key]: value }));
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    const msg = validate(draft);
    if (msg) {
      setError(msg);
      return;
    }
    setError('');
    // Build the API body from the typed draft (so callers don't
    // need to know the field names) and forward BOTH the draft +
    // body — the section uses draft.type/name and forwards body
    // as-is to api().
    const body = buildBody(draft);
    try {
      await onSubmit({ draft, body });
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Submit failed');
    }
  };

  return (
    <form className="notebook-create-form" onSubmit={handleSubmit}>
      <label>
        <span>Display name</span>
        <input
          type="text"
          required
          maxLength={64}
          value={draft.name}
          onChange={(e) => update('name', e.target.value)}
          placeholder="Okta corp"
          aria-label="Provider name"
        />
      </label>

      {type === 'oidc' ? (
        <>
          <label>
            <span>Client ID</span>
            <input
              type="text"
              required
              value={draft.client_id || ''}
              onChange={(e) => update('client_id', e.target.value)}
              placeholder="0oab1234cd…"
              aria-label="Client ID"
            />
          </label>
          <label>
            <span>Client secret (encrypted at rest)</span>
            <input
              type="password"
              required
              value={draft.client_secret || ''}
              onChange={(e) => update('client_secret', e.target.value)}
              placeholder="••••••••••"
              aria-label="Client secret"
            />
          </label>
          <label>
            <span>Discovery URL</span>
            <input
              type="url"
              required
              value={draft.discovery_url || ''}
              onChange={(e) => update('discovery_url', e.target.value)}
              placeholder="https://your-tenant.okta.com/.well-known/openid-configuration"
              aria-label="Discovery URL"
            />
          </label>
          <label>
            <span>Redirect URI (optional)</span>
            <input
              type="url"
              value={draft.redirect_uri || ''}
              onChange={(e) => update('redirect_uri', e.target.value)}
              placeholder="https://stackwatch.example.com/auth/sso-done"
              aria-label="Redirect URI"
            />
          </label>
          <label>
            <span>Scopes (comma-separated)</span>
            <input
              type="text"
              value={draft.scopes || ''}
              onChange={(e) => update('scopes', e.target.value)}
              placeholder="openid,email,profile"
              aria-label="Scopes"
            />
          </label>
        </>
      ) : (
        <>
          <label>
            <span>Entity ID</span>
            <input
              type="text"
              required
              value={draft.entity_id || ''}
              onChange={(e) => update('entity_id', e.target.value)}
              placeholder="https://stackwatch.example.com/saml/metadata"
              aria-label="Entity ID"
            />
          </label>
          <label>
            <span>SSO URL (IdP SSO endpoint)</span>
            <input
              type="url"
              required
              value={draft.sso_url || ''}
              onChange={(e) => update('sso_url', e.target.value)}
              placeholder="https://your-idp.example.com/saml2/sso"
              aria-label="SSO URL"
            />
          </label>
          <label>
            <span>Metadata URL (optional — paste XML below if blank)</span>
            <input
              type="url"
              value={draft.metadata_url || ''}
              onChange={(e) => update('metadata_url', e.target.value)}
              placeholder="https://your-idp.example.com/saml/metadata"
              aria-label="Metadata URL"
            />
          </label>
          <label>
            <span>Metadata XML (optional — if not using Metadata URL)</span>
            <textarea
              value={draft.metadata_xml || ''}
              onChange={(e) => update('metadata_xml', e.target.value)}
              placeholder="<EntityDescriptor>…</EntityDescriptor>"
              aria-label="Metadata XML"
              rows={4}
              style={{
                fontFamily:
                  'var(--mono, ui-monospace, SFMono-Regular, Menlo, monospace)',
                fontSize: 12,
              }}
            />
          </label>
          <label>
            <span>X509 cert (PEM, encrypted at rest)</span>
            <textarea
              required
              value={draft.x509_cert || ''}
              onChange={(e) => update('x509_cert', e.target.value)}
              placeholder="-----BEGIN CERTIFICATE-----…"
              aria-label="X509 cert"
              rows={4}
              style={{
                fontFamily:
                  'var(--mono, ui-monospace, SFMono-Regular, Menlo, monospace)',
                fontSize: 12,
              }}
            />
          </label>
        </>
      )}

      {error ? (
        <div className="dash-error" role="alert" style={{ fontSize: 12 }}>
          {error}
        </div>
      ) : null}

      <div className="incident-create-actions">
        <motion.button
          type="button"
          className="dash-icon-button"
          onClick={onCancel}
          whileHover={reduce ? undefined : buttonSpring.whileHover}
          whileTap={reduce ? undefined : buttonSpring.whileTap}
          transition={buttonSpring.transition}
          disabled={busy}
        >
          Cancel
        </motion.button>
        <motion.button
          type="submit"
          className="empty-state-cta"
          whileHover={reduce ? undefined : buttonSpring.whileHover}
          whileTap={reduce ? undefined : buttonSpring.whileTap}
          transition={buttonSpring.transition}
          disabled={busy || !draft.name.trim()}
        >
          Create provider
        </motion.button>
      </div>
    </form>
  );
}

export { buildBody };
