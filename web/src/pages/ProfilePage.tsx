import { FormEvent, useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { ApiError, api, clearToken, getToken, me } from '../lib/api';
import ProfileMenu from '../components/ProfileMenu';

type Profile = {
  email: string;
  full_name: string;
  role: string;
  status: string;
  user_id: string;
  tenant_id: string;
  tenant_name: string;
  tenant_slug: string;
  tenant_plan: string;
};

const emptyProfile: Profile = {
  email: '',
  full_name: '',
  role: '',
  status: '',
  user_id: '',
  tenant_id: '',
  tenant_name: '',
  tenant_slug: '',
  tenant_plan: '',
};

export default function ProfilePage() {
  const nav = useNavigate();
  const [profile, setProfile] = useState<Profile>(emptyProfile);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  // Name-edit form state
  const [nameDraft, setNameDraft] = useState('');
  const [nameSaving, setNameSaving] = useState(false);
  const [nameMessage, setNameMessage] = useState<string | null>(null);
  const [nameError, setNameError] = useState<string | null>(null);

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
          role: data?.user?.role ?? '',
          status: data?.user?.status ?? '',
          user_id: data?.user?.id ?? '',
          tenant_id: data?.tenant?.id ?? '',
          tenant_name: data?.tenant?.name ?? '',
          tenant_slug: data?.tenant?.slug ?? '',
          tenant_plan: data?.tenant?.plan ?? '',
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

  function logout() {
    clearToken();
    nav('/login');
  }

  return (
    <div className="dash-app">
      <aside className="dash-sidebar">
        <Link className="dash-brand" to="/dashboard">
          <span className="dash-brand-mark">S</span>
          <span><strong>StackWatch</strong><small>Infrastructure control plane</small></span>
        </Link>
        <div className="dash-nav-section"><span className="dash-nav-heading">Workspace</span>
          <Link className="dash-nav-item" to="/dashboard"><span>⌂</span>Overview</Link>
          <Link className="dash-nav-item dash-nav-active" to="/profile"><span>◉</span>Profile</Link>
          <Link className="dash-nav-item" to="/billing"><span>$</span>Billing</Link>
          <Link className="dash-nav-item" to="/settings"><span>⚙</span>Settings</Link>
        </div>
        <div className="dash-nav-section"><span className="dash-nav-heading">Infrastructure</span>
          <Link className="dash-nav-item" to="/proxmox"><span>◈</span>Proxmox</Link>
          <Link className="dash-nav-item" to="/truenas"><span>▤</span>TrueNAS</Link>
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
            </div>
          </div>
          <div className="dash-top-actions">
            <button className="dash-icon-button" onClick={() => window.location.reload()} aria-label="Refresh page" title="Refresh page">↻</button>
            <ProfileMenu firstName={(profile.full_name || '').split(' ')[0] || 'there'} fullName={profile.full_name} tenantName={profile.tenant_name} initials={(profile.full_name || '?').charAt(0).toUpperCase()} />
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
            <section className="dash-panel"><div className="dash-panel-empty"><strong>Loading profile…</strong></div></section>
          ) : (
            <>
              <section className="dash-grid-main">
                <article className="dash-panel">
                  <div className="dash-panel-head">
                    <div><span className="dash-eyebrow">Account</span><h3>Identity</h3></div>
                  </div>
                  <div className="dash-resource-grid">
                    <div className="dash-resource-card">
                      <div className="dash-resource-head"><span className="dash-resource-type">Email</span></div>
                      <strong>{profile.email || '—'}</strong>
                      <div className="dash-resource-meta"><span>user id <b>{profile.user_id ? profile.user_id.slice(0, 8) + '…' : '—'}</b></span></div>
                    </div>
                    <div className="dash-resource-card">
                      <div className="dash-resource-head"><span className="dash-resource-type">Role</span></div>
                      <strong>{profile.role || '—'}</strong>
                      <div className="dash-resource-meta"><span>status <b>{profile.status || '—'}</b></span></div>
                    </div>
                    <div className="dash-resource-card">
                      <div className="dash-resource-head"><span className="dash-resource-type">Workspace</span></div>
                      <strong>{profile.tenant_name || '—'}</strong>
                      <div className="dash-resource-meta"><span>plan <b>{profile.tenant_plan || '—'}</b></span><span>slug <b>{profile.tenant_slug || '—'}</b></span></div>
                    </div>
                    <div className="dash-resource-card">
                      <div className="dash-resource-head"><span className="dash-resource-type">Tenant id</span></div>
                      <strong>{profile.tenant_id ? profile.tenant_id.slice(0, 8) + '…' : '—'}</strong>
                      <div className="dash-resource-meta"><span>plan <b>{profile.tenant_plan || '—'}</b></span></div>
                    </div>
                  </div>
                </article>
                <article className="dash-panel">
                  <div className="dash-panel-head">
                    <div><span className="dash-eyebrow">Profile</span><h3>Edit your name</h3></div>
                  </div>
                  <form onSubmit={onSaveName} className="sw-form-grid" style={{ padding: '18px 20px 20px' }}>
                    <label className="sw-field">
                      <span>Display name</span>
                      <input
                        type="text"
                        value={nameDraft}
                        onChange={(e) => { setNameDraft(e.target.value); setNameError(null); setNameMessage(null); }}
                        placeholder="Your full name"
                        required
                        autoFocus
                      />
                    </label>
                    {nameError && <div className="auth-error" style={{ gridColumn: '1 / -1' }}>{nameError}</div>}
                    {nameMessage && !nameError && <div className="dash-banner-ok" style={{ gridColumn: '1 / -1' }}>{nameMessage}</div>}
                    <div className="sw-form-actions">
                      <button type="button" className="sw-button sw-button-quiet" onClick={() => { setNameDraft(profile.full_name); setNameError(null); setNameMessage(null); }} disabled={nameSaving}>Discard</button>
                      <button type="submit" className="sw-button sw-button-primary" disabled={nameSaving || !nameDraft.trim() || nameDraft.trim() === profile.full_name}>{nameSaving ? 'Saving...' : 'Save name'}</button>
                    </div>
                  </form>
                </article>
              </section>
              <p className="dash-foot-note">Email and tenant id are tied to your account — change them via your administrator.</p>
            </>
          )}
        </div>
      </main>
    </div>
  );
}
