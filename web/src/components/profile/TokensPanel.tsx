import { useEffect, useState } from 'react';
import { ApiError, api } from '../../lib/api';

interface ApiKey {
  id: string;
  name: string;
  prefix: string;
  created_at?: string;
  revoked_at?: string | null;
}

interface TokensPanelProps {
  open: boolean;
}

/**
 * TokensPanel — the inline API-tokens manager. Loaded lazily on
 * first expand. Reveals a new token once (then hides); supports
 * create + revoke.
 */
export default function TokensPanel({ open }: TokensPanelProps) {
  const [keys, setKeys] = useState<ApiKey[]>([]);
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState('');
  const [revealed, setRevealed] = useState<string | null>(null);
  // Inline error for token operations — surfaces just above the list
  // so the user sees it without dismissing the panel.
  const [error, setError] = useState<string | null>(null);

  function friendlyError(cause: unknown, fallback: string): string {
    if (cause instanceof ApiError) return cause.friendlyMessage;
    if (cause instanceof Error) return cause.message;
    return fallback;
  }

  async function loadKeys() {
    setLoading(true);
    setError(null);
    try {
      const data = await api<{ api_keys?: ApiKey[] }>('GET', '/api/v1/api-keys');
      setKeys(Array.isArray(data?.api_keys) ? data.api_keys : []);
    } catch (cause) {
      setError(friendlyError(cause, 'Could not load tokens.'));
      setKeys([]);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (open && keys.length === 0 && !loading) {
      void loadKeys();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  if (!open) return null;

  async function onCreate() {
    const trimmed = name.trim() || 'Unnamed token';
    setCreating(true);
    setError(null);
    try {
      const res = await api<{ token?: string }>('POST', '/api/v1/api-keys', { name: trimmed });
      if (res?.token) {
        setRevealed(res.token);
        try {
          await navigator.clipboard?.writeText(res.token);
        } catch {
          /* clipboard optional */
        }
      }
      setName('');
      await loadKeys();
    } catch (cause) {
      setError(friendlyError(cause, 'Could not create token.'));
    } finally {
      setCreating(false);
    }
  }

  async function onRevoke(id: string) {
    setError(null);
    try {
      await api('POST', `/api/v1/api-keys/${id}/revoke`);
      await loadKeys();
    } catch (cause) {
      setError(friendlyError(cause, 'Could not revoke token.'));
    }
  }

  return (
    <div className="prof-tokens-panel">
      <div className="prof-tokens-header">
        <p className="prof-tokens-help">
          Personal access tokens let CLI tools talk to the API without sharing your password.
          Each token is shown <strong>once</strong> when created - copy it then.
        </p>
      </div>
      {revealed && (
        <div className="prof-token-revealed">
          <span className="dash-eyebrow">New token (copy now)</span>
          <code className="prof-token-revealed-value">{revealed}</code>
          <div className="prof-token-revealed-actions">
            <button
              type="button"
              className="sw-button sw-button-quiet"
              onClick={async () => {
                try {
                  await navigator.clipboard?.writeText(revealed);
                } catch {
                  /* ignore */
                }
              }}
            >
              Copy
            </button>
            <button type="button" className="sw-button" onClick={() => setRevealed(null)}>
              I've saved it
            </button>
          </div>
        </div>
      )}
      {error && (
        <div className="prof-tokens-error" role="alert">{error}</div>
      )}
      <div className="prof-tokens-create">
        <input
          type="text"
          className="sw-input"
          placeholder="Token name (e.g. laptop-cli)"
          value={name}
          onChange={(e) => setName(e.target.value)}
          disabled={creating}
        />
        <button
          type="button"
          className="sw-button sw-button-primary"
          onClick={() => void onCreate()}
          disabled={creating}
        >
          {creating ? 'Creating…' : 'Create token'}
        </button>
      </div>
      {loading ? (
        <div className="prof-tokens-empty">Loading tokens…</div>
      ) : keys.length === 0 ? (
        <div className="prof-tokens-empty">No tokens yet. Create one above.</div>
      ) : (
        <ul className="prof-tokens-list">
          {keys.map((k) => (
            <li key={k.id} className={`prof-token-row ${k.revoked_at ? 'is-revoked' : ''}`}>
              <div className="prof-token-row-text">
                <strong>{k.name || 'Unnamed'}</strong>
                <code>{k.prefix}…</code>
                <small>Created {k.created_at ? new Date(k.created_at).toLocaleDateString() : '—'}</small>
              </div>
              <div className="prof-token-row-actions">
                {k.revoked_at ? (
                  <span className="prof-badge-bad">Revoked</span>
                ) : (
                  <button
                    type="button"
                    className="sw-button sw-button-quiet sw-button-danger"
                    onClick={() => void onRevoke(k.id)}
                  >
                    Revoke
                  </button>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
