import { FormEvent, useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { ApiError, api, clearToken, getToken, setToken } from '../lib/api';
import ProfileMenu from '../components/ProfileMenu';
import PasswordInput from '../components/PasswordInput';
import { friendlyPasswordMessage } from '../lib/password';

type Tenant = {
  id: string;
  name: string;
  slug: string;
  plan: string;
  status: string;
};

export default function SettingsPage() {
  const nav = useNavigate();
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  // Rename workspace state
  const [nameDraft, setNameDraft] = useState('');
  const [nameSaving, setNameSaving] = useState(false);
  const [nameMessage, setNameMessage] = useState<string | null>(null);
  const [nameError, setNameError] = useState<string | null>(null);

  // Change password state
  const [oldPw, setOldPw] = useState('');
  const [newPw, setNewPw] = useState('');
  const [newPw2, setNewPw2] = useState('');
  const [pwSaving, setPwSaving] = useState(false);
  const [pwMessage, setPwMessage] = useState<string | null>(null);
  const [pwError, setPwError] = useState<string | null>(null);
  const [pwErrorCode, setPwErrorCode] = useState<string | undefined>(undefined);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      if (!getToken()) {
        setError('You are not signed in.');
        setLoading(false);
        return;
      }
      try {
        const me = await api<{ tenant?: Tenant }>('GET', '/api/v1/auth/me');
        if (cancelled) return;
        setTenant(me.tenant ?? null);
      } catch (cause) {
        if (cancelled) return;
        if (cause instanceof ApiError) {
          setError(cause.friendlyMessage);
        } else {
          setError(cause instanceof Error ? cause.message : 'Unable to load your settings.');
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

  useEffect(() => {
    setNameDraft(tenant?.name || '');
  }, [tenant?.name]);

  async function onSaveName(event: FormEvent) {
    event.preventDefault();
    setNameMessage(null);
    setNameError(null);
    const trimmed = nameDraft.trim();
    if (!trimmed) {
      setNameError('Please enter a workspace name.');
      return;
    }
    if (!tenant || trimmed === tenant.name) {
      setNameMessage('No changes to save.');
      return;
    }
    setNameSaving(true);
    try {
      const updated = await api<Tenant>('PATCH', `/api/v1/tenants/${tenant.id}`, { name: trimmed });
      setTenant(updated);
      setNameMessage('Saved.');
    } catch (cause) {
      if (cause instanceof ApiError) {
        setNameError(cause.friendlyMessage);
      } else {
        setNameError(cause instanceof Error ? cause.message : 'Could not save your workspace name.');
      }
    } finally {
      setNameSaving(false);
    }
  }

  async function onChangePassword(event: FormEvent) {
    event.preventDefault();
    setPwMessage(null);
    setPwError(null);
    setPwErrorCode(undefined);
    if (!oldPw) {
      setPwError('Please enter your current password.');
      return;
    }
    if (newPw.length < 10) {
      setPwError('New password must be at least 10 characters.');
      return;
    }
    if (newPw !== newPw2) {
      setPwError('New password and confirmation do not match.');
      return;
    }
    setPwSaving(true);
    try {
      const body = await api<{ ok: boolean; token?: string }>(
        'POST',
        '/api/v1/auth/change-password',
        { old_password: oldPw, new_password: newPw },
      );
      // Backend returns a fresh token so we can keep this device signed in.
      if (body?.token) {
        setToken(body.token);
      }
      setPwMessage('Password updated.');
      setOldPw('');
      setNewPw('');
      setNewPw2('');
    } catch (cause) {
      if (cause instanceof ApiError) {
        const code = (cause as ApiError & { code?: string }).code;
        if (code?.startsWith('password_')) {
          setPwErrorCode(code);
          setPwError(friendlyPasswordMessage(code, cause.friendlyMessage));
        } else {
          setPwError(cause.friendlyMessage);
        }
      } else {
        setPwError(cause instanceof Error ? cause.message : 'Could not change your password.');
      }
    } finally {
      setPwSaving(false);
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
          <Link className="dash-nav-item" to="/profile"><span>◉</span>Profile</Link>
          <Link className="dash-nav-item" to="/billing"><span>$</span>Billing</Link>
          <Link className="dash-nav-item dash-nav-active" to="/settings"><span>⚙</span>Settings</Link>
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
            <span className="dash-greeting-eyebrow">Settings</span>
            <div className="dash-greeting-row">
              <strong className="dash-greeting-text">{tenant?.name || 'Workspace'}</strong>
            </div>
          </div>
          <div className="dash-top-actions">
            <button className="dash-icon-button" onClick={() => window.location.reload()} aria-label="Refresh page" title="Refresh page">↻</button>
            <ProfileMenu firstName={tenant?.name ? tenant.name.split(' ')[0] : 'there'} fullName={tenant?.name || ''} tenantName={tenant?.name || 'Workspace'} initials={(tenant?.name || '?').charAt(0).toUpperCase()} />
          </div>
        </header>
        <div className="dash-content">
          {error && (
            <div className="dash-error">
              <strong>Could not load your settings</strong>
              <span>{error}</span>
              <button onClick={() => window.location.reload()}>Retry</button>
            </div>
          )}
          {loading ? (
            <section className="dash-panel"><div className="dash-panel-empty"><strong>Loading settings…</strong></div></section>
          ) : (
            <>
              <section className="dash-grid-main">
                <article className="dash-panel">
                  <div className="dash-panel-head">
                    <div><span className="dash-eyebrow">Workspace</span><h3>Rename workspace</h3></div>
                  </div>
                  <form onSubmit={onSaveName} className="sw-form-grid" style={{ padding: '18px 20px 20px' }}>
                    <label className="sw-field">
                      <span>Workspace name</span>
                      <input
                        type="text"
                        value={nameDraft}
                        onChange={(e) => { setNameDraft(e.target.value); setNameError(null); setNameMessage(null); }}
                        placeholder="Your team or project name"
                        required
                      />
                    </label>
                    {nameError && <div className="auth-error" style={{ gridColumn: '1 / -1' }}>{nameError}</div>}
                    {nameMessage && !nameError && <div className="dash-banner-ok" style={{ gridColumn: '1 / -1' }}>{nameMessage}</div>}
                    <div className="sw-form-actions">
                      <button type="button" className="sw-button sw-button-quiet" onClick={() => { setNameDraft(tenant?.name || ''); setNameError(null); setNameMessage(null); }} disabled={nameSaving}>Discard</button>
                      <button type="submit" className="sw-button sw-button-primary" disabled={nameSaving || !nameDraft.trim() || nameDraft.trim() === (tenant?.name || '')}>{nameSaving ? 'Saving...' : 'Save name'}</button>
                    </div>
                  </form>
                </article>

                <article className="dash-panel">
                  <div className="dash-panel-head">
                    <div><span className="dash-eyebrow">Security</span><h3>Change password</h3></div>
                  </div>
                  <form onSubmit={onChangePassword} className="sw-form-grid" style={{ padding: '18px 20px 20px' }}>
                    <label className="sw-field">
                      <span>Current password</span>
                      <input type="password" value={oldPw} onChange={(e) => { setOldPw(e.target.value); setPwError(null); setPwErrorCode(undefined); setPwMessage(null); }} autoComplete="current-password" required />
                    </label>
                    <label className="sw-field">
                      <span>New password</span>
                      <PasswordInput
                        value={newPw}
                        onChange={(v) => { setNewPw(v); setPwError(null); setPwErrorCode(undefined); setPwMessage(null); }}
                        errorCode={pwErrorCode}
                        confirmOldPassword={oldPw}
                        autoComplete="new-password"
                        required
                      />
                      <small>Minimum 10 characters. Must differ from your current password.</small>
                    </label>
                    <label className="sw-field">
                      <span>Confirm new password</span>
                      <input type="password" value={newPw2} onChange={(e) => { setNewPw2(e.target.value); setPwError(null); setPwMessage(null); }} autoComplete="new-password" required minLength={10} />
                    </label>
                    {pwError && <div className="auth-error" style={{ gridColumn: '1 / -1' }}>{pwError}</div>}
                    {pwMessage && !pwError && <div className="dash-banner-ok" style={{ gridColumn: '1 / -1' }}>{pwMessage}</div>}
                    <div className="sw-form-actions">
                      <button type="button" className="sw-button sw-button-quiet" onClick={() => { setOldPw(''); setNewPw(''); setNewPw2(''); setPwError(null); setPwErrorCode(undefined); setPwMessage(null); }} disabled={pwSaving}>Discard</button>
                      <button type="submit" className="sw-button sw-button-primary" disabled={pwSaving || !oldPw || !newPw || !newPw2}>{pwSaving ? 'Saving...' : 'Update password'}</button>
                    </div>
                  </form>
                </article>
              </section>
              <p className="dash-foot-note">Changing your password invalidates this device&apos;s session. You&apos;ll be sent back to the sign-in screen.</p>
            </>
          )}
        </div>
      </main>
    </div>
  );
}
