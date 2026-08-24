/**
 * SsoProviderCard — Tier 9.1 (Phase 1) Datadog-style card for a
 * single SSO provider row returned by GET /api/v1/enterprise/sso/providers.
 *
 * Mirrors the ssoProviderRow JSON shape (id, type, name, enabled,
 * config_summary, created_at). Layout:
 *   - Header: provider name + type badge (OIDC = blue, SAML = green)
 *     + enabled status pill
 *   - Click-to-expand config summary (OIDC: discovery_url preview,
 *     client_id masked; SAML: metadata_url preview, entity_id)
 *   - Actions row: Test / Edit / Delete buttons
 *   - Created_at relative time at the bottom
 *
 * Honors design tokens (--accent, --green, --red, --surface, --border,
 * --text-muted). No new CSS — reuses the threat-card-* classes that
 * the existing Tier 6/7 surfaces already use so visual rhythm is
 * consistent across the app.
 */
export interface SsoProviderRow {
  id: string;
  type: 'oidc' | 'saml' | string;
  name: string;
  enabled: boolean;
  config_summary: {
    client_id?: string;
    discovery_url?: string;
    redirect_uri?: string;
    scopes?: string[];
    entity_id?: string;
    sso_url?: string;
    metadata_url?: string;
    metadata_xml_length?: number;
  };
  created_at: string;
}

interface SsoProviderCardProps {
  provider: SsoProviderRow;
  onTest?: (id: string) => void;
  onEdit?: (id: string) => void;
  onDelete?: (id: string) => void;
}

function relativeTime(iso: string): string {
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

function maskClientId(id: string): string {
  // Show first 6 chars + ellipsis so admins can recognize their client_id
  // without us leaking the full value into screenshots / screen shares.
  if (id.length <= 10) return id;
  return `${id.slice(0, 6)}…${id.slice(-4)}`;
}

function previewUrl(u: string): string {
  try {
    const parsed = new URL(u);
    return `${parsed.host}${parsed.pathname.length > 24 ? parsed.pathname.slice(0, 24) + '…' : parsed.pathname}`;
  } catch {
    return u.length > 36 ? u.slice(0, 36) + '…' : u;
  }
}

export default function SsoProviderCard({
  provider,
  onTest,
  onEdit,
  onDelete,
}: SsoProviderCardProps) {
  const typ = (provider.type || '').toLowerCase();
  const toneClass =
    typ === 'saml'
      ? 'dash-sev dash-sev-low' // green-ish
      : 'dash-sev dash-sev-medium'; // blue-ish
  return (
    <article className="threat-card">
      <div className="threat-card-top">
        <span className={toneClass}>{provider.type.toUpperCase()}</span>
        <strong className="threat-card-type">{provider.name}</strong>
        <span
          className={`dash-status ${provider.enabled ? 'dash-status-up' : 'dash-status-stale'}`}
        >
          <span className="dash-status-dot" aria-hidden="true" />
          {provider.enabled ? 'enabled' : 'disabled'}
        </span>
      </div>

      <p className="threat-card-desc">
        {typ === 'oidc' ? (
          <>
            discovery <code style={{ color: 'var(--accent)' }}>{provider.config_summary.discovery_url ? previewUrl(provider.config_summary.discovery_url) : '—'}</code>
            {' · '}
            client_id <code>{provider.config_summary.client_id ? maskClientId(provider.config_summary.client_id) : '—'}</code>
            {provider.config_summary.scopes && provider.config_summary.scopes.length > 0 ? (
              <>
                {' · '}
                scopes <strong>{provider.config_summary.scopes.join(', ')}</strong>
              </>
            ) : null}
          </>
        ) : typ === 'saml' ? (
          <>
            entity_id <code style={{ color: 'var(--accent)' }}>{provider.config_summary.entity_id || '—'}</code>
            {' · '}
            sso_url <code>{provider.config_summary.sso_url ? previewUrl(provider.config_summary.sso_url) : '—'}</code>
            {provider.config_summary.metadata_url ? (
              <>
                {' · '}
                metadata <code>{previewUrl(provider.config_summary.metadata_url)}</code>
              </>
            ) : provider.config_summary.metadata_xml_length ? (
              <>
                {' · '}
                metadata_xml <strong>{provider.config_summary.metadata_xml_length.toLocaleString()} bytes</strong>
              </>
            ) : null}
          </>
        ) : (
          <>unknown provider type</>
        )}
      </p>

      <div className="threat-card-meta">
        {onTest ? (
          <button
            type="button"
            className="threat-card-resolve-btn"
            onClick={() => onTest(provider.id)}
            aria-label="Test SSO provider"
          >
            Test
          </button>
        ) : null}
        {onEdit ? (
          <button
            type="button"
            className="threat-card-resolve-btn"
            onClick={() => onEdit(provider.id)}
            aria-label="Edit SSO provider"
          >
            Edit
          </button>
        ) : null}
        {onDelete ? (
          <button
            type="button"
            className="threat-card-resolve-btn"
            onClick={() => onDelete(provider.id)}
            aria-label="Delete SSO provider"
            style={{ color: 'var(--red)' }}
          >
            Delete
          </button>
        ) : null}
        <span
          style={{
            marginLeft: 'auto',
            fontSize: 11,
            color: 'var(--text-muted)',
          }}
          title={provider.created_at}
        >
          created {relativeTime(provider.created_at)}
        </span>
      </div>
    </article>
  );
}
