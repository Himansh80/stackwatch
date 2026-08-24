import { FormEvent, useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { ApiError, api, clearToken, getToken, me } from '../lib/api';
import ProfileMenu from '../components/ProfileMenu';
import CommandPalette from '../components/CommandPalette';
import TimeWidget from '../components/TimeWidget';

type Profile = {
  email: string;
  full_name: string;
  avatar_url: string;
  role: string;
  status: string;
  user_id: string;
  tenant_id: string;
  tenant_name: string;
  tenant_slug: string;
  tenant_plan: string;
  created_at?: string;
  last_login_at?: string;
};

const emptyProfile: Profile = {
  email: '',
  full_name: '',
  avatar_url: '',
  role: '',
  status: '',
  user_id: '',
  tenant_id: '',
  tenant_name: '',
  tenant_slug: '',
  tenant_plan: '',
};

/**
 * Compute a stable accent color for the avatar based on the email —
 * gives each user a personal color even when full_name is generic.
 * Same algo as the topbar ProfileMenu so the two avatars match.
 */
function avatarColor(seed: string): string {
  let hash = 0;
  for (let i = 0; i < seed.length; i += 1) {
    hash = (hash * 31 + seed.charCodeAt(i)) >>> 0;
  }
  const palette = [
    'linear-gradient(135deg,#38bdf8,#22d3ee)', // cyan
    'linear-gradient(135deg,#a78bfa,#818cf8)', // violet
    'linear-gradient(135deg,#fb923c,#f59e0b)', // amber
    'linear-gradient(135deg,#34d399,#10b981)', // emerald
    'linear-gradient(135deg,#f472b6,#ec4899)', // pink
    'linear-gradient(135deg,#60a5fa,#3b82f6)', // blue
  ];
  return palette[hash % palette.length] || palette[0];
}

/**
 * Role tone — visual treatment of the role badge. super_admin gets
 * the warm "elevated" tone; admin/operator get blue; everything else
 * stays neutral. Matches the sidebar nav item styling language.
 */
function roleTone(role: string): 'admin' | 'operator' | 'viewer' | 'neutral' {
  const lower = role.toLowerCase();
  if (lower === 'super_admin' || lower === 'owner') return 'admin';
  if (lower === 'admin' || lower === 'operator') return 'operator';
  if (lower === 'viewer' || lower === 'member') return 'viewer';
  return 'neutral';
}

/**
 * Copy `text` to the clipboard. Falls back to a manual selection
 * if the async Clipboard API isn't available (older browsers, http
 * origins without user gesture).
 */
async function copyToClipboard(text: string): Promise<boolean> {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text);
      return true;
    }
    const range = document.createRange();
    const sel = window.getSelection();
    const el = document.createElement('textarea');
    el.value = text;
    el.style.position = 'fixed';
    el.style.opacity = '0';
    document.body.appendChild(el);
    range.selectNodeContents(el);
    if (sel) { sel.removeAllRanges(); sel.addRange(range); }
    document.execCommand('copy');
    document.body.removeChild(el);
    if (sel) sel.removeAllRanges();
    return true;
  } catch {
    return false;
  }
}

/**
 * Read a File from the browser File API, render it into a
 * hidden canvas, resize it to fit within maxSide x maxSide
 * preserving aspect ratio, and re-export as JPEG.
 *
 * Why the resize: storing full-resolution selfies as data
 * URLs would blow past the 500KB backend cap quickly. A
 * 256x256 JPEG at quality 0.7 is typically 10-25KB - small
 * enough to send in a single PATCH, sharp enough for an
 * avatar in any UI context.
 */
