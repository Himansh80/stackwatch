import { FormEvent, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ApiError, clearToken, getToken, me } from '../lib/api';
import { motion, pageEnter } from '../lib/motion';
import HeroCard from '../components/profile/HeroCard';
import IdentityPanel, { buildIdentityRows, copyToClipboard } from '../components/profile/IdentityPanel';
import WorkspacePanel from '../components/profile/WorkspacePanel';
import NameFormPanel from '../components/profile/NameFormPanel';
import SecurityPanel from '../components/profile/SecurityPanel';
import TokensPanel from '../components/profile/TokensPanel';
import { api } from '../lib/api';
import Button from '../components/shared/Button';

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

function roleTone(role: string): 'admin' | 'operator' | 'viewer' | 'neutral' {
  const lower = role.toLowerCase();
  if (lower === 'super_admin' || lower === 'owner') return 'admin';
  if (lower === 'admin' || lower === 'operator') return 'operator';
  if (lower === 'viewer' || lower === 'member') return 'viewer';
  return 'neutral';
}

export default function ProfilePage() {
  const nav = useNavigate();
  const [profile, setProfile] = useState<Profile>(emptyProfile);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [copiedField, setCopiedField] = useState<string | null>(null);

  const [nameDraft, setNameDraft] = useState('');
  const [nameSaving, setNameSaving] = useState(false);
  const [nameMessage, setNameMessage] = useState<string | null>(null);
  const [nameError, setNameError] = useState<string | null>(null);

  const [showTokens, _setShowTokens] = useState(false);

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

  async function onCopy(value: string, field: string) {
    const ok = await copyToClipboard(value);
    if (ok) {
      setCopiedField(field);
      window.setTimeout(() => {
        setCopiedField((current) => (current === field ? null : current));
      }, 1600);
    }
  }

  function onLogout() {
    clearToken();
    nav('/login');
  }

  const tone = roleTone(profile.role);
  const initials = profile.full_name
    ?.split(' ')
    .map((w) => w[0])
    .join('')
    .toUpperCase()
    .slice(0, 2);

  const identityRows = buildIdentityRows(profile);
  const nameUnchanged = nameDraft === profile.full_name;

  return (
        <motion.div
          className="dash-content"
          initial="hidden"
          animate="show"
          variants={pageEnter}
        >
          {error && (
            <div className="dash-error">
              <strong>Could not load your profile</strong>
              <span>{error}</span>
              <Button variant="primary" onClick={() => window.location.reload()}>
                Retry
              </Button>
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
              <div className="prof-grid">
                <IdentityPanel rows={identityRows} onCopy={onCopy} copiedField={copiedField} />
                <WorkspacePanel
                  tenantName={profile.tenant_name}
                  tenantPlan={profile.tenant_plan}
                  tenantSlug={profile.tenant_slug}
                  status={profile.status}
                  role={profile.role}
                  tone={tone}
                />
              </div>
              <NameFormPanel
                draft={nameDraft}
                onDraftChange={setNameDraft}
                onSubmit={onSaveName}
                onDiscard={() => setNameDraft(profile.full_name)}
                saving={nameSaving}
                unchanged={nameUnchanged}
                error={nameError}
                message={nameMessage}
              />
              <SecurityPanel
                onSignOut={onLogout}
                onToggleTokens={() => _setShowTokens((v) => !v)}
                tokensOpen={showTokens}
                tokenCount={0}
              />
              {showTokens && <TokensPanel open={showTokens} />}
            </>
          )}
        </motion.div>
  );
}