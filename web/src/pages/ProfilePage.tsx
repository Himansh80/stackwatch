import { FormEvent, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ApiError, clearToken, getToken, me } from '../lib/api';
import ProfileMenu from '../components/ProfileMenu';
import CommandPalette from '../components/CommandPalette';
import TimeWidget from '../components/TimeWidget';
import AppSidebar from '../components/AppSidebar';
import HeroCard from '../components/profile/HeroCard';
import IdentityPanel, { buildIdentityRows, copyToClipboard } from '../components/profile/IdentityPanel';
import WorkspacePanel from '../components/profile/WorkspacePanel';
import NameFormPanel from '../components/profile/NameFormPanel';
import SecurityPanel from '../components/profile/SecurityPanel';
import TokensPanel from '../components/profile/TokensPanel';
import { api } from '../lib/api';

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
 * Role tone — visual treatment of the role badge. super_admin gets
 * the warm "elevated" tone; admin/operator get blue; everything else
 * stays neutral.
 */
function roleTone(role: string): 'admin' | 'operator' | 'viewer' | 'neutral' {
  const lower = role.toLowerCase();
  if (lower === 'super_admin' || lower === 'owner') return 'admin';
  if (lower === 'admin' || lower === 'operator') return 'operator';
  if (lower === 'viewer' || lower === 'member') return 'viewer';
  return 'neutral';
}

/**
 * ProfilePage — top-level route at /profile.
 *
 * Owns: profile fetch, name-edit form state, avatar-upload state,
 * copied-field indicator, tokens panel state, Cmd+K binding,
 * logout. All visible sections delegate to components in
 * `components/profile/`.
 */
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

  // Inline panels
  const [showTokens, setShowTokens] = useState(false);

  // Cmd+K palette open state
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
  const identityRows = buildIdentityRows(profile);
  const nameUnchanged = nameDraft.trim() === profile.full_name;

  return (
    <div className="dash-app">
      <AppSidebar active="profile" onLogout={logout} show={['dashboard', 'profile', 'billing', 'settings']} />
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
          <div className="dash-topbar-center">
            <button
              className="dash-topbar-search"
              onClick={() => setPaletteOpen(true)}
              title="Search & navigate (Cmd+K)"
              aria-label="Open command palette"
            >
              <span className="dash-topbar-search-icon" aria-hidden="true">⌕</span>
              <span className="dash-topbar-search-placeholder">Search & navigate…</span>
              <kbd className="dash-topbar-search-kbd">⌘</kbd>
              <kbd className="dash-topbar-search-kbd">K</kbd>
            </button>
          </div>
          <div className="dash-top-actions">
            <TimeWidget />
            <button
              className="dash-icon-button"
              onClick={() => window.location.reload()}
              aria-label="Refresh page"
              title="Refresh page"
            >↻</button>
            <ProfileMenu
              firstName={(profile.full_name || '').split(' ')[0] || 'there'}
              fullName={profile.full_name}
              tenantName={profile.tenant_name}
              initials={initials}
              avatarUrl={profile.avatar_url}
            />
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
              <HeroCard
                profile={profile}
                initials={initials}
                tone={tone}
                onAvatarUploaded={(dataUrl) => setProfile((p) => ({ ...p, avatar_url: dataUrl }))}
                onAvatarRemoved={() => setProfile((p) => ({ ...p, avatar_url: '' }))}
              />

              <section className="prof-grid-main">
                <IdentityPanel rows={identityRows} onCopy={onCopy} copiedField={copiedField} />
                <WorkspacePanel
                  tenantName={profile.tenant_name}
                  tenantPlan={profile.tenant_plan}
                  tenantSlug={profile.tenant_slug}
                  status={profile.status}
                  role={profile.role}
                  tone={tone}
                />
              </section>

              <section className="prof-grid-main">
                <NameFormPanel
                  draft={nameDraft}
                  onDraftChange={(next) => {
                    setNameDraft(next);
                    setNameError(null);
                    setNameMessage(null);
                  }}
                  onSubmit={onSaveName}
                  onDiscard={() => {
                    setNameDraft(profile.full_name);
                    setNameError(null);
                    setNameMessage(null);
                  }}
                  saving={nameSaving}
                  unchanged={nameUnchanged || !nameDraft.trim()}
                  error={nameError}
                  message={nameMessage}
                />
                <div>
                  <SecurityPanel
                    onSignOut={logout}
                    onToggleTokens={() => setShowTokens((v) => !v)}
                    tokensOpen={showTokens}
                    tokenCount={0}
                  />
                  <TokensPanel open={showTokens} />
                </div>
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