async function fileToResizedDataUrl(file: File, maxSide = 256, quality = 0.7): Promise<string> {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file);
    const img = new Image();
    img.onload = () => {
      try {
        const ratio = Math.min(1, maxSide / Math.max(img.width, img.height));
        const w = Math.max(1, Math.round(img.width * ratio));
        const h = Math.max(1, Math.round(img.height * ratio));
        const canvas = document.createElement('canvas');
        canvas.width = w;
        canvas.height = h;
        const ctx = canvas.getContext('2d');
        if (!ctx) throw new Error('canvas 2d unavailable');
        ctx.drawImage(img, 0, 0, w, h);
        const dataUrl = canvas.toDataURL('image/jpeg', quality);
        URL.revokeObjectURL(url);
        resolve(dataUrl);
      } catch (err) {
        URL.revokeObjectURL(url);
        reject(err);
      }
    };
    img.onerror = (err) => {
      URL.revokeObjectURL(url);
      reject(err);
    };
    img.src = url;
  });
}

export default function ProfilePage() {
  const nav = useNavigate();
  const [profile, setProfile] = useState<Profile>(emptyProfile);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [copiedField, setCopiedField] = useState<string | null>(null);

  // Name-edit form state
  const [nameDraft, setNameDraft] = useState('');
  const [nameSaving, setNameSaving] = useState(false);
  const [nameMessage, setNameMessage] = useState<string | null>(null);
  const [nameError, setNameError] = useState<string | null>(null);

  // Avatar upload state
  const [avatarUploading, setAvatarUploading] = useState(false);
  const [avatarError, setAvatarError] = useState<string | null>(null);

  // Inline panels (so "API tokens" + "2FA" expand on the page
  // instead of bouncing to /settings where they don't exist yet).
  const [showTokens, setShowTokens] = useState(false);
  const [apiKeys, setApiKeys] = useState<Array<{ id: string; name: string; prefix: string; created_at?: string; revoked_at?: string | null }>>([]);
  const [tokensLoading, setTokensLoading] = useState(false);
  const [creatingToken, setCreatingToken] = useState(false);
  const [newTokenName, setNewTokenName] = useState('');
  const [revealedToken, setRevealedToken] = useState<string | null>(null);

  // Cmd+K palette open state — same pattern as Dashboard.
  const [paletteOpen, setPaletteOpen] = useState(false);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      if (!getToken()) {
        setError('You are not signed in.');
        setLoading(false);
        return;
      }
      try {
        const data = await me();
        if (cancelled) return;
        setProfile({
          email: data?.user?.email ?? '',
          full_name: data?.user?.full_name ?? '',
          avatar_url: data?.user?.avatar_url ?? '',
          role: data?.user?.role ?? '',
          status: data?.user?.status ?? '',
          user_id: data?.user?.id ?? '',
          tenant_id: data?.tenant?.id ?? '',
          tenant_name: data?.tenant?.name ?? '',
          tenant_slug: data?.tenant?.slug ?? '',
          tenant_plan: data?.tenant?.plan ?? '',
          created_at: data?.user?.created_at ?? '',
          last_login_at: data?.user?.last_login_at ?? '',
        });
      } catch (cause) {
        if (cancelled) return;
        if (cause instanceof ApiError) {
          setError(cause.friendlyMessage);
        } else {
          setError(cause instanceof Error ? cause.message : 'Unable to load your profile.');
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  // Keep the name form in sync with the loaded profile.
  useEffect(() => {
    setNameDraft(profile.full_name);
  }, [profile.full_name]);

  // Cmd+K / Ctrl+K opens the global command palette.
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        setPaletteOpen(true);
      }
      if (event.key === 'Escape' && paletteOpen) {
        setPaletteOpen(false);
      }
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [paletteOpen]);

  async function onSaveName(event: FormEvent) {
    event.preventDefault();
    setNameMessage(null);
    setNameError(null);
    const trimmed = nameDraft.trim();
    if (!trimmed) {
      setNameError('Please enter your name.');
      return;
    }
    if (trimmed === profile.full_name) {
      setNameMessage('No changes to save.');
      return;
    }
    setNameSaving(true);
    try {
      await api('PATCH', '/api/v1/auth/profile', { full_name: trimmed });
      setProfile((p) => ({ ...p, full_name: trimmed }));
      setNameMessage('Saved.');
    } catch (cause) {
      if (cause instanceof ApiError) {
        setNameError(cause.friendlyMessage);
      } else {
        setNameError(cause instanceof Error ? cause.message : 'Could not save your name.');
      }
    } finally {
      setNameSaving(false);
    }
  }

  async function onPickAvatar(event: FormEvent<HTMLInputElement>) {
    const file = event.currentTarget.files?.[0];
    event.currentTarget.value = ''; // reset so picking the same file twice fires onChange again
    if (!file) return;
    if (!file.type.startsWith('image/')) {
      setAvatarError('Please choose an image file (PNG, JPG, GIF, WebP).');
      return;
    }
    setAvatarError(null);
    setAvatarUploading(true);
    try {
      const dataUrl = await fileToResizedDataUrl(file, 256, 0.7);
      await api('PATCH', '/api/v1/auth/profile', { avatar_url: dataUrl });
      setProfile((p) => ({ ...p, avatar_url: dataUrl }));
    } catch (cause) {
      if (cause instanceof ApiError) {
        setAvatarError(cause.friendlyMessage);
      } else {
        setAvatarError(cause instanceof Error ? cause.message : 'Could not upload your photo.');
      }
    } finally {
      setAvatarUploading(false);
    }
  }

  async function onRemoveAvatar() {
    if (!profile.avatar_url) return;
    // Backend treats empty string as "no change" so we use a sentinel
    // data URL to clear the field - the avatar_url column is plain
    // text and accepts an empty string as a valid value.
    setAvatarUploading(true);
    setAvatarError(null);
    try {
      await api('PATCH', '/api/v1/auth/profile', { avatar_url: '' });
      setProfile((p) => ({ ...p, avatar_url: '' }));
    } catch (cause) {
      if (cause instanceof ApiError) {
        setAvatarError(cause.friendlyMessage);
      } else {
        setAvatarError(cause instanceof Error ? cause.message : 'Could not remove your photo.');
      }
    } finally {
      setAvatarUploading(false);
    }
  }

  async function loadApiKeys() {
    setTokensLoading(true);
    try {
      const data = await api<{ api_keys?: Array<{ id: string; name: string; prefix: string; created_at?: string; revoked_at?: string | null }> }>('GET', '/api/v1/api-keys');
      setApiKeys(Array.isArray(data?.api_keys) ? data.api_keys : []);
    } catch (cause) {
      if (cause instanceof ApiError) {
        setAvatarError(cause.friendlyMessage);
      }
      setApiKeys([]);
    } finally {
      setTokensLoading(false);
    }
  }

  async function onCreateToken() {
    const name = newTokenName.trim() || 'Unnamed token';
    setCreatingToken(true);
    setAvatarError(null);
    try {
      const res = await api<{ token?: string; api_key?: { id: string; name: string; prefix: string } }>(
        'POST', '/api/v1/api-keys', { name },
      );
      // The token is only returned on creation - copy to clipboard
      // and reveal it inline so the user can paste it into their tool.
      if (res?.token) {
        setRevealedToken(res.token);
        try { await navigator.clipboard?.writeText(res.token); } catch { /* clipboard optional */ }
      }
      setNewTokenName('');
      await loadApiKeys();
    } catch (cause) {
      if (cause instanceof ApiError) {
        setAvatarError(cause.friendlyMessage);
      } else {
        setAvatarError(cause instanceof Error ? cause.message : 'Could not create token.');
      }
    } finally {
      setCreatingToken(false);
    }
  }

  async function onRevokeToken(id: string) {
    setAvatarError(null);
    try {
      await api('POST', `/api/v1/api-keys/${id}/revoke`);
      await loadApiKeys();
    } catch (cause) {
      if (cause instanceof ApiError) {
        setAvatarError(cause.friendlyMessage);
      } else {
        setAvatarError(cause instanceof Error ? cause.message : 'Could not revoke token.');
      }
    }
  }

  function onToggleTokens() {
    const next = !showTokens;
    setShowTokens(next);
    if (next && apiKeys.length === 0 && !tokensLoading) {
      void loadApiKeys();
    }
  }

  async function onCopy(value: string, field: string) {
    const ok = await copyToClipboard(value);
    if (ok) {
      setCopiedField(field);
      window.setTimeout(() => {
        setCopiedField((current) => (current === field ? null : current));
      }, 1600);
    }
  }

  function logout() {
    clearToken();
    nav('/login');
  }

  const initials = (profile.full_name || profile.email || '?').slice(0, 1).toUpperCase();
  const tone = roleTone(profile.role);

  return (
    <div className="dash-app">
      <aside className="dash-sidebar">
        <Link className="dash-brand" to="/dashboard">
          <span className="dash-brand-mark">S</span>
          <span><strong>StackWatch</strong><small>Infrastructure control plane</small></span>
        </Link>
        <div className="dash-nav-section"><span className="dash-nav-heading">Workspace</span>
          <Link className="dash-nav-item" to="/dashboard"><span className="dash-nav-icon">⌂</span><span className="dash-nav-label">Overview</span></Link>
          <Link className="dash-nav-item dash-nav-active" to="/profile"><span className="dash-nav-icon">◉</span><span className="dash-nav-label">Profile</span></Link>
          <Link className="dash-nav-item" to="/billing"><span className="dash-nav-icon">$</span><span className="dash-nav-label">Billing</span></Link>
          <Link className="dash-nav-item" to="/settings"><span className="dash-nav-icon">⚙</span><span className="dash-nav-label">Settings</span></Link>
        </div>
        <div className="dash-nav-section"><span className="dash-nav-heading">Infrastructure</span>
          <Link className="dash-nav-item" to="/proxmox"><span className="dash-nav-icon">◈</span><span className="dash-nav-label">Proxmox</span></Link>
          <Link className="dash-nav-item" to="/truenas"><span className="dash-nav-icon">▤</span><span className="dash-nav-label">TrueNAS</span></Link>
        </div>
        <div className="dash-sidebar-bottom">
          <button className="dash-sidebar-logout" onClick={logout}>↪ Sign out</button>
        </div>
      </aside>
      <main className="dash-main">
        <header className="dash-topbar">
          <div className="dash-greeting">
            <span className="dash-greeting-eyebrow">Your account</span>
            <div className="dash-greeting-row">
              <strong className="dash-greeting-text">{profile.full_name || 'Profile'}</strong>
              <span className="dash-greeting-clock">
                <span className="dash-greeting-clock-time">{profile.email || 'no email on file'}</span>
              </span>
            </div>
          </div>
          <div className="dash-topbar-center"><button className="dash-topbar-search" onClick={() => setPaletteOpen(true)} title="Search & navigate (Cmd+K)" aria-label="Open command palette"><span className="dash-topbar-search-icon" aria-hidden="true">⌕</span><span className="dash-topbar-search-placeholder">Search & navigate…</span><kbd className="dash-topbar-search-kbd">⌘</kbd><kbd className="dash-topbar-search-kbd">K</kbd></button></div><div className="dash-top-actions"><TimeWidget />
            <button className="dash-icon-button" onClick={() => window.location.reload()} aria-label="Refresh page" title="Refresh page">↻</button>
            <ProfileMenu firstName={(profile.full_name || '').split(' ')[0] || 'there'} fullName={profile.full_name} tenantName={profile.tenant_name} initials={initials} avatarUrl={profile.avatar_url} />
          </div>
        </header>
        <div className="dash-content">
          {error && (
            <div className="dash-error">
              <strong>Could not load your profile</strong>
              <span>{error}</span>
              <button onClick={() => window.location.reload()}>Retry</button>
            </div>
          )}
          {loading ? (
            <div className="prof-skeleton-stack">
              <div className="prof-skel prof-skel-hero" />
              <div className="prof-skel-row">
                <div className="prof-skel prof-skel-card" />
                <div className="prof-skel prof-skel-card" />
              </div>
            </div>
          ) : (
            <>
              {/* HERO CARD — avatar + name + status pills + copy-id action.
                  Avatar is now a real upload control: click the camera
                  icon or the avatar itself to pick a file. Stored as
                  a 256x256 JPEG data URL on the backend. */}
              <section className="prof-hero" aria-labelledby="prof-hero-name">
                <div className="prof-hero-avatar-wrap">
                  <label
                    className={`prof-hero-avatar ${profile.avatar_url ? 'has-image' : ''}`}
                    style={profile.avatar_url ? undefined : { background: avatarColor(profile.email || profile.user_id) }}
                    aria-hidden="true"
                  >
                    {profile.avatar_url ? (
                      <img src={profile.avatar_url} alt="" />
                    ) : (
                      <span>{initials}</span>
                    )}
                    <span className="prof-hero-avatar-overlay" aria-hidden="true">📷</span>
                    {avatarUploading && <span className="prof-hero-avatar-spinner" aria-hidden="true" />}
                  </label>
                  <input
                    type="file"
                    accept="image/*"
                    className="prof-hero-avatar-input"
                    onChange={(e) => void onPickAvatar(e)}
                    disabled={avatarUploading}
                    aria-label="Upload profile photo"
                  />
                  {profile.avatar_url && !avatarUploading && (
                    <button
                      type="button"
                      className="prof-hero-avatar-remove"
                      onClick={() => void onRemoveAvatar()}
                      aria-label="Remove profile photo"
                      title="Remove photo"
                    >×</button>
                  )}
                </div>
                <div className="prof-hero-body">
                  <span className="dash-eyebrow">Account</span>
                  <h2 id="prof-hero-name" className="prof-hero-name">{profile.full_name || '—'}</h2>
                  <div className="prof-hero-meta">
                    <span className={`prof-role prof-role-${tone}`}>{profile.role || 'no role'}</span>
                    <span className={`prof-status prof-status-${profile.status === 'active' ? 'active' : 'inactive'}`}>
                      <span className="prof-status-dot" />
                      {profile.status || 'unknown'}
                    </span>
                    <span className="prof-hero-plan">{profile.tenant_plan || 'free'} plan</span>
                  </div>
                  <div className="prof-hero-email">{profile.email || 'no email'}</div>
                  {avatarError && <div className="prof-hero-avatar-error">{avatarError}</div>}
                </div>
                <div className="prof-hero-actions">
                  <a className="sw-button sw-button-quiet" href="/billing">Manage plan</a>
                </div>
              </section>

              {/* Two-column layout — identity (left) + session (right). The
                  identity column shows the technical IDs the user
                  occasionally needs to copy (user_id, tenant_id, slugs).
                  Each row has its own copy button. */}
              <section className="prof-grid-main">
                <article className="dash-panel prof-panel">
                  <div className="dash-panel-head">
                    <div>
                      <span className="dash-eyebrow">Identity</span>
                      <h3>Technical identifiers</h3>
                    </div>
                    <span className="dash-panel-context">Needed for API calls and scripts</span>
                  </div>
                  <div className="prof-id-list">
                    {[
                      { key: 'user_id', label: 'User ID', value: profile.user_id, mono: true },
                      { key: 'tenant_id', label: 'Tenant ID', value: profile.tenant_id, mono: true },
                      { key: 'tenant_slug', label: 'Workspace slug', value: profile.tenant_slug, mono: true },
                      { key: 'email', label: 'Email address', value: profile.email, mono: false },
                    ].map((row) => (
                      <div key={row.key} className="prof-id-row">
                        <span className="prof-id-label">{row.label}</span>
                        <span className={`prof-id-value ${row.mono ? 'prof-id-mono' : ''}`}>
                          {row.value || '—'}
                        </span>
                        <button
                          type="button"
                          className="prof-id-copy"
                          onClick={() => row.value && void onCopy(row.value, row.key)}
                          disabled={!row.value}
                          aria-label={`Copy ${row.label}`}
                          title={row.value ? 'Copy to clipboard' : 'No value to copy'}
                        >
                          {copiedField === row.key ? '✓ Copied' : '⧉ Copy'}
                        </button>
                      </div>
                    ))}
                  </div>
                </article>

                <article className="dash-panel prof-panel">
                  <div className="dash-panel-head">
                    <div>
                      <span className="dash-eyebrow">Workspace</span>
                      <h3>{profile.tenant_name || 'Your workspace'}</h3>
                    </div>
                    <span className="prof-plan-badge">{profile.tenant_plan || 'free'}</span>
                  </div>
                  <div className="prof-workspace-body">
                    <Stat label="Plan" value={profile.tenant_plan || 'free'} hint="Subscription tier" tone="cyan" />
                    <Stat label="Workspace" value={profile.tenant_name || '—'} hint={`slug · ${profile.tenant_slug || '—'}`} tone="indigo" />
                    <Stat label="Status" value={profile.status || 'unknown'} hint="Account state" tone={profile.status === 'active' ? 'green' : 'amber'} />
                    <Stat label="Role" value={profile.role || '—'} hint="Permission level" tone={tone === 'admin' ? 'amber' : 'cyan'} />
                  </div>
                </article>
              </section>

              {/* EDIT NAME + SECURITY — two side-by-side panels.
                  The name-edit uses the polished .sw-form-* classes so
                  it matches the rest of the dashboard. The security
                  panel shows what we can detect about the user's
                  session — gives them a clear next step (change
                  password / enable MFA) without inventing UI that
                  doesn't match the backend. */}
              <section className="prof-grid-main">
                <article className="dash-panel prof-panel">
                  <div className="dash-panel-head">
                    <div>
                      <span className="dash-eyebrow">Profile</span>
                      <h3>Display name</h3>
                    </div>
                  </div>
                  <form onSubmit={onSaveName} className="sw-form-grid prof-form">
                    <label className="sw-field">
                      <span>Display name</span>
                      <input
                        type="text"
                        value={nameDraft}
                        onChange={(e) => { setNameDraft(e.target.value); setNameError(null); setNameMessage(null); }}
                        placeholder="Your full name"
                        maxLength={255}
                        required
                        autoFocus
                      />
                      <small>This is the name shown across the dashboard and in alerts.</small>
                    </label>
                    {nameError && <div className="auth-error prof-form-msg">{nameError}</div>}
                    {nameMessage && !nameError && <div className="dash-banner-ok prof-form-msg">{nameMessage}</div>}
                    <div className="sw-form-actions">
                      <button type="button" className="sw-button sw-button-quiet" onClick={() => { setNameDraft(profile.full_name); setNameError(null); setNameMessage(null); }} disabled={nameSaving}>Discard</button>
                      <button type="submit" className="sw-button sw-button-primary" disabled={nameSaving || !nameDraft.trim() || nameDraft.trim() === profile.full_name}>{nameSaving ? 'Saving…' : 'Save name'}</button>
                    </div>
                  </form>
                </article>

                <article className="dash-panel prof-panel prof-panel-security">
                  <div className="dash-panel-head">
                    <div>
                      <span className="dash-eyebrow">Security</span>
                      <h3>Sign-in & access</h3>
                    </div>
                  </div>
                  <div className="prof-security-list">
                    <SecurityRow
                      label="Password"
                      value="Last changed at signup"
                      action={<Link className="sw-button sw-button-quiet" to="/settings">Change password</Link>}
                    />
                    <SecurityRow
                      label="Two-factor authentication"
                      value="Not enabled"
                      badge={<span className="prof-badge-warn">Recommended</span>}
                      action={
                        <button type="button" className="sw-button sw-button-quiet" disabled title="Coming soon — 2FA rolls out in a future release">
                          Configure
                        </button>
                      }
                    />
                    <SecurityRow
                      label="Active sessions"
                      value="1 device · this browser"
                      action={<button type="button" className="sw-button sw-button-quiet" onClick={logout}>Sign out</button>}
                    />
                    <SecurityRow
                      label="API tokens"
                      value={showTokens ? `${apiKeys.length} token${apiKeys.length === 1 ? '' : 's'}` : 'Manage personal access tokens'}
                      action={
                        <button type="button" className="sw-button sw-button-quiet" onClick={onToggleTokens}>
                          {showTokens ? 'Hide' : 'Manage'}
                        </button>
                      }
                    />
                  </div>
                  {showTokens && (
                    <div className="prof-tokens-panel">
                      <div className="prof-tokens-header">
                        <p className="prof-tokens-help">
                          Personal access tokens let CLI tools talk to the API without sharing your password.
                          Each token is shown <strong>once</strong> when created - copy it then.
                        </p>
                      </div>
                      {revealedToken && (
                        <div className="prof-token-revealed">
                          <span className="dash-eyebrow">New token (copy now)</span>
                          <code className="prof-token-revealed-value">{revealedToken}</code>
                          <div className="prof-token-revealed-actions">
                            <button
                              type="button"
                              className="sw-button sw-button-quiet"
                              onClick={async () => { try { await navigator.clipboard?.writeText(revealedToken); } catch { /* ignore */ } }}
                            >Copy</button>
                            <button type="button" className="sw-button" onClick={() => setRevealedToken(null)}>I've saved it</button>
                          </div>
                        </div>
                      )}
                      <div className="prof-tokens-create">
                        <input
                          type="text"
                          className="sw-input"
                          placeholder="Token name (e.g. laptop-cli)"
                          value={newTokenName}
                          onChange={(e) => setNewTokenName(e.target.value)}
                          disabled={creatingToken}
                        />
                        <button
                          type="button"
                          className="sw-button sw-button-primary"
                          onClick={() => void onCreateToken()}
                          disabled={creatingToken}
                        >
                          {creatingToken ? 'Creating…' : 'Create token'}
                        </button>
                      </div>
                      {tokensLoading ? (
                        <div className="prof-tokens-empty">Loading tokens…</div>
                      ) : apiKeys.length === 0 ? (
                        <div className="prof-tokens-empty">No tokens yet. Create one above.</div>
                      ) : (
                        <ul className="prof-tokens-list">
                          {apiKeys.map((k) => (
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
                                    onClick={() => void onRevokeToken(k.id)}
                                  >Revoke</button>
                                )}
                              </div>
                            </li>
                          ))}
                        </ul>
                      )}
                    </div>
                  )}
                </article>
              </section>

              <p className="dash-foot-note">Email and tenant ID are tied to your account — change them via your administrator.</p>
            </>
          )}
        </div>
      </main>
      <CommandPalette open={paletteOpen} onClose={() => setPaletteOpen(false)} />
    </div>
  );
}

/* Small inline helper components — kept in the same file so the
   polish work doesn't have to ship yet another module. */

function Stat({ label, value, hint, tone }: { label: string; value: string; hint: string; tone: string }) {
  return (
    <div className={`prof-stat prof-stat-${tone}`}>
      <span className="prof-stat-label">{label}</span>
      <strong className="prof-stat-value">{value}</strong>
      <span className="prof-stat-hint">{hint}</span>
    </div>
  );
}

function SecurityRow({ label, value, action, badge }: { label: string; value: string; action?: React.ReactNode; badge?: React.ReactNode }) {
  return (
    <div className="prof-security-row">
      <div className="prof-security-text">
        <span className="prof-security-label">{label}</span>
        <span className="prof-security-value">{value}</span>
      </div>
      <div className="prof-security-actions">
        {badge}
        {action}
      </div>
    </div>
  );
}
